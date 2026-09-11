package customers

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
	router.GET("/customers", h.list)
	router.GET("/customers/:id", h.get)
	router.POST("/customers", h.create)
	router.PATCH("/customers/:id", h.update)
}

// Customer is one loyalty participant in a tenant.
type Customer struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Email              string `json:"email"`
	Phone              string `json:"phone"`
	LoyaltyPoints      int64  `json:"loyalty_points"`
	LoyaltyPointsTotal int64  `json:"loyalty_points_total"`
	CreatedAt          string `json:"created_at"`
}

type LoyaltyEntry struct {
	ID        string `json:"id"`
	SaleID    string `json:"sale_id"`
	Delta     int64  `json:"points_delta"`
	Reason    string `json:"reason"`
	CreatedAt string `json:"created_at"`
}

type createCustomerRequest struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type updateCustomerRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

func (h *Handler) list(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "customers", "read") {
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
	query := strings.TrimSpace(c.Query("q"))

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

	where := ""
	whereCount := ""
	args := []interface{}{limit, offset}
	if query != "" {
		pattern := "%" + query + "%"
		where = "WHERE name ILIKE $3 OR COALESCE(email, '') ILIKE $3 OR COALESCE(phone, '') ILIKE $3"
		whereCount = "WHERE name ILIKE $1 OR COALESCE(email, '') ILIKE $1 OR COALESCE(phone, '') ILIKE $1"
		args = append(args, pattern)
	}

	var total int64
	if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM customers `+whereCount, args[2:]...).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count customers")
		return
	}

	sqlQuery := `SELECT id::text, name, COALESCE(email, ''), COALESCE(phone, ''), loyalty_points, loyalty_points_total, created_at::text
		FROM customers ` + where + ` ORDER BY name LIMIT $1 OFFSET $2`
	rows, err := tx.Query(ctx, sqlQuery, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load customers")
		return
	}
	defer rows.Close()

	items := make([]Customer, 0)
	for rows.Next() {
		var item Customer
		if err := rows.Scan(&item.ID, &item.Name, &item.Email, &item.Phone, &item.LoyaltyPoints, &item.LoyaltyPointsTotal, &item.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load customers")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load customers")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load customers")
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

func (h *Handler) get(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "customers", "read") {
		return
	}
	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid customer id")
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

	var customer Customer
	err = tx.QueryRow(ctx, `
		SELECT id::text, name, COALESCE(email, ''), COALESCE(phone, ''), loyalty_points, loyalty_points_total, created_at::text
		FROM customers WHERE id = $1`, customerID).Scan(
		&customer.ID, &customer.Name, &customer.Email, &customer.Phone, &customer.LoyaltyPoints, &customer.LoyaltyPointsTotal, &customer.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "customer_not_found", "customer not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load customer")
		return
	}

	rows, err := tx.Query(ctx, `
		SELECT id::text, COALESCE(sale_id::text, ''), points_delta, reason, created_at::text
		FROM customer_loyalty_log WHERE customer_id = $1 ORDER BY created_at DESC LIMIT 20`, customerID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load loyalty log")
		return
	}
	defer rows.Close()

	entries := make([]LoyaltyEntry, 0)
	for rows.Next() {
		var entry LoyaltyEntry
		if err := rows.Scan(&entry.ID, &entry.SaleID, &entry.Delta, &entry.Reason, &entry.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load loyalty log")
			return
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load loyalty log")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load customer")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"customer":        customer,
			"loyalty_entries": entries,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) create(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	var request createCustomerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid customer request")
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	request.Email = strings.TrimSpace(request.Email)
	request.Phone = strings.TrimSpace(request.Phone)
	if request.Name == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "name is required")
		return
	}
	if request.Email != "" && !strings.Contains(request.Email, "@") {
		writeError(c, http.StatusBadRequest, "validation_error", "email is invalid")
		return
	}
	if len(request.Name) > 128 || len(request.Email) > 254 || len(request.Phone) > 32 {
		writeError(c, http.StatusBadRequest, "validation_error", "customer fields are too long")
		return
	}
	if !h.can(c, claims.Role, "customers", "write") {
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

	customerID := uuid.New()
	var createdAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO customers (id, tenant_id, name, email, phone)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''))
		RETURNING created_at::text`,
		customerID, tenantID, request.Name, request.Email, request.Phone).Scan(&createdAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save customer")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save customer")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": Customer{
			ID: customerID.String(), Name: request.Name, Email: request.Email, Phone: request.Phone,
			CreatedAt: createdAt,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) update(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "customers", "write") {
		return
	}
	var request updateCustomerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid customer request")
		return
	}
	customerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid customer id")
		return
	}
	if request.Name != nil && strings.TrimSpace(*request.Name) == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "name is required")
		return
	}
	if request.Email != nil && *request.Email != "" && !strings.Contains(*request.Email, "@") {
		writeError(c, http.StatusBadRequest, "validation_error", "email is invalid")
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

	name, email, phone, createdAt := "", "", "", ""
	// Load current values so PATCH defaults to keeping them.
	err = tx.QueryRow(ctx, `
		SELECT name, COALESCE(email, ''), COALESCE(phone, ''), created_at::text
		FROM customers WHERE id = $1`, customerID).Scan(&name, &email, &phone, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "customer_not_found", "customer not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load customer")
		return
	}
	if request.Name != nil {
		newName := strings.TrimSpace(*request.Name)
		if newName == "" {
			writeError(c, http.StatusBadRequest, "validation_error", "name is required")
			return
		}
		name = newName
	}
	if request.Email != nil {
		email = strings.TrimSpace(*request.Email)
		if email != "" && !strings.Contains(email, "@") {
			writeError(c, http.StatusBadRequest, "validation_error", "email is invalid")
			return
		}
	}
	if request.Phone != nil {
		phone = strings.TrimSpace(*request.Phone)
	}

	row := tx.QueryRow(ctx, `
		UPDATE customers SET name = $1, email = NULLIF($2, ''), phone = NULLIF($3, '')
		WHERE id = $4
		RETURNING name, COALESCE(email, ''), COALESCE(phone, ''), loyalty_points, loyalty_points_total, created_at::text`,
		name, email, phone, customerID)
	var customer Customer
	customer.ID = customerID.String()
	if err := row.Scan(&customer.Name, &customer.Email, &customer.Phone, &customer.LoyaltyPoints, &customer.LoyaltyPointsTotal, &customer.CreatedAt); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update customer")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update customer")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": customer,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) can(c *gin.Context, role, resource, action string) bool {
	if !httptransport.HasPermission(role, resource, action) {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return false
	}
	return true
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
