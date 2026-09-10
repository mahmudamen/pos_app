package saas

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
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
		SELECT id, name, slug, business_type, country_code, currency_code, default_language
		FROM tenants ORDER BY name LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenants")
		return
	}
	type tenantInfo struct {
		id, name, slug, bType, country, currency, lang string
	}
	list := make([]tenantInfo, 0)
	for rows.Next() {
		var t tenantInfo
		if err := rows.Scan(&t.id, &t.name, &t.slug, &t.bType, &t.country, &t.currency, &t.lang); err != nil {
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
			"users": users, "products": products,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": tenants,
		"meta": gin.H{"request_id": c.GetString("request_id"), "page": page, "limit": limit, "total": total},
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
