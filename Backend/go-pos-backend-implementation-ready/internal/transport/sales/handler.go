package sales

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool             *pgxpool.Pool
	tokens           security.TokenManager
	discountLimitPct int
}

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
}

type createSaleRequest struct {
	Items             []saleItemRequest `json:"items" binding:"required,min=1"`
	Payments          []paymentRequest  `json:"payments"`
	RegisterSessionID string            `json:"session_id"`
	DiscountMinor     int64             `json:"discount_minor"`
}

type saleItemRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int64  `json:"quantity" binding:"required,gt=0"`
}

type Sale struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	SubtotalMinor int64  `json:"subtotal_minor"`
	DiscountMinor int64  `json:"discount_minor"`
	TaxMinor      int64  `json:"tax_minor"`
	TotalMinor    int64  `json:"total_minor"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method"`
	CreatedAt     string `json:"created_at"`
}

type SaleItem struct {
	ID             string `json:"id"`
	ProductName    string `json:"product_name"`
	SKU            string `json:"sku"`
	Quantity       int64  `json:"quantity"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	TotalMinor     int64  `json:"total_minor"`
}

type Payment struct {
	Method      string `json:"method"`
	AmountMinor int64  `json:"amount_minor"`
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
	defer tx.Rollback(ctx)
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
		SELECT id, status, subtotal_minor, discount_minor, tax_minor, total_minor, currency,
		       COALESCE(payment_method, 'cash'), created_at::text
		FROM sales ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sales")
		return
	}
	defer rows.Close()

	sales := make([]Sale, 0)
	for rows.Next() {
		var s Sale
		if err := rows.Scan(&s.ID, &s.Status, &s.SubtotalMinor, &s.DiscountMinor, &s.TaxMinor, &s.TotalMinor, &s.Currency, &s.PaymentMethod, &s.CreatedAt); err != nil {
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
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var detail SaleDetail
	err = tx.QueryRow(ctx, `
		SELECT id, status, subtotal_minor, discount_minor, tax_minor, total_minor, currency,
		       COALESCE(payment_method, 'cash'), created_at::text,
		       COALESCE(created_by::text, '')
		FROM sales WHERE id = $1`, saleID).Scan(
		&detail.ID, &detail.Status, &detail.SubtotalMinor, &detail.DiscountMinor, &detail.TaxMinor,
		&detail.TotalMinor, &detail.Currency, &detail.PaymentMethod, &detail.CreatedAt, &detail.CreatedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "sale_not_found", "sale not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale")
		return
	}

	itemRows, err := tx.Query(ctx, `
		SELECT id, product_name, sku, quantity, unit_price_minor, total_minor
		FROM sale_items WHERE sale_id = $1 ORDER BY created_at`, saleID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale items")
		return
	}
	defer itemRows.Close()

	detail.Items = make([]SaleItem, 0)
	for itemRows.Next() {
		var item SaleItem
		if err := itemRows.Scan(&item.ID, &item.ProductName, &item.SKU, &item.Quantity, &item.UnitPriceMinor, &item.TotalMinor); err != nil {
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
		SELECT method, amount_minor FROM sale_payments WHERE sale_id = $1 ORDER BY amount_minor DESC, created_at`, saleID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale payments")
		return
	}
	defer paymentRows.Close()

	detail.Payments = make([]Payment, 0)
	for paymentRows.Next() {
		var p Payment
		if err := paymentRows.Scan(&p.Method, &p.AmountMinor); err != nil {
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
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var requestSessionID *uuid.UUID
	if request.RegisterSessionID != "" {
		parsed, parseErr := uuid.Parse(request.RegisterSessionID)
		if parseErr != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "session_id is invalid")
			return
		}
		var status string
		err = tx.QueryRow(ctx, `
			SELECT status FROM register_sessions
			WHERE id = $1 AND tenant_id = $2 AND device_id = $3`,
			parsed, tenantID, deviceID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(c, http.StatusNotFound, "session_not_found", "session not found on this terminal")
			return
		}
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to check session")
			return
		}
		if status != "open" {
			writeError(c, http.StatusConflict, "session_not_open", "session is already closed")
			return
		}
		requestSessionID = &parsed
	}

	var existing Sale
	err = tx.QueryRow(ctx, `SELECT id, status, subtotal_minor, discount_minor, tax_minor, total_minor, currency,
		COALESCE(payment_method, 'cash'), created_at::text FROM sales WHERE idempotency_key = $1`, idempotencyKey).Scan(
		&existing.ID, &existing.Status, &existing.SubtotalMinor, &existing.DiscountMinor, &existing.TaxMinor, &existing.TotalMinor, &existing.Currency, &existing.PaymentMethod, &existing.CreatedAt)
	if err == nil {
		if err = tx.Commit(ctx); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale")
			return
		}
		writeSale(c, existing)
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to check idempotency")
		return
	}

	var subtotal int64
	currency := ""
	type lockedItem struct {
		productID uuid.UUID
		name      string
		sku       string
		quantity  int64
		price     int64
	}
	lockedItems := make([]lockedItem, 0, len(request.Items))
	for _, item := range request.Items {
		productID, parseErr := uuid.Parse(item.ProductID)
		if parseErr != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "product_id is invalid")
			return
		}
		var product lockedItem
		var stock int64
		err = tx.QueryRow(ctx, `
			SELECT id, name, sku, price_minor, currency, stock_quantity
			FROM products WHERE id = $1 AND is_active FOR UPDATE`, productID).Scan(
			&product.productID, &product.name, &product.sku, &product.price, &currency, &stock)
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(c, http.StatusNotFound, "product_not_found", "product not found")
			return
		}
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
			return
		}
		if stock < item.Quantity {
			writeError(c, http.StatusConflict, "insufficient_stock", "insufficient stock")
			return
		}
		product.quantity = item.Quantity
		subtotal += product.price * item.Quantity
		lockedItems = append(lockedItems, product)
	}

	discount := request.DiscountMinor
	total := subtotal
	if discount > 0 {
		if !httptransport.HasPermission(claims.Role, "pos", "discount") {
			writeError(c, http.StatusForbidden, "permission_denied", "discount not allowed for this role")
			return
		}
		validated, err := validateDiscount(claims.Role, subtotal, discount, h.discountLimitPct)
		if err != nil {
			writeError(c, http.StatusBadRequest, "discount_error", err.Error())
			return
		}
		total = validated
	} else if discount < 0 {
		writeError(c, http.StatusBadRequest, "discount_error", string(ErrNegativeDiscount))
		return
	}

	payments, err := normalizePayments(request.Payments, total)
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}

	saleID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO sales (id, tenant_id, device_id, created_by, idempotency_key, subtotal_minor, discount_minor, total_minor, currency, payment_method, register_session_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`, saleID, tenantID, deviceID, userID, idempotencyKey, subtotal, discount, total, currency, primaryPaymentMethod(payments), requestSessionID)
	if err != nil {
		writeError(c, http.StatusConflict, "idempotency_conflict", "idempotency key is already in use")
		return
	}
	for _, item := range lockedItems {
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (tenant_id, sale_id, product_id, product_name, sku, quantity, unit_price_minor, total_minor)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, tenantID, saleID, item.productID, item.name, item.sku, item.quantity, item.price, item.price*item.quantity)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to save sale items")
			return
		}
		_, err = tx.Exec(ctx, `UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2`, item.quantity, item.productID)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to update stock")
			return
		}
	}
	for _, p := range payments {
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_payments (tenant_id, sale_id, method, amount_minor)
			VALUES ($1, $2, $3, $4)`, tenantID, saleID, p.Method, p.AmountMinor)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to save sale payments")
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit sale")
		return
	}
	writeSale(c, Sale{ID: saleID.String(), Status: "completed", SubtotalMinor: subtotal, DiscountMinor: discount, TotalMinor: total, Currency: currency, PaymentMethod: primaryPaymentMethod(payments)})
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
