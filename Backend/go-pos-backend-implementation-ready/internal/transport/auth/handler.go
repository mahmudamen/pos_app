package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pinPattern = regexp.MustCompile(`^[0-9]{4,8}$`)

type Handler struct {
	pool        *pgxpool.Pool
	tokens      security.TokenManager
	cost        int
	maxSessions int
}

func NewHandler(pool *pgxpool.Pool, cfg config.Config) *Handler {
	return &Handler{
		pool: pool,
		tokens: security.TokenManager{
			Issuer: cfg.JWTIssuer, AccessSecret: []byte(cfg.JWTAccessSecret), RefreshSecret: []byte(cfg.JWTRefreshSecret),
			AccessTTL: cfg.JWTAccessTTL, RefreshTTL: cfg.JWTRefreshTTL,
		},
		cost:        cfg.BcryptCost,
		maxSessions: cfg.MaxSessionsPerUser,
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
	defer tx.Rollback(ctx)
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
// state local.
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
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var pinHash string
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(manager_pin_hash, '') FROM users
		WHERE id = $1::uuid AND tenant_id = $2::uuid`, claims.UserID, claims.TenantID).Scan(&pinHash)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "user_not_found", "user no longer exists")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit pin check")
		return
	}
	valid := pinHash != "" && security.CheckPassword(pinHash, request.Pin)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"valid": valid, "has_pin": pinHash != ""}, "meta": gin.H{"request_id": c.GetString("request_id")}})
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
	defer tx.Rollback(ctx)
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
	defer tx.Rollback(ctx)

	var tenantID, businessType, countryCode, currencyCode, defaultLanguage string
	var plan string
	var trialEndsAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT id, business_type, country_code, currency_code, default_language, plan, trial_ends_at
		FROM tenants WHERE id::text = $1 OR slug = $1`, request.TenantID).
		Scan(&tenantID, &businessType, &countryCode, &currencyCode, &defaultLanguage, &plan, &trialEndsAt)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
		return
	}
	if plan == "trial" && trialEndsAt != nil && trialEndsAt.Before(time.Now()) {
		writeError(c, http.StatusForbidden, "trial_expired", "the 15-day trial has ended; renew your plan to continue")
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
		"user": gin.H{"id": userID, "display_name": displayName, "role": role, "account_type": accountType},
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
	defer tx.Rollback(ctx)
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
