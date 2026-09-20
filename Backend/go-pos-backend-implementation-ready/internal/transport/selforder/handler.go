// Package selforder implements the "order from the table / online menu"
// vertical. A cashier prints a QR code whose payload is ONLY the public
// self-order URL (e.g. https://api.xamltech.com/selforder?tenant=slug). Any
// browser can open it: the page shows the store's published products
// (products.selforder_enabled AND in stock), lets a customer place an order
// (lands here as a pending self_order) and request a product for back-in-stock
// browser notifications. The cashier approves or cancels pending orders; only
// approval moves stock (it reuses the sales.CreateSale path), so orders never
// oversell even if stock changed between ordering and paying at the counter.
//
// Everything is tenant-RLS scoped: the public routes resolve the tenant by
// slug or UUID and run inside a transaction with app.current_tenant set.
package selforder

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	salestransport "github.com/example/pos-api/internal/transport/sales"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const maxOrderItems = 50
const maxOrderQuantity = 99

// WebPushConfig is the VAPID identity for back-in-stock browser notifications.
// An empty PrivateKey disables sending (the page then hides the notify button).
type WebPushConfig struct {
	PublicKey  string
	PrivateKey string
	Subject    string
}

// Handler exposes the public self-order surface and the staff management
// routes. pool may be nil (OpenAPI generator) — every route then answers 503.
type Handler struct {
	pool             *pgxpool.Pool
	tokens           security.TokenManager
	vapid            WebPushConfig
	discountLimitPct int
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager, vapid WebPushConfig, cashierDiscountPct int) *Handler {
	return &Handler{pool: pool, tokens: tokens, vapid: vapid, discountLimitPct: cashierDiscountPct}
}

// RegisterPublic mounts the routes any browser can call without signing in.
func (h *Handler) RegisterPublic(api *gin.RouterGroup) {
	api.GET("/selforder/menu/:tenant", h.menu)
	api.POST("/selforder/orders", h.createOrder)
	api.POST("/selforder/requests", h.createRequest)
}

// Register mounts the staff-facing routes (all bearer-auth + RBAC gated).
func (h *Handler) Register(api *gin.RouterGroup) {
	api.GET("/self-orders", h.listSelfOrders)
	api.POST("/self-orders/:id/approve", h.approveSelfOrder)
	api.POST("/self-orders/:id/cancel", h.cancelSelfOrder)
	api.GET("/product-requests", h.listRequests)
	api.POST("/product-requests/:id/fulfill", h.fulfillRequest)
	api.POST("/product-requests/:id/close", h.closeRequest)
}

type tenantInfo struct {
	ID       uuid.UUID
	Name     string
	Slug     string
	Currency string
	Language string
	Address  string
	Status   string
}

// resolveTenant maps a URL slug or UUID to the tenant row. It is the only
// public entry that touches the tenants table (no RLS on tenants) and it
// hides suspended/closed stores so a deactivated shop's menu disappears.
func (h *Handler) resolveTenant(ctx context.Context, ref string) (tenantInfo, error) {
	if h.pool == nil {
		return tenantInfo{}, errDBUnavailable{}
	}
	var t tenantInfo
	err := h.pool.QueryRow(ctx, `
		SELECT id, name, slug, currency_code, COALESCE(default_language, 'ar'), COALESCE(address, ''), status
		FROM tenants WHERE id::text = $1 OR slug = $1`, ref).Scan(
		&t.ID, &t.Name, &t.Slug, &t.Currency, &t.Language, &t.Address, &t.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return tenantInfo{}, errNotFound{"tenant_not_found", "store not found"}
	}
	if err != nil {
		return tenantInfo{}, errInternal{"unable to load store", err}
	}
	if t.Status != "active" {
		return tenantInfo{}, errNotFound{"tenant_not_found", "store is not available"}
	}
	return t, nil
}

// tenantTx begins a transaction with the tenant RLS context set.
func (h *Handler) tenantTx(ctx context.Context, tenantID uuid.UUID) (pgx.Tx, error) {
	if h.pool == nil {
		return nil, errDBUnavailable{}
	}
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, errDBUnavailable{}
	}
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		_ = tx.Rollback(ctx)
		return nil, errInternal{"unable to establish tenant context", err}
	}
	return tx, nil
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

// menuProduct is what a shopper sees for one published, in-stock item.
type menuProduct struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	PriceMinor    int64  `json:"price_minor"`
	Unit          string `json:"unit"`
	ImageURL      string `json:"image_url"`
	StockQuantity int64  `json:"stock_quantity"`
}

type menuCategory struct {
	ID       string        `json:"id,omitempty"`
	Name     string        `json:"name"`
	Slug     string        `json:"slug"`
	Products []menuProduct `json:"products"`
}

func (h *Handler) menu(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tenant, err := h.resolveTenant(c.Request.Context(), c.Param("tenant"))
	if err != nil {
		writeHTTPError(c, err)
		return
	}
	tx, err := h.tenantTx(c.Request.Context(), tenant.ID)
	if err != nil {
		writeHTTPError(c, err)
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()

	// Categories first, then published & in-stock products grouped by
	// category. Anything unpublished (selforder_enabled=false) or with zero
	// stock never appears — the "can't order when inventory is 0 / out of
	// stock" rule is enforced at read time.
	catRows, err := tx.Query(c.Request.Context(),
		`SELECT id, name, slug FROM categories WHERE is_active ORDER BY name`)
	if err == nil {
		defer catRows.Close()
	}
	var categories []menuCategory
	if err == nil {
		for catRows.Next() {
			var mc menuCategory
			if err = catRows.Scan(&mc.ID, &mc.Name, &mc.Slug); err != nil {
				break
			}
			mc.Products = []menuProduct{}
			categories = append(categories, mc)
		}
	}
	if err == nil {
		err = catRows.Err()
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load menu")
		return
	}

	prodRows, err := tx.Query(c.Request.Context(), `
		SELECT p.id, p.name, COALESCE(p.description, ''), p.price_minor, COALESCE(p.unit, 'piece'),
		       COALESCE(p.image_url, ''), p.stock_quantity, COALESCE(p.category_id::text, '')
		FROM products p
		WHERE p.is_active AND p.selforder_enabled AND p.stock_quantity > 0
		ORDER BY p.name`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load menu")
		return
	}
	defer prodRows.Close()

	byID := make(map[string]*menuCategory, len(categories))
	uncategorized := &menuCategory{Name: "uncategorized", Products: []menuProduct{}}
	for i := range categories {
		byID[categories[i].ID] = &categories[i]
	}
	published := 0
	for prodRows.Next() {
		var p menuProduct
		var catID string
		if err = prodRows.Scan(&p.ID, &p.Name, &p.Description, &p.PriceMinor, &p.Unit, &p.ImageURL, &p.StockQuantity, &catID); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load menu")
			return
		}
		published++
		if byID[catID] != nil {
			byID[catID].Products = append(byID[catID].Products, p)
		} else {
			uncategorized.Products = append(uncategorized.Products, p)
		}
	}
	if err = prodRows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load menu")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load menu")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"tenant": gin.H{
				"id":   tenant.ID.String(),
				"name": tenant.Name, "slug": tenant.Slug,
				"currency": tenant.Currency,
				"language": tenant.Language, "address": tenant.Address,
			},
			"published":     published,
			"categories":    categories,
			"uncategorized": uncategorized.Products,
			"request_id":    c.GetString("request_id"),
		},
	})
}

type orderItemRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	Quantity  int64  `json:"quantity" binding:"required"`
}

type createOrderRequest struct {
	Tenant       string             `json:"tenant" binding:"required"`
	TableName    string             `json:"table_name"`
	CustomerName string             `json:"customer_name"`
	Items        []orderItemRequest `json:"items" binding:"required,min=1"`
}

type selfOrder struct {
	ID             string            `json:"id"`
	Reference      string            `json:"reference"`
	CustomerName   string            `json:"customer_name"`
	TableName      string            `json:"table_name"`
	Status         string            `json:"status"`
	SubtotalMinor  int64             `json:"subtotal_minor"`
	TotalMinor     int64             `json:"total_minor"`
	Currency       string            `json:"currency"`
	Items          []orderItemResult `json:"items"`
	ApprovedSaleID string            `json:"approved_sale_id,omitempty"`
	CreatedAt      string            `json:"created_at"`
}

type orderItemResult struct {
	ProductID string `json:"product_id"`
	Name      string `json:"name"`
	SKU       string `json:"sku"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unit_price_minor"`
	Total     int64  `json:"total_minor"`
}

func (h *Handler) createOrder(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	var request createOrderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "items are required")
		return
	}
	if len(request.Items) > maxOrderItems {
		writeError(c, http.StatusBadRequest, "validation_error", "too many items")
		return
	}
	for _, item := range request.Items {
		if item.Quantity < 1 || item.Quantity > maxOrderQuantity {
			writeError(c, http.StatusBadRequest, "validation_error", "quantity must be between 1 and 99")
			return
		}
	}
	tenant, err := h.resolveTenant(c.Request.Context(), request.Tenant)
	if err != nil {
		writeHTTPError(c, err)
		return
	}
	tx, err := h.tenantTx(c.Request.Context(), tenant.ID)
	if err != nil {
		writeHTTPError(c, err)
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()

	items := make([]orderItemResult, 0, len(request.Items))
	var subtotal int64
	currency := tenant.Currency
	for _, item := range request.Items {
		productID, parseErr := uuid.Parse(item.ProductID)
		if parseErr != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "product_id is invalid")
			return
		}
		var name, sku string
		var stock, price int64
		err = tx.QueryRow(c.Request.Context(), `
			SELECT name, sku, stock_quantity, price_minor
			FROM products WHERE id = $1 AND is_active FOR UPDATE`, productID).Scan(&name, &sku, &stock, &price)
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(c, http.StatusNotFound, "product_not_found", "product not found")
			return
		}
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
			return
		}
		if stock <= 0 || item.Quantity > stock {
			writeError(c, http.StatusConflict, "out_of_stock", fmt.Sprintf("%s is currently out of stock", name))
			return
		}
		total := price * item.Quantity
		subtotal += total
		items = append(items, orderItemResult{ProductID: productID.String(), Name: name, SKU: sku, Quantity: item.Quantity, UnitPrice: price, Total: total})
	}

	orderID := uuid.New()
	reference := "SO-" + strings.ToUpper(orderID.String()[:8])
	encoded, _ := json.Marshal(items)
	var order selfOrder
	err = tx.QueryRow(c.Request.Context(), `
		INSERT INTO self_orders (id, tenant_id, reference, customer_name, table_name, items, subtotal_minor, total_minor, currency, status)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, 'pending')
		RETURNING id, reference, customer_name, table_name, status, subtotal_minor, total_minor, currency, created_at::text`,
		orderID, tenant.ID, reference, strings.TrimSpace(request.CustomerName), strings.TrimSpace(request.TableName), encoded, subtotal, subtotal, currency).Scan(
		&order.ID, &order.Reference, &order.CustomerName, &order.TableName, &order.Status, &order.SubtotalMinor, &order.TotalMinor, &order.Currency, &order.CreatedAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to place order")
		return
	}
	order.Items = items
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to place order")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": order, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

type webPushKeys struct {
	P256DH string `json:"p256dh"`
	Auth   string `json:"auth"`
}

type webPushSub struct {
	Endpoint string      `json:"endpoint"`
	Keys     webPushKeys `json:"keys"`
}

type createRequestRequest struct {
	Tenant      string      `json:"tenant" binding:"required"`
	ProductID   string      `json:"product_id"`
	ProductName string      `json:"product_name"`
	Note        string      `json:"note"`
	Contact     string      `json:"contact"`
	WebPush     *webPushSub `json:"webpush"`
}

type productRequest struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Note        string `json:"note"`
	Contact     string `json:"contact"`
	HasWebPush  bool   `json:"has_webpush"`
	Status      string `json:"status"`
	NotifiedAt  string `json:"notified_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func validWebPushSub(sub *webPushSub) bool {
	if sub == nil {
		return true
	}
	return strings.HasPrefix(sub.Endpoint, "https://") &&
		sub.Keys.P256DH != "" && sub.Keys.Auth != ""
}

func (h *Handler) createRequest(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	var request createRequestRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid request")
		return
	}
	productName := strings.TrimSpace(request.ProductName)
	if request.ProductID != "" {
		productName = "" // resolved below, server-side
	} else if productName == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "product_name is required")
		return
	}
	if !validWebPushSub(request.WebPush) {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid webpush subscription")
		return
	}
	tenant, err := h.resolveTenant(c.Request.Context(), request.Tenant)
	if err != nil {
		writeHTTPError(c, err)
		return
	}
	tx, err := h.tenantTx(c.Request.Context(), tenant.ID)
	if err != nil {
		writeHTTPError(c, err)
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()

	var productID *uuid.UUID
	if request.ProductID != "" {
		parsed, parseErr := uuid.Parse(request.ProductID)
		if parseErr != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "product_id is invalid")
			return
		}
		var existingName string
		err = tx.QueryRow(c.Request.Context(), `SELECT name FROM products WHERE id = $1 AND is_active`, parsed).Scan(&existingName)
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(c, http.StatusNotFound, "product_not_found", "product not found")
			return
		}
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
			return
		}
		productID = &parsed
		productName = existingName
	}

	var webpushJSON []byte
	if request.WebPush != nil {
		webpushJSON, _ = json.Marshal(request.WebPush)
	}
	var result productRequest
	err = tx.QueryRow(c.Request.Context(), `
		INSERT INTO product_requests (tenant_id, product_id, product_name, note, contact, webpush, status)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, 'null')::jsonb, 'open')
		RETURNING id, COALESCE(product_id::text, ''), product_name, note, contact, status, created_at::text`,
		tenant.ID, productID, productName, strings.TrimSpace(request.Note), strings.TrimSpace(request.Contact), webpushJSON).Scan(
		&result.ID, &result.ProductID, &result.ProductName, &result.Note, &result.Contact, &result.Status, &result.CreatedAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save request")
		return
	}
	result.HasWebPush = request.WebPush != nil
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save request")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": result, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) tenantClaims(c *gin.Context) (security.Claims, pgx.Tx, bool) {
	claims, ok := h.authenticate(c)
	if !ok {
		return security.Claims{}, nil, false
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return security.Claims{}, nil, false
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return security.Claims{}, nil, false
	}
	tx, err := h.tenantTx(c.Request.Context(), tenantID)
	if err != nil {
		writeHTTPError(c, err)
		return security.Claims{}, nil, false
	}
	return claims, tx, true
}

func (h *Handler) listSelfOrders(c *gin.Context) {
	claims, tx, ok := h.tenantClaims(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	if !httptransport.HasPermission(claims.Role, "pos", "read") {
		writeError(c, http.StatusForbidden, "permission_denied", "reading self-orders is not allowed for this role")
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != "pending" && status != "approved" && status != "cancelled" {
		writeError(c, http.StatusBadRequest, "validation_error", "status must be pending, approved or cancelled")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 50
	}
	where := ""
	arg := []any{}
	if status != "" {
		where = "WHERE status = $1"
		arg = append(arg, status)
	}
	arg = append(arg, limit, (page-1)*limit)

	var total int64
	if err := tx.QueryRow(c.Request.Context(), `SELECT COUNT(*) FROM self_orders `+where, arg[:1]...).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load self-orders")
		return
	}

	rows, err := tx.Query(c.Request.Context(), `
		SELECT id, reference, customer_name, table_name, status, subtotal_minor, total_minor, currency,
		       items, COALESCE(approved_sale_id::text, ''), created_at::text
		FROM self_orders `+where+`
		ORDER BY created_at DESC LIMIT $`+strconv.Itoa(len(arg)-1)+` OFFSET $`+strconv.Itoa(len(arg)),
		arg...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load self-orders")
		return
	}
	defer rows.Close()
	orders := make([]selfOrder, 0)
	for rows.Next() {
		var o selfOrder
		var rawItems []byte
		if err = rows.Scan(&o.ID, &o.Reference, &o.CustomerName, &o.TableName, &o.Status, &o.SubtotalMinor, &o.TotalMinor, &o.Currency, &rawItems, &o.ApprovedSaleID, &o.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load self-orders")
			return
		}
		_ = json.Unmarshal(rawItems, &o.Items)
		orders = append(orders, o)
	}
	if err = rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load self-orders")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load self-orders")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": orders,
		"meta": gin.H{"request_id": c.GetString("request_id"), "page": page, "limit": limit, "total": total},
	})
}

func (h *Handler) approveSelfOrder(c *gin.Context) {
	claims, tx, ok := h.tenantClaims(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	if !httptransport.HasPermission(claims.Role, "pos", "sale") {
		writeError(c, http.StatusForbidden, "permission_denied", "approving self-orders is not allowed for this role")
		return
	}
	orderID, parseErr := uuid.Parse(c.Param("id"))
	if parseErr != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid self-order id")
		return
	}
	tenantID, _ := uuid.Parse(claims.TenantID)
	userID, _ := uuid.Parse(claims.UserID)
	deviceID, _ := uuid.Parse(claims.DeviceID)

	var request struct {
		SessionID string `json:"session_id"`
	}
	_ = c.ShouldBindJSON(&request)

	var status string
	var rawItems []byte
	var subtotal int64
	var currency string
	err := tx.QueryRow(c.Request.Context(), `
		SELECT status, items, subtotal_minor, currency FROM self_orders WHERE id = $1`, orderID).
		Scan(&status, &rawItems, &subtotal, &currency)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "self_order_not_found", "self-order not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load self-order")
		return
	}
	if status == "approved" {
		writeError(c, http.StatusConflict, "already_approved", "self-order is already approved")
		return
	}
	if status == "cancelled" {
		writeError(c, http.StatusConflict, "already_cancelled", "self-order is already cancelled")
		return
	}

	var items []orderItemResult
	if err = json.Unmarshal(rawItems, &items); err != nil || len(items) == 0 {
		writeError(c, http.StatusInternalServerError, "internal_error", "self-order has no items")
		return
	}

	// Re-validate the basket against the live catalog: products that were
	// unpublished or ran out since ordering cannot be fulfilled — the counter
	// sale must match the shopper's intent. The stock check is advisory here;
	// CreateSale below locks the rows and is authoritative.
	for i := range items {
		var active, selfEnabled bool
		var stock int64
		if err = tx.QueryRow(c.Request.Context(), `
			SELECT is_active, selforder_enabled, stock_quantity
			FROM products WHERE id = $1::uuid`, items[i].ProductID).Scan(&active, &selfEnabled, &stock); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to re-check product")
			return
		}
		if !active {
			writeError(c, http.StatusConflict, "product_removed", fmt.Sprintf("%s is no longer sold", items[i].Name))
			return
		}
		if !selfEnabled {
			writeError(c, http.StatusConflict, "product_offline", fmt.Sprintf("%s is not published for online ordering", items[i].Name))
			return
		}
		if stock <= 0 {
			writeError(c, http.StatusConflict, "out_of_stock", fmt.Sprintf("%s is currently out of stock", items[i].Name))
			return
		}
	}
	_ = subtotal
	_ = currency

	saleReq := salestransport.CreateSaleRequest{
		Items: make([]salestransport.SaleItemRequest, 0, len(items)),
	}
	for _, item := range items {
		saleReq.Items = append(saleReq.Items, salestransport.SaleItemRequest{ProductID: item.ProductID, Quantity: item.Quantity})
	}
	saleReq.RegisterSessionID = request.SessionID

	sale, saleErr := salestransport.CreateSale(
		c.Request.Context(), tx, tenantID, userID, deviceID, claims.Role,
		"selforder:"+orderID.String(), h.discountLimitPct, saleReq)
	if saleErr != nil {
		writeHTTPError(c, saleErr)
		return
	}
	if _, err = tx.Exec(c.Request.Context(), `
		UPDATE self_orders SET status = 'approved', approved_sale_id = $1, processed_by = $2, updated_at = NOW()
		WHERE id = $3 AND status = 'pending'`, sale.ID, userID, orderID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to approve self-order")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to approve self-order")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"order_id": orderID.String(),
			"status":   "approved",
			"sale": gin.H{
				"id": sale.ID, "subtotal_minor": sale.SubtotalMinor, "total_minor": sale.TotalMinor,
				"currency": sale.Currency, "payment_method": sale.PaymentMethod, "created_at": sale.CreatedAt,
			},
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) cancelSelfOrder(c *gin.Context) {
	claims, tx, ok := h.tenantClaims(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	if !httptransport.HasPermission(claims.Role, "pos", "sale") {
		writeError(c, http.StatusForbidden, "permission_denied", "cancelling self-orders is not allowed for this role")
		return
	}
	orderID, parseErr := uuid.Parse(c.Param("id"))
	if parseErr != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid self-order id")
		return
	}
	userID, _ := uuid.Parse(claims.UserID)
	var status string
	err := tx.QueryRow(c.Request.Context(), `SELECT status FROM self_orders WHERE id = $1`, orderID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "self_order_not_found", "self-order not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load self-order")
		return
	}
	if status == "approved" {
		writeError(c, http.StatusConflict, "already_approved", "an approved self-order cannot be cancelled")
		return
	}
	if status == "cancelled" {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"order_id": orderID.String(), "status": "cancelled"}, "meta": gin.H{"request_id": c.GetString("request_id")}})
		return
	}
	if _, err = tx.Exec(c.Request.Context(), `
		UPDATE self_orders SET status = 'cancelled', processed_by = $1, updated_at = NOW()
		WHERE id = $2 AND status = 'pending'`, userID, orderID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to cancel self-order")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to cancel self-order")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"order_id": orderID.String(), "status": "cancelled"}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) listRequests(c *gin.Context) {
	claims, tx, ok := h.tenantClaims(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	if !httptransport.HasPermission(claims.Role, "pos", "read") {
		writeError(c, http.StatusForbidden, "permission_denied", "reading product requests is not allowed for this role")
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	if status != "" && status != "open" && status != "fulfilled" && status != "closed" {
		writeError(c, http.StatusBadRequest, "validation_error", "status must be open, fulfilled or closed")
		return
	}
	where := ""
	arg := []any{}
	if status != "" {
		where = "WHERE status = $1"
		arg = append(arg, status)
	}
	rows, err := tx.Query(c.Request.Context(), `
		SELECT id, COALESCE(product_id::text, ''), product_name, note, contact,
		       (webpush IS NOT NULL), status, COALESCE(notified_at::text, ''), created_at::text
		FROM product_requests `+where+`
		ORDER BY created_at DESC`, arg...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product requests")
		return
	}
	defer rows.Close()
	requests := make([]productRequest, 0)
	for rows.Next() {
		var r productRequest
		if err = rows.Scan(&r.ID, &r.ProductID, &r.ProductName, &r.Note, &r.Contact, &r.HasWebPush, &r.Status, &r.NotifiedAt, &r.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product requests")
			return
		}
		requests = append(requests, r)
	}
	if err = rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product requests")
		return
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product requests")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": requests, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) fulfillRequest(c *gin.Context) {
	h.transitionRequest(c, "fulfilled")
}

func (h *Handler) closeRequest(c *gin.Context) {
	h.transitionRequest(c, "closed")
}

func (h *Handler) transitionRequest(c *gin.Context, target string) {
	claims, tx, ok := h.tenantClaims(c)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	if !httptransport.HasPermission(claims.Role, "catalog", "write") {
		writeError(c, http.StatusForbidden, "permission_denied", "changing product requests is not allowed for this role")
		return
	}
	id, parseErr := uuid.Parse(c.Param("id"))
	if parseErr != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid product request id")
		return
	}
	ct, err := tx.Exec(c.Request.Context(), `
		UPDATE product_requests SET status = $1, updated_at = NOW() WHERE id = $2 AND status <> $1`, target, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update product request")
		return
	}
	if ct.RowsAffected() == 0 {
		var exists bool
		if err = tx.QueryRow(c.Request.Context(), `SELECT EXISTS(SELECT 1 FROM product_requests WHERE id = $1)`, id).Scan(&exists); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product request")
			return
		}
		if !exists {
			writeError(c, http.StatusNotFound, "product_request_not_found", "product request not found")
			return
		}
	}
	if err := tx.Commit(c.Request.Context()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update product request")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id.String(), "status": target}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

// writeHTTPError maps the package's internal error helpers onto HTTP JSON.

type errDBUnavailable struct{}

func (errDBUnavailable) Error() string { return "database unavailable" }

type errNotFound struct {
	code, message string
}

func (e errNotFound) Error() string { return e.message }

type errInternal struct {
	message string
	cause   error
}

func (e errInternal) Error() string { return e.message }

func (e errInternal) Unwrap() error { return e.cause }

// writeHTTPError maps the package's internal error helpers onto HTTP JSON.
func writeHTTPError(c *gin.Context, err error) {
	var notFound errNotFound
	switch {
	case errors.As(err, &notFound):
		writeError(c, http.StatusNotFound, notFound.code, notFound.message)
	case errors.As(err, &errInternal{}):
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to process request")
	case errors.As(err, &errDBUnavailable{}):
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to process request")
	}
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
