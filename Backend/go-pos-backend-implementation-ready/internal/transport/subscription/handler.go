package subscription

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/example/pos-api/internal/billing"
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
	router.POST("/subscription/change-plan", httptransport.RequireAccessToken(h.tokens), h.changePlan)
}

type changePlanRequest struct {
	PlanCode string `json:"plan_code" binding:"required"`
}

// changePlan is the self-service plan upgrade/downgrade for the tenant owner
// (or a manager acting on the store's behalf). The subscription is resolved by
// the caller's tenant — never from a client-supplied id — and the plan limits
// are re-checked inside the tenant's RLS context, mirroring the saas_admin
// change-plan path in billing.
func (h *Handler) changePlan(c *gin.Context) {
	var req changePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid change-plan payload")
		return
	}
	claims, ok := httptransport.Claims(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is required")
		return
	}
	if claims.Role != "owner" && claims.Role != "manager" {
		writeError(c, http.StatusForbidden, "permission_denied", "only the store owner or a manager can change the plan")
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

	var subID, subStatus string
	err = tx.QueryRow(ctx, `
		SELECT id::text, status FROM subscriptions
		WHERE tenant_id = $1::uuid AND status <> 'cancelled'
		ORDER BY created_at DESC LIMIT 1`, claims.TenantID).
		Scan(&subID, &subStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "subscription_not_found", "no active subscription")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load subscription")
		return
	}
	if subStatus == billing.StatusCancelled {
		writeError(c, http.StatusConflict, "invalid_transition", "cannot change the plan of a cancelled subscription")
		return
	}

	var target billing.Plan
	if err := tx.QueryRow(ctx, `
		SELECT code, name, price_minor, currency, billing_period, max_users, max_products
		FROM plans WHERE code = $1 AND is_active`, req.PlanCode).
		Scan(&target.Code, &target.Name, &target.PriceMinor, &target.Currency, &target.BillingPeriod, &target.MaxUsers, &target.MaxProducts); errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "plan_not_found", "plan not found or inactive")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load plan")
		return
	}

	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var users, products int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count users")
		return
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&products); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count products")
		return
	}
	decision := billing.EvaluatePlanChange(int(users), int(products), target)
	if !decision.Allowed {
		c.JSON(http.StatusConflict, gin.H{
			"error": gin.H{
				"code": "plan_downgrade_blocked", "message": decision.Remediation,
				"request_id": c.GetString("request_id"),
			},
			"data": gin.H{
				"code": decision.Code, "required": decision.Required,
				"limit": decision.Limit, "usage": gin.H{"users": users, "products": products},
			},
		})
		return
	}

	if _, err := tx.Exec(ctx, `
		UPDATE subscriptions SET plan_id = (SELECT id FROM plans WHERE code = $1) WHERE id = $2::uuid`,
		req.PlanCode, subID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to change plan")
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE tenants SET plan = $2, max_users = $3, max_products = $4 WHERE id = $1::uuid`,
		claims.TenantID, target.Code, target.MaxUsers, target.MaxProducts); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to sync tenant plan")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to change plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"id": subID, "plan_code": req.PlanCode, "status": subStatus},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
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
