package catalog

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BackInStockFunc is invoked after a product PATCH commits when the product
// just became available for online self-ordering again (was out of stock or
// not published, now in stock and published). The selforder package wires it
// up via SetBackInStockNotifier to fire web-push back-in-stock notifications.
type BackInStockFunc func(ctx context.Context, tenantID, productID string)

type Handler struct {
	pool        *pgxpool.Pool
	tokens      security.TokenManager
	backInStock BackInStockFunc
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

// SetBackInStockNotifier registers the back-in-stock callback (selforder).
func (h *Handler) SetBackInStockNotifier(fn BackInStockFunc) {
	h.backInStock = fn
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/categories", h.listCategories)
	router.POST("/categories", h.createCategory)
	router.PATCH("/categories/:id", h.updateCategory)
	router.DELETE("/categories/:id", h.deleteCategory)
	router.GET("/products", h.listProducts)
	router.POST("/products", h.createProduct)
	router.GET("/products/:id", h.getProduct)
	router.PATCH("/products/:id", h.updateProduct)
	router.DELETE("/products/:id", h.deleteProduct)
	router.GET("/products/barcode/:barcode", h.getByBarcode)
}

type Category struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	NameAr   string `json:"name_ar,omitempty"`
	Slug     string `json:"slug"`
	IsActive bool   `json:"is_active"`
}

type categoryRequest struct {
	Name   string  `json:"name" binding:"required"`
	NameAr *string `json:"name_ar"`
	Slug   string  `json:"slug" binding:"required"`
}

type categoryPatchRequest struct {
	Name     *string `json:"name"`
	NameAr   *string `json:"name_ar"`
	Slug     *string `json:"slug"`
	IsActive *bool   `json:"is_active"`
}

type productRequest struct {
	CategoryID       *string `json:"category_id"`
	Name             string  `json:"name" binding:"required"`
	NameAr           *string `json:"name_ar"`
	SKU              string  `json:"sku" binding:"required"`
	Barcode          string  `json:"barcode"`
	PriceMinor       int64   `json:"price_minor" binding:"gte=0"`
	CostMinor        int64   `json:"cost_minor" binding:"gte=0"`
	Currency         string  `json:"currency" binding:"required,len=3"`
	StockQuantity    int64   `json:"stock_quantity" binding:"gte=0"`
	ImageURL         string  `json:"image_url"`
	Description      string  `json:"description"`
	DescriptionAr    *string `json:"description_ar"`
	Unit             string  `json:"unit"`
	SelforderEnabled bool    `json:"selforder_enabled"`
}

type productPatchRequest struct {
	CategoryID       *string `json:"category_id"`
	Name             *string `json:"name"`
	NameAr           *string `json:"name_ar"`
	SKU              *string `json:"sku"`
	Barcode          *string `json:"barcode"`
	PriceMinor       *int64  `json:"price_minor"`
	CostMinor        *int64  `json:"cost_minor"`
	Currency         *string `json:"currency"`
	StockQuantity    *int64  `json:"stock_quantity"`
	ImageURL         *string `json:"image_url"`
	Description      *string `json:"description"`
	DescriptionAr    *string `json:"description_ar"`
	IsActive         *bool   `json:"is_active"`
	Unit             *string `json:"unit"`
	SelforderEnabled *bool   `json:"selforder_enabled"`
}

func (h *Handler) listCategories(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	tx, ok := h.tenantTx(c, claims)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	rows, err := tx.Query(c.Request.Context(), `SELECT id, name, COALESCE(name_ar, ''), slug, is_active FROM categories WHERE is_active ORDER BY name`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load categories")
		return
	}
	defer rows.Close()
	categories := make([]Category, 0)
	for rows.Next() {
		var category Category
		if err := rows.Scan(&category.ID, &category.Name, &category.NameAr, &category.Slug, &category.IsActive); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load categories")
			return
		}
		categories = append(categories, category)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load categories")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load categories")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": categories, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) createCategory(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	var request categoryRequest
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Name) == "" || strings.TrimSpace(request.Slug) == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "name and slug are required")
		return
	}
	tx, ok := h.tenantTx(c, claims)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	var category Category
	err := tx.QueryRow(c.Request.Context(), `
		INSERT INTO categories (tenant_id, name, name_ar, slug) VALUES ($1::uuid, $2, NULLIF($3, ''), $4)
		RETURNING id, name, COALESCE(name_ar, ''), slug, is_active`, claims.TenantID, strings.TrimSpace(request.Name), strings.TrimSpace(nonNilString(request.NameAr)), strings.TrimSpace(request.Slug)).Scan(&category.ID, &category.Name, &category.NameAr, &category.Slug, &category.IsActive)
	if err != nil {
		writeError(c, http.StatusConflict, "category_conflict", "category slug is already in use")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create category")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": category, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) updateCategory(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	var request categoryPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid category request")
		return
	}
	if request.Name != nil && strings.TrimSpace(*request.Name) == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "name cannot be empty")
		return
	}
	if request.Slug != nil && strings.TrimSpace(*request.Slug) == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "slug cannot be empty")
		return
	}
	tx, ok := h.tenantTx(c, claims)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	var current Category
	err := tx.QueryRow(c.Request.Context(),
		`SELECT id, name, COALESCE(name_ar, ''), slug, is_active FROM categories WHERE id = $1::uuid`, c.Param("id")).Scan(
		&current.ID, &current.Name, &current.NameAr, &current.Slug, &current.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "category_not_found", "category not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load category")
		return
	}
	if request.Name != nil {
		current.Name = strings.TrimSpace(*request.Name)
	}
	if request.NameAr != nil {
		current.NameAr = strings.TrimSpace(*request.NameAr)
	}
	if request.Slug != nil {
		current.Slug = strings.TrimSpace(*request.Slug)
	}
	if request.IsActive != nil {
		current.IsActive = *request.IsActive
	}
	_, err = tx.Exec(c.Request.Context(),
		`UPDATE categories SET name = $1, name_ar = NULLIF($2, ''), slug = $3, is_active = $4 WHERE id = $5::uuid`,
		current.Name, current.NameAr, current.Slug, current.IsActive, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusConflict, "category_conflict", "category slug is already in use")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update category")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": current, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) deleteCategory(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	tx, ok := h.tenantTx(c, claims)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	var category Category
	err := tx.QueryRow(c.Request.Context(),
		`SELECT id, name, COALESCE(name_ar, ''), slug, is_active FROM categories WHERE id = $1::uuid`, c.Param("id")).Scan(
		&category.ID, &category.Name, &category.NameAr, &category.Slug, &category.IsActive)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "category_not_found", "category not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load category")
		return
	}
	_, err = tx.Exec(c.Request.Context(),
		`UPDATE categories SET is_active = false WHERE id = $1::uuid`, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete category")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete category")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": category, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) deleteProduct(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	tx, ok := h.tenantTx(c, claims)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	product, err := queryProduct(c, tx, c.Param("id"))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "product_not_found", "product not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
		return
	}
	_, err = tx.Exec(c.Request.Context(),
		`UPDATE products SET is_active = false WHERE id = $1::uuid`, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete product")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete product")
		return
	}
	product.IsActive = false
	c.JSON(http.StatusOK, gin.H{"data": product, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) createProduct(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	var request productRequest
	if err := c.ShouldBindJSON(&request); err != nil || !validProduct(request.Name, request.SKU, request.Currency, request.PriceMinor, request.CostMinor, request.StockQuantity) {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid product request")
		return
	}
	tx, ok := h.tenantTx(c, claims)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()

	// D2 plan limits: max_products = 0 means unlimited; the check runs inside
	// the tenant tx so COUNT(*) is RLS-filtered to this tenant's rows.
	var maxProducts int
	if err := tx.QueryRow(c.Request.Context(), `SELECT max_products FROM tenants WHERE id = $1::uuid`, claims.TenantID).Scan(&maxProducts); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenant plan")
		return
	}
	if maxProducts > 0 {
		var existing int
		if err := tx.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM products`).Scan(&existing); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to count products")
			return
		}
		if existing >= maxProducts {
			_ = tx.Rollback(c.Request.Context())
			writeError(c, http.StatusConflict, "plan_limit_exceeded", "tenant product limit reached")
			return
		}
	}

	product, err := insertProduct(c, tx, claims.TenantID, request)
	if err != nil {
		writeError(c, http.StatusConflict, "product_conflict", "product SKU or barcode is already in use")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create product")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": product, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) getProduct(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	tx, ok := h.tenantTx(c, claims)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	product, err := queryProduct(c, tx, c.Param("id"))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "product_not_found", "product not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) updateProduct(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	var request productPatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid product request")
		return
	}
	tx, ok := h.tenantTx(c, claims)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	current, err := queryProduct(c, tx, c.Param("id"))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "product_not_found", "product not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
		return
	}
	wasAvailableOnline := current.IsActive && current.SelforderEnabled && current.StockQuantity > 0
	applyPatch(&current, request)
	if !validProduct(current.Name, current.SKU, current.Currency, current.PriceMinor, current.CostMinor, current.StockQuantity) {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid product request")
		return
	}
	nowAvailableOnline := current.IsActive && current.SelforderEnabled && current.StockQuantity > 0
	_, err = tx.Exec(c.Request.Context(), `
		UPDATE products SET category_id = NULLIF($1, '')::uuid, name = $2, name_ar = NULLIF($3, ''), sku = $4, barcode = NULLIF($5, ''),
		price_minor = $6, cost_minor = $7, currency = $8, stock_quantity = $9, image_url = $10, description = $11, description_ar = NULLIF($12, ''), is_active = $13, unit = $15, selforder_enabled = $16
		WHERE id = $14::uuid`, current.CategoryID, current.Name, current.NameAr, current.SKU, current.Barcode, current.PriceMinor, current.CostMinor, current.Currency, current.StockQuantity, current.ImageURL, current.Description, current.DescriptionAr, current.IsActive, c.Param("id"), normalizeUnit(current.Unit), current.SelforderEnabled)
	if err != nil {
		writeError(c, http.StatusConflict, "product_conflict", "product SKU or barcode is already in use")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update product")
		return
	}
	if nowAvailableOnline && !wasAvailableOnline && h.backInStock != nil {
		h.backInStock(c.Request.Context(), claims.TenantID, c.Param("id"))
	}
	c.JSON(http.StatusOK, gin.H{"data": current, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) listProducts(c *gin.Context) {
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
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 100
	}
	offset := int64((page - 1) * limit)

	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	if _, err = tx.Exec(c.Request.Context(), "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	search := strings.TrimSpace(c.Query("search"))

	var total int64
	err = tx.QueryRow(c.Request.Context(), `
		SELECT COUNT(*) FROM products WHERE is_active
		  AND ($1 = '' OR name ILIKE '%' || $1 || '%' OR sku ILIKE '%' || $1 || '%' OR barcode ILIKE '%' || $1 || '%')`, search).Scan(&total)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count products")
		return
	}

	rows, err := tx.Query(c.Request.Context(), `
		SELECT id, name, COALESCE(name_ar, ''), sku, COALESCE(barcode, ''), price_minor, cost_minor, currency, stock_quantity, COALESCE(image_url, ''), COALESCE(description, ''), COALESCE(description_ar, ''), unit, selforder_enabled, created_at
		FROM products WHERE is_active
		  AND ($1 = '' OR name ILIKE '%' || $1 || '%' OR sku ILIKE '%' || $1 || '%' OR barcode ILIKE '%' || $1 || '%')
		ORDER BY name
		LIMIT $2 OFFSET $3`, search, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load products")
		return
	}
	defer rows.Close()
	products := make([]Product, 0)
	for rows.Next() {
		var product Product
		if err := rows.Scan(&product.ID, &product.Name, &product.NameAr, &product.SKU, &product.Barcode, &product.PriceMinor, &product.CostMinor, &product.Currency, &product.StockQuantity, &product.ImageURL, &product.Description, &product.DescriptionAr, &product.Unit, &product.SelforderEnabled, &product.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load products")
			return
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load products")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load products")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": products,
		"meta": gin.H{
			"request_id": c.GetString("request_id"),
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
	})
}

func (h *Handler) getByBarcode(c *gin.Context) {
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
	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	if _, err = tx.Exec(c.Request.Context(), "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var product Product
	err = tx.QueryRow(c.Request.Context(), `
		SELECT id, name, COALESCE(name_ar, ''), sku, COALESCE(barcode, ''), price_minor, cost_minor, currency, stock_quantity, COALESCE(image_url, ''), COALESCE(description, ''), COALESCE(description_ar, ''), unit, selforder_enabled, created_at
		FROM products WHERE barcode = $1 AND is_active`, c.Param("barcode")).Scan(
		&product.ID, &product.Name, &product.NameAr, &product.SKU, &product.Barcode, &product.PriceMinor, &product.CostMinor, &product.Currency, &product.StockQuantity, &product.ImageURL, &product.Description, &product.DescriptionAr, &product.Unit, &product.SelforderEnabled, &product.CreatedAt)
	if err != nil {
		writeError(c, http.StatusNotFound, "product_not_found", "product not found")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": product, "meta": gin.H{"request_id": c.GetString("request_id")}})
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

type Product struct {
	ID               string    `json:"id"`
	CategoryID       string    `json:"category_id,omitempty"`
	Name             string    `json:"name"`
	NameAr           string    `json:"name_ar,omitempty"`
	SKU              string    `json:"sku"`
	Barcode          string    `json:"barcode"`
	PriceMinor       int64     `json:"price_minor"`
	CostMinor        int64     `json:"cost_minor"`
	Currency         string    `json:"currency"`
	StockQuantity    int64     `json:"stock_quantity"`
	ImageURL         string    `json:"image_url"`
	Description      string    `json:"description"`
	DescriptionAr    string    `json:"description_ar,omitempty"`
	Unit             string    `json:"unit"`
	IsActive         bool      `json:"is_active"`
	SelforderEnabled bool      `json:"selforder_enabled"`
	CreatedAt        time.Time `json:"created_at"`
}

func (h *Handler) tenantTx(c *gin.Context, claims security.Claims) (pgx.Tx, bool) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return nil, false
	}
	tx, err := h.pool.Begin(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return nil, false
	}
	if _, err = tx.Exec(c.Request.Context(), "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		_ = tx.Rollback(c.Request.Context())
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return nil, false
	}
	return tx, true
}

func validProduct(name, sku, currency string, price, cost, stock int64) bool {
	return strings.TrimSpace(name) != "" && strings.TrimSpace(sku) != "" &&
		len(currency) == 3 && price >= 0 && cost >= 0 && stock >= 0
}

func normalizeUnit(unit string) string {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return "piece"
	}
	if len(unit) > 16 {
		unit = unit[:16]
	}
	return unit
}

func insertProduct(c *gin.Context, tx pgx.Tx, tenantID string, request productRequest) (Product, error) {
	categoryID := ""
	if request.CategoryID != nil {
		categoryID = strings.TrimSpace(*request.CategoryID)
	}
	var product Product
	err := tx.QueryRow(c.Request.Context(), `
		INSERT INTO products (tenant_id, category_id, name, name_ar, sku, barcode, price_minor, cost_minor, currency, stock_quantity, image_url, description, description_ar, unit, selforder_enabled)
		VALUES ($1::uuid, NULLIF($2, '')::uuid, $3, NULLIF($4, ''), $5, NULLIF($6, ''), $7, $8, $9, $10, $11, $12, NULLIF($13, ''), $14, $15)
		RETURNING id, COALESCE(category_id::text, ''), name, COALESCE(name_ar, ''), sku, COALESCE(barcode, ''), price_minor, currency, stock_quantity, COALESCE(image_url, ''), COALESCE(description, ''), COALESCE(description_ar, ''), unit, is_active, selforder_enabled`,
		tenantID, categoryID, strings.TrimSpace(request.Name), strings.TrimSpace(nonNilString(request.NameAr)), strings.TrimSpace(request.SKU), strings.TrimSpace(request.Barcode), request.PriceMinor, request.CostMinor, strings.ToUpper(strings.TrimSpace(request.Currency)), request.StockQuantity, strings.TrimSpace(request.ImageURL), strings.TrimSpace(request.Description), strings.TrimSpace(nonNilString(request.DescriptionAr)), normalizeUnit(request.Unit), request.SelforderEnabled).Scan(
		&product.ID, &product.CategoryID, &product.Name, &product.NameAr, &product.SKU, &product.Barcode, &product.PriceMinor, &product.Currency, &product.StockQuantity, &product.ImageURL, &product.Description, &product.DescriptionAr, &product.Unit, &product.IsActive, &product.SelforderEnabled)
	return product, err
}

func nonNilString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func queryProduct(c *gin.Context, tx pgx.Tx, id string) (Product, error) {
	var product Product
	err := tx.QueryRow(c.Request.Context(), `
		SELECT id, COALESCE(category_id::text, ''), name, COALESCE(name_ar, ''), sku, COALESCE(barcode, ''), price_minor, currency, stock_quantity, COALESCE(image_url, ''), COALESCE(description, ''), COALESCE(description_ar, ''), unit, is_active, selforder_enabled, created_at
		FROM products WHERE id = $1::uuid`, id).Scan(
		&product.ID, &product.CategoryID, &product.Name, &product.NameAr, &product.SKU, &product.Barcode, &product.PriceMinor, &product.Currency, &product.StockQuantity, &product.ImageURL, &product.Description, &product.DescriptionAr, &product.Unit, &product.IsActive, &product.SelforderEnabled, &product.CreatedAt)
	return product, err
}

func applyPatch(product *Product, request productPatchRequest) {
	if request.CategoryID != nil {
		product.CategoryID = strings.TrimSpace(*request.CategoryID)
	}
	if request.Name != nil {
		product.Name = strings.TrimSpace(*request.Name)
	}
	if request.NameAr != nil {
		product.NameAr = strings.TrimSpace(*request.NameAr)
	}
	if request.SKU != nil {
		product.SKU = strings.TrimSpace(*request.SKU)
	}
	if request.Barcode != nil {
		product.Barcode = strings.TrimSpace(*request.Barcode)
	}
	if request.PriceMinor != nil {
		product.PriceMinor = *request.PriceMinor
	}
	if request.CostMinor != nil {
		product.CostMinor = *request.CostMinor
	}
	if request.Currency != nil {
		product.Currency = strings.ToUpper(strings.TrimSpace(*request.Currency))
	}
	if request.StockQuantity != nil {
		product.StockQuantity = *request.StockQuantity
	}
	if request.ImageURL != nil {
		product.ImageURL = strings.TrimSpace(*request.ImageURL)
	}
	if request.Description != nil {
		product.Description = strings.TrimSpace(*request.Description)
	}
	if request.DescriptionAr != nil {
		product.DescriptionAr = strings.TrimSpace(*request.DescriptionAr)
	}
	if request.Unit != nil {
		product.Unit = normalizeUnit(*request.Unit)
	}
	if request.IsActive != nil {
		product.IsActive = *request.IsActive
	}
	if request.SelforderEnabled != nil {
		product.SelforderEnabled = *request.SelforderEnabled
	}
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
