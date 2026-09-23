package platform

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler exposes the SaaS-platform trial-management surface. Every route is
// behind the exact saas_admin role; every mutating call writes an audit row.
type Handler struct {
	pool     *pgxpool.Pool
	tokens   security.TokenManager
	cfg      config.Config
	accounts *identity.AccountStore
	audit    identity.AuditRecorder
	trials   *identity.TrialStore
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager, cfg config.Config) *Handler {
	audit := identity.NewAuditRecorder()
	return &Handler{
		pool:     pool,
		tokens:   tokens,
		cfg:      cfg,
		accounts: identity.NewAccountStore(audit),
		audit:    audit,
		trials:   identity.NewTrialStore(),
	}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	admin := router.Group("/platform", httptransport.RequireSaasAdmin(h.tokens))
	{
		admin.GET("/trial/entitlements", h.listEntitlements)
		admin.GET("/trial/entitlements/:id", h.getEntitlement)
		admin.POST("/trial/entitlements/:id/extend", h.extendEntitlement)
		admin.POST("/trial/entitlements/:id/revoke", h.revokeEntitlement)
		admin.POST("/trial/entitlements/:id/convert", h.convertEntitlement)
		admin.POST("/trial/grant", h.manualGrant)
		admin.GET("/trial/settings", h.getSettings)
		admin.PUT("/trial/settings", h.putSettings)
		admin.GET("/audit", h.listAudit)
		admin.POST("/accounts/:id/suspend", h.suspendAccount)
		admin.POST("/tenants/:id/suspend", h.suspendTenant)
		admin.POST("/tenants/:id/activate", h.activateTenant)
		admin.POST("/tenants/:id/stop", h.stopTenant)
		admin.POST("/tenants/:id/backup", h.backupTenant)
	}
}

func (h *Handler) listEntitlements(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	items, err := h.trials.List(c.Request.Context(), tx, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load entitlements")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"entitlements": entitlementPayloads(items)}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) getEntitlement(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	item, err := h.trials.GetByID(c.Request.Context(), tx, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, "entitlement_not_found", "entitlement not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": entitlementPayloads([]identity.TrialEntitlement{*item})[0], "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) extendEntitlement(c *gin.Context) {
	var req struct {
		ExtraDays int `json:"extra_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.ExtraDays < 1 {
		writeError(c, http.StatusBadRequest, "validation_error", "extra_days must be a positive integer")
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
	claims, _ := httptransport.Claims(c)
	updated, err := h.trials.Extend(ctx, tx, c.Param("id"), req.ExtraDays)
	if err != nil {
		writeError(c, http.StatusNotFound, "entitlement_not_found", "entitlement not found or not extendable")
		return
	}
	if err := h.audit.Record(ctx, tx, identity.AuditEntry{
		Action:      identity.ActionTrialExtended,
		ActorUserID: claims.UserID,
		TenantID:    claims.TenantID,
		EntityID:    updated.ID,
		EntityType:  identity.EntityTrial,
		Reason:      "admin extend by " + strconv.Itoa(req.ExtraDays) + " days",
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record audit")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to extend trial")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": entitlementPayloads([]identity.TrialEntitlement{*updated})[0], "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) revokeEntitlement(c *gin.Context) {
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
	claims, _ := httptransport.Claims(c)
	updated, err := h.trials.Revoke(ctx, tx, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, "entitlement_not_found", "entitlement not found")
		return
	}
	if err := h.audit.Record(ctx, tx, identity.AuditEntry{
		Action:      identity.ActionTrialRevoked,
		ActorUserID: claims.UserID,
		TenantID:    claims.TenantID,
		EntityID:    updated.ID,
		EntityType:  identity.EntityTrial,
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record audit")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to revoke trial")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": entitlementPayloads([]identity.TrialEntitlement{*updated})[0], "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) convertEntitlement(c *gin.Context) {
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
	claims, _ := httptransport.Claims(c)
	updated, err := h.trials.Convert(ctx, tx, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, "entitlement_not_found", "entitlement not found or not convertible")
		return
	}
	if err := h.audit.Record(ctx, tx, identity.AuditEntry{
		Action:      identity.ActionTrialConverted,
		ActorUserID: claims.UserID,
		TenantID:    claims.TenantID,
		EntityID:    updated.ID,
		EntityType:  identity.EntityTrial,
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record audit")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to convert trial")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": entitlementPayloads([]identity.TrialEntitlement{*updated})[0], "meta": gin.H{"request_id": c.GetString("request_id")}})
}

// manualGrant is the support/override grant: it bypasses the eligibility
// policy on purpose (audited), projects the tenant, and links the account.
func (h *Handler) manualGrant(c *gin.Context) {
	var req struct {
		AccountID      string `json:"account_id"`
		OrganizationID string `json:"organization_id"`
		Days           int    `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "account_id, organization_id and days are required")
		return
	}
	if req.Days < 1 || req.Days > 3650 {
		writeError(c, http.StatusBadRequest, "validation_error", "days must be between 1 and 3650")
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
	// Validate existence up-front; org_live_uq catches a second live grant.
	var ownerUserID string
	if err := tx.QueryRow(ctx, `SELECT owner_user_id::text FROM tenants WHERE id = $1::uuid`, req.OrganizationID).Scan(&ownerUserID); err != nil {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found")
		return
	}
	if _, err := h.accounts.GetByID(ctx, tx, req.AccountID); err != nil {
		writeError(c, http.StatusNotFound, "account_not_found", "account not found")
		return
	}
	claims, _ := httptransport.Claims(c)
	now := time.Now().UTC()
	entitlement, err := identity.GrantTrial(ctx, tx, identity.TrialPolicy{DurationDays: req.Days}, identity.GrantTrialInput{
		AccountID:      req.AccountID,
		TenantID:       req.OrganizationID,
		OwnerUserID:    ownerUserID,
		TrialType:      "standard",
		EligibilityKey: identity.EligibilityKeyFor(identity.ScopeOrganization, req.OrganizationID),
		Status:         identity.TrialStatusActive,
		DurationDays:   req.Days,
		Source:         "admin_grant",
		Reason:         "manual admin grant",
	}, now)
	if err != nil {
		writeError(c, http.StatusConflict, "trial_conflict", err.Error())
		return
	}
	if err := h.audit.Record(ctx, tx, identity.AuditEntry{
		Action:      identity.ActionManualTrialOverride,
		ActorUserID: claims.UserID,
		TenantID:    claims.TenantID,
		EntityID:    entitlement.ID,
		EntityType:  identity.EntityTrial,
		Reason:      "manual grant for org " + req.OrganizationID,
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record audit")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to grant trial")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": entitlementPayloads([]identity.TrialEntitlement{*entitlement})[0], "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) getSettings(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	policy, err := identity.LoadPolicy(c.Request.Context(), h.pool, h.cfg.Trial)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load settings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": identity.PolicyJSON(policy), "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) putSettings(c *gin.Context) {
	var raw map[string]any
	if err := c.ShouldBindJSON(&raw); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid settings payload")
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
	// Start from the env-derived defaults and layer the admin patch on top.
	policy, err := identity.LoadPolicy(ctx, tx, h.cfg.Trial)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load policy")
		return
	}
	patch, err := json.Marshal(raw)
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid policy values")
		return
	}
	if err := identity.OverlayPolicyJSON(&policy, patch); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid policy values")
		return
	}
	if err := identity.SavePolicy(ctx, tx, policy); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	claims, _ := httptransport.Claims(c)
	if err := h.audit.Record(ctx, tx, identity.AuditEntry{
		Action:      identity.ActionSubscriptionChanged,
		ActorUserID: claims.UserID,
		TenantID:    claims.TenantID,
		EntityType:  identity.EntityTrial,
		Reason:      "trial_settings updated by admin",
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record audit")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to persist settings")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": identity.PolicyJSON(policy), "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) listAudit(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit < 1 || limit > 500 {
		limit = 100
	}
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	rows, err := h.pool.Query(c.Request.Context(), `
		SELECT id::text, created_at, COALESCE(actor_user_id::text, ''), COALESCE(account_id::text, ''),
		       COALESCE(tenant_id::text, ''), action, entity_type, COALESCE(entity_id::text, ''),
		       reason, COALESCE(ip::text, '')
		FROM audit_log ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load audit")
		return
	}
	defer rows.Close()
	type entry struct {
		ID, CreatedAt, ActorID, AccountID, TenantID, Action, EntityType, EntityID, Reason, IP string
	}
	items := make([]entry, 0, limit)
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.ID, &e.CreatedAt, &e.ActorID, &e.AccountID, &e.TenantID,
			&e.Action, &e.EntityType, &e.EntityID, &e.Reason, &e.IP); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load audit")
			return
		}
		items = append(items, e)
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"entries": items}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) suspendAccount(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { /* reason optional */
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
	claims, _ := httptransport.Claims(c)
	account, err := h.accounts.GetByID(ctx, tx, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusNotFound, "account_not_found", "account not found")
		return
	}
	if err := h.accounts.Suspend(ctx, tx, account.ID, req.Reason); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to suspend account")
		return
	}
	if err := h.audit.Record(ctx, tx, identity.AuditEntry{
		Action:      identity.ActionAccountSuspended,
		ActorUserID: claims.UserID,
		AccountID:   account.ID,
		TenantID:    claims.TenantID,
		EntityID:    account.ID,
		EntityType:  identity.EntityAccount,
		Reason:      req.Reason,
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record audit")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to suspend account")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"suspended": true, "account_id": account.ID}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) suspendTenant(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { /* reason optional */
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
	claims, _ := httptransport.Claims(c)
	tag, err := tx.Exec(ctx, `
		UPDATE tenants SET status = 'suspended', updated_at = now() WHERE id = $1::uuid AND status <> 'suspended'`,
		c.Param("id"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to suspend tenant")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found or already suspended")
		return
	}
	// Fold the suspension into the subscription row too, so /v1/subscription
	// reports suspended even when the trial projection is still valid.
	if _, err := tx.Exec(ctx, `
		UPDATE subscriptions SET status = 'suspended', updated_at = now()
		WHERE tenant_id = $1::uuid AND status NOT IN ('cancelled')`, c.Param("id")); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to suspend subscription")
		return
	}
	if err := h.audit.Record(ctx, tx, identity.AuditEntry{
		Action:      identity.ActionOrganizationSuspended,
		ActorUserID: claims.UserID,
		TenantID:    c.Param("id"),
		EntityID:    c.Param("id"),
		EntityType:  identity.EntityOrganization,
		Reason:      req.Reason,
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record audit")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to suspend tenant")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"suspended": true, "tenant_id": c.Param("id")}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

// activateTenant reverses a suspension: clears tenants.status back to 'active'
// and lifts any subscription suspension (cancelled subscriptions stay put).
func (h *Handler) activateTenant(c *gin.Context) {
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
	claims, _ := httptransport.Claims(c)
	tag, err := tx.Exec(ctx, `
		UPDATE tenants SET status = 'active', updated_at = now() WHERE id = $1::uuid AND status = 'suspended'`,
		c.Param("id"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to activate tenant")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found or not suspended")
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE subscriptions SET status = 'active', updated_at = now()
		WHERE tenant_id = $1::uuid AND status = 'suspended'`, c.Param("id")); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to activate subscription")
		return
	}
	if err := h.audit.Record(ctx, tx, identity.AuditEntry{
		Action:      identity.ActionOrganizationActivated,
		ActorUserID: claims.UserID,
		TenantID:    c.Param("id"),
		EntityID:    c.Param("id"),
		EntityType:  identity.EntityOrganization,
		Reason:      "admin reactivated tenant",
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record audit")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to activate tenant")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"activated": true, "tenant_id": c.Param("id")}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

// stopTenant is the permanent sibling of suspendTenant: it disables the tenant
// (login returns organization_stopped) and cancels its live subscription(s) so
// billing stops accruing. Data is kept for restore; only access is closed.
func (h *Handler) stopTenant(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil { /* reason optional */
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
	claims, _ := httptransport.Claims(c)
	tag, err := tx.Exec(ctx, `
		UPDATE tenants SET status = 'disabled', updated_at = now() WHERE id = $1::uuid AND status <> 'disabled'`,
		c.Param("id"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to stop tenant")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found or already stopped")
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE subscriptions SET status = 'cancelled', cancelled_at = now(), updated_at = now()
		WHERE tenant_id = $1::uuid AND status NOT IN ('cancelled')`, c.Param("id")); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to cancel subscription")
		return
	}
	if err := h.audit.Record(ctx, tx, identity.AuditEntry{
		Action:      identity.ActionOrganizationStopped,
		ActorUserID: claims.UserID,
		TenantID:    c.Param("id"),
		EntityID:    c.Param("id"),
		EntityType:  identity.EntityOrganization,
		Reason:      req.Reason,
	}); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record audit")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to stop tenant")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"stopped": true, "tenant_id": c.Param("id")}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

// tenantTables is the tenant-scoped business-state catalog mirrored by the sync
// change feed. Every table carries tenant_id + FORCE RLS, so SELECTs below are
// scoped by app.current_tenant and CANNOT read another tenant's rows even by
// owner/superuser. Platform-level tables (plans, subscriptions, invoices,
// accounts, audit_log, countries, currency, ...) have no RLS and are NOT
// enumerated here, so a cross-tenant leak via the backup is impossible by
// construction. client_events is deliberately excluded (observability, not
// business state — matches the sync feed's exclusion).
var tenantTables = []string{
	"categories", "products",
	"product_variants", "product_lots",
	"customers", "customer_loyalty_log",
	"floors", "restaurant_tables",
	"register_sessions", "inventory_adjustments",
	"tenant_settings",
	"sales", "sale_items", "sale_payments", "sale_refunds",
	"users", "users_pos_security",
	"devices", "sessions", "refresh_tokens", "sync_commands",
	"purchases", "purchase_items", "ocr_usage",
	"self_orders", "product_requests",
	"notifications", "notification_push_sub",
}

// backupTenant streams a per-tenant, RLS-scoped JSON export of every
// tenant-scoped table. One JSON document: {"tenant_id", "slug",
// "exported_at", "tables": {name: [rows...]}}. Rows are streamed as
// row_to_json(text) raw bytes, so even large tenants never buffer a whole
// table in memory. The tx runs with app.current_tenant set for the whole
// export; FORCE RLS guarantees only this tenant's rows are visible.
func (h *Handler) backupTenant(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_tenant_id", "tenant id is invalid")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var slug string
	if err := tx.QueryRow(ctx, `SELECT slug FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&slug); err != nil {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found")
		return
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	// Stream the export. The tenant row lookup above validated existence; the
	// handler is saas_admin-gated. Everything below runs under the RLS context
	// just established, so only this tenant's rows are read.
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", `attachment; filename="`+slug+`-backup.json"`)

	if _, err := c.Writer.WriteString(`{"tenant_id":"` + tenantID.String() + `","slug":"` + jsonEscape(slug) +
		`","subdomain":"` + jsonEscape(slug+".xamltech.com") + `","exported_at":"` + time.Now().UTC().Format(time.RFC3339) + `","tables":{`); err != nil {
		return
	}
	for i, table := range tenantTables {
		if i > 0 {
			if _, err := c.Writer.WriteString(`,`); err != nil {
				return
			}
		}
		if _, err := c.Writer.WriteString(`"` + table + `":[`); err != nil {
			return
		}
		rows, err := tx.Query(ctx, `SELECT row_to_json(t)::text FROM `+table+` t`)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to export "+table)
			return
		}
		first := true
		for rows.Next() {
			var rowBytes []byte
			if err := rows.Scan(&rowBytes); err != nil {
				rows.Close()
				writeError(c, http.StatusInternalServerError, "internal_error", "unable to export "+table)
				return
			}
			if !first {
				if _, err := c.Writer.WriteString(`,`); err != nil {
					rows.Close()
					return
				}
			}
			first = false
			if _, err := c.Writer.Write(rowBytes); err != nil {
				rows.Close()
				return
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to export "+table)
			return
		}
		if _, err := c.Writer.WriteString(`]`); err != nil {
			return
		}
	}
	if _, err := c.Writer.WriteString(`}}`); err != nil {
		return
	}
	c.Status(http.StatusOK)
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b[1 : len(b)-1])
}

func entitlementPayloads(items []identity.TrialEntitlement) []gin.H {
	out := make([]gin.H, 0, len(items))
	for _, e := range items {
		out = append(out, gin.H{
			"id":              e.ID,
			"organization_id": e.OrganizationID,
			"owner_user_id":   e.OwnerUserID,
			"account_id":      e.AccountID,
			"trial_type":      e.TrialType,
			"status":          e.Status,
			"started_at":      fmtTime(e.StartedAt),
			"expires_at":      fmtTime(e.ExpiresAt),
			"trial_days":      e.TrialDays,
			"consumed_at":     fmtTime(e.ConsumedAt),
			"source":          e.Source,
			"eligibility_key": e.EligibilityKey,
			"reason":          e.Reason,
		})
	}
	return out
}

func fmtTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
