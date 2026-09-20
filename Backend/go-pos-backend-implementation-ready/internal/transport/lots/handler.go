package lots

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler is the C2 pharmacy module: expiry-date + lot/serial ledger. Lots are
// the stock sub-ledger for products with track_lots; checkout consumes them
// first-expiry-first-out (see businesssales.CreateSale).
type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/products/:id/lots", h.list)
	router.POST("/products/:id/lots", h.create)
	router.PATCH("/lots/:id", h.patch)
}

type Lot struct {
	ID         string `json:"id"`
	ProductID  string `json:"product_id"`
	Product    string `json:"product_name"`
	SKU        string `json:"sku"`
	LotNumber  string `json:"lot_number"`
	ExpiryDate string `json:"expiry_date"`
	Quantity   int64  `json:"quantity"`
	Expired    bool   `json:"expired"`
	CreatedAt  string `json:"created_at"`
}

func expiryDateOrNull(s string) any {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func parseIncludeEmpty(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "":
		return false, nil
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, errors.New("invalid include_empty")
	}
}

func isExpired(expiry string) bool {
	if expiry == "" {
		return false
	}
	parsed, err := time.Parse("2006-01-02", expiry)
	if err != nil {
		return false
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return parsed.Before(today)
}

func (h *Handler) list(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "product id is invalid")
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
	if !productExists(ctx, tx, productID) {
		writeError(c, http.StatusNotFound, "product_not_found", "product not found")
		return
	}
	includeEmpty, err := parseIncludeEmpty(c.Query("include_empty"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "include_empty must be true or false")
		return
	}
	rows, err := tx.Query(ctx, `
		SELECT l.id, l.product_id::text, p.name, p.sku, l.lot_number,
		       COALESCE(l.expiry_date::text, ''), l.quantity, l.created_at::text
		FROM product_lots l JOIN products p ON p.tenant_id = l.tenant_id AND p.id = l.product_id
		WHERE l.product_id = $1`+includeEmptySQL(includeEmpty)+`
		ORDER BY l.expiry_date NULLS LAST, l.created_at`, productID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load lots")
		return
	}
	defer rows.Close()
	items := make([]Lot, 0)
	for rows.Next() {
		var lot Lot
		if err := rows.Scan(&lot.ID, &lot.ProductID, &lot.Product, &lot.SKU, &lot.LotNumber, &lot.ExpiryDate, &lot.Quantity, &lot.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to read lots")
			return
		}
		lot.Expired = isExpired(lot.ExpiryDate)
		items = append(items, lot)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to read lots")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load lots")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func includeEmptySQL(include bool) string {
	if include {
		return ""
	}
	return " AND l.quantity > 0"
}

func (h *Handler) create(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "inventory", "adjust") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "product id is invalid")
		return
	}
	var request struct {
		LotNumber  string `json:"lot_number" binding:"required"`
		ExpiryDate string `json:"expiry_date"`
		Quantity   int64  `json:"quantity" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.LotNumber) == "" || len(request.LotNumber) > 64 {
		writeError(c, http.StatusBadRequest, "validation_error", "lot_number is required and must be under 64 characters")
		return
	}
	if request.Quantity < 1 {
		writeError(c, http.StatusBadRequest, "validation_error", "quantity must be positive")
		return
	}
	if request.ExpiryDate != "" {
		if _, err := time.Parse("2006-01-02", request.ExpiryDate); err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "expiry_date must be YYYY-MM-DD")
			return
		}
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
	var trackLots bool
	err = tx.QueryRow(ctx, `SELECT track_lots FROM products WHERE id = $1 FOR UPDATE`, productID).Scan(&trackLots)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "product_not_found", "product not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
		return
	}
	if !trackLots {
		writeError(c, http.StatusConflict, "lot_tracking_not_enabled", "product does not track lots")
		return
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO product_lots (id, tenant_id, product_id, lot_number, expiry_date, quantity)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		uuid.New(), tenantID, productID, strings.TrimSpace(request.LotNumber), expiryDateOrNull(request.ExpiryDate), request.Quantity).Scan(&id)
	if err != nil {
		writeError(c, http.StatusConflict, "duplicate_lot", "lot number already exists for this product")
		return
	}
	// Keep the product's aggregate stock_quantity in sync with the sum of lots.
	if _, err = tx.Exec(ctx, `UPDATE products SET stock_quantity = stock_quantity + $1 WHERE id = $2`, request.Quantity, productID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update stock")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save lot")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"id": id.String(), "product_id": productID.String(), "lot_number": request.LotNumber,
			"expiry_date": request.ExpiryDate, "quantity": request.Quantity,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) patch(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "inventory", "adjust") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	lotID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "lot id is invalid")
		return
	}
	var request struct {
		Quantity   *int64  `json:"quantity"`
		ExpiryDate *string `json:"expiry_date"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid lot update")
		return
	}
	if request.Quantity == nil && request.ExpiryDate == nil {
		writeError(c, http.StatusBadRequest, "validation_error", "nothing to update")
		return
	}
	if request.ExpiryDate != nil && *request.ExpiryDate != "" {
		if _, err := time.Parse("2006-01-02", *request.ExpiryDate); err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "expiry_date must be YYYY-MM-DD")
			return
		}
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
	var productID uuid.UUID
	var oldQty int64
	err = tx.QueryRow(ctx, `SELECT product_id, quantity FROM product_lots WHERE id = $1 FOR UPDATE`, lotID).Scan(&productID, &oldQty)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "lot_not_found", "lot not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load lot")
		return
	}
	newQty := oldQty
	if request.Quantity != nil {
		if *request.Quantity < 0 {
			writeError(c, http.StatusBadRequest, "validation_error", "quantity must not be negative")
			return
		}
		newQty = *request.Quantity
	}
	expiryUpdate := any(nil)
	if request.ExpiryDate != nil {
		expiryUpdate = expiryDateOrNull(*request.ExpiryDate)
	}
	tag, err := tx.Exec(ctx, `UPDATE product_lots SET quantity = $1, expiry_date = COALESCE($2, expiry_date) WHERE id = $3`,
		newQty, expiryUpdate, lotID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update lot")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "lot_not_found", "lot not found")
		return
	}
	// products.stock_quantity tracks the sum of lots.
	if _, err = tx.Exec(ctx, `UPDATE products SET stock_quantity = stock_quantity + $1 WHERE id = $2`, newQty-oldQty, productID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update stock")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update lot")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": lotID.String(), "quantity": newQty}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func productExists(ctx context.Context, tx pgx.Tx, productID uuid.UUID) bool {
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id = $1)`, productID).Scan(&exists); err != nil {
		return false
	}
	return exists
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
