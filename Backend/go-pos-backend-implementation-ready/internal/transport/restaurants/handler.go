package restaurants

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Handler is the C1 restaurant module: floor plans, table status management,
// and the split-bill endpoint. KDS (kitchen display system) is a realtime
// push feature and is deferred; the floor/table + split/tip surface is here.
type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/floors", h.listFloors)
	router.POST("/floors", h.createFloor)
	router.GET("/tables", h.listTables)
	router.POST("/tables", h.createTable)
	router.PATCH("/tables/:id", h.patchTable)
	router.POST("/sales/:id/split", h.splitSale)
}

type Floor struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type Table struct {
	ID      string `json:"id"`
	FloorID string `json:"floor_id"`
	Name    string `json:"name"`
	Seats   int    `json:"seats"`
	Status  string `json:"status"`
	Floor   string `json:"floor_name,omitempty"`
}

// validTableStatus mirrors the DB CHECK constraint.
func validTableStatus(status string) bool {
	switch status {
	case "free", "occupied", "reserved", "closed":
		return true
	default:
		return false
	}
}

func (h *Handler) listFloors(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "restaurant", "read") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
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
	rows, err := tx.Query(ctx, `SELECT id, name, sort_order FROM floors ORDER BY sort_order, name`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load floors")
		return
	}
	defer rows.Close()
	items := make([]Floor, 0)
	for rows.Next() {
		var f Floor
		if err := rows.Scan(&f.ID, &f.Name, &f.SortOrder); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load floors")
			return
		}
		items = append(items, f)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load floors")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load floors")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) createFloor(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "restaurant", "write") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	var request struct {
		Name      string `json:"name" binding:"required"`
		SortOrder int    `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Name) == "" || len(request.Name) > 80 {
		writeError(c, http.StatusBadRequest, "validation_error", "name is required and must be under 80 characters")
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
	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO floors (id, tenant_id, name, sort_order) VALUES ($1, $2, $3, $4)
		RETURNING id`, uuid.New(), tenantID, strings.TrimSpace(request.Name), request.SortOrder).Scan(&id)
	if err != nil {
		writeError(c, http.StatusConflict, "duplicate_floor", "floor name already exists")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save floor")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": Floor{ID: id.String(), Name: request.Name, SortOrder: request.SortOrder}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) listTables(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "restaurant", "read") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
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
	floorID := c.Query("floor_id")
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
	query := `
		SELECT t.id, t.floor_id::text, t.name, t.seats, t.status, f.name
		FROM restaurant_tables t JOIN floors f ON f.tenant_id = t.tenant_id AND f.id = t.floor_id`
	args := []any{}
	if floorID != "" {
		query += ` WHERE t.floor_id = $1::uuid`
		args = append(args, floorID)
	}
	query += ` ORDER BY f.sort_order, t.name`
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tables")
		return
	}
	defer rows.Close()
	items := make([]Table, 0)
	for rows.Next() {
		var t Table
		if err := rows.Scan(&t.ID, &t.FloorID, &t.Name, &t.Seats, &t.Status, &t.Floor); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tables")
			return
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tables")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tables")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) createTable(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "restaurant", "write") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	var request struct {
		FloorID string `json:"floor_id" binding:"required"`
		Name    string `json:"name" binding:"required"`
		Seats   int    `json:"seats"`
		Status  string `json:"status"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || strings.TrimSpace(request.Name) == "" || len(request.Name) > 80 {
		writeError(c, http.StatusBadRequest, "validation_error", "floor_id and name are required")
		return
	}
	floorID, err := uuid.Parse(request.FloorID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "floor_id is invalid")
		return
	}
	status := request.Status
	if status == "" {
		status = "free"
	}
	if !validTableStatus(status) {
		writeError(c, http.StatusBadRequest, "validation_error", "status must be one of free, occupied, reserved, closed")
		return
	}
	if request.Seats < 1 {
		request.Seats = 2
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
	var floorExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM floors WHERE id = $1)`, floorID).Scan(&floorExists); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to check floor")
		return
	}
	if !floorExists {
		writeError(c, http.StatusNotFound, "floor_not_found", "floor not found")
		return
	}
	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO restaurant_tables (id, tenant_id, floor_id, name, seats, status)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		uuid.New(), tenantID, floorID, strings.TrimSpace(request.Name), request.Seats, status).Scan(&id)
	if err != nil {
		writeError(c, http.StatusConflict, "duplicate_table", "a table with this name already exists on the floor")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save table")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": Table{ID: id.String(), FloorID: request.FloorID, Name: request.Name, Seats: request.Seats, Status: status},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) patchTable(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "restaurant", "write") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	tableID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "table id is invalid")
		return
	}
	var request struct {
		Status *string `json:"status"`
		Seats  *int    `json:"seats"`
		Name   *string `json:"name"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid table update")
		return
	}
	if request.Status != nil && !validTableStatus(*request.Status) {
		writeError(c, http.StatusBadRequest, "validation_error", "status must be one of free, occupied, reserved, closed")
		return
	}
	if request.Seats != nil && *request.Seats < 1 {
		writeError(c, http.StatusBadRequest, "validation_error", "seats must be at least 1")
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
	tag, err := tx.Exec(ctx, `
		UPDATE restaurant_tables
		SET status = COALESCE($1, status), seats = COALESCE($2, seats), name = COALESCE($3, name)
		WHERE id = $4`, request.Status, request.Seats, request.Name, tableID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update table")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "table_not_found", "table not found")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update table")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": tableID.String()}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

// splitSale breaks an existing sale into child sales (split bills). The client
// supplies, per child, the line quantities to take; per line the child
// quantities must sum to the parent's quantity. Children reuse the product
// rows (stock was already decremented on the parent), carry the same currency,
// and are paid cash. KDS/tips are outside this operation.
func (h *Handler) splitSale(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "pos", "sale") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "sale id is invalid")
		return
	}
	var request struct {
		Children []struct {
			Lines []struct {
				SaleItemID string `json:"sale_item_id" binding:"required"`
				Quantity   int64  `json:"quantity" binding:"required"`
			} `json:"lines" binding:"required,min=1"`
		} `json:"children" binding:"required,min=2"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "children with line splits are required")
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

	var currency string
	err = tx.QueryRow(ctx, `SELECT currency FROM sales WHERE id = $1`, saleID).Scan(&currency)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "sale_not_found", "sale not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale")
		return
	}

	// Load parent items and their unit prices.
	rows, err := tx.Query(ctx, `SELECT id::text, product_id, product_name, sku, quantity, unit_price_minor FROM sale_items WHERE sale_id = $1`, saleID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sale items")
		return
	}
	type parentItem struct {
		ID        string
		ProductID uuid.UUID
		Name      string
		SKU       string
		Price     int64
		Quantity  int64
	}
	parents := map[string]parentItem{}
	for rows.Next() {
		var p parentItem
		var price int64
		if err := rows.Scan(&p.ID, &p.ProductID, &p.Name, &p.SKU, &p.Quantity, &price); err != nil {
			rows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to read sale items")
			return
		}
		p.Price = price / p.Quantity
		parents[p.ID] = p
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to read sale items")
		return
	}

	// Verify every requested line exists and quantities per parent line match.
	perParent := map[string]int64{}
	for _, child := range request.Children {
		for _, line := range child.Lines {
			if _, exists := parents[line.SaleItemID]; !exists {
				writeError(c, http.StatusBadRequest, "validation_error", "sale_item_id does not belong to this sale")
				return
			}
			if line.Quantity <= 0 {
				writeError(c, http.StatusBadRequest, "validation_error", "quantities must be positive")
				return
			}
			perParent[line.SaleItemID] += line.Quantity
		}
	}
	for itemID, wanted := range perParent {
		if wanted != parents[itemID].Quantity {
			writeError(c, http.StatusBadRequest, "validation_error", "split quantities must match the sale line quantities")
			return
		}
	}

	created := make([]string, 0, len(request.Children))
	for _, child := range request.Children {
		var childSubtotal int64
		childID := uuid.New()
		_, err = tx.Exec(ctx, `
			INSERT INTO sales (id, tenant_id, device_id, created_by, subtotal_minor, total_minor, currency, payment_method, parent_sale_id, idempotency_key)
			VALUES ($1, $2, (SELECT device_id FROM sales WHERE id = $3), (SELECT created_by FROM sales WHERE id = $3), $4, $4, (SELECT currency FROM sales WHERE id = $3), 'cash', $3, $5)`,
			childID, tenantID, saleID, 0, "split-of:"+saleID.String()+":"+childID.String())
		if err != nil {
			slog.Error("split sale: create child sale failed", "sale_id", saleID.String(), "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to create split sale")
			return
		}
		for _, line := range child.Lines {
			p := parents[line.SaleItemID]
			lineTotal := p.Price * line.Quantity
			if _, err = tx.Exec(ctx, `
				INSERT INTO sale_items (tenant_id, sale_id, product_id, product_name, sku, quantity, unit_price_minor, total_minor)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				tenantID, childID, p.ProductID, p.Name, p.SKU, line.Quantity, p.Price, lineTotal); err != nil {
				writeError(c, http.StatusInternalServerError, "internal_error", "unable to save split line")
				return
			}
			childSubtotal += lineTotal
		}
		if _, err = tx.Exec(ctx, `UPDATE sales SET subtotal_minor = $1, total_minor = $1 WHERE id = $2`, childSubtotal, childID); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to finalize split sale")
			return
		}
		if _, err = tx.Exec(ctx, `
			INSERT INTO sale_payments (tenant_id, sale_id, method, amount_minor)
			VALUES ($1, $2, 'cash', $3)`, tenantID, childID, childSubtotal); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to save split payment")
			return
		}
		created = append(created, childID.String())
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit split")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{"parent_sale_id": saleID.String(), "children": created, "currency": currency},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
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
