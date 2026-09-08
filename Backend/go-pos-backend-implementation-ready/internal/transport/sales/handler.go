package sales

import (
	"errors"
	"net/http"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	router.POST("/sales", h.createSale)
}

type createSaleRequest struct {
	Items []saleItemRequest `json:"items" binding:"required,min=1"`
}

type saleItemRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int64  `json:"quantity" binding:"required,gt=0"`
}

type Sale struct {
	ID            string `json:"id"`
	SubtotalMinor int64  `json:"subtotal_minor"`
	TotalMinor    int64  `json:"total_minor"`
	Currency      string `json:"currency"`
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

	var existing Sale
	err = tx.QueryRow(ctx, `SELECT id, subtotal_minor, total_minor, currency FROM sales WHERE idempotency_key = $1`, idempotencyKey).Scan(
		&existing.ID, &existing.SubtotalMinor, &existing.TotalMinor, &existing.Currency)
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

	saleID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO sales (id, tenant_id, device_id, created_by, idempotency_key, subtotal_minor, total_minor, currency)
		VALUES ($1, $2, $3, $4, $5, $6, $6, $7)`, saleID, tenantID, deviceID, userID, idempotencyKey, subtotal, currency)
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
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit sale")
		return
	}
	writeSale(c, Sale{ID: saleID.String(), SubtotalMinor: subtotal, TotalMinor: subtotal, Currency: currency})
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
