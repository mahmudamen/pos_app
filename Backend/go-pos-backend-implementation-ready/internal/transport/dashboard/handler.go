package dashboard

import (
	"net/http"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	router.GET("/dashboard/summary", h.summary)
}

type TodayStats struct {
	RevenueMinor int64 `json:"revenue_minor"`
	SalesCount   int64 `json:"sales_count"`
	AvgSaleMinor int64 `json:"avg_sale_minor"`
	ItemsSold    int64 `json:"items_sold"`
}

type TopProduct struct {
	ProductName  string `json:"product_name"`
	SKU          string `json:"sku"`
	Quantity     int64  `json:"quantity"`
	RevenueMinor int64  `json:"revenue_minor"`
}

type CashierStat struct {
	Cashier      string `json:"cashier"`
	SalesCount   int64  `json:"sales_count"`
	RevenueMinor int64  `json:"revenue_minor"`
}

type PaymentMix struct {
	Method      string `json:"method"`
	AmountMinor int64  `json:"amount_minor"`
}

type RecentSale struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	TotalMinor    int64  `json:"total_minor"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method"`
	CreatedAt     string `json:"created_at"`
}

type Summary struct {
	Date        string        `json:"date"`
	Today       TodayStats    `json:"today"`
	TopProducts []TopProduct  `json:"top_products"`
	RecentSales []RecentSale  `json:"recent_sales"`
	PerCashier  []CashierStat `json:"per_cashier"`
	PaymentMix  []PaymentMix  `json:"payment_mix"`
}

func (h *Handler) summary(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	summary := Summary{Today: TodayStats{}, TopProducts: []TopProduct{}, RecentSales: []RecentSale{}, PerCashier: []CashierStat{}, PaymentMix: []PaymentMix{}}

	if err = tx.QueryRow(ctx, `SELECT CURRENT_DATE::text`).Scan(&summary.Date); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load date")
		return
	}

	if err = tx.QueryRow(ctx, `
		SELECT COALESCE(SUM(total_minor), 0), COUNT(*), COALESCE(SUM((SELECT COALESCE(SUM(quantity), 0) FROM sale_items si WHERE si.sale_id = s.id)), 0)
		FROM sales s WHERE created_at::date = CURRENT_DATE`).
		Scan(&summary.Today.RevenueMinor, &summary.Today.SalesCount, &summary.Today.ItemsSold); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to tally today")
		return
	}
	if summary.Today.SalesCount > 0 {
		summary.Today.AvgSaleMinor = summary.Today.RevenueMinor / summary.Today.SalesCount
	}

	rows, err := tx.Query(ctx, `
		SELECT product_name, sku, SUM(quantity)::bigint, SUM(total_minor)
		FROM sale_items
		WHERE created_at::date = CURRENT_DATE
		GROUP BY product_name, sku
		ORDER BY 3 DESC
		LIMIT 5`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load top products")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var p TopProduct
		if err := rows.Scan(&p.ProductName, &p.SKU, &p.Quantity, &p.RevenueMinor); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load top products")
			return
		}
		summary.TopProducts = append(summary.TopProducts, p)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load top products")
		return
	}

	rows, err = tx.Query(ctx, `
		SELECT id, status, total_minor, currency, COALESCE(payment_method, 'cash'), created_at::text
		FROM sales ORDER BY created_at DESC LIMIT 5`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load recent sales")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var s RecentSale
		if err := rows.Scan(&s.ID, &s.Status, &s.TotalMinor, &s.Currency, &s.PaymentMethod, &s.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load recent sales")
			return
		}
		summary.RecentSales = append(summary.RecentSales, s)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load recent sales")
		return
	}

	rows, err = tx.Query(ctx, `
		SELECT COALESCE(u.display_name, ''), COUNT(*), SUM(s.total_minor)
		FROM sales s
		LEFT JOIN users u ON u.tenant_id = s.tenant_id AND u.id = s.created_by
		WHERE s.created_at::date = CURRENT_DATE
		GROUP BY u.display_name
		ORDER BY 3 DESC`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load cashier breakdown")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var cs CashierStat
		if err := rows.Scan(&cs.Cashier, &cs.SalesCount, &cs.RevenueMinor); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load cashier breakdown")
			return
		}
		summary.PerCashier = append(summary.PerCashier, cs)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load cashier breakdown")
		return
	}

	rows, err = tx.Query(ctx, `
		SELECT method, SUM(amount_minor)
		FROM sale_payments
		WHERE created_at::date = CURRENT_DATE
		GROUP BY method
		ORDER BY 2 DESC`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load payment mix")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var pm PaymentMix
		if err := rows.Scan(&pm.Method, &pm.AmountMinor); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load payment mix")
			return
		}
		summary.PaymentMix = append(summary.PaymentMix, pm)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load payment mix")
		return
	}

	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load summary")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summary, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) authenticate(c *gin.Context) (security.Claims, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is required")
		return security.Claims{}, false
	}
	claims, err := h.tokens.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), security.AccessToken)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return security.Claims{}, false
	}
	return claims, true
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
