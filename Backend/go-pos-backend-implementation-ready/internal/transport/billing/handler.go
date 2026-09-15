// Package billing exposes the SaaS billing control plane to the web admin:
// the plan catalog, tenant subscriptions (with an explicit state machine), and
// invoices charged through the payment gateway abstraction. Every route is
// gated to the saas_admin platform role and reads/writes the non-RLS platform
// tables (plans / subscriptions / invoices); per-tenant usage checks run in a
// per-tenant RLS context like the rest of the SaaS handlers.
package billing

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/example/pos-api/internal/billing"
	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultCurrency = "EGP"

type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	admin := httptransport.RequireSaasAdmin(h.tokens)
	router.GET("/plans", admin, h.listPlans)
	router.POST("/plans", admin, h.createPlan)
	router.PATCH("/plans/:id", admin, h.updatePlan)
	router.DELETE("/plans/:id", admin, h.deletePlan)

	router.GET("/subscriptions", admin, h.listSubscriptions)
	router.GET("/tenants/:id/subscription", admin, h.tenantSubscription)
	router.POST("/tenants/:id/subscription", admin, h.assignSubscription)
	router.POST("/subscriptions/:id/status", admin, h.setSubscriptionStatus)
	router.POST("/subscriptions/:id/change-plan", admin, h.changeSubscriptionPlan)

	router.GET("/invoices", admin, h.listInvoices)
	router.POST("/tenants/:id/invoices", admin, h.createInvoice)
	router.POST("/invoices/:id/pay", admin, h.payInvoice)
	router.POST("/invoices/:id/void", admin, h.voidInvoice)
	router.POST("/invoices/:id/refund", admin, h.refundInvoice)

	router.GET("/payment-providers", admin, h.paymentProviders)
	router.GET("/billing/summary", admin, h.billingSummary)
}

// ---------------------------------------------------------------------------
// Plans
// ---------------------------------------------------------------------------

type planPayload struct {
	ID            string   `json:"id"`
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	PriceMinor    int64    `json:"price_minor"`
	Currency      string   `json:"currency"`
	BillingPeriod string   `json:"billing_period"`
	Features      []string `json:"features"`
	MaxUsers      int      `json:"max_users"`
	MaxProducts   int      `json:"max_products"`
	IsActive      bool     `json:"is_active"`
}

func (h *Handler) listPlans(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id, code, name, description, price_minor, currency, billing_period,
		       COALESCE(features, '[]'::jsonb), max_users, max_products, is_active
		FROM plans ORDER BY price_minor, code`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load plans")
		return
	}
	defer rows.Close()
	plans := make([]planPayload, 0)
	for rows.Next() {
		var p planPayload
		if err := rows.Scan(&p.ID, &p.Code, &p.Name, &p.Description, &p.PriceMinor, &p.Currency,
			&p.BillingPeriod, &p.Features, &p.MaxUsers, &p.MaxProducts, &p.IsActive); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load plans")
			return
		}
		plans = append(plans, p)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load plans")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": plans,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

type createPlanRequest struct {
	Code          string   `json:"code" binding:"required"`
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	PriceMinor    int64    `json:"price_minor"`
	Currency      string   `json:"currency"`
	BillingPeriod string   `json:"billing_period" binding:"required"`
	Features      []string `json:"features"`
	MaxUsers      int      `json:"max_users"`
	MaxProducts   int      `json:"max_products"`
}

func (h *Handler) createPlan(c *gin.Context) {
	var req createPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid plan payload")
		return
	}
	if req.Currency == "" {
		req.Currency = defaultCurrency
	}
	plan := billing.Plan{
		Code: req.Code, Name: req.Name, Description: req.Description,
		PriceMinor: req.PriceMinor, Currency: req.Currency, BillingPeriod: req.BillingPeriod,
		Features: req.Features, MaxUsers: req.MaxUsers, MaxProducts: req.MaxProducts, IsActive: true,
	}
	if err := plan.Validate(); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	var id string
	err := h.pool.QueryRow(c.Request.Context(), `
		INSERT INTO plans (code, name, description, price_minor, currency, billing_period, features, max_users, max_products)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9)
		RETURNING id::text`,
		req.Code, req.Name, req.Description, req.PriceMinor, req.Currency, req.BillingPeriod, jsonFeatures(req.Features), req.MaxUsers, req.MaxProducts).Scan(&id)
	if err != nil {
		writeError(c, http.StatusConflict, "plan_code_taken", "a plan with this code already exists")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{"id": id, "code": req.Code, "name": req.Name},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

type updatePlanRequest struct {
	Name          *string   `json:"name"`
	Description   *string   `json:"description"`
	PriceMinor    *int64    `json:"price_minor"`
	Currency      *string   `json:"currency"`
	BillingPeriod *string   `json:"billing_period"`
	Features      *[]string `json:"features"`
	MaxUsers      *int      `json:"max_users"`
	MaxProducts   *int      `json:"max_products"`
	IsActive      *bool     `json:"is_active"`
}

func (h *Handler) updatePlan(c *gin.Context) {
	var req updatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid plan payload")
		return
	}
	if req.BillingPeriod != nil && !billing.ValidBillingPeriod(*req.BillingPeriod) {
		writeError(c, http.StatusBadRequest, "validation_error", "billing_period must be monthly or yearly")
		return
	}
	if req.PriceMinor != nil && *req.PriceMinor < 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "price must not be negative")
		return
	}
	if req.Currency != nil && len(*req.Currency) != 3 {
		writeError(c, http.StatusBadRequest, "validation_error", "currency must be a 3-letter code")
		return
	}
	if (req.MaxUsers != nil && *req.MaxUsers < 0) || (req.MaxProducts != nil && *req.MaxProducts < 0) {
		writeError(c, http.StatusBadRequest, "validation_error", "limits must not be negative")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	var features any
	if req.Features != nil {
		features = jsonFeatures(*req.Features)
	}
	tag, err := h.pool.Exec(c.Request.Context(), `
		UPDATE plans SET
			name           = COALESCE($2, name),
			description    = COALESCE($3, description),
			price_minor    = COALESCE($4, price_minor),
			currency       = COALESCE($5, currency),
			billing_period = COALESCE($6, billing_period),
			features       = COALESCE($7::jsonb, features),
			max_users      = COALESCE($8, max_users),
			max_products   = COALESCE($9, max_products),
			is_active      = COALESCE($10, is_active)
		WHERE id = $1::uuid`,
		c.Param("id"), req.Name, req.Description, req.PriceMinor, req.Currency, req.BillingPeriod,
		features, req.MaxUsers, req.MaxProducts, req.IsActive)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update plan")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "plan_not_found", "plan not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"id": c.Param("id"), "updated": true},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) deletePlan(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(),
		`UPDATE plans SET is_active = false WHERE id = $1::uuid AND is_active`, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete plan")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "plan_not_found", "plan not found")
		return
	}
	c.Status(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Subscriptions
// ---------------------------------------------------------------------------

type subscriptionPayload struct {
	ID                 string `json:"id"`
	TenantID           string `json:"tenant_id"`
	TenantName         string `json:"tenant_name"`
	TenantSlug         string `json:"tenant_slug"`
	PlanID             string `json:"plan_id"`
	PlanCode           string `json:"plan_code"`
	PlanName           string `json:"plan_name"`
	PriceMinor         int64  `json:"price_minor"`
	Currency           string `json:"currency"`
	BillingPeriod      string `json:"billing_period"`
	Status             string `json:"status"`
	Provider           string `json:"provider"`
	TrialEndsAt        string `json:"trial_ends_at,omitempty"`
	CurrentPeriodStart string `json:"current_period_start"`
	CurrentPeriodEnd   string `json:"current_period_end,omitempty"`
	CancelAtPeriodEnd  bool   `json:"cancel_at_period_end"`
	CancelledAt        string `json:"cancelled_at,omitempty"`
	CreatedAt          string `json:"created_at"`
}

func (h *Handler) listSubscriptions(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	status := c.DefaultQuery("status", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	ctx := c.Request.Context()
	var total int64
	if err := h.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM subscriptions s
		WHERE ($1 = '' OR s.status = $1)`, status).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count subscriptions")
		return
	}
	rows, err := h.pool.Query(ctx, `
		SELECT s.id, s.tenant_id, t.name, t.slug, s.plan_id, p.code, p.name, p.price_minor,
		       p.currency, p.billing_period, s.status, s.provider,
		       COALESCE(s.trial_ends_at::text, ''), s.current_period_start::text,
		       COALESCE(s.current_period_end::text, ''), s.cancel_at_period_end,
		       COALESCE(s.cancelled_at::text, ''), s.created_at::text
		FROM subscriptions s
		JOIN tenants t ON t.id = s.tenant_id
		JOIN plans p ON p.id = s.plan_id
		WHERE ($1 = '' OR s.status = $1)
		ORDER BY s.created_at DESC
		LIMIT $2 OFFSET $3`, status, limit, (page-1)*limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load subscriptions")
		return
	}
	defer rows.Close()
	list := make([]subscriptionPayload, 0)
	for rows.Next() {
		var s subscriptionPayload
		if err := rows.Scan(&s.ID, &s.TenantID, &s.TenantName, &s.TenantSlug, &s.PlanID, &s.PlanCode,
			&s.PlanName, &s.PriceMinor, &s.Currency, &s.BillingPeriod, &s.Status, &s.Provider,
			&s.TrialEndsAt, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelAtPeriodEnd,
			&s.CancelledAt, &s.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load subscriptions")
			return
		}
		list = append(list, s)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load subscriptions")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": list,
		"meta": gin.H{"request_id": c.GetString("request_id"), "page": page, "limit": limit, "total": total},
	})
}

func (h *Handler) tenantSubscription(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	subscription, found, err := h.loadLiveSubscription(c, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load subscription")
		return
	}
	if !found {
		writeError(c, http.StatusNotFound, "no_active_subscription", "tenant has no active subscription")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": subscription, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

type assignSubscriptionRequest struct {
	PlanCode  string `json:"plan_code" binding:"required"`
	Status    string `json:"status"`
	TrialDays int    `json:"trial_days"`
}

func (h *Handler) assignSubscription(c *gin.Context) {
	var req assignSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid subscription payload")
		return
	}
	status := req.Status
	if status == "" {
		status = billing.StatusTrial
	}
	if !billing.ValidSubscriptionStatus(status) {
		writeError(c, http.StatusBadRequest, "validation_error", "unknown subscription status")
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

	var tenantName string
	if err := tx.QueryRow(ctx, `SELECT name FROM tenants WHERE id = $1::uuid`, c.Param("id")).Scan(&tenantName); errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenant")
		return
	}

	var existing string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM subscriptions WHERE tenant_id = $1::uuid AND status <> 'cancelled'`, c.Param("id")).Scan(&existing); err == nil {
		writeError(c, http.StatusConflict, "subscription_active", "tenant already has an active subscription; cancel it first")
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to check existing subscription")
		return
	}

	var plan billing.Plan
	if err := tx.QueryRow(ctx, `
		SELECT code, name, price_minor, currency, billing_period FROM plans WHERE code = $1 AND is_active`,
		req.PlanCode).Scan(&plan.Code, &plan.Name, &plan.PriceMinor, &plan.Currency, &plan.BillingPeriod); errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "plan_not_found", "plan not found or inactive")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load plan")
		return
	}

	now := time.Now().UTC()
	var trialEndsAt *time.Time
	if req.TrialDays > 0 {
		end := now.AddDate(0, 0, req.TrialDays)
		trialEndsAt = &end
	}
	periodEnd := billing.PeriodEnd(now, plan.BillingPeriod)

	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO subscriptions (tenant_id, plan_id, status, provider, trial_ends_at, current_period_start, current_period_end)
		VALUES ($1::uuid, (SELECT id FROM plans WHERE code = $2), $3, 'mock', $4, $5, $6)
		RETURNING id::text`,
		c.Param("id"), req.PlanCode, status, trialEndsAt, now, periodEnd).Scan(&id); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create subscription")
		return
	}

	// Keep tenants.plan / max_* in sync so the existing plan-limit enforcement
	// in the users/products handlers reflects the subscribed plan.
	if _, err := tx.Exec(ctx, `
		UPDATE tenants SET plan = $2, max_users = $3, max_products = $4 WHERE id = $1::uuid`,
		c.Param("id"), plan.Code, plan.MaxUsers, plan.MaxProducts); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to sync tenant plan")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to assign subscription")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"id": id, "tenant_id": c.Param("id"), "tenant_name": tenantName,
			"plan_code": req.PlanCode, "status": status, "provider": "mock",
			"current_period_start": now.Format(time.RFC3339), "current_period_end": periodEnd.Format(time.RFC3339),
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

type setSubscriptionStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

func (h *Handler) setSubscriptionStatus(c *gin.Context) {
	var req setSubscriptionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid status payload")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	var current string
	err := h.pool.QueryRow(ctx,
		`SELECT status FROM subscriptions WHERE id = $1::uuid`, c.Param("id")).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "subscription_not_found", "subscription not found")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load subscription")
		return
	}
	if !billing.ValidSubscriptionStatus(req.Status) {
		writeError(c, http.StatusBadRequest, "validation_error", "unknown subscription status")
		return
	}
	if err := billing.ValidateTransition(current, req.Status); err != nil {
		writeError(c, http.StatusConflict, "invalid_transition", err.Error())
		return
	}
	now := time.Now().UTC()
	tag, err := h.pool.Exec(ctx, `
		UPDATE subscriptions SET
			status               = $2,
			cancelled_at         = CASE WHEN $2 = 'cancelled' THEN $3 ELSE cancelled_at END,
			cancel_at_period_end = CASE WHEN $2 = 'cancelled' THEN true ELSE cancel_at_period_end END
		WHERE id = $1::uuid`, c.Param("id"), req.Status, now)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update subscription")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "subscription_not_found", "subscription not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"id": c.Param("id"), "status": req.Status, "changed": current != req.Status},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

type changePlanRequest struct {
	PlanCode string `json:"plan_code" binding:"required"`
}

func (h *Handler) changeSubscriptionPlan(c *gin.Context) {
	var req changePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid change-plan payload")
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

	var tenantID string
	var subStatus string
	err = tx.QueryRow(ctx,
		`SELECT tenant_id::text, status FROM subscriptions WHERE id = $1::uuid`, c.Param("id")).Scan(&tenantID, &subStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "subscription_not_found", "subscription not found")
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

	// Usage check must run in the tenant's RLS context.
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var users, products int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count users")
		return
	}
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&products); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count products")
		return
	}
	decision := billing.EvaluatePlanChange(users, products, target)
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

	if _, err := tx.Exec(ctx, `UPDATE subscriptions SET plan_id = (SELECT id FROM plans WHERE code = $1) WHERE id = $2::uuid`, req.PlanCode, c.Param("id")); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to change plan")
		return
	}
	if _, err := tx.Exec(ctx, `UPDATE tenants SET plan = $2, max_users = $3, max_products = $4 WHERE id = $1::uuid`, tenantID, target.Code, target.MaxUsers, target.MaxProducts); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to sync tenant plan")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to change plan")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"id": c.Param("id"), "plan_code": req.PlanCode, "status": subStatus},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

// loadLiveSubscription returns the tenant's live (non-cancelled) subscription
// joined with its tenant and plan.
func (h *Handler) loadLiveSubscription(c *gin.Context, tenantID string) (subscriptionPayload, bool, error) {
	var s subscriptionPayload
	err := h.pool.QueryRow(c.Request.Context(), `
		SELECT s.id, s.tenant_id, t.name, t.slug, s.plan_id, p.code, p.name, p.price_minor,
		       p.currency, p.billing_period, s.status, s.provider,
		       COALESCE(s.trial_ends_at::text, ''), s.current_period_start::text,
		       COALESCE(s.current_period_end::text, ''), s.cancel_at_period_end,
		       COALESCE(s.cancelled_at::text, ''), s.created_at::text
		FROM subscriptions s
		JOIN tenants t ON t.id = s.tenant_id
		JOIN plans p ON p.id = s.plan_id
		WHERE s.tenant_id = $1::uuid AND s.status <> 'cancelled'
		LIMIT 1`, tenantID).
		Scan(&s.ID, &s.TenantID, &s.TenantName, &s.TenantSlug, &s.PlanID, &s.PlanCode,
			&s.PlanName, &s.PriceMinor, &s.Currency, &s.BillingPeriod, &s.Status, &s.Provider,
			&s.TrialEndsAt, &s.CurrentPeriodStart, &s.CurrentPeriodEnd, &s.CancelAtPeriodEnd,
			&s.CancelledAt, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return subscriptionPayload{}, false, nil
	}
	if err != nil {
		return subscriptionPayload{}, false, err
	}
	return s, true, nil
}

// ---------------------------------------------------------------------------
// Invoices
// ---------------------------------------------------------------------------

type invoicePayload struct {
	ID          string `json:"id"`
	TenantID    string `json:"tenant_id"`
	TenantName  string `json:"tenant_name"`
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
	Status      string `json:"status"`
	Provider    string `json:"provider"`
	ProviderRef string `json:"provider_ref,omitempty"`
	Description string `json:"description"`
	DueAt       string `json:"due_at,omitempty"`
	PaidAt      string `json:"paid_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func (h *Handler) listInvoices(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	status := c.DefaultQuery("status", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	ctx := c.Request.Context()
	var total int64
	if err := h.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM invoices i WHERE ($1 = '' OR i.status = $1)`, status).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count invoices")
		return
	}
	rows, err := h.pool.Query(ctx, `
		SELECT i.id, i.tenant_id, t.name, i.amount_minor, i.currency, i.status, i.provider,
		       COALESCE(i.provider_ref, ''), i.description, COALESCE(i.due_at::text, ''),
		       COALESCE(i.paid_at::text, ''), i.created_at::text
		FROM invoices i JOIN tenants t ON t.id = i.tenant_id
		WHERE ($1 = '' OR i.status = $1)
		ORDER BY i.created_at DESC
		LIMIT $2 OFFSET $3`, status, limit, (page-1)*limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load invoices")
		return
	}
	defer rows.Close()
	list := make([]invoicePayload, 0)
	for rows.Next() {
		var inv invoicePayload
		if err := rows.Scan(&inv.ID, &inv.TenantID, &inv.TenantName, &inv.AmountMinor, &inv.Currency,
			&inv.Status, &inv.Provider, &inv.ProviderRef, &inv.Description, &inv.DueAt, &inv.PaidAt, &inv.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load invoices")
			return
		}
		list = append(list, inv)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load invoices")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": list,
		"meta": gin.H{"request_id": c.GetString("request_id"), "page": page, "limit": limit, "total": total},
	})
}

type createInvoiceRequest struct {
	AmountMinor    int64  `json:"amount_minor" binding:"required"`
	Currency       string `json:"currency"`
	Description    string `json:"description" binding:"required"`
	DueAt          string `json:"due_at"`
	Provider       string `json:"provider"`
	SubscriptionID string `json:"subscription_id"`
}

func (h *Handler) createInvoice(c *gin.Context) {
	var req createInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid invoice payload")
		return
	}
	if req.Currency == "" {
		req.Currency = defaultCurrency
	}
	inv := billing.Invoice{AmountMinor: req.AmountMinor, Currency: req.Currency, Description: req.Description}
	if err := inv.Validate(); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if req.Provider == "" {
		req.Provider = "manual"
	}
	if _, ok := billing.ResolveGateway(req.Provider); !ok {
		writeError(c, http.StatusBadRequest, "validation_error", "unknown payment provider")
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

	var tenantName string
	if err := tx.QueryRow(ctx, `SELECT name FROM tenants WHERE id = $1::uuid`, c.Param("id")).Scan(&tenantName); errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenant")
		return
	}

	var dueAt *time.Time
	if req.DueAt != "" {
		parsed, err := time.Parse(time.RFC3339, req.DueAt)
		if err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "due_at must be RFC3339")
			return
		}
		dueAt = &parsed
	}
	var subID *string
	if req.SubscriptionID != "" {
		var ok string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM subscriptions WHERE id = $1::uuid AND tenant_id = $2::uuid`, req.SubscriptionID, c.Param("id")).Scan(&ok); errors.Is(err, pgx.ErrNoRows) {
			writeError(c, http.StatusNotFound, "subscription_not_found", "subscription not found for tenant")
			return
		} else if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to check subscription")
			return
		}
		subID = &req.SubscriptionID
	}

	var id string
	if err := tx.QueryRow(ctx, `
		INSERT INTO invoices (tenant_id, subscription_id, amount_minor, currency, provider, description, due_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7)
		RETURNING id::text`, c.Param("id"), subID, req.AmountMinor, req.Currency, req.Provider, req.Description, dueAt).Scan(&id); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create invoice")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create invoice")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{"id": id, "tenant_id": c.Param("id"), "tenant_name": tenantName,
			"amount_minor": req.AmountMinor, "currency": req.Currency, "status": billing.InvoiceOpen,
			"provider": req.Provider, "description": req.Description},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

type payInvoiceRequest struct {
	Provider string `json:"provider"`
}

func (h *Handler) payInvoice(c *gin.Context) {
	var req payInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil { // empty body is also fine
		writeError(c, http.StatusBadRequest, "validation_error", "invalid pay payload")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	var amount int64
	var currency, status, provider, tenantID string
	err := h.pool.QueryRow(ctx, `
		SELECT tenant_id::text, amount_minor, currency, status, provider FROM invoices WHERE id = $1::uuid`, c.Param("id")).
		Scan(&tenantID, &amount, &currency, &status, &provider)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "invoice_not_found", "invoice not found")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load invoice")
		return
	}
	if status != billing.InvoiceOpen {
		writeError(c, http.StatusConflict, "invalid_transition", "only open invoices can be paid")
		return
	}
	gatewayName := provider
	if req.Provider != "" {
		gatewayName = req.Provider
	}
	gateway, ok := billing.ResolveGateway(gatewayName)
	if !ok {
		writeError(c, http.StatusBadRequest, "validation_error", "unknown payment provider")
		return
	}
	charge, err := gateway.Charge(ctx, billing.ChargeRequest{
		TenantID: tenantID, InvoiceID: c.Param("id"), AmountMinor: amount, Currency: currency,
		Description: "invoice " + c.Param("id"), IdempotencyKey: "inv:" + c.Param("id") + ":pay",
	})
	if err != nil {
		switch {
		case errors.Is(err, billing.ErrDeclined):
			writeError(c, http.StatusPaymentRequired, "payment_declined", "payment was declined")
		case errors.Is(err, billing.ErrInvalidAmount):
			writeError(c, http.StatusBadRequest, "validation_error", "invalid charge amount")
		case errors.Is(err, billing.ErrUnsupportedCurrency):
			writeError(c, http.StatusBadRequest, "unsupported_currency", "provider does not support this currency")
		default:
			writeError(c, http.StatusBadGateway, "gateway_error", "payment gateway error")
		}
		return
	}
	if charge.Status != billing.PaymentSucceeded {
		writeError(c, http.StatusPaymentRequired, "payment_declined", "payment was not completed")
		return
	}
	if _, err := h.pool.Exec(ctx, `
		UPDATE invoices SET status = 'paid', paid_at = $2, provider = $3, provider_ref = $4
		WHERE id = $1::uuid`, c.Param("id"), charge.PaidAt, charge.Provider, charge.Reference); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to settle invoice")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"id": c.Param("id"), "status": billing.InvoicePaid,
			"provider": charge.Provider, "provider_ref": charge.Reference, "paid_at": charge.PaidAt.Format(time.RFC3339)},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) voidInvoice(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	h.transitionInvoice(c, "void")
}

func (h *Handler) refundInvoice(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	h.transitionInvoice(c, "refunded")
}

// transitionInvoice moves an invoice to `to` only from its expected source
// state, mirroring payInvoice's strictness so void/refund are not idempotent
// on already-transitioned invoices.
func (h *Handler) transitionInvoice(c *gin.Context, to string) {
	ctx := c.Request.Context()
	var current string
	err := h.pool.QueryRow(ctx,
		`SELECT status FROM invoices WHERE id = $1::uuid`, c.Param("id")).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "invoice_not_found", "invoice not found")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load invoice")
		return
	}
	source := map[string]string{"void": billing.InvoiceOpen, "refunded": billing.InvoicePaid}[to]
	if current != source {
		writeError(c, http.StatusConflict, "invalid_transition", "invoice is not in the required state for this action")
		return
	}
	if _, err := h.pool.Exec(ctx, `UPDATE invoices SET status = $2 WHERE id = $1::uuid`, c.Param("id"), to); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update invoice")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"id": c.Param("id"), "status": to},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) paymentProviders(c *gin.Context) {
	descriptions := map[string]string{
		"mock":   "Deterministic test provider; charges always succeed unless configured to decline.",
		"manual": "Offline/manual recording; no external call is made.",
	}
	names := billing.GatewayNames()
	providers := make([]gin.H, 0, len(names))
	for _, name := range names {
		providers = append(providers, gin.H{
			"name": name, "description": descriptions[name],
		})
	}
	c.JSON(http.StatusOK, gin.H{"data": providers, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) billingSummary(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	var activePlans int
	if err := h.pool.QueryRow(ctx, `SELECT COUNT(*) FROM plans WHERE is_active`).Scan(&activePlans); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load plans")
		return
	}

	counts := map[string]int{
		billing.StatusTrial: 0, billing.StatusActive: 0, billing.StatusGracePeriod: 0,
		billing.StatusPastDue: 0, billing.StatusSuspended: 0, billing.StatusCancelled: 0,
	}
	rows, err := h.pool.Query(ctx, `SELECT status, COUNT(*) FROM subscriptions GROUP BY status`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to tally subscriptions")
		return
	}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			rows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to tally subscriptions")
			return
		}
		counts[status] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to tally subscriptions")
		return
	}

	var mrrMinor int64
	if err := h.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(
			CASE WHEN pl.billing_period = 'yearly' THEN (pl.price_minor::numeric / 12)::bigint ELSE pl.price_minor END
		), 0)::bigint
		FROM subscriptions s JOIN plans pl ON pl.id = s.plan_id
		WHERE s.status IN ('trial', 'active', 'grace_period', 'past_due')`).Scan(&mrrMinor); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to total MRR")
		return
	}

	var outstandingMinor int64
	if err := h.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount_minor), 0) FROM invoices WHERE status = 'open'`).Scan(&outstandingMinor); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to total outstanding")
		return
	}

	type recentInvoice struct {
		ID          string `json:"id"`
		TenantName  string `json:"tenant_name"`
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
		Status      string `json:"status"`
		CreatedAt   string `json:"created_at"`
	}
	recent := make([]recentInvoice, 0)
	invRows, err := h.pool.Query(ctx, `
		SELECT i.id, t.name, i.amount_minor, i.currency, i.status, i.created_at::text
		FROM invoices i JOIN tenants t ON t.id = i.tenant_id
		ORDER BY i.created_at DESC LIMIT 5`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load recent invoices")
		return
	}
	for invRows.Next() {
		var r recentInvoice
		if err := invRows.Scan(&r.ID, &r.TenantName, &r.AmountMinor, &r.Currency, &r.Status, &r.CreatedAt); err != nil {
			invRows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load recent invoices")
			return
		}
		recent = append(recent, r)
	}
	invRows.Close()
	if err := invRows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load recent invoices")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"active_plans":      activePlans,
			"subscriptions":     counts,
			"mrr_minor":         mrrMinor,
			"currency":          defaultCurrency,
			"outstanding_minor": outstandingMinor,
			"providers":         billing.GatewayNames(),
			"recent_invoices":   recent,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}

// jsonFeatures renders a []string as JSON bytes for a jsonb column.
func jsonFeatures(features []string) []byte {
	if features == nil {
		features = []string{}
	}
	encoded, err := json.Marshal(features)
	if err != nil {
		return []byte("[]")
	}
	return encoded
}
