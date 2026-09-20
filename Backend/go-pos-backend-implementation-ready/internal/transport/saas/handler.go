package saas

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/example/pos-api/internal/transport/access"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	admin := httptransport.RequireSaasAdmin(h.tokens)
	router.GET("/summary", admin, h.summary)
	router.GET("/tenants", admin, h.listTenants)
	router.POST("/tenants", admin, h.createTenant)
	router.PATCH("/tenants/:id", admin, h.updateTenant)
	router.GET("/tenants/:id/analytics", admin, h.tenantAnalytics)
	router.GET("/users", admin, h.listUsers)
	router.POST("/tenants/:id/users", admin, h.createUser)
}

var (
	validTenantBusinessTypes = map[string]bool{
		"coffee_shop": true, "restaurant": true, "retail": true, "book_store": true,
		"mobile_shop": true, "computer_shop": true, "grocery": true, "bakery": true,
		"shawerma": true, "falafel": true, "pharmacy": true, "butcher": true,
		"fruits_veg": true, "clothing": true, "sweets": true, "jewelry": true,
		"hardware": true,
	}
	slugClean = regexp.MustCompile(`[^a-z0-9]+`)
)

// summary returns platform-wide counts broken down by business type and country.
// RLS is force-enabled on tenant-scoped tables, so we aggregate per tenant by
// setting app.current_tenant for each tenant within a single transaction. The
// internal platform tenant (slug 'saas', used by the SaaS staff roles) is
// excluded from every aggregate; only real merchants are counted. Revenue and
// sales count only completed sales (voided/refunded/draft sales are not income).
func (h *Handler) summary(c *gin.Context) {
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

	type tenantRow struct {
		ID   string
		Type string
	}
	tenantRows, err := tx.Query(ctx, `SELECT id, business_type FROM tenants WHERE slug <> 'saas' ORDER BY business_type`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenants")
		return
	}
	tenants := make([]tenantRow, 0)
	for tenantRows.Next() {
		var t tenantRow
		if err := tenantRows.Scan(&t.ID, &t.Type); err != nil {
			tenantRows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenants")
			return
		}
		tenants = append(tenants, t)
	}
	tenantRows.Close()
	if err := tenantRows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenants")
		return
	}

	totalUsers := 0
	var totalSales, revenueMinor int64
	typeUsers := make(map[string]int, len(tenants))
	typeRevenue := make(map[string]int64, len(tenants))
	for _, t := range tenants {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, t.ID); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
			return
		}
		var users int
		var sales int64
		var revenue int64
		if err := tx.QueryRow(ctx, `
			SELECT (SELECT COUNT(*) FROM users),
			       (SELECT COUNT(*) FROM sales WHERE status = 'completed'),
			       COALESCE((SELECT SUM(total_minor) FROM sales WHERE status = 'completed'), 0)
		`).Scan(&users, &sales, &revenue); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to tally tenant")
			return
		}
		totalUsers += users
		typeUsers[t.Type] += users
		totalSales += sales
		revenueMinor += revenue
		typeRevenue[t.Type] += revenue
	}

	typeCount := make(map[string]int, len(tenants))
	for _, t := range tenants {
		typeCount[t.Type]++
	}
	byType := make([]gin.H, 0, len(typeCount))
	for bType, count := range typeCount {
		byType = append(byType, gin.H{
			"business_type": bType,
			"tenants":       count,
			"users":         typeUsers[bType],
			"revenue_minor": typeRevenue[bType],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"total_tenants":      len(tenants),
			"total_users":        totalUsers,
			"total_sales":        totalSales,
			"revenue_minor":      revenueMinor,
			"by_business":        byType,
			"supported_currency": "EGP",
			"supported_country":  "EG",
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

// listTenants returns a paginated list of tenants for the SaaS control panel.
// Filters: q (name/slug contains), business_type, plan, status. The internal
// platform tenant (slug 'saas') is hidden unless include_internal=1. Each row
// carries live user/product counts and completed-only revenue plus sales, all
// aggregated inside a tenant-scoped RLS context.
func (h *Handler) listTenants(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := int64((page - 1) * limit)
	ctx := c.Request.Context()

	q := strings.TrimSpace(c.Query("q"))
	bType := c.Query("business_type")
	plan := c.Query("plan")
	status := c.Query("status")
	includeInternal := c.Query("include_internal") == "1"

	where := make([]string, 0, 5)
	args := make([]any, 0, 6)
	if !includeInternal {
		where = append(where, "slug <> 'saas'")
	}
	if q != "" {
		args = append(args, "%"+q+"%")
		where = append(where, fmt.Sprintf("(name ILIKE $%d OR slug ILIKE $%d)", len(args), len(args)))
	}
	if bType != "" {
		args = append(args, bType)
		where = append(where, fmt.Sprintf("business_type = $%d", len(args)))
	}
	if plan != "" {
		args = append(args, plan)
		where = append(where, fmt.Sprintf("plan = $%d", len(args)))
	}
	if status != "" {
		args = append(args, status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	whereSQL := ""
	if len(where) > 0 {
		whereSQL = " WHERE " + strings.Join(where, " AND ")
	}

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var total int64
	countQ := `SELECT COUNT(*) FROM tenants` + whereSQL
	if err := tx.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count tenants")
		return
	}

	listArgs := append(append([]any{}, args...), limit, offset)
	rows, err := tx.Query(ctx, `
		SELECT id, name, slug, business_type, country_code, currency_code, default_language, plan,
		       max_users, max_products, status, created_at, trial_ends_at,
		       COALESCE(owner_user_id::text, '')
		FROM tenants`+whereSQL+` ORDER BY created_at DESC, name LIMIT $`+strconv.Itoa(len(args)+1)+` OFFSET $`+strconv.Itoa(len(args)+2), listArgs...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenants")
		return
	}
	type tenantInfo struct {
		id, name, slug, bType, country, currency, lang, plan, status, ownerUserID string
		maxUsers, maxProducts                                                     int
		createdAt, trialEndsAt                                                    *time.Time
	}
	list := make([]tenantInfo, 0)
	for rows.Next() {
		var t tenantInfo
		if err := rows.Scan(&t.id, &t.name, &t.slug, &t.bType, &t.country, &t.currency, &t.lang, &t.plan,
			&t.maxUsers, &t.maxProducts, &t.status, &t.createdAt, &t.trialEndsAt, &t.ownerUserID); err != nil {
			rows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenants")
			return
		}
		list = append(list, t)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenants")
		return
	}

	tenants := make([]gin.H, 0, len(list))
	for _, t := range list {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, t.id); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
			return
		}
		var users, products int
		var totalSales int64
		var revenue int64
		if err := tx.QueryRow(ctx, `
			SELECT (SELECT COUNT(*) FROM users),
			       (SELECT COUNT(*) FROM products),
			       (SELECT COUNT(*) FROM sales WHERE status = 'completed'),
			       COALESCE((SELECT SUM(total_minor) FROM sales WHERE status = 'completed'), 0)
		`).Scan(&users, &products, &totalSales, &revenue); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to tally tenant")
			return
		}
		tenants = append(tenants, gin.H{
			"id": t.id, "name": t.name, "slug": t.slug, "business_type": t.bType,
			"country_code": t.country, "currency_code": t.currency, "default_language": t.lang,
			"plan": t.plan, "max_users": t.maxUsers, "max_products": t.maxProducts,
			"status": t.status, "users": users, "products": products,
			"owner_user_id": t.ownerUserID, "created_at": fmtTime(t.createdAt), "trial_ends_at": fmtTime(t.trialEndsAt),
			"revenue_minor": revenue, "total_sales": totalSales,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tenants,
		"meta": gin.H{"request_id": c.GetString("request_id"), "page": page, "limit": limit, "total": total},
	})
}

// createTenant provisions a brand-new store shell from the control planel:
// the tenant row only (slug auto-generated, plan defaults to 'standard',
// status 'active'). Users, catalog and subscriptions are added later by the
// store owner or the platform staff. saas_admin only.
func (h *Handler) createTenant(c *gin.Context) {
	var req struct {
		Name            string `json:"name" binding:"required"`
		BusinessType    string `json:"business_type" binding:"required"`
		CountryCode     string `json:"country_code"`
		CurrencyCode    string `json:"currency_code"`
		DefaultLanguage string `json:"default_language"`
		Plan            string `json:"plan"`
		MaxUsers        int    `json:"max_users"`
		MaxProducts     int    `json:"max_products"`
		Status          string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid tenant payload")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "name is required")
		return
	}
	if !validTenantBusinessTypes[req.BusinessType] {
		writeError(c, http.StatusBadRequest, "validation_error", "unsupported business_type")
		return
	}
	country, currency, language := tenantDefaults(req.CountryCode, req.CurrencyCode, req.DefaultLanguage)
	if req.Plan == "" {
		req.Plan = "standard"
	}
	if req.Status == "" {
		req.Status = "active"
	}
	if req.Status != "active" && req.Status != "suspended" {
		writeError(c, http.StatusBadRequest, "validation_error", "status must be active or suspended")
		return
	}
	if req.MaxUsers < 0 || req.MaxProducts < 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "limits must not be negative")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	id := uuid.New()
	_, err := h.pool.Exec(ctx, `
		INSERT INTO tenants (id, name, slug, business_type, country_code, currency_code,
		                     default_language, plan, max_users, max_products, status, is_demo_seeded)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, FALSE)`,
		id, req.Name, tenantSlug(req.Name), req.BusinessType, country, currency, language,
		req.Plan, req.MaxUsers, req.MaxProducts, req.Status)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(c, http.StatusConflict, "tenant_conflict", "a store with this name/slug already exists")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create tenant")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": tenantShell(id.String(), req.Name, tenantSlug(req.Name), req.BusinessType,
			country, currency, language, req.Plan, req.MaxUsers, req.MaxProducts, req.Status, nil),
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

type updateTenantRequest struct {
	Name            *string `json:"name"`
	BusinessType    *string `json:"business_type"`
	CountryCode     *string `json:"country_code"`
	CurrencyCode    *string `json:"currency_code"`
	DefaultLanguage *string `json:"default_language"`
	Plan            *string `json:"plan"`
	MaxUsers        *int    `json:"max_users"`
	MaxProducts     *int    `json:"max_products"`
	Status          *string `json:"status"`
}

// updateTenant edits a store's plan, quotas and localization profile from the
// control panel — the "manage the user's plan" workhorse. Only the fields that
// are present in the payload change; nothing else is touched.
func (h *Handler) updateTenant(c *gin.Context) {
	var req updateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid tenant payload")
		return
	}
	if req.BusinessType != nil && !validTenantBusinessTypes[*req.BusinessType] {
		writeError(c, http.StatusBadRequest, "validation_error", "unsupported business_type")
		return
	}
	if req.Status != nil && *req.Status != "active" && *req.Status != "suspended" {
		writeError(c, http.StatusBadRequest, "validation_error", "status must be active or suspended")
		return
	}
	if (req.MaxUsers != nil && *req.MaxUsers < 0) || (req.MaxProducts != nil && *req.MaxProducts < 0) {
		writeError(c, http.StatusBadRequest, "validation_error", "limits must not be negative")
		return
	}
	if req.CurrencyCode != nil && (len(*req.CurrencyCode) < 2 || len(*req.CurrencyCode) > 3) {
		writeError(c, http.StatusBadRequest, "validation_error", "currency_code must be a 2 or 3-letter code")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tag, err := h.pool.Exec(ctx, `
		UPDATE tenants SET
			name             = COALESCE($2, name),
			business_type    = COALESCE($3, business_type),
			country_code     = COALESCE($4, country_code),
			currency_code    = COALESCE($5, currency_code),
			default_language = COALESCE($6, default_language),
			plan             = COALESCE($7, plan),
			max_users        = COALESCE($8, max_users),
			max_products     = COALESCE($9, max_products),
			status           = COALESCE($10, status),
			updated_at       = now()
		WHERE id = $1::uuid`,
		c.Param("id"), req.Name, req.BusinessType, req.CountryCode, req.CurrencyCode,
		req.DefaultLanguage, req.Plan, req.MaxUsers, req.MaxProducts, req.Status)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update tenant")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found")
		return
	}
	shell, found, err := h.loadTenantShell(ctx, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to reload tenant")
		return
	}
	if !found {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": shell, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

// loadTenantShell reads one tenant's shell fields (no usage aggregates).
func (h *Handler) loadTenantShell(ctx context.Context, id string) (tenantShellView, bool, error) {
	var row struct {
		id, name, slug, bType, country, currency, lang, plan, status string
		maxUsers, maxProducts                                        int
		createdAt, trialEndsAt                                       *time.Time
	}
	err := h.pool.QueryRow(ctx, `
		SELECT id, name, slug, business_type, country_code, currency_code, default_language,
		       plan, max_users, max_products, status, created_at, trial_ends_at
		FROM tenants WHERE id = $1::uuid`, id).
		Scan(&row.id, &row.name, &row.slug, &row.bType, &row.country, &row.currency, &row.lang,
			&row.plan, &row.maxUsers, &row.maxProducts, &row.status, &row.createdAt, &row.trialEndsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return tenantShellView{}, false, nil
	}
	if err != nil {
		return tenantShellView{}, false, err
	}
	return tenantShell(row.id, row.name, row.slug, row.bType, row.country, row.currency,
		row.lang, row.plan, row.maxUsers, row.maxProducts, row.status, row.trialEndsAt), true, nil
}

type tenantShellView = gin.H

func tenantShell(id, name, slug, bType, country, currency, lang, plan string, maxUsers, maxProducts int, status string, createdAt *time.Time) tenantShellView {
	return tenantShellView{
		"id": id, "name": name, "slug": slug, "business_type": bType,
		"country_code": country, "currency_code": currency, "default_language": lang,
		"plan": plan, "max_users": maxUsers, "max_products": maxProducts, "status": status,
		"created_at": fmtTime(createdAt),
	}
}

func tenantDefaults(country, currency, language string) (string, string, string) {
	if strings.TrimSpace(country) == "" {
		country = "EG"
	}
	if strings.TrimSpace(currency) == "" {
		currency = "EGP"
	}
	if strings.TrimSpace(language) == "" {
		language = "ar"
	}
	return country, currency, language
}

func tenantSlug(name string) string {
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

// tenantAnalytics is the per-tenant drill-down for the SaaS control panel:
// plan/resource facts, user & product counts, today's stats, a 7-day revenue
// trend, top products, and recent sales. Everything that touches tenant data
// runs inside one transaction with app.current_tenant set for that tenant
// (RLS FORCE would otherwise hide every row).
func (h *Handler) tenantAnalytics(c *gin.Context) {
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

	var name, slug, bType, country, currency, lang, plan, status string
	var maxUsers, maxProducts int
	var createdAt, trialEndsAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT name, slug, business_type, country_code, currency_code, default_language, plan, max_users, max_products, status, created_at, trial_ends_at
		FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&name, &slug, &bType, &country, &currency, &lang, &plan, &maxUsers, &maxProducts, &status, &createdAt, &trialEndsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenant")
		return
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID.String()); err != nil {
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

	var date string
	var todayRevenue, todaySales, todayItems int64
	if err := tx.QueryRow(ctx, `SELECT CURRENT_DATE::text`).Scan(&date); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load date")
		return
	}
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_minor), 0), COUNT(*),
		       COALESCE(SUM((SELECT COALESCE(SUM(quantity), 0) FROM sale_items si WHERE si.sale_id = s.id)), 0)
		FROM sales s WHERE status = 'completed' AND created_at::date = CURRENT_DATE`).
		Scan(&todayRevenue, &todaySales, &todayItems); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to tally today")
		return
	}
	todayAvg := int64(0)
	if todaySales > 0 {
		todayAvg = todayRevenue / todaySales
	}

	type trendPoint struct {
		Day          string `json:"day"`
		RevenueMinor int64  `json:"revenue_minor"`
	}
	trend := make([]trendPoint, 0)
	trendRows, err := tx.Query(ctx, `
		SELECT (created_at::date)::text AS day, COALESCE(SUM(total_minor), 0)
		FROM sales
		WHERE status = 'completed' AND created_at::date >= CURRENT_DATE - 6
		GROUP BY created_at::date
		ORDER BY created_at::date
		LIMIT 7`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load revenue trend")
		return
	}
	for trendRows.Next() {
		var p trendPoint
		if err := trendRows.Scan(&p.Day, &p.RevenueMinor); err != nil {
			trendRows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load revenue trend")
			return
		}
		trend = append(trend, p)
	}
	trendRows.Close()
	if err := trendRows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load revenue trend")
		return
	}

	type topProduct struct {
		ProductName  string `json:"product_name"`
		SKU          string `json:"sku"`
		Quantity     int64  `json:"quantity"`
		RevenueMinor int64  `json:"revenue_minor"`
	}
	topProducts := make([]topProduct, 0)
	topRows, err := tx.Query(ctx, `
		SELECT si.product_name, si.sku, SUM(si.quantity)::bigint, SUM(si.total_minor)
		FROM sale_items si
		JOIN sales s ON s.id = si.sale_id AND s.status = 'completed'
		WHERE si.created_at::date >= CURRENT_DATE - 6
		GROUP BY si.product_name, si.sku
		ORDER BY 3 DESC
		LIMIT 5`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load top products")
		return
	}
	for topRows.Next() {
		var p topProduct
		if err := topRows.Scan(&p.ProductName, &p.SKU, &p.Quantity, &p.RevenueMinor); err != nil {
			topRows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load top products")
			return
		}
		topProducts = append(topProducts, p)
	}
	topRows.Close()
	if err := topRows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load top products")
		return
	}

	type recentSale struct {
		ID            string `json:"id"`
		Status        string `json:"status"`
		TotalMinor    int64  `json:"total_minor"`
		Currency      string `json:"currency"`
		PaymentMethod string `json:"payment_method"`
		Cashier       string `json:"cashier"`
		CreatedAt     string `json:"created_at"`
	}
	recentSales := make([]recentSale, 0)
	saleRows, err := tx.Query(ctx, `
		SELECT s.id, s.status, s.total_minor, s.currency, COALESCE(s.payment_method, 'cash'),
		       COALESCE(u.display_name, ''), s.created_at::text
		FROM sales s
		LEFT JOIN users u ON u.tenant_id = s.tenant_id AND u.id = s.created_by
		ORDER BY s.created_at DESC
		LIMIT 5`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load recent sales")
		return
	}
	for saleRows.Next() {
		var s recentSale
		if err := saleRows.Scan(&s.ID, &s.Status, &s.TotalMinor, &s.Currency, &s.PaymentMethod, &s.Cashier, &s.CreatedAt); err != nil {
			saleRows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load recent sales")
			return
		}
		recentSales = append(recentSales, s)
	}
	saleRows.Close()
	if err := saleRows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load recent sales")
		return
	}

	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to finalize analytics")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"tenant": gin.H{
				"id": tenantID.String(), "name": name, "slug": slug, "business_type": bType,
				"country_code": country, "currency_code": currency, "default_language": lang,
				"plan": plan, "max_users": maxUsers, "max_products": maxProducts, "status": status,
				"created_at": fmtTime(createdAt), "trial_ends_at": fmtTime(trialEndsAt),
			},
			"counts":        gin.H{"users": users, "products": products},
			"today":         gin.H{"date": date, "revenue_minor": todayRevenue, "sales_count": todaySales, "avg_sale_minor": todayAvg, "items_sold": todayItems},
			"revenue_trend": trend,
			"top_products":  topProducts,
			"recent_sales":  recentSales,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

// listUsers returns every user across all merchant tenants (or just one tenant
// when ?tenant_id= is given). Each user row carries its owning tenant so the
// control panel can manage staff per store. Users are RLS-scoped, so each
// tenant is aggregated inside its own app.current_tenant context within one tx.
func (h *Handler) listUsers(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	tenantFilter := strings.TrimSpace(c.Query("tenant_id"))
	if tenantFilter != "" {
		if _, err := uuid.Parse(tenantFilter); err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "tenant_id is invalid")
			return
		}
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tenantRows, err := tx.Query(ctx, `
		SELECT id, name FROM tenants WHERE slug <> 'saas'
		AND ($1 = '' OR id::text = $1) ORDER BY name`, tenantFilter)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenants")
		return
	}
	type tenantPair struct{ id, name string }
	tenants := make([]tenantPair, 0)
	for tenantRows.Next() {
		var t tenantPair
		if err := tenantRows.Scan(&t.id, &t.name); err != nil {
			tenantRows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenants")
			return
		}
		tenants = append(tenants, t)
	}
	tenantRows.Close()
	if err := tenantRows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenants")
		return
	}

	type userView struct {
		ID, TenantID, TenantName, Email, DisplayName, Role, AccountType string
		IsActive                                                        bool
		AccessLevel                                                     string
	}
	users := make([]userView, 0)
	for _, t := range tenants {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, t.id); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
			return
		}
		rows, err := tx.Query(ctx, `
			SELECT u.id, u.email, u.display_name, u.role, u.account_type, u.is_active,
			       COALESCE(s.access_level, '')
			FROM users u
			LEFT JOIN users_pos_security s ON s.tenant_id = u.tenant_id AND s.user_id = u.id
			ORDER BY u.display_name`)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load users")
			return
		}
		for rows.Next() {
			var u userView
			u.TenantID = t.id
			u.TenantName = t.name
			if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.AccountType,
				&u.IsActive, &u.AccessLevel); err != nil {
				rows.Close()
				writeError(c, http.StatusInternalServerError, "internal_error", "unable to load users")
				return
			}
			users = append(users, u)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load users")
			return
		}
	}
	sort.Slice(users, func(i, j int) bool {
		if users[i].TenantName != users[j].TenantName {
			return users[i].TenantName < users[j].TenantName
		}
		return users[i].DisplayName < users[j].DisplayName
	})
	total := len(users)
	start := (page - 1) * limit
	end := start + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	out := make([]gin.H, 0, end-start)
	for _, u := range users[start:end] {
		out = append(out, gin.H{
			"id": u.ID, "tenant_id": u.TenantID, "tenant_name": u.TenantName,
			"email": u.Email, "display_name": u.DisplayName, "role": u.Role,
			"account_type": u.AccountType, "is_active": u.IsActive, "access_level": u.AccessLevel,
		})
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load users")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": out,
		"meta": gin.H{"request_id": c.GetString("request_id"), "page": page, "limit": limit, "total": total},
	})
}

type createSassUserRequest struct {
	Email       string `json:"email" binding:"required,email"`
	DisplayName string `json:"display_name" binding:"required"`
	Password    string `json:"password" binding:"required,min=8"`
	Role        string `json:"role" binding:"required,oneof=owner manager cashier"`
	AccountType string `json:"account_type" binding:"omitempty,oneof=standard demo guest"`
}

// createUser provisions a staff user inside a specific merchant store. It runs
// inside the store's RLS context, honors its plan user cap, and seeds the POS
// security profile exactly like the tenant-scoped POST /v1/users endpoint.
func (h *Handler) createUser(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	var req createSassUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid user payload")
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

	var tenantName string
	if err := tx.QueryRow(ctx, `SELECT name FROM tenants WHERE id = $1::uuid`, tenantID.String()).Scan(&tenantName); errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "tenant_not_found", "tenant not found")
		return
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenant")
		return
	}
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var maxUsers int
	if err := tx.QueryRow(ctx, `SELECT max_users FROM tenants WHERE id = $1::uuid`, tenantID.String()).Scan(&maxUsers); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenant plan")
		return
	}
	if maxUsers > 0 {
		var existing int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&existing); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to count users")
			return
		}
		if existing >= maxUsers {
			writeError(c, http.StatusConflict, "plan_limit_exceeded", "tenant user limit reached")
			return
		}
	}

	accountType := req.AccountType
	if accountType == "" {
		accountType = "standard"
	}
	passwordHash, err := security.HashPassword(req.Password, 10)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to hash password")
		return
	}

	type userRow struct {
		ID, Email, DisplayName, Role, AccountType string
		IsActive                                  bool
	}
	var u userRow
	err = tx.QueryRow(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)
		RETURNING id, email, display_name, role, account_type, is_active`,
		tenantID.String(), strings.TrimSpace(strings.ToLower(req.Email)),
		passwordHash, strings.TrimSpace(req.DisplayName), req.Role, accountType).Scan(
		&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.AccountType, &u.IsActive)
	if err != nil {
		writeError(c, http.StatusConflict, "user_conflict", "email is already in use in this store")
		return
	}
	level := access.DefaultLevelForRole(u.Role)
	if _, err := tx.Exec(ctx, `
		INSERT INTO users_pos_security (tenant_id, user_id, access_level)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO NOTHING`,
		tenantID.String(), u.ID, level); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to provision user security")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create user")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"id": u.ID, "tenant_id": tenantID.String(), "tenant_name": tenantName,
			"email": u.Email, "display_name": u.DisplayName, "role": u.Role,
			"account_type": u.AccountType, "is_active": u.IsActive, "access_level": level,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
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
