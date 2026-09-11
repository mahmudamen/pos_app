package inventory

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
	pool   *pgxpool.Pool
	tokens security.TokenManager
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.POST("/inventory/adjustments", h.create)
	router.GET("/inventory/adjustments", h.list)
}

type createAdjustmentRequest struct {
	ProductID     string `json:"product_id" binding:"required"`
	Reason        string `json:"reason" binding:"required"`
	QuantityDelta int64  `json:"quantity_delta" binding:"required"`
	Note          string `json:"note"`
}

// Adjustment is one stock movement that is not a sale. It is the only way a
// product's stock_quantity may be changed outside of checkout.
type Adjustment struct {
	ID            string `json:"id"`
	ProductID     string `json:"product_id"`
	ProductName   string `json:"product_name"`
	SKU           string `json:"sku"`
	Reason        string `json:"reason"`
	QuantityDelta int64  `json:"quantity_delta"`
	Note          string `json:"note"`
	CreatedBy     string `json:"created_by"`
	CreatedAt     string `json:"created_at"`
}

// validReason returns whether reason is an allowed adjustment reason code.
func validReason(reason string) bool {
	switch reason {
	case "damaged", "restock", "count":
		return true
	default:
		return false
	}
}

func (h *Handler) create(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	var request createAdjustmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid adjustment request")
		return
	}
	if !validReason(request.Reason) {
		writeError(c, http.StatusBadRequest, "validation_error", "reason must be one of damaged, restock, count")
		return
	}
	if request.QuantityDelta == 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "quantity_delta must not be zero")
		return
	}
	if len(request.Note) > 255 {
		writeError(c, http.StatusBadRequest, "validation_error", "note is too long")
		return
	}
	productID, err := uuid.Parse(request.ProductID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "product_id is invalid")
		return
	}
	if !httptransport.HasPermission(claims.Role, "inventory", "adjust") {
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
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var name, sku string
	var stock int64
	err = tx.QueryRow(ctx, `
		SELECT name, sku, stock_quantity FROM products WHERE id = $1 AND is_active FOR UPDATE`, productID).
		Scan(&name, &sku, &stock)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "product_not_found", "product not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
		return
	}
	if stock+request.QuantityDelta < 0 {
		writeError(c, http.StatusConflict, "insufficient_stock", "adjustment would make stock negative")
		return
	}

	adjustmentID := uuid.New()
	var createdAt, actorName string
	err = tx.QueryRow(ctx, `
		INSERT INTO inventory_adjustments (id, tenant_id, product_id, reason, quantity_delta, note, created_by)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)
		RETURNING created_at::text,
		          (SELECT display_name FROM users u WHERE u.tenant_id = inventory_adjustments.tenant_id AND u.id = inventory_adjustments.created_by)`,
		adjustmentID, tenantID, productID, request.Reason, request.QuantityDelta, request.Note, userID).Scan(&createdAt, &actorName)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save adjustment")
		return
	}
	if _, err = tx.Exec(ctx, `
		UPDATE products SET stock_quantity = stock_quantity + $1 WHERE id = $2`,
		request.QuantityDelta, productID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update stock")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save adjustment")
		return
	}
	stock += request.QuantityDelta

	c.JSON(http.StatusCreated, gin.H{
		"data": Adjustment{
			ID: adjustmentID.String(), ProductID: request.ProductID, ProductName: name, SKU: sku,
			Reason: request.Reason, QuantityDelta: request.QuantityDelta, Note: request.Note,
			CreatedBy: actorName, CreatedAt: createdAt,
		},
		"meta": gin.H{"request_id": c.GetString("request_id"), "stock_quantity": stock},
	})
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
	if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM inventory_adjustments`).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count adjustments")
		return
	}

	rows, err := tx.Query(ctx, `
		SELECT ia.id, ia.product_id::text, p.name, p.sku, ia.reason, ia.quantity_delta,
		       COALESCE(ia.note, ''), COALESCE(u.display_name, ''), ia.created_at::text
		FROM inventory_adjustments ia
		JOIN products p ON p.tenant_id = ia.tenant_id AND p.id = ia.product_id
		LEFT JOIN users u ON u.tenant_id = ia.tenant_id AND u.id = ia.created_by
		ORDER BY ia.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load adjustments")
		return
	}
	defer rows.Close()

	items := make([]Adjustment, 0)
	for rows.Next() {
		var item Adjustment
		if err := rows.Scan(&item.ID, &item.ProductID, &item.ProductName, &item.SKU, &item.Reason,
			&item.QuantityDelta, &item.Note, &item.CreatedBy, &item.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load adjustments")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load adjustments")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load adjustments")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{
			"request_id": c.GetString("request_id"),
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
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
