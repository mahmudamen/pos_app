package variants

import (
	"errors"
	"net/http"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler is the C3 fashion/textile module: size/color variants. Parents with
// has_variants hold the template; price and stock live on product_variants and
// are authoritative at checkout.
type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/products/:id/variants", h.list)
	router.POST("/products/:id/variants", h.create)
	router.PATCH("/variants/:id", h.patch)
	router.DELETE("/variants/:id", h.delete)
}

type Variant struct {
	ID            string `json:"id"`
	ProductID     string `json:"product_id"`
	Name          string `json:"name"`
	Size          string `json:"size"`
	Color         string `json:"color"`
	SKU           string `json:"sku"`
	PriceMinor    int64  `json:"price_minor"`
	StockQuantity int64  `json:"stock_quantity"`
	IsActive      bool   `json:"is_active"`
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
	var hasVariants bool
	err = tx.QueryRow(ctx, `SELECT has_variants FROM products WHERE id = $1`, productID).Scan(&hasVariants)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "product_not_found", "product not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
		return
	}
	if !hasVariants {
		writeError(c, http.StatusConflict, "variants_not_enabled", "product does not have variants")
		return
	}
	rows, err := tx.Query(ctx, `
		SELECT id, product_id::text, name, size, color, sku, price_minor, stock_quantity, is_active
		FROM product_variants WHERE product_id = $1 ORDER BY name`, productID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load variants")
		return
	}
	defer rows.Close()
	items := make([]Variant, 0)
	for rows.Next() {
		var v Variant
		if err := rows.Scan(&v.ID, &v.ProductID, &v.Name, &v.Size, &v.Color, &v.SKU, &v.PriceMinor, &v.StockQuantity, &v.IsActive); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to read variants")
			return
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to read variants")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load variants")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) create(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "catalog", "write") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "product id is invalid")
		return
	}
	var request struct {
		Name          string `json:"name" binding:"required"`
		Size          string `json:"size"`
		Color         string `json:"color"`
		SKU           string `json:"sku" binding:"required"`
		PriceMinor    int64  `json:"price_minor" binding:"required"`
		StockQuantity int64  `json:"stock_quantity"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Name) == "" || request.PriceMinor < 0 || strings.TrimSpace(request.SKU) == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "name, sku and a non-negative price_minor are required")
		return
	}
	if request.StockQuantity < 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "stock_quantity must not be negative")
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
	var hasVariants bool
	err = tx.QueryRow(ctx, `SELECT has_variants FROM products WHERE id = $1`, productID).Scan(&hasVariants)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "product_not_found", "product not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
		return
	}
	if !hasVariants {
		writeError(c, http.StatusConflict, "variants_not_enabled", "product does not have variants")
		return
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO product_variants (id, tenant_id, product_id, name, size, color, sku, price_minor, stock_quantity)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		uuid.New(), tenantID, productID, strings.TrimSpace(request.Name),
		request.Size, request.Color, strings.TrimSpace(request.SKU), request.PriceMinor, request.StockQuantity).Scan(&id)
	if err != nil {
		writeError(c, http.StatusConflict, "duplicate_sku", "sku already exists on this tenant")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save variant")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": Variant{ID: id.String(), ProductID: productID.String(), Name: request.Name, Size: request.Size,
			Color: request.Color, SKU: request.SKU, PriceMinor: request.PriceMinor, StockQuantity: request.StockQuantity, IsActive: true},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) patch(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "catalog", "write") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	variantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "variant id is invalid")
		return
	}
	var request struct {
		Name          *string `json:"name"`
		Size          *string `json:"size"`
		Color         *string `json:"color"`
		SKU           *string `json:"sku"`
		PriceMinor    *int64  `json:"price_minor"`
		StockQuantity *int64  `json:"stock_quantity"`
		IsActive      *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid variant update")
		return
	}
	if request.PriceMinor != nil && *request.PriceMinor < 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "price_minor must not be negative")
		return
	}
	if request.StockQuantity != nil && *request.StockQuantity < 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "stock_quantity must not be negative")
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
	tag, err := tx.Exec(ctx, `
		UPDATE product_variants
		SET name = COALESCE($1, name), size = COALESCE($2, size), color = COALESCE($3, color),
		    sku = COALESCE($4, sku), price_minor = COALESCE($5, price_minor),
		    stock_quantity = COALESCE($6, stock_quantity), is_active = COALESCE($7, is_active)
		WHERE id = $8`,
		request.Name, request.Size, request.Color, request.SKU, request.PriceMinor, request.StockQuantity, request.IsActive, variantID)
	if err != nil {
		writeError(c, http.StatusConflict, "duplicate_sku", "sku already exists on this tenant")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "variant_not_found", "variant not found")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update variant")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": variantID.String()}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) delete(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "catalog", "write") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	variantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "variant id is invalid")
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
	tag, err := tx.Exec(ctx, `UPDATE product_variants SET is_active = false WHERE id = $1`, variantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete variant")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "variant_not_found", "variant not found")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete variant")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": variantID.String()}, "meta": gin.H{"request_id": c.GetString("request_id")}})
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
