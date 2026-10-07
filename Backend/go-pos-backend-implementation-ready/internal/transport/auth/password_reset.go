package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/example/pos-api/internal/email"
	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

// passwordResetExpiryLabel is the human-readable lifetime put in the mail and
// matched by identity.PasswordResetTTL.
const passwordResetExpiryLabel = "30 minutes"

type forgotPasswordRequest struct {
	TenantID string `json:"tenant_id" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// ForgotPassword mints a single-use reset link for one login and emails it.
//
// The request names the store exactly like POST /v1/auth/login does: `users`
// rows are per-tenant and FORCE-RLS'd, so a bare email cannot be resolved
// without either a tenant context or a cross-tenant scan. Scoping up front
// keeps the route O(1) and keeps RLS doing its job.
//
// The response is deliberately uniform — 202 whether or not the store/email
// matched — so this route cannot be used to enumerate tenants or addresses.
// Rate limiting is per IP and runs before any database work.
func (h *Handler) ForgotPassword(c *gin.Context) {
	var request forgotPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid forgot-password request")
		return
	}
	request.TenantID = strings.TrimSpace(request.TenantID)
	request.Email = identity.NormalizeEmail(request.Email)
	if request.Email == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid email")
		return
	}
	ctx := c.Request.Context()
	if !h.resetLimiter.Take(ctx, "password_reset:ip:"+c.ClientIP(),
		identity.PasswordResetRateLimit, identity.PasswordResetRateWindow) {
		writeError(c, http.StatusTooManyRequests, "rate_limited",
			"too many password reset requests, please try again later")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}

	accepted := func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{
			"data": gin.H{"status": "sent"},
			"meta": gin.H{"request_id": c.GetString("request_id")},
		})
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var tenantID string
	if err = tx.QueryRow(ctx, `SELECT id::text FROM tenants WHERE id::text = $1 OR slug = $1`,
		request.TenantID).Scan(&tenantID); err != nil {
		// Unknown store: answer exactly as for a known one.
		accepted(c)
		return
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var userID, displayName string
	if err = tx.QueryRow(ctx, `
		SELECT id::text, display_name FROM users
		WHERE tenant_id = $1::uuid AND lower(email) = lower($2) AND is_active`,
		tenantID, request.Email).Scan(&userID, &displayName); err != nil {
		accepted(c)
		return
	}
	token, err := identity.CreatePasswordResetToken(ctx, tx, tenantID, userID, identity.PasswordResetTTL)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create reset link")
		return
	}
	logger := identity.NewAuditRecorder()
	if err = logger.Record(ctx, tx, identity.AuditEntry{
		TenantID:    tenantID,
		ActorUserID: userID,
		Action:      identity.ActionPasswordResetRequested,
		EntityType:  "user",
		EntityID:    userID,
		IP:          c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record reset request")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit reset request")
		return
	}

	// Delivered after the commit, exactly like the welcome email: a mailer
	// failure must not fail a request that already succeeded, and the token
	// simply expires unused. The link is a plain GET on the public reset page,
	// so it works from any browser, whichever device asked.
	if h.mail != nil {
		resetURL := requestOrigin(c) + "/reset-password?token=" + token
		if err = h.mail.SendTemplate(ctx, email.TemplatePasswordReset, request.Email, map[string]string{
			"name":      displayName,
			"reset_url": resetURL,
			"expires":   passwordResetExpiryLabel,
		}); err != nil {
			slog.Warn("auth: password reset email delivery failed", "err", err,
				"tenant_id", tenantID, "user_id", userID)
		}
	}
	accepted(c)
}

// ResetPassword consumes a presented token and sets the new password for the
// one login it is bound to, revoking that user's live sessions so a stolen
// refresh token dies with the old password.
func (h *Handler) ResetPassword(c *gin.Context) {
	var request resetPasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid reset request")
		return
	}
	request.Token = strings.TrimSpace(request.Token)
	if request.Token == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid reset request")
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

	invalid := func(c *gin.Context) {
		writeError(c, http.StatusBadRequest, "invalid_token", "reset link is invalid or has expired")
	}
	scope, err := identity.ConsumePasswordResetToken(ctx, tx, request.Token)
	if errors.Is(err, identity.ErrInvalidToken) {
		invalid(c)
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to validate reset token")
		return
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, scope.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	hash, err := security.HashPassword(request.Password, h.cost)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to hash password")
		return
	}
	res, err := tx.Exec(ctx, `
		UPDATE users SET password_hash = $1, updated_at = now()
		WHERE id = $2::uuid AND tenant_id = $3::uuid`,
		hash, scope.UserID, scope.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update password")
		return
	}
	if res.RowsAffected() != 1 {
		invalid(c)
		return
	}
	// Every session for that login is dead: the old password may have been
	// stolen, so the refresh tokens minted with it must not outlive the reset.
	if _, err = tx.Exec(ctx, `
		UPDATE sessions SET revoked_at = now()
		WHERE tenant_id = $1::uuid AND user_id = $2::uuid AND revoked_at IS NULL`,
		scope.TenantID, scope.UserID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to revoke sessions")
		return
	}
	logger := identity.NewAuditRecorder()
	if err = logger.Record(ctx, tx, identity.AuditEntry{
		TenantID:    scope.TenantID,
		ActorUserID: scope.UserID,
		Action:      identity.ActionPasswordResetCompleted,
		EntityType:  "user",
		EntityID:    scope.UserID,
		IP:          c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record password reset")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit password reset")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"status": "reset"},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

// requestOrigin rebuilds the absolute origin a request arrived on, so an
// emailed link points back at this server (the public reset page lives here).
// Behind the production nginx, X-Forwarded-Proto carries the terminated scheme;
// direct local development falls back to the plain scheme of the connection.
func requestOrigin(c *gin.Context) string {
	scheme := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto"))
	if scheme != "http" && scheme != "https" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		return ""
	}
	return scheme + "://" + host
}

// resetLinkTTL is the formatted lifetime used when rendering the mail; it is
// asserted against identity.PasswordResetTTL by a unit test so the two cannot
// drift.
func resetLinkTTL() time.Duration { return identity.PasswordResetTTL }
