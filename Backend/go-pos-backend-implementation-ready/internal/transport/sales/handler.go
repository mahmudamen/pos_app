package sales

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	notificationstransport "github.com/example/pos-api/internal/transport/notifications"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool             *pgxpool.Pool
	tokens           security.TokenManager
	discountLimitPct int
	notify           notificationstransport.Emit
}

// SetNotifier wires the notifications emitter; a nil emitter makes the
// handler a no-op (unit tests, OpenAPI generator).
func (h *Handler) SetNotifier(fn notificationstransport.Emit) { h.notify = fn }

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens, discountLimitPct: 5}
}

func NewHandlerWithDiscountLimit(pool *pgxpool.Pool, tokens security.TokenManager, limitPct int) *Handler {
	if limitPct < 0 {
		limitPct = 0
	}
	if limitPct > 100 {
		limitPct = 100
	}
	return &Handler{pool: pool, tokens: tokens, discountLimitPct: limitPct}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/sales", h.listSales)
	router.GET("/sales/:id", h.getSale)
	router.POST("/sales", h.createSale)
	router.POST("/sales/:id/refund", h.refundSale)
}

type createSaleRequest struct {
	Items             []saleItemRequest `json:"items" binding:"required,min=1"`
	Payments          []paymentRequest  `json:"payments"`
	RegisterSessionID string            `json:"session_id"`
	DiscountMinor     int64             `json:"discount_minor"`
	CustomerID        string            `json:"customer_id"`
	TableID           string            `json:"table_id"`
	ManagerPIN        string            `json:"manager_pin"`
	RoundingMinor     int64             `json:"rounding_minor"`
}

type saleItemRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int64  `json:"quantity" binding:"required,gt=0"`
	VariantID string `json:"variant_id"`
}

type Sale struct {
	ID                  string `json:"id"`
	Status              string `json:"status"`
	SubtotalMinor       int64  `json:"subtotal_minor"`
	DiscountMinor       int64  `json:"discount_minor"`
	TaxMinor            int64  `json:"tax_minor"`
	TotalMinor          int64  `json:"total_minor"`
	RoundingMinor       int64  `json:"rounding_minor"`
	Currency            string `json:"currency"`
	PaymentMethod       string `json:"payment_method"`
	CustomerID          string `json:"customer_id"`
	LoyaltyPointsEarned int64  `json:"loyalty_points_earned"`
	TipsMinor           int64  `json:"tips_minor"`
	TableID             string `json:"table_id"`
	CreatedAt           string `json:"created_at"`
	DiscountCapped      bool   `json:"discount_capped,omitempty"`
	DiscountWarning     string `json:"discount_warning,omitempty"`
}

type SaleItem struct {
	ID             string `json:"id"`
	ProductName    string `json:"product_name"`
	SKU            string `json:"sku"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	TotalMinor     int64  `json:"total_minor"`
	VariantName    string `json:"variant_name,omitempty"`
	LotNumber      string `json:"lot_number,omitempty"`
}

type Payment struct {
	Method      string `json:"method"`
	AmountMinor int64  `json:"amount_minor"`
	TipMinor    int64  `json:"tip_minor"`
}

type SaleDetail struct {
	Sale
	CreatedBy string     `json:"created_by"`
	Items     []SaleItem `json:"items"`
	Payments  []Payment  `json:"payments"`
}

func (h *Handler) listSales(c *gin.Context) {
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
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var total int64
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM sales`).Scan(&total)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count sales")
		return
	}

	rows, err := tx.Query(ctx, `
		SELECT id, status, subtotal_minor, discount_minor, tax_minor, total_minor, rounding_minor, currency,
		       COALESCE(payment_method, 'cash'), tips_minor, COALESCE(table_id::text, ''), created_at::text
		FROM sales ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sales")
		return
	}
	defer rows.Close()

	sales := make([]Sale, 0)
	for rows.Next() {
		var s Sale
		if err := rows.Scan(&s.ID, &s.Status, &s.SubtotalMinor, &s.DiscountMinor, &s.TaxMinor, &s.TotalMinor, &s.RoundingMinor, &s.Currency, &s.PaymentMethod, &s.TipsMinor, &s.TableID, &s.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sales")
			return
		}
		sales = append(sales, s)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sales")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sales")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": sales,
		"meta": gin.H{
			"request_id": c.GetString("request_id"),
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
	})
}

func (h *Handler) getSale(c *gin.Context) {
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

	saleID := c.Param("id")
	if _, err := uuid.Parse(saleID); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid sale id")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var detail SaleDetail
	err = tx.QueryRow(ctx, `
		SELECT id, status, subtotal_minor, discount_minor, tax_minor, total_minor, rounding_minor, currency,
		       COALESCE(payment_method, 'cash'), COALESCE(customer_id::text, ''),
		       COALESCE((SELECT SUM(points_delta) FROM customer_loyalty_log cl WHERE cl.sale_id = sales.id), 0),
		       tips_minor, COALESCE(table_id::text, ''),
		       created_at::text, COALESCE(created_by::text, '')
		FROM sales WHERE id = $1`, saleID).Scan(
		&detail.ID, &detail.Status, &detail.SubtotalMinor, &detail.DiscountMinor, &detail.TaxMinor,
		&detail.TotalMinor, &detail.RoundingMinor, &detail.Currency, &detail.PaymentMethod, &detail.CustomerID,
		&detail.LoyaltyPointsEarned, &detail.TipsMinor, &detail.TableID,
		&detail.CreatedAt, &detail.CreatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "sale_not_found", "sale not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale")
		return
	}

	itemRows, err := tx.Query(ctx, `
		SELECT id, product_name, sku, quantity, unit_price_minor, total_minor,
		       COALESCE(variant_name, ''), COALESCE(lot_number, '')
		FROM sale_items WHERE sale_id = $1 ORDER BY created_at`, saleID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale items")
		return
	}
	defer itemRows.Close()

	detail.Items = make([]SaleItem, 0)
	for itemRows.Next() {
		var item SaleItem
		if err := itemRows.Scan(&item.ID, &item.ProductName, &item.SKU, &item.Quantity, &item.UnitPriceMinor, &item.TotalMinor, &item.VariantName, &item.LotNumber); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale items")
			return
		}
		detail.Items = append(detail.Items, item)
	}
	if err := itemRows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale items")
		return
	}

	paymentRows, err := tx.Query(ctx, `
		SELECT method, amount_minor, tip_minor FROM sale_payments WHERE sale_id = $1 ORDER BY amount_minor DESC, created_at`, saleID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale payments")
		return
	}
	defer paymentRows.Close()

	detail.Payments = make([]Payment, 0)
	for paymentRows.Next() {
		var p Payment
		if err := paymentRows.Scan(&p.Method, &p.AmountMinor, &p.TipMinor); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale payments")
			return
		}
		detail.Payments = append(detail.Payments, p)
	}
	if err := paymentRows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale payments")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": detail, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) createSale(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" || len(idempotencyKey) > 128 {
		writeError(c, http.StatusBadRequest, "validation_error", "Idempotency-Key is required")
		return
	}
	var request createSaleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid sale request")
		return
	}

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}
	deviceID, err := uuid.Parse(claims.DeviceID)
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
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	sale, applyErr := CreateSale(ctx, tx, tenantID, userID, deviceID, claims.Role, idempotencyKey, h.discountLimitPct, request)
	if applyErr != nil {
		status, code, message := SaleError(applyErr)
		writeError(c, status, code, message)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit sale")
		return
	}
	if saleID, err := uuid.Parse(sale.ID); err == nil {
		h.notifyLowStock(ctx, tenantID, saleID)
	}
	writeSale(c, sale)
}

func (h *Handler) refundSale(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	if idempotencyKey == "" || len(idempotencyKey) > 128 {
		writeError(c, http.StatusBadRequest, "validation_error", "Idempotency-Key is required")
		return
	}
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid sale id")
		return
	}
	var request RefundRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid refund request")
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
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
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	refund, applyErr := RefundSale(ctx, tx, tenantID, userID, claims.Role, saleID, idempotencyKey, request)
	if applyErr != nil {
		status, code, message := SaleError(applyErr)
		writeError(c, status, code, message)
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit refund")
		return
	}
	if h.notify != nil {
		h.notify(ctx, tenantID.String(), userID.String(), notificationstransport.TypeRefund,
			"refund:"+saleID.String(), notificationstransport.SeverityCritical,
			"Refund applied", "A refund was issued against sale "+saleID.String()+".",
			map[string]any{"sale_id": saleID.String()})
	}
	c.JSON(http.StatusCreated, gin.H{"data": refund, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

// notifyLowStock emits a low-stock notification for every product on this sale
// whose post-sale on-hand count dropped to (or below) the tenant's
// pos.low_stock_threshold. Runs in its own connection after the sale commit, so
// failures only degrade the inbox, never the sale response.
func (h *Handler) notifyLowStock(ctx context.Context, tenantID uuid.UUID, saleID uuid.UUID) {
	if h.pool == nil || h.notify == nil {
		return
	}
	threshold := int64(5) // tenant_settings default
	if err := h.pool.QueryRow(ctx, `
		SELECT COALESCE((SELECT value FROM tenant_settings WHERE tenant_id = $1 AND key = 'pos.low_stock_threshold'), '5')`,
		tenantID).Scan(&threshold); err != nil {
		return
	}
	rows, err := h.pool.Query(ctx, `
		SELECT DISTINCT p.id::text, p.name, p.stock_quantity
		FROM sale_items si
		JOIN products p ON p.id = si.product_id
		WHERE si.sale_id = $1 AND p.stock_quantity <= $2`,
		saleID, threshold)
	if err != nil {
		return
	}
	type lowProduct struct {
		id, name string
		stock    int64
	}
	lows := make([]lowProduct, 0)
	for rows.Next() {
		var lp lowProduct
		if rows.Scan(&lp.id, &lp.name, &lp.stock) != nil {
			continue
		}
		lows = append(lows, lp)
	}
	rows.Close()
	for _, lp := range lows {
		h.notify(ctx, tenantID.String(), "", notificationstransport.TypeLowStock,
			"low_stock:"+lp.id, notificationstransport.SeverityWarning,
			"Low stock: "+lp.name, "Only "+strconv.FormatInt(lp.stock, 10)+" left.",
			map[string]any{"product_id": lp.id, "stock_quantity": lp.stock})
	}
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

func writeSale(c *gin.Context, sale Sale) {
	c.JSON(http.StatusCreated, gin.H{"data": sale, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
