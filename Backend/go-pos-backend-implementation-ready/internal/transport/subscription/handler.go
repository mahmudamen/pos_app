package subscription

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler serves the trial/subscription state that the client displays.
// Nothing here trusts the request clock: now comes from the database.
type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
	cfg    config.Config
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager, cfg config.Config) *Handler {
	return &Handler{pool: pool, tokens: tokens, cfg: cfg}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/subscription", httptransport.RequireAccessToken(h.tokens), h.subscription)
}

func (h *Handler) subscription(c *gin.Context) {
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

	dbNow, err := dbNow(ctx, tx)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to read server time")
		return
	}

	type tenantRow struct {
		name, plan     string
		trialEndsAt    *time.Time
		maxUsers       int
		maxProducts    int
		ownerAccountID string
	}
	var tenant tenantRow
	err = tx.QueryRow(ctx, `
		SELECT name, plan, trial_ends_at, max_users, max_products, COALESCE(owner_account_id::text, '')
		FROM tenants WHERE id = $1::uuid`, claims.TenantID).
		Scan(&tenant.name, &tenant.plan, &tenant.trialEndsAt, &tenant.maxUsers, &tenant.maxProducts, &tenant.ownerAccountID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenant")
		return
	}

	entitlement, err := identity.NewTrialStore().GetByOrganization(ctx, tx, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load trial")
		return
	}

	type subRow struct {
		status         string
		planCode       string
		trialEndsAt    *time.Time
		cancelAtPeriod bool
		periodEnd      *time.Time
	}
	var sub subRow
	hasSub := false
	err = tx.QueryRow(ctx, `
		SELECT status, code, trial_ends_at, cancel_at_period_end, current_period_end
		FROM subscriptions s JOIN plans p ON p.id = s.plan_id
		WHERE s.tenant_id = $1::uuid AND s.status <> 'cancelled'
		ORDER BY s.created_at DESC LIMIT 1`, claims.TenantID).
		Scan(&sub.status, &sub.planCode, &sub.trialEndsAt, &sub.cancelAtPeriod, &sub.periodEnd)
	if errors.Is(err, pgx.ErrNoRows) {
		hasSub = false
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load subscription")
		return
	} else {
		hasSub = true
	}

	// Account-level suspension (admin action) is folded into the sub status.
	accountStatus := "active"
	suspendedReason := ""
	if tenant.ownerAccountID != "" {
		_ = tx.QueryRow(ctx,
			`SELECT status FROM accounts WHERE id = $1::uuid`, tenant.ownerAccountID).Scan(&accountStatus)
		if accountStatus == "suspended" {
			suspendedReason = "account_suspended_by_admin"
		}
	}

	policy, err := identity.LoadPolicy(ctx, tx, h.cfg.Trial)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load trial policy")
		return
	}

	trialStatus := resolveTrialStatus(entitlement, dbNow)
	subStatus := resolveSubscriptionStatus(sub.status, accountStatus, hasSub, accountStatus == "suspended", trialStatus)

	// The trial row's expiry is authoritative for the plan projection; fall back
	// to tenants.trial_ends_at (legacy registrations) when no entitlement row
	// exists yet.
	var trialExpires *time.Time
	if entitlement != nil && entitlement.ExpiresAt != nil {
		trialExpires = entitlement.ExpiresAt
	} else if tenant.trialEndsAt != nil {
		trialExpires = tenant.trialEndsAt
	}
	if sub.status == "trial" && sub.trialEndsAt != nil {
		trialExpires = sub.trialEndsAt
	}
	var trialStart *time.Time
	if entitlement != nil && entitlement.StartedAt != nil {
		trialStart = entitlement.StartedAt
	}

	derived := DerivedStatus{
		Plan:               tenant.plan,
		SubscriptionStatus: subStatus,
		TrialStatus:        trialStatus,
		TrialStart:         trialStart,
		TrialExpires:       trialExpires,
		DaysRemaining:      daysRemaining(trialExpires, dbNow),
		ServerTime:         dbNow,
		AccessAllowed:      accessAllowed(subStatus, trialStatus),
		RenewalRequired:    renewalRequired(subStatus, trialStatus),
		SuspensionReason:   suspendedReason,
	}
	if derived.DaysRemaining < 0 {
		derived.DaysRemaining = 0
	}

	entitlementID := ""
	if entitlement != nil {
		entitlementID = entitlement.ID
	}

	response := gin.H{
		"plan":                 derived.Plan,
		"subscription_status":  derived.SubscriptionStatus,
		"trial_status":         derived.TrialStatus,
		"trial_started_at":     formatTime(trialStart),
		"trial_expires_at":     formatTime(trialExpires),
		"days_remaining":       derived.DaysRemaining,
		"server_time":          dbNow.Format(time.RFC3339),
		"renewal_required":     derived.RenewalRequired,
		"access_allowed":       derived.AccessAllowed,
		"suspension_reason":    derived.SuspensionReason,
		"trial_entitlement_id": entitlementID,
		"features":             featuresFor(derived.Plan),
		"limits": gin.H{
			"max_users":    tenant.maxUsers,
			"max_products": tenant.maxProducts,
		},
		"offline": gin.H{
			"policy":      emptyIfZero(policy.OfflinePolicy, "grace24h"),
			"grace_hours": int64(identity.OfflineGraceHours(policy.OfflinePolicy).Hours()),
		},
	}
	c.JSON(http.StatusOK, gin.H{"data": response, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func featuresFor(plan string) []gin.H {
	features := []gin.H{
		{"code": "pos", "title": "Unlimited POS", "description": "Unlimited sku scanning and checkout"},
		{"code": "offline_mode", "title": "Offline mode", "description": "Continue selling without internet"},
	}
	if _, ok := map[string]bool{"trial": true, "starter": true}[plan]; ok {
		features = append(features, gin.H{"code": "users_3", "title": "1 user", "description": "Single cashier seat"})
	} else {
		features = append(features, gin.H{"code": "users_unlimited", "title": "Multiple users", "description": "Cashiers and managers"})
	}
	return features
}

func dbNow(ctx context.Context, tx pgx.Tx) (time.Time, error) {
	var now time.Time
	err := tx.QueryRow(ctx, `SELECT now()`).Scan(&now)
	return now.UTC(), err
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func emptyIfZero(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}
