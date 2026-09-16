package receipts

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler serves printable receipts for a closed sale. Two views over the same
// data: JSON (GET /sales/:id/receipt) for clients that render their own UI and
// ESC/POS bytes (GET /sales/:id/receipt/print) for direct thermal printing.
type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/sales/:id/receipt", h.get)
	router.GET("/sales/:id/receipt/print", h.print)
}

func (h *Handler) get(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "sale id is invalid")
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
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	receipt, err := loadReceipt(ctx, tx, saleID)
	if err != nil {
		writeReceiptError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": receipt, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) print(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "sale id is invalid")
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
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	receipt, err := loadReceipt(ctx, tx, saleID)
	if err != nil {
		writeReceiptError(c, err)
		return
	}
	c.Header("Content-Type", "application/vnd.escpos")
	c.Header("Cache-Control", "no-store")
	c.Writer.WriteHeader(http.StatusOK)
	cols := cols80
	if c.Query("cols") == "24" {
		cols = cols58
	}
	opts := ReceiptOptions{
		Cols:    cols,
		Cut:     c.Query("cut") != "0",
		Compact: c.Query("compact") == "1",
	}
	_, _ = c.Writer.Write(BuildBytes(receipt, opts))
}

func loadReceipt(ctx context.Context, tx pgx.Tx, saleID uuid.UUID) (Receipt, error) {
	var r Receipt
	err := tx.QueryRow(ctx, `
		SELECT s.status, s.subtotal_minor, s.discount_minor, s.tips_minor, s.total_minor, s.currency,
		       s.created_at::text, s.idempotency_key,
		       COALESCE((SELECT SUM(points_delta) FROM customer_loyalty_log cl WHERE cl.sale_id = s.id), 0),
		       COALESCE(s.created_by::text, ''), COALESCE(s.device_id::text, ''),
		       COALESCE(cu.name, ''), COALESCE(rt.name, ''), COALESCE(fl.name, ''),
		       COALESCE(t.name, ''), COALESCE(t.address, '')
		FROM sales s
		LEFT JOIN customers cu ON cu.id = s.customer_id
		LEFT JOIN restaurant_tables rt ON rt.id = s.table_id
		LEFT JOIN floors fl ON fl.id = rt.floor_id
		LEFT JOIN tenants t ON t.id = s.tenant_id
		WHERE s.id = $1`, saleID).Scan(
		&r.Status, &r.SubtotalMinor, &r.DiscountMinor, &r.TipsMinor, &r.TotalMinor, &r.Currency,
		&r.CreatedAt, &r.IdempotencyKey,
		&r.LoyaltyPoints,
		&r.Cashier, &r.Device,
		&r.CustomerName, &r.TableName, &r.FloorName,
		&r.TenantName, &r.TenantAddress)
	if errors.Is(err, pgx.ErrNoRows) {
		return Receipt{}, errSaleNotFound{code: "sale_not_found", message: "sale not found"}
	}
	if err != nil {
		return Receipt{}, errInternal{message: "unable to load sale"}
	}
	r.SaleID = saleID.String()
	r.Lines = make([]ReceiptLine, 0)
	rows, err := tx.Query(ctx, `
		SELECT product_name, COALESCE(sku, ''), quantity, unit_price_minor, total_minor
		FROM sale_items WHERE sale_id = $1 ORDER BY created_at`, saleID)
	if err != nil {
		return Receipt{}, errInternal{message: "unable to load sale items"}
	}
	for rows.Next() {
		var line ReceiptLine
		if err := rows.Scan(&line.Name, &line.Sku, &line.Qty, &line.Price, &line.Total); err != nil {
			rows.Close()
			return Receipt{}, errInternal{message: "unable to read sale items"}
		}
		r.Lines = append(r.Lines, line)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Receipt{}, errInternal{message: "unable to read sale items"}
	}
	r.Payments = make([]ReceiptPayment, 0)
	payRows, err := tx.Query(ctx, `
		SELECT method, amount_minor, tip_minor FROM sale_payments WHERE sale_id = $1 ORDER BY amount_minor DESC`, saleID)
	if err != nil {
		return Receipt{}, errInternal{message: "unable to load sale payments"}
	}
	for payRows.Next() {
		var p ReceiptPayment
		if err := payRows.Scan(&p.Method, &p.Amount, &p.Tip); err != nil {
			payRows.Close()
			return Receipt{}, errInternal{message: "unable to read sale payments"}
		}
		r.Payments = append(r.Payments, p)
	}
	payRows.Close()
	if err := payRows.Err(); err != nil {
		return Receipt{}, errInternal{message: "unable to read sale payments"}
	}
	return r, nil
}

type errSaleNotFound struct{ code, message string }
type errInternal struct{ message string }

func (e errSaleNotFound) Error() string { return e.message }
func (e errInternal) Error() string     { return e.message }

func writeReceiptError(c *gin.Context, err error) {
	var notFound errSaleNotFound
	if errors.As(err, &notFound) {
		writeError(c, http.StatusNotFound, notFound.code, notFound.message)
		return
	}
	writeError(c, http.StatusInternalServerError, "internal_error", err.Error())
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
