package users

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/example/pos-api/internal/transport/access"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	router.GET("/users", h.listUsers)
	router.POST("/users", h.createUser)
	router.PUT("/users/:id/security", h.updateSecurity)
}

type User struct {
	ID          string             `json:"id"`
	Email       string             `json:"email"`
	DisplayName string             `json:"display_name"`
	Role        string             `json:"role"`
	AccountType string             `json:"account_type"`
	IsActive    bool               `json:"is_active"`
	Permissions access.Permissions `json:"permissions"`
}

type createUserRequest struct {
	Email       string `json:"email" binding:"required,email"`
	DisplayName string `json:"display_name" binding:"required"`
	Password    string `json:"password" binding:"required,min=8"`
	Role        string `json:"role" binding:"required,oneof=owner manager cashier"`
	AccountType string `json:"account_type" binding:"omitempty,oneof=standard demo guest"`
}

func (h *Handler) listUsers(c *gin.Context) {
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
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var total int64
	err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count users")
		return
	}

	rows, err := tx.Query(ctx, `
		SELECT u.id, u.email, u.display_name, u.role, u.account_type, u.is_active,
		       s.access_level, s.max_discount_pct, COALESCE(s.use_custom_permissions, FALSE),
		       COALESCE(s.can_delete_order, FALSE), COALESCE(s.can_delete_line, FALSE),
		       COALESCE(s.can_change_qty, FALSE), COALESCE(s.can_negative_qty, FALSE),
		       COALESCE(s.can_price_change, FALSE), COALESCE(s.can_discount, FALSE),
		       COALESCE(s.can_open_session, FALSE), COALESCE(s.can_close_session, FALSE),
		       COALESCE(s.can_payment_modification, FALSE), COALESCE(s.can_refund, FALSE),
		       COALESCE(s.can_negative_stock, FALSE)
		FROM users u
		LEFT JOIN users_pos_security s ON s.tenant_id = u.tenant_id AND s.user_id = u.id
		ORDER BY u.display_name
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load users")
		return
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		var level *string
		var maxDiscount *int
		var custom, do, dl, cq, nq, pc, d, os, cs, pm, r, ns bool
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.AccountType, &u.IsActive,
			&level, &maxDiscount, &custom, &do, &dl, &cq, &nq, &pc, &d, &os, &cs, &pm, &r, &ns); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load users")
			return
		}
		var row *access.Row
		if level != nil {
			row = access.ScanRow(*level, maxDiscount, custom, do, dl, cq, nq, pc, d, os, cs, pm, r, ns)
		}
		u.Permissions = access.Resolve(u.Role, row)
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load users")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load users")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": users,
		"meta": gin.H{
			"request_id": c.GetString("request_id"),
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
	})
}

func (h *Handler) createUser(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	var request createUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid user request")
		return
	}
	if strings.TrimSpace(request.Email) == "" || strings.TrimSpace(request.DisplayName) == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "email and display_name are required")
		return
	}
	if !canCreateRole(claims.Role, request.Role) {
		writeError(c, http.StatusForbidden, "forbidden", "insufficient role to create this user")
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

	accountType := request.AccountType
	if accountType == "" {
		accountType = "standard"
	}
	passwordHash, err := security.HashPassword(request.Password, 10)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to hash password")
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

	// D2 plan limits: max_users = 0 means unlimited; otherwise refuse to grow
	// past the cap (checked inside the tx so the RLS tenant context is active).
	var maxUsers int
	if err := tx.QueryRow(ctx, `SELECT max_users FROM tenants WHERE id = $1::uuid`, tenantID.String()).Scan(&maxUsers); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenant plan")
		return
	}
	if maxUsers > 0 {
		var existing int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&existing); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to count users")
			return
		}
		if existing >= maxUsers {
			_ = tx.Rollback(ctx)
			writeError(c, http.StatusConflict, "plan_limit_exceeded", "tenant user limit reached")
			return
		}
	}

	var user User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (tenant_id, email, password_hash, display_name, role, account_type)
		VALUES ($1::uuid, $2, $3, $4, $5, $6)
		RETURNING id, email, display_name, role, account_type, is_active`,
		tenantID.String(), strings.TrimSpace(strings.ToLower(request.Email)),
		passwordHash, strings.TrimSpace(request.DisplayName), request.Role, accountType).Scan(
		&user.ID, &user.Email, &user.DisplayName, &user.Role, &user.AccountType, &user.IsActive)
	if err != nil {
		writeError(c, http.StatusConflict, "user_conflict", "email is already in use")
		return
	}
	level := access.DefaultLevelForRole(user.Role)
	if _, err = tx.Exec(ctx, `
		INSERT INTO users_pos_security (tenant_id, user_id, access_level)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO NOTHING`,
		tenantID.String(), user.ID, level); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to provision user security")
		return
	}
	user.Permissions = access.Resolve(user.Role, &access.Row{AccessLevel: level})
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create user")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": user, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

type updateSecurityRequest struct {
	AccessLevel            string `json:"access_level" binding:"required"`
	MaxDiscountPct         *int   `json:"max_discount_pct"`
	UseCustomPermissions   bool   `json:"use_custom_permissions"`
	CanDeleteOrder         bool   `json:"can_delete_order"`
	CanDeleteLine          bool   `json:"can_delete_line"`
	CanChangeQty           bool   `json:"can_change_qty"`
	CanNegativeQty         bool   `json:"can_negative_qty"`
	CanPriceChange         bool   `json:"can_price_change"`
	CanDiscount            bool   `json:"can_discount"`
	CanOpenSession         bool   `json:"can_open_session"`
	CanCloseSession        bool   `json:"can_close_session"`
	CanPaymentModification bool   `json:"can_payment_modification"`
	CanRefund              bool   `json:"can_refund"`
	CanNegativeStock       bool   `json:"can_negative_stock"`
}

// updateSecurity upserts a user's POS security profile (ma_pos_base parity):
// 5-tier access level, optional per-user discount cap and granular operation
// overrides. Owner/saas_admin may edit anyone; managers may only edit
// cashiers (mirrors canCreateRole). Returns the resolved effective permissions.
func (h *Handler) updateSecurity(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if claims.Role != "owner" && claims.Role != "manager" && claims.Role != "saas_admin" {
		writeError(c, http.StatusForbidden, "permission_denied", "owner or manager role is required")
		return
	}
	var request updateSecurityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid security request")
		return
	}
	level, err := access.ParseLevel(request.AccessLevel)
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	if request.MaxDiscountPct != nil && (*request.MaxDiscountPct < 0 || *request.MaxDiscountPct > 100) {
		writeError(c, http.StatusBadRequest, "validation_error", "max_discount_pct must be between 0 and 100")
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
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "user id is invalid")
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
	var targetRole string
	if err = tx.QueryRow(ctx, `SELECT role FROM users WHERE id = $1 AND tenant_id = $2`,
		targetID.String(), tenantID.String()).Scan(&targetRole); err != nil {
		writeError(c, http.StatusNotFound, "user_not_found", "user not found")
		return
	}
	if claims.Role == "manager" && targetRole != "cashier" {
		writeError(c, http.StatusForbidden, "permission_denied", "managers may only configure cashiers")
		return
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO users_pos_security
			(tenant_id, user_id, access_level, max_discount_pct, use_custom_permissions,
			 can_delete_order, can_delete_line, can_change_qty, can_negative_qty,
			 can_price_change, can_discount, can_open_session, can_close_session,
			 can_payment_modification, can_refund, can_negative_stock, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, now())
		ON CONFLICT (user_id) DO UPDATE SET
			access_level = EXCLUDED.access_level,
			max_discount_pct = EXCLUDED.max_discount_pct,
			use_custom_permissions = EXCLUDED.use_custom_permissions,
			can_delete_order = EXCLUDED.can_delete_order,
			can_delete_line = EXCLUDED.can_delete_line,
			can_change_qty = EXCLUDED.can_change_qty,
			can_negative_qty = EXCLUDED.can_negative_qty,
			can_price_change = EXCLUDED.can_price_change,
			can_discount = EXCLUDED.can_discount,
			can_open_session = EXCLUDED.can_open_session,
			can_close_session = EXCLUDED.can_close_session,
			can_payment_modification = EXCLUDED.can_payment_modification,
			can_refund = EXCLUDED.can_refund,
			can_negative_stock = EXCLUDED.can_negative_stock,
			updated_at = now()`,
		tenantID.String(), targetID.String(), level, request.MaxDiscountPct, request.UseCustomPermissions,
		request.CanDeleteOrder, request.CanDeleteLine, request.CanChangeQty, request.CanNegativeQty,
		request.CanPriceChange, request.CanDiscount, request.CanOpenSession, request.CanCloseSession,
		request.CanPaymentModification, request.CanRefund, request.CanNegativeStock); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save user security")
		return
	}
	row := access.ScanRow(level, request.MaxDiscountPct, request.UseCustomPermissions,
		request.CanDeleteOrder, request.CanDeleteLine, request.CanChangeQty, request.CanNegativeQty,
		request.CanPriceChange, request.CanDiscount, request.CanOpenSession, request.CanCloseSession,
		request.CanPaymentModification, request.CanRefund, request.CanNegativeStock)
	permissions := access.Resolve(targetRole, row)
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save user security")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"user_id": targetID.String(), "permissions": permissions}, "meta": gin.H{"request_id": c.GetString("request_id")}})
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

// canCreateRole reports whether a caller with the given role may provision a
// new user with targetRole. Hierarchy: tenant owner > manager > cashier;
// saas_admin (platform staff) may provision any tenant role.
func canCreateRole(callerRole, targetRole string) bool {
	switch callerRole {
	case "owner", "saas_admin":
		return true
	case "manager":
		return targetRole == "cashier"
	default:
		return false
	}
}
