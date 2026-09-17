package auth

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	defaultCountry    = "EG"
	defaultCurrency   = "EGP"
	defaultLanguage   = "ar"
	passwordMinLength = 8
	termsVersion      = "2026-01"
	privacyVersion    = "2026-01"
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
		"bakery":        true,
		"shawerma":      true,
		"falafel":       true,
		"pharmacy":      true,
		"butcher":       true,
		"fruits_veg":    true,
		"clothing":      true,
		"sweets":        true,
		"jewelry":       true,
		"hardware":      true,
	}
	slugClean = regexp.MustCompile(`[^a-z0-9]+`)
)

type registerRequest struct {
	StoreName            string `json:"store_name" binding:"required"`
	BusinessType         string `json:"business_type" binding:"required"`
	Email                string `json:"email" binding:"required,email"`
	Password             string `json:"password" binding:"required,min=8"`
	DisplayName          string `json:"display_name" binding:"required"`
	Phone                string `json:"phone"`
	CountryCode          string `json:"country_code"`
	CurrencyCode         string `json:"currency_code"`
	Language             string `json:"language"`
	DeviceID             string `json:"device_id" binding:"required"`
	DeviceName           string `json:"device_name" binding:"required"`
	InstallationPublicID string `json:"installation_public_id"`
	Platform             string `json:"platform"`
	AppVersion           string `json:"app_version"`
}

// Signup provisions a merchant store: a global account, a new tenant, the owner
// user, an installation link, an authoritative trial entitlement (granted,
// pending-while-verification, or denied-but-created), a demo catalog, and an
// immediate session. Public endpoint (no auth required).
//
// The trial decision lives entirely server-side: duration, scope, dates, and
// denial all come from trial_entitlements, never from the client.
func (h *Handler) Signup(c *gin.Context) {
	var request registerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid registration request")
		return
	}
	request.Email = strings.TrimSpace(strings.ToLower(request.Email))
	request.StoreName = strings.TrimSpace(request.StoreName)
	request.DisplayName = strings.TrimSpace(request.DisplayName)
	request.Phone = identity.NormalizePhone(request.Phone)
	// The installation public id is a client-generated random UUID. Older
	// clients only send an opaque device_id, which is not a UUID, so the
	// installation layer is simply skipped rather than failing signup.
	request.InstallationPublicID = strings.TrimSpace(request.InstallationPublicID)
	if _, parseErr := uuid.Parse(request.InstallationPublicID); parseErr != nil {
		request.InstallationPublicID = ""
	}
	if !validBusinessTypes[request.BusinessType] {
		writeError(c, http.StatusBadRequest, "validation_error", "unsupported business_type")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}

	ctx := c.Request.Context()
	ip := c.ClientIP()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Global uniqueness of the email across tenants (store_emails lives outside
	// RLS so the check is not hidden by any tenant's row policy).
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

	logger := identity.NewAuditRecorder()
	accounts := identity.NewAccountStore(logger)
	persistedTrialPolicy, err := identity.LoadPolicy(ctx, tx, h.cfg.Trial)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load trial policy")
		return
	}

	// 1. Account — create on first sighting, reuse on re-registration attempts.
	account, err := accounts.GetByEmail(ctx, tx, request.Email)
	if errors.Is(err, identity.ErrNotFound) {
		account, err = accounts.Create(ctx, tx, request.Email, request.Phone,
			persistedTrialPolicy.RequireEmailVerification, persistedTrialPolicy.RequirePhoneVerification)
		if errors.Is(err, identity.ErrStateConflict) {
			writeError(c, http.StatusConflict, "email_taken", "an account with this email already exists")
			return
		}
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to create account")
			return
		}
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load account")
		return
	}
	if account.Status == securityStatusSuspended() || account.Status == "disabled" {
		writeError(c, http.StatusConflict, "account_suspended", "this account has been suspended; contact support")
		return
	}

	// 2. Organization + owner (public registration is always a new store).
	tenantID := uuid.New()
	countryCode, currencyCode, language := defaults(request.CountryCode, request.CurrencyCode, request.Language)
	_, err = tx.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, business_type, country_code, currency_code, default_language,
		                     plan, trial_ends_at, address, is_demo_seeded, owner_account_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'standard', NULL, '', TRUE, $8)`,
		tenantID, request.StoreName, slugFor(request.StoreName), request.BusinessType,
		countryCode, currencyCode, language, account.ID)
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
		INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type, account_id)
		VALUES ($1, $2, $3, $4, 'owner', 'standard', $5)
		RETURNING id`,
		tenantID, request.Email, passwordHash, request.DisplayName, account.ID).Scan(&userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create owner")
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE tenants SET owner_user_id = $2 WHERE id = $1`, tenantID, userID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to set owner")
		return
	}

	// 3. Installation — server-side upsert + link (optional: only when the
	// client presented a valid installation public id).
	installations := identity.NewInstallationStore()
	var installationID string
	if request.InstallationPublicID != "" {
		installation, err := installations.Upsert(ctx, tx, request.InstallationPublicID, request.Platform, request.AppVersion)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to register installation")
			return
		}
		if err := installations.Link(ctx, tx, installation.ID, account.ID, tenantID.String()); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to link installation")
			return
		}
		installationID = installation.ID
		if err := logger.Record(ctx, tx, identity.AuditEntry{
			Action:     identity.ActionInstallationRegistered,
			AccountID:  account.ID,
			TenantID:   tenantID.String(),
			EntityID:   installation.ID,
			EntityType: identity.EntityInstallation,
		}); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to record installation")
			return
		}
	}

	// 4. Eligibility → grant | pending (verification/review) | denied.
	service := identity.NewEligibilityService(persistedTrialPolicy, h.registerLimiter, logger)
	check, err := service.CheckEligibility(ctx, tx, identity.EligibilityRequest{
		AccountID:      account.ID,
		TenantID:       tenantID.String(),
		OwnerUserID:    userID.String(),
		TrialType:      "standard",
		EmailVerified:  account.IsEmailVerified(),
		PhoneVerified:  account.IsPhoneVerified(),
		InstallationID: installationID,
		IP:             ip,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to evaluate trial eligibility")
		return
	}
	_ = logger.Record(ctx, tx, identity.AuditEntry{
		Action:     identity.ActionTrialEligibilityChecked,
		AccountID:  account.ID,
		TenantID:   tenantID.String(),
		EntityType: identity.EntityTrial,
		Reason:     check.Reason,
	})

	now := time.Now().UTC()
	eligKey := identity.EligibilityKeyFor(persistedTrialPolicy.Scope, account.ID)
	if persistedTrialPolicy.Scope == "organization" {
		eligKey = identity.EligibilityKeyFor("organization", tenantID.String())
	}

	var entitlement *identity.TrialEntitlement
	denied := true
	switch {
	case check.Eligible:
		entitlement, err = identity.GrantTrial(ctx, tx, persistedTrialPolicy, identity.GrantTrialInput{
			AccountID: account.ID, TenantID: tenantID.String(), OwnerUserID: userID.String(),
			TrialType: "standard", EligibilityKey: eligKey,
			Status: identity.TrialStatusActive, DurationDays: persistedTrialPolicy.DurationDays,
			Source: "signup", Reason: "eligible",
		}, now)
		denied = false
	case check.VerificationRequired:
		entitlement, err = identity.GrantTrial(ctx, tx, persistedTrialPolicy, identity.GrantTrialInput{
			AccountID: account.ID, TenantID: tenantID.String(), OwnerUserID: userID.String(),
			TrialType: "standard", EligibilityKey: eligKey,
			Status: identity.TrialStatusPending, DurationDays: 0,
			Source: "signup", Reason: "verification_pending:" + strings.Join(check.VerificationHints, ","),
		}, now)
		denied = false
	case check.NeedsReview:
		entitlement, err = identity.GrantTrial(ctx, tx, persistedTrialPolicy, identity.GrantTrialInput{
			AccountID: account.ID, TenantID: tenantID.String(), OwnerUserID: userID.String(),
			TrialType: "standard", EligibilityKey: eligKey,
			Status: identity.TrialStatusPending, DurationDays: 0,
			Source: "signup", Reason: "review_pending:" + check.Reason,
		}, now)
		denied = false
	default: // hard denial — the org still exists, but with no usable trial.
		entitlement = &identity.TrialEntitlement{}
		if err := identity.DenyTrialProjection(ctx, tx, tenantID.String()); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to finalize trial")
			return
		}
		_ = logger.Record(ctx, tx, identity.AuditEntry{
			Action: identity.ActionTrialDenied, AccountID: account.ID, TenantID: tenantID.String(),
			EntityType: identity.EntityTrial, Reason: check.Reason,
		})
	}
	if err != nil {
		if errors.Is(err, identity.ErrTrialAlreadyExists) {
			// A race lost to a concurrent grant never produces a usable trial.
			if err := identity.DenyTrialProjection(ctx, tx, tenantID.String()); err != nil {
				writeError(c, http.StatusInternalServerError, "internal_error", "unable to finalize trial")
				return
			}
			denied = true
		} else {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to create trial")
			return
		}
	}

	trialState := gin.H{
		"status":                "denied",
		"expires_at":            "",
		"denied":                denied,
		"requires_verification": false,
		"verification_hints":    []string{},
		"days_remaining":        0,
	}
	if entitlement != nil {
		switch entitlement.Status {
		case identity.TrialStatusActive:
			trialState["status"] = "active"
			trialState["expires_at"] = entitlement.ExpiresAt.UTC().Format(time.RFC3339)
			trialState["days_remaining"] = daysUntil(entitlement.ExpiresAt, now)
		case identity.TrialStatusPending:
			trialState["status"] = "pending"
			if check.VerificationRequired {
				trialState["requires_verification"] = true
				trialState["verification_hints"] = check.VerificationHints
			}
		}
	}

	if err := seedCatalog(ctx, tx, tenantID.String(), request.BusinessType); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to seed demo catalog")
		return
	}

	sessionID := uuid.New()
	deviceUUID, err := h.registerDevice(ctx, tx, tenantID.String(), request.DeviceID, request.DeviceName, installationID)
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

	if err := logger.Record(ctx, tx, identity.AuditEntry{
		Action: identity.ActionAccountRegistered, AccountID: account.ID, TenantID: tenantID.String(),
		EntityID: account.ID, EntityType: identity.EntityAccount,
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record registration")
		return
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit registration")
		return
	}

	accessToken, err := h.tokens.IssueWithRole(time.Now(), security.AccessToken, tenantID.String(), userID.String(), deviceUUID.String(), sessionID.String(), "owner")
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to issue access token")
		return
	}
	response := gin.H{
		"access_token": accessToken, "refresh_token": refreshToken, "expires_in": int(h.tokens.AccessTTL.Seconds()),
		"user": gin.H{"id": userID.String(), "display_name": request.DisplayName, "role": "owner", "account_type": "standard"},
		"tenant": gin.H{
			"id": tenantID.String(), "business_type": request.BusinessType,
			"country_code": countryCode, "currency_code": currencyCode, "default_language": language,
			"plan": "trial", "trial_ends_at": "",
		},
		"identity": gin.H{
			"account_id": account.ID, "email": account.PrimaryEmail,
			"email_verified": account.IsEmailVerified(), "phone_verified": account.IsPhoneVerified(),
		},
		"installation": gin.H{
			"installation_public_id": request.InstallationPublicID,
			"status":                 installationStatus(installationID),
		},
		"trial": trialState,
	}
	if entitlement != nil && entitlement.ExpiresAt != nil {
		response["tenant"].(gin.H)["trial_ends_at"] = entitlement.ExpiresAt.UTC().Format(time.RFC3339)
	}
	c.JSON(http.StatusCreated, gin.H{"data": response, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func securityStatusSuspended() string { return "suspended" }

func installationStatus(installationID string) string {
	if installationID == "" {
		return "none"
	}
	return "linked"
}

func daysUntil(expires *time.Time, now time.Time) int {
	if expires == nil {
		return 0
	}
	days := int(expires.Sub(now).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
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

func (h *Handler) registerDevice(ctx context.Context, tx pgx.Tx, tenantID, deviceID, deviceName, installationID string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO devices (id, tenant_id, client_device_id, name, last_seen_at, installation_id)
		VALUES ($1, $2, $3, $4, now(), NULLIF($5, '')::uuid)
		ON CONFLICT (tenant_id, client_device_id)
		DO UPDATE SET name = EXCLUDED.name, last_seen_at = now(), revoked_at = NULL,
			installation_id = COALESCE(devices.installation_id, EXCLUDED.installation_id)
		RETURNING id`, uuid.New(), tenantID, deviceID, deviceName, installationID).Scan(&id)
	return id, err
}

func (h *Handler) createSession(ctx context.Context, tx pgx.Tx, sessionID uuid.UUID, tenantID, userID string, deviceUUID uuid.UUID, refreshToken string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO sessions (id, tenant_id, user_id, device_id, refresh_token_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5, now() + ($6::bigint * interval '1 second'))`,
		sessionID, tenantID, userID, deviceUUID, hashToken(refreshToken), int64(h.tokens.RefreshTTL.Seconds()))
	return err
}
