package identitytransport

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler serves the identity self-service endpoints: email verification,
// email change, phone OTP, installations, and logout-all. These mutate the
// global account, so only the tenant owner (or SaaS admin) may call them.
type Handler struct {
	pool     *pgxpool.Pool
	tokens   security.TokenManager
	cfg      config.Config
	store    *identity.AccountStore
	audit    identity.AuditRecorder
	installs *identity.InstallationStore
	counters identity.Counter
	mailer   identity.Mailer
	sms      identity.SMSSender
	outbox   *identity.DevOutbox
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager, cfg config.Config) *Handler {
	audit := identity.NewAuditRecorder()
	outbox := identity.NewDevOutbox()
	return &Handler{
		pool:     pool,
		tokens:   tokens,
		cfg:      cfg,
		store:    identity.NewAccountStore(audit),
		audit:    audit,
		installs: identity.NewInstallationStore(),
		counters: identity.NewMemoryCounter("identity:"),
		mailer:   identity.NewMailer(cfg.Trial.Mailer, outbox),
		sms:      identity.NewSMSSender(cfg.Trial.SMSSender, outbox),
		outbox:   outbox,
	}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	auth := httptransport.RequireAccessToken(h.tokens)
	identityV1 := router.Group("/identity", auth, h.requireAccountManager)
	{
		identityV1.GET("", h.accountSummary)
		identityV1.POST("/email/verify", h.emailVerify)
		identityV1.POST("/email/resend", h.emailResend)
		identityV1.POST("/email/change", h.emailChange)
		identityV1.POST("/email/verify_change", h.emailVerifyChange)
		identityV1.POST("/phone/send", h.phoneSend)
		identityV1.POST("/phone/verify", h.phoneVerify)
		identityV1.GET("/installations", h.listInstallations)
		identityV1.POST("/installations/:id/revoke", h.revokeInstallation)
	}
	if h.cfg.Trial.Mailer == "noop" {
		debug := router.Group("/debug", httptransport.RequireAccessToken(h.tokens), h.requireAccountManager)
		debug.GET("/email-outbox", h.debugEmailOutbox)
	}
}

// requireAccountManager restricts identity self-service to the tenant owner
// (or SaaS admin). All account-level mutations are audited.
func (h *Handler) requireAccountManager(c *gin.Context) {
	claims, _ := httptransport.Claims(c)
	if claims.Role != "owner" && claims.Role != "saas_admin" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": gin.H{
			"code": "permission_denied", "message": "only the account owner can manage identity", "request_id": c.GetString("request_id"),
		}})
		return
	}
	c.Next()
}

// resolveAccount returns the global account for the caller's tenant. Identity
// is keyed on accounts; tenants link via tenants.owner_account_id (034).
func (h *Handler) resolveAccount(ctx context.Context, q identity.Querier, tenantID string) (*identity.Account, error) {
	var accountID string
	err := q.QueryRow(ctx,
		`SELECT owner_account_id::text FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("tenant_not_found")
	}
	if err != nil {
		return nil, err
	}
	if accountID == "" {
		return nil, errors.New("account_not_linked")
	}
	account, err := h.store.GetByID(ctx, q, accountID)
	if err != nil {
		return nil, err
	}
	return &account, nil
}

func (h *Handler) accountSummary(c *gin.Context) {
	claims, _ := httptransport.Claims(c)
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
	account, err := h.resolveAccount(ctx, tx, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	installations, err := h.installs.ListByAccount(ctx, tx, account.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load installations")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"account_id":     account.ID,
			"email":          account.PrimaryEmail,
			"email_verified": account.IsEmailVerified(),
			"phone":          account.PhoneE164,
			"phone_verified": account.IsPhoneVerified(),
			"account_status": account.Status,
			"installations":  installationPayloads(installations),
			"org_count_hint": "",
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func installationPayloads(installations []identity.Installation) []gin.H {
	out := make([]gin.H, 0, len(installations))
	for _, inst := range installations {
		out = append(out, gin.H{
			"id":            inst.InstallationPublicID,
			"platform":      inst.Platform,
			"app_version":   inst.AppVersion,
			"first_seen_at": inst.FirstSeenAt.UTC().Format("2006-01-02T15:04:05Z"),
			"last_seen_at":  inst.LastSeenAt.UTC().Format("2006-01-02T15:04:05Z"),
			"integrity":     inst.DeviceIntegrityStatus,
			"risk_level":    inst.RiskLevel,
		})
	}
	return out
}

func (h *Handler) emailVerify(c *gin.Context) {
	var req struct {
		VerificationToken string `json:"verification_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.VerificationToken == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "verification_token is required")
		return
	}
	claims, _ := httptransport.Claims(c)
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
	account, err := h.resolveAccount(ctx, tx, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	payload, err := identity.ConsumeEmailToken(ctx, tx, req.VerificationToken, identity.TokenPurposeVerify, h.cfg.Trial.EmailTokenTTL)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_token", "the verification link is invalid or expired")
		return
	}
	if payload.AccountID != account.ID {
		writeError(c, http.StatusBadRequest, "invalid_token", "the verification link does not match this account")
		return
	}
	if err := h.store.MarkEmailVerified(ctx, tx, account.ID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify email")
		return
	}
	// A pending trial self-activates as soon as its verification requirements
	// are satisfied (safe no-op without a pending entitlement).
	if refreshed, err := h.store.GetByID(ctx, tx, account.ID); err == nil {
		h.activatePendingTrial(ctx, tx, claims.TenantID, &refreshed)
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify email")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"account_id": account.ID, "email": account.PrimaryEmail, "email_verified": true},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

// activatePendingTrial flips a pending entitlement to active once the account
// has satisfied every configured requirement. It never errors the request: a
// trial still under review, or with an unmet sibling requirement, simply stays
// pending and the admin surface remains the backstop.
func (h *Handler) activatePendingTrial(ctx context.Context, tx pgx.Tx, tenantID string, account *identity.Account) {
	policy, err := identity.LoadPolicy(ctx, tx, h.cfg.Trial)
	if err != nil {
		return
	}
	_, _ = identity.NewTrialStore().ActivatePending(ctx, tx, tenantID, account.ID, *account, policy)
}

func (h *Handler) emailResend(c *gin.Context) {
	claims, _ := httptransport.Claims(c)
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
	account, err := h.resolveAccount(ctx, tx, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if !h.counters.Take(ctx, "email:resend:"+account.ID, 3, h.cfg.Trial.EmailTokenTTL) {
		writeError(c, http.StatusTooManyRequests, "rate_limited", "resend requests too frequent; try again later")
		return
	}
	if account.IsEmailVerified() {
		writeError(c, http.StatusConflict, "already_verified", "email is already verified")
		return
	}
	plain, err := identity.CreateEmailToken(ctx, tx, account.ID, identity.TokenPurposeVerify, "", h.cfg.Trial.EmailTokenTTL)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create verification token")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to issue verification")
		return
	}
	err = h.mailer.Send(ctx, identity.EmailMessage{
		To:      account.PrimaryEmail,
		Subject: "Verify your email",
		Text:    fmt.Sprintf("Your verification code is:\n%s\nIt expires in %s.", plain, h.cfg.Trial.EmailTokenTTL),
	})
	if err != nil {
		writeError(c, http.StatusBadGateway, "delivery_failed", "verification email could not be sent")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"sent": true}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) emailChange(c *gin.Context) {
	var req struct {
		NewEmail string `json:"new_email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "new_email and password are required")
		return
	}
	newEmail := identity.NormalizeEmail(req.NewEmail)
	if newEmail == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "new_email is not a valid email")
		return
	}
	if req.Password == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "password is required")
		return
	}
	claims, _ := httptransport.Claims(c)
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
	account, err := h.resolveAccount(ctx, tx, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	var passwordHash string
	if err := tx.QueryRow(ctx, `
		SELECT u.password_hash FROM users u JOIN tenants t ON u.id = t.owner_user_id
		WHERE t.id = $1::uuid AND u.status = 'active'`, claims.TenantID).Scan(&passwordHash); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load owner credentials")
		return
	}
	passwordOK := security.CheckPassword(passwordHash, req.Password)
	if !passwordOK {
		writeError(c, http.StatusUnauthorized, "invalid_password", "current password is incorrect")
		return
	}
	if !(account.Status == "active" || account.Status == "pending") {
		writeError(c, http.StatusForbidden, "account_suspended", "suspended accounts cannot change their email")
		return
	}
	used := false
	_ = tx.QueryRow(ctx, `SELECT true FROM email_verification_tokens
		WHERE account_id = $1::uuid AND purpose = 'email_change' AND consumed_at IS NULL AND expires_at > now()`,
		account.ID).Scan(&used)
	if used {
		writeError(c, http.StatusConflict, "change_pending", "an email change is already pending")
		return
	}
	plain, err := identity.CreateEmailToken(ctx, tx, account.ID, identity.TokenPurposeEmailChange, newEmail, h.cfg.Trial.EmailTokenTTL)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create change token")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to issue email change")
		return
	}
	_ = h.mailer.Send(ctx, identity.EmailMessage{
		To:      newEmail,
		Subject: "Confirm your new email",
		Text:    fmt.Sprintf("Confirm your new email with code:\n%s\nIt expires in %s.", plain, h.cfg.Trial.EmailTokenTTL),
	})
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"requires_verification": true, "sent_to": newEmail},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) emailVerifyChange(c *gin.Context) {
	var req struct {
		VerificationToken string `json:"verification_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.VerificationToken == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "verification_token is required")
		return
	}
	claims, _ := httptransport.Claims(c)
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
	account, err := h.resolveAccount(ctx, tx, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	payload, err := identity.ConsumeEmailToken(ctx, tx, req.VerificationToken, identity.TokenPurposeEmailChange, h.cfg.Trial.EmailTokenTTL)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_token", "the change link is invalid or expired")
		return
	}
	if payload.AccountID != account.ID || payload.NewEmail == "" {
		writeError(c, http.StatusBadRequest, "invalid_token", "the change link does not match this account")
		return
	}
	if err := h.store.ChangeEmail(ctx, tx, account.ID, payload.NewEmail); err != nil {
		writeError(c, http.StatusConflict, "email_taken", "that email is already in use")
		return
	}
	if err := h.store.MarkEmailVerified(ctx, tx, account.ID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to finalize email change")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to finalize email change")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"account_id": account.ID, "email": payload.NewEmail, "email_verified": true},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) phoneSend(c *gin.Context) {
	claims, _ := httptransport.Claims(c)
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
	account, err := h.resolveAccount(ctx, tx, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if account.PhoneE164 == "" {
		writeError(c, http.StatusBadRequest, "no_phone", "account has no phone number; phone is set at registration")
		return
	}
	if !h.counters.Take(ctx, "sms:send:"+account.ID, 3, identity.OTPResendWindow()) {
		writeError(c, http.StatusTooManyRequests, "rate_limited", "too many OTP requests; try again later")
		return
	}
	plain, err := identity.CreateOTPChallenge(ctx, tx, account.ID, account.PhoneE164, h.cfg.Trial.OTPTTL)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create OTP")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to issue OTP")
		return
	}
	_ = h.sms.Send(ctx, identity.SMSMessage{To: account.PhoneE164, Text: "Your POS verification code is: " + plain})
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"sent": true}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) phoneVerify(c *gin.Context) {
	var req struct {
		Otp string `json:"otp"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Otp == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "otp is required")
		return
	}
	claims, _ := httptransport.Claims(c)
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
	account, err := h.resolveAccount(ctx, tx, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	if err := identity.VerifyOTP(ctx, tx, account.ID, account.PhoneE164, req.Otp, h.cfg.Trial.OTPTTL); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_otp", "invalid or expired code")
		return
	}
	if err := h.store.VerifyPhone(ctx, tx, account.ID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify phone")
		return
	}
	if refreshed, err := h.store.GetByID(ctx, tx, account.ID); err == nil {
		h.activatePendingTrial(ctx, tx, claims.TenantID, &refreshed)
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify phone")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"account_id": account.ID, "phone": account.PhoneE164, "phone_verified": true},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) listInstallations(c *gin.Context) {
	claims, _ := httptransport.Claims(c)
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
	account, err := h.resolveAccount(ctx, tx, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	installations, err := h.installs.ListByAccount(ctx, tx, account.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load installations")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"installations": installationPayloads(installations)},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) revokeInstallation(c *gin.Context) {
	claims, _ := httptransport.Claims(c)
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
	account, err := h.resolveAccount(ctx, tx, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	installation, err := h.installs.GetByPublicID(ctx, tx, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, "installation_not_found", "installation not found")
		return
	}
	if installation.AccountID != account.ID {
		writeError(c, http.StatusNotFound, "installation_not_found", "installation not found")
		return
	}
	if err := h.installs.Revoke(ctx, tx, installation.ID, account.ID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to revoke installation")
		return
	}
	// Kill every live session whose device carries this installation's public
	// id within the caller's tenant (cascade-logout-all for the device).
	if _, err := tx.Exec(ctx, `
		UPDATE sessions s SET revoked_at = now()
		FROM devices d
		WHERE d.id = s.device_id AND d.tenant_id = s.tenant_id
		  AND d.tenant_id = $1::uuid AND d.client_device_id = $2::text AND s.revoked_at IS NULL`,
		claims.TenantID, installation.InstallationPublicID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to revoke sessions")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to revoke installation")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"revoked": true, "installation_id": installation.InstallationPublicID},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) debugEmailOutbox(c *gin.Context) {
	items := h.outbox.List(20)
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"items": items},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}
