package saas

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const platformRole = "saas_admin"

type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/saas/summary", h.requirePlatformRole, h.summary)
	router.GET("/saas/tenants", h.requirePlatformRole, h.listTenants)
	router.GET("/saas/tenants/:id/analytics", h.requirePlatformRole, h.tenantAnalytics)
}

// summary returns platform-wide counts broken down by business type and country.
// RLS is force-enabled on tenant-scoped tables, so we aggregate per tenant by
// setting app.current_tenant for each tenant within a single transaction.
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
	defer tx.Rollback(ctx)

	type tenantRow struct {
		ID   string
		Type string
	}
	tenantRows, err := tx.Query(ctx, `SELECT id, business_type FROM tenants ORDER BY business_type`)
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
	for _, t := range tenants {
		if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, t.ID); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
			return
		}
		var users int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to count users")
			return
		}
		totalUsers += users
		var sales int64
		var revenue int64
		if err := tx.QueryRow(ctx, `SELECT COUNT(*), COALESCE(SUM(total_minor), 0) FROM sales`).Scan(&sales, &revenue); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to total sales")
			return
		}
		totalSales += sales
		revenueMinor += revenue
	}

	typeCount := make(map[string]int, len(tenants))
	for _, t := range tenants {
		typeCount[t.Type]++
	}
	byType := make([]gin.H, 0, len(typeCount))
	for bType, count := range typeCount {
		byType = append(byType, gin.H{"business_type": bType, "tenants": count})
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

	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer tx.Rollback(ctx)

	var total int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count tenants")
		return
	}

	rows, err := tx.Query(ctx, `
		SELECT id, name, slug, business_type, country_code, currency_code, default_language, plan, max_users, max_products
		FROM tenants ORDER BY name LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenants")
		return
	}
	type tenantInfo struct {
		id, name, slug, bType, country, currency, lang, plan string
		maxUsers, maxProducts                                int
	}
	list := make([]tenantInfo, 0)
	for rows.Next() {
		var t tenantInfo
		if err := rows.Scan(&t.id, &t.name, &t.slug, &t.bType, &t.country, &t.currency, &t.lang, &t.plan, &t.maxUsers, &t.maxProducts); err != nil {
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
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&users); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to count users")
			return
		}
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&products); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to count products")
			return
		}
		tenants = append(tenants, gin.H{
			"id": t.id, "name": t.name, "slug": t.slug, "business_type": t.bType,
			"country_code": t.country, "currency_code": t.currency, "default_language": t.lang,
			"plan": t.plan, "max_users": t.maxUsers, "max_products": t.maxProducts,
			"users": users, "products": products,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tenants,
		"meta": gin.H{"request_id": c.GetString("request_id"), "page": page, "limit": limit, "total": total},
	})
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
	defer tx.Rollback(ctx)

	var name, slug, bType, country, currency, lang, plan string
	var maxUsers, maxProducts int
	err = tx.QueryRow(ctx, `
		SELECT name, slug, business_type, country_code, currency_code, default_language, plan, max_users, max_products
		FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&name, &slug, &bType, &country, &currency, &lang, &plan, &maxUsers, &maxProducts)
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
		FROM sales s WHERE created_at::date = CURRENT_DATE`).
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
		WHERE created_at::date >= CURRENT_DATE - 6
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
		SELECT product_name, sku, SUM(quantity)::bigint, SUM(total_minor)
		FROM sale_items
		WHERE created_at::date >= CURRENT_DATE - 6
		GROUP BY product_name, sku
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
				"plan": plan, "max_users": maxUsers, "max_products": maxProducts,
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

func (h *Handler) requirePlatformRole(c *gin.Context) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is required")
		c.Abort()
		return
	}
	claims, err := h.tokens.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), security.AccessToken)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		c.Abort()
		return
	}
	if claims.Role != platformRole {
		writeError(c, http.StatusForbidden, "forbidden", "saas_admin role is required")
		c.Abort()
		return
	}
	c.Next()
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
