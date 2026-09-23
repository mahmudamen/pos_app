package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/example/pos-api/internal/transport/access"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pinPattern = regexp.MustCompile(`^[0-9]{4,8}$`)

// PinLockThreshold and PinLockWindow mirror ma_pos_base's smart lockout:
// five consecutive failed PIN attempts lock the credential for 15 minutes.
const (
	PinLockThreshold = 5
	PinLockWindow    = 15 * time.Minute
)

type Handler struct {
	pool            *pgxpool.Pool
	tokens          security.TokenManager
	cost            int
	maxSessions     int
	cfg             config.Config
	registerLimiter identity.Counter
}

func NewHandler(pool *pgxpool.Pool, cfg config.Config) *Handler {
	return &Handler{
		pool: pool,
		tokens: security.TokenManager{
			Issuer: cfg.JWTIssuer, AccessSecret: []byte(cfg.JWTAccessSecret), RefreshSecret: []byte(cfg.JWTRefreshSecret),
			AccessTTL: cfg.JWTAccessTTL, RefreshTTL: cfg.JWTRefreshTTL,
		},
		cost:            cfg.BcryptCost,
		maxSessions:     cfg.MaxSessionsPerUser,
		cfg:             cfg,
		registerLimiter: identity.NewMemoryCounter("register:"),
	}
}

func (h *Handler) Tokens() security.TokenManager {
	return h.tokens
}

type loginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required"`
	TenantID   string `json:"tenant_id" binding:"required"`
	DeviceID   string `json:"device_id" binding:"required"`
	DeviceName string `json:"device_name" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.POST("/auth/register", h.Signup)
	router.POST("/auth/login", h.Login())
	router.POST("/auth/refresh", h.Refresh())
	router.POST("/auth/logout", h.Logout())
	router.POST("/auth/set-pin", h.SetPin())
	router.POST("/auth/verify-pin", h.VerifyPin())
	router.POST("/auth/unlock-pin", h.UnlockPin)
}

func (h *Handler) Login() gin.HandlerFunc {
	return h.login
}

func (h *Handler) Refresh() gin.HandlerFunc {
	return h.refresh
}

func (h *Handler) Logout() gin.HandlerFunc {
	return h.logout
}

type pinRequest struct {
	Pin string `json:"pin" binding:"required"`
}

func validPIN(pin string) bool {
	return pinPattern.MatchString(pin)
}

func (h *Handler) SetPin() gin.HandlerFunc {
	return h.setPin
}

func (h *Handler) VerifyPin() gin.HandlerFunc {
	return h.verifyPin
}

func (h *Handler) UnlockPin(c *gin.Context) {
	h.unlockPin(c)
}

// lockAgain returns the UTC expiry of the current lockout, or nil when the
// credential is usable.
func lockStillActive(lockedUntil *time.Time) *time.Time {
	if lockedUntil == nil || lockedUntil.Before(time.Now()) {
		return nil
	}
	expiry := lockedUntil.UTC()
	return &expiry
}

// pinState is the mutable per-user PIN credential state.
type pinState struct {
	Hash        string
	Failed      int
	LockedUntil *time.Time
	HasPin      bool
}

// loadPinState reads the acting user's manager PIN credential state within the
// caller's transaction (tenant context assumed active).
func loadPinState(ctx context.Context, tx pgx.Tx, tenantID, userID string) (pinState, error) {
	var state pinState
	err := tx.QueryRow(ctx,
		`SELECT COALESCE(manager_pin_hash, ''), pos_pin_failed_attempts, pos_pin_locked_until
		 FROM users WHERE id = $1::uuid AND tenant_id = $2::uuid`,
		userID, tenantID).Scan(&state.Hash, &state.Failed, &state.LockedUntil)
	if err != nil {
		return pinState{}, err
	}
	state.HasPin = state.Hash != ""
	return state, nil
}

// setPin stores the acting manager's bcrypt-hashed PIN. Cashiers cannot set
// one (they have no refund rights to gate anyway).
func (h *Handler) setPin(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if claims.Role != "manager" && claims.Role != "owner" && claims.Role != "saas_admin" {
		writeError(c, http.StatusForbidden, "permission_denied", "only managers may set a PIN")
		return
	}
	var request pinRequest
	if err := c.ShouldBindJSON(&request); err != nil || !validPIN(request.Pin) {
		writeError(c, http.StatusBadRequest, "validation_error", "pin must be 4-8 digits")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	hash, err := security.HashPassword(request.Pin, h.cost)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to hash pin")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	result, err := tx.Exec(ctx, `
		UPDATE users SET manager_pin_hash = $1, updated_at = now()
		WHERE id = $2::uuid AND tenant_id = $3::uuid`,
		hash, claims.UserID, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save pin")
		return
	}
	if result.RowsAffected() != 1 {
		writeError(c, http.StatusUnauthorized, "user_not_found", "user no longer exists")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit pin")
		return
	}
	c.Status(http.StatusNoContent)
}

// verifyPin reports whether the submitted PIN matches the acting user's stored
// manager PIN. Returns valid:false (never 401) so the client keeps the derived
// state local. ma_pos_base parity: five consecutive failures lock the pin for
// 15 minutes (423 pin_locked); a success resets the counter.
func (h *Handler) verifyPin(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	var request pinRequest
	if err := c.ShouldBindJSON(&request); err != nil || !validPIN(request.Pin) {
		writeError(c, http.StatusBadRequest, "validation_error", "pin must be 4-8 digits")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	state, err := loadPinState(ctx, tx, claims.TenantID, claims.UserID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "user_not_found", "user no longer exists")
		return
	}
	if expiry := lockStillActive(state.LockedUntil); expiry != nil {
		if err = tx.Commit(ctx); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit pin check")
			return
		}
		writeError(c, http.StatusLocked, "pin_locked", "too many failed PIN attempts; try again later")
		return
	}

	if state.Hash != "" && security.CheckPassword(state.Hash, request.Pin) {
		if _, err = tx.Exec(ctx, `
			UPDATE users SET pos_pin_failed_attempts = 0, pos_pin_locked_until = NULL,
				pos_pin_last_validated_at = now() WHERE id = $1::uuid`,
			claims.UserID); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to update pin state")
			return
		}
		if err = tx.Commit(ctx); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit pin check")
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"valid": true, "has_pin": true}, "meta": gin.H{"request_id": c.GetString("request_id")}})
		return
	}

	// Wrong PIN (or no PIN set): increment the counter, locking at the threshold.
	failed := state.Failed + 1
	lockedUntil := (*time.Time)(nil)
	if failed >= PinLockThreshold {
		now := time.Now()
		until := now.Add(PinLockWindow)
		lockedUntil = &until
		failed = 0
	}
	if _, err = tx.Exec(ctx, `
		UPDATE users SET pos_pin_failed_attempts = $2, pos_pin_locked_until = $3
		WHERE id = $1::uuid`,
		claims.UserID, failed, lockedUntil); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update pin state")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit pin check")
		return
	}
	if lockedUntil != nil {
		writeError(c, http.StatusLocked, "pin_locked", "too many failed PIN attempts; the PIN is locked for 15 minutes")
		return
	}
	attemptsLeft := PinLockThreshold - failed
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"valid": false, "has_pin": state.HasPin, "attempts_left": attemptsLeft}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

// unlockPin clears a user's PIN lockout state (admin/manager action, ma_pos_base
// parity "Unlock"). Managers may only unlock cashiers, mirroring user creation.
func (h *Handler) unlockPin(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if claims.Role != "owner" && claims.Role != "manager" && claims.Role != "saas_admin" {
		writeError(c, http.StatusForbidden, "permission_denied", "owner or manager role is required")
		return
	}
	var request struct {
		UserID string `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "user_id is required")
		return
	}
	targetID, err := uuid.Parse(request.UserID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "user_id is invalid")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var targetRole string
	if err = tx.QueryRow(ctx, `SELECT role FROM users WHERE id = $1 AND tenant_id = $2`,
		targetID.String(), claims.TenantID).Scan(&targetRole); err != nil {
		writeError(c, http.StatusNotFound, "user_not_found", "user not found")
		return
	}
	if claims.Role == "manager" && targetRole != "cashier" {
		writeError(c, http.StatusForbidden, "permission_denied", "managers may only manage cashiers")
		return
	}
	if _, err = tx.Exec(ctx, `
		UPDATE users SET pos_pin_failed_attempts = 0, pos_pin_locked_until = NULL
		WHERE id = $1::uuid`, targetID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to unlock pin")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit pin unlock")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) logout(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	result, err := tx.Exec(ctx, `
		UPDATE sessions SET revoked_at = now()
		WHERE id = $1::uuid AND tenant_id = $2::uuid AND user_id = $3::uuid AND revoked_at IS NULL`,
		claims.SessionID, claims.TenantID, claims.UserID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to revoke session")
		return
	}
	if result.RowsAffected() != 1 {
		writeError(c, http.StatusUnauthorized, "invalid_session", "session is no longer active")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit logout")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) authenticate(c *gin.Context) (security.Claims, bool) {
	header := c.GetHeader("Authorization")
	if len(header) < len("Bearer ") || header[:len("Bearer ")] != "Bearer " {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is required")
		return security.Claims{}, false
	}
	claims, err := h.tokens.Parse(header[len("Bearer "):], security.AccessToken)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return security.Claims{}, false
	}
	return claims, true
}

func (h *Handler) login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid login request")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var tenantID, businessType, countryCode, currencyCode, defaultLanguage, tenantStatus, ownerAccountID string
	var plan string
	var trialEndsAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, business_type, country_code, currency_code, default_language, plan, trial_ends_at, status,
		       COALESCE(owner_account_id::text, '')
		FROM tenants WHERE id::text = $1 OR slug = $1`, request.TenantID).
		Scan(&tenantID, &businessType, &countryCode, &currencyCode, &defaultLanguage, &plan, &trialEndsAt,
			&tenantStatus, &ownerAccountID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
		return
	}
	var dbNow time.Time
	if err := tx.QueryRow(ctx, `SELECT now()`).Scan(&dbNow); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to read server time")
		return
	}
	dbNow = dbNow.UTC()
	if tenantStatus == "suspended" || tenantStatus == "closed" {
		writeError(c, http.StatusForbidden, "organization_suspended", "this organization has been suspended; contact support")
		return
	}
	if tenantStatus == "disabled" {
		writeError(c, http.StatusForbidden, "organization_stopped", "this organization has been stopped; contact support")
		return
	}
	if ownerAccountID != "" {
		var accountStatus string
		if err := tx.QueryRow(ctx, `SELECT status FROM accounts WHERE id = $1::uuid`, ownerAccountID).Scan(&accountStatus); err == nil {
			if accountStatus == "suspended" || accountStatus == "disabled" {
				writeError(c, http.StatusForbidden, "account_suspended", "this account has been suspended; contact support")
				return
			}
		}
	}
	if plan == "trial" && trialEndsAt != nil && trialEndsAt.Before(dbNow) {
		writeError(c, http.StatusForbidden, "trial_expired", "the free trial has ended; renew your plan to continue")
		return
	}
	if plan == "trial" && trialEndsAt == nil {
		writeError(c, http.StatusForbidden, "trial_pending", "your trial is not activated yet; verify your email/phone to start it")
		return
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var userID, passwordHash, displayName, role, accountType string
	err = tx.QueryRow(ctx, `
		SELECT id, password_hash, display_name, role, account_type FROM users
		WHERE tenant_id = $1 AND lower(email) = lower($2) AND is_active`, tenantID, request.Email).Scan(&userID, &passwordHash, &displayName, &role, &accountType)
	if err != nil || !security.CheckPassword(passwordHash, request.Password) {
		writeError(c, http.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
		return
	}

	permissions := access.ResolveFromDB(ctx, tx, tenantID, userID, role)

	var deviceUUID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO devices (id, tenant_id, client_device_id, name, last_seen_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (tenant_id, client_device_id)
		DO UPDATE SET name = EXCLUDED.name, last_seen_at = now(), revoked_at = NULL
		RETURNING id`, uuid.New(), tenantID, request.DeviceID, request.DeviceName).Scan(&deviceUUID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to register device")
		return
	}

	sessionID := uuid.New()
	refreshToken, err := h.tokens.Issue(time.Now(), security.RefreshToken, tenantID, userID, deviceUUID.String(), sessionID.String())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to issue session")
		return
	}
	refreshHash := hashToken(refreshToken)
	_, err = tx.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, user_id, device_id, refresh_token_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5, now() + ($6::bigint * interval '1 second'))`,
		sessionID, tenantID, userID, deviceUUID, refreshHash, int64(h.tokens.RefreshTTL.Seconds()))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create session")
		return
	}
	// AUTH-010: cap active sessions per user, evicting the oldest (explicit ordering).
	if h.maxSessions > 0 {
		if _, err = tx.Exec(ctx, `
			UPDATE sessions AS s SET revoked_at = now()
			FROM (
				SELECT id FROM sessions
				WHERE tenant_id = $1 AND user_id = $2 AND revoked_at IS NULL
				ORDER BY created_at DESC, id DESC
				OFFSET $3
			) AS evicted
			WHERE s.id = evicted.id`,
			tenantID, userID, h.maxSessions); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to enforce session limit")
			return
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, userID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update login state")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit session")
		return
	}

	accessToken, err := h.tokens.IssueWithRole(time.Now(), security.AccessToken, tenantID, userID, deviceUUID.String(), sessionID.String(), role)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to issue access token")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"access_token": accessToken, "refresh_token": refreshToken, "expires_in": int(h.tokens.AccessTTL.Seconds()),
		"user": gin.H{"id": userID, "display_name": displayName, "role": role, "account_type": accountType, "permissions": permissions},
		"tenant": gin.H{
			"id": tenantID, "business_type": businessType,
			"country_code": countryCode, "currency_code": currencyCode, "default_language": defaultLanguage,
			"plan": plan, "trial_ends_at": formatTrialEndsAt(trialEndsAt),
		},
	}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) refresh(c *gin.Context) {
	var request refreshRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid refresh request")
		return
	}
	claims, err := h.tokens.Parse(request.RefreshToken, security.RefreshToken)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "invalid_refresh", "refresh token is invalid or expired")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	newRefresh, err := h.tokens.Issue(time.Now(), security.RefreshToken, claims.TenantID, claims.UserID, claims.DeviceID, claims.SessionID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to issue refresh token")
		return
	}
	result, err := tx.Exec(ctx, `
		UPDATE sessions SET refresh_token_hash = $1, last_used_at = now()
		WHERE id = $2::uuid AND tenant_id = $3::uuid AND user_id = $4::uuid AND device_id = $5::uuid
		AND refresh_token_hash = $6 AND revoked_at IS NULL AND expires_at > now()`,
		hashToken(newRefresh), claims.SessionID, claims.TenantID, claims.UserID, claims.DeviceID, hashToken(request.RefreshToken))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to rotate refresh token")
		return
	}
	if result.RowsAffected() != 1 {
		// AUTH-007 replay detection: the presented refresh token has already
		// been rotated, so all sessions for this user+device are compromised.
		_, revokeErr := tx.Exec(ctx, `
			UPDATE sessions SET revoked_at = now()
			WHERE tenant_id = $1::uuid AND user_id = $2::uuid AND device_id = $3::uuid
			AND revoked_at IS NULL`, claims.TenantID, claims.UserID, claims.DeviceID)
		if revokeErr != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to revoke reused session")
			return
		}
		if err = tx.Commit(ctx); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit session revocation")
			return
		}
		writeError(c, http.StatusUnauthorized, "invalid_refresh", "refresh token is invalid or already used")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit refresh")
		return
	}
	access, err := h.tokens.Issue(time.Now(), security.AccessToken, claims.TenantID, claims.UserID, claims.DeviceID, claims.SessionID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to issue access token")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"access_token": access, "refresh_token": newRefresh, "expires_in": int(h.tokens.AccessTTL.Seconds())}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}

func formatTrialEndsAt(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
