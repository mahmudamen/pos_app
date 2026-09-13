package auth

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	trialDays         = 15
	defaultCountry    = "EG"
	defaultCurrency   = "EGP"
	defaultLanguage   = "ar"
	passwordMinLength = 8
)

var (
	validBusinessTypes = map[string]bool{
		"coffee_shop":   true,
		"restaurant":    true,
		"retail":        true,
		"book_store":    true,
		"mobile_shop":   true,
		"computer_shop": true,
		"grocery":       true,
	}
	slugClean = regexp.MustCompile(`[^a-z0-9]+`)
)

type registerRequest struct {
	StoreName    string `json:"store_name" binding:"required"`
	BusinessType string `json:"business_type" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=8"`
	DisplayName  string `json:"display_name" binding:"required"`
	CountryCode  string `json:"country_code"`
	CurrencyCode string `json:"currency_code"`
	Language     string `json:"language"`
	DeviceID     string `json:"device_id" binding:"required"`
	DeviceName   string `json:"device_name" binding:"required"`
}

// register provisions a merchant store: a new tenant on a 15-day trial, the
// owner account, a demo catalog seeded with real products (price, image, info,
// barcode), and an immediate session so the first log-in is the store itself.
// Public endpoint (no auth required).
func (h *Handler) Signup(c *gin.Context) {
	var request registerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid registration request")
		return
	}
	request.Email = strings.TrimSpace(strings.ToLower(request.Email))
	request.StoreName = strings.TrimSpace(request.StoreName)
	request.DisplayName = strings.TrimSpace(request.DisplayName)
	if !validBusinessTypes[request.BusinessType] {
		writeError(c, http.StatusBadRequest, "validation_error", "unsupported business_type")
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

	// Global uniqueness across tenants: store_emails lives outside RLS so the
	// check is not hidden by the current tenant's row policy. Inserting in the
	// same transaction makes concurrent registrations with one email safe.
	res, err := tx.Exec(ctx, `INSERT INTO store_emails (email) VALUES (LOWER($1))`, request.Email)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(c, http.StatusConflict, "email_taken", "an account with this email already exists")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to check email")
		return
	}
	if res.RowsAffected() != 1 {
		writeError(c, http.StatusConflict, "email_taken", "an account with this email already exists")
		return
	}

	tenantID := uuid.New()
	trialEndsAt := time.Now().Add(trialDays * 24 * time.Hour)
	countryCode, currencyCode, language := defaults(request.CountryCode, request.CurrencyCode, request.Language)
	_, err = tx.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, business_type, country_code, currency_code, default_language, plan, trial_ends_at, address)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'trial', $8, '')`,
		tenantID, request.StoreName, slugFor(request.StoreName), request.BusinessType,
		countryCode, currencyCode, language, trialEndsAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create tenant")
		return
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	passwordHash, err := security.HashPassword(request.Password, h.cost)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to hash password")
		return
	}
	var userID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type)
		VALUES ($1, $2, $3, $4, 'owner', 'standard')
		RETURNING id`,
		tenantID, request.Email, passwordHash, request.DisplayName).Scan(&userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create owner")
		return
	}

	if err = seedCatalog(ctx, tx, tenantID.String(), request.BusinessType); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to seed demo catalog")
		return
	}

	sessionID := uuid.New()
	deviceUUID, err := h.registerDevice(ctx, tx, tenantID.String(), request.DeviceID, request.DeviceName)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to register device")
		return
	}
	refreshToken, err := h.tokens.Issue(time.Now(), security.RefreshToken, tenantID.String(), userID.String(), deviceUUID.String(), sessionID.String())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to issue session")
		return
	}
	if err := h.createSession(ctx, tx, sessionID, tenantID.String(), userID.String(), deviceUUID, refreshToken); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create session")
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, userID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update login state")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit registration")
		return
	}

	accessToken, err := h.tokens.IssueWithRole(time.Now(), security.AccessToken, tenantID.String(), userID.String(), deviceUUID.String(), sessionID.String(), "owner")
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to issue access token")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"access_token": accessToken, "refresh_token": refreshToken, "expires_in": int(h.tokens.AccessTTL.Seconds()),
		"user": gin.H{"id": userID.String(), "display_name": request.DisplayName, "role": "owner", "account_type": "standard"},
		"tenant": gin.H{
			"id": tenantID.String(), "business_type": request.BusinessType,
			"country_code": countryCode, "currency_code": currencyCode, "default_language": language,
			"plan": "trial", "trial_ends_at": trialEndsAt.UTC().Format(time.RFC3339),
		},
	}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func defaults(country, currency, language string) (string, string, string) {
	if strings.TrimSpace(country) == "" {
		country = defaultCountry
	}
	if strings.TrimSpace(currency) == "" {
		currency = defaultCurrency
	}
	if strings.TrimSpace(language) == "" {
		language = defaultLanguage
	}
	return country, currency, language
}

func slugFor(name string) string {
	slug := slugClean.ReplaceAllString(strings.ToLower(name), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 40 {
		slug = slug[:40]
	}
	if slug == "" {
		slug = "store"
	}
	return slug + "-" + strings.ToLower(uuid.NewString()[:8])
}

func (h *Handler) registerDevice(ctx context.Context, tx pgx.Tx, tenantID, deviceID, deviceName string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO devices (id, tenant_id, client_device_id, name, last_seen_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (tenant_id, client_device_id)
		DO UPDATE SET name = EXCLUDED.name, last_seen_at = now(), revoked_at = NULL
		RETURNING id`, uuid.New(), tenantID, deviceID, deviceName).Scan(&id)
	return id, err
}

func (h *Handler) createSession(ctx context.Context, tx pgx.Tx, sessionID uuid.UUID, tenantID, userID string, deviceUUID uuid.UUID, refreshToken string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, user_id, device_id, refresh_token_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5, now() + ($6::bigint * interval '1 second'))`,
		sessionID, tenantID, userID, deviceUUID, hashToken(refreshToken), int64(h.tokens.RefreshTTL.Seconds()))
	return err
}
