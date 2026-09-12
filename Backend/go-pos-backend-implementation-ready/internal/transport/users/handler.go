package users

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
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
}

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	AccountType string `json:"account_type"`
	IsActive    bool   `json:"is_active"`
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
	defer tx.Rollback(ctx)
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
		SELECT id, email, display_name, role, account_type, is_active
		FROM users ORDER BY display_name
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load users")
		return
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.Role, &u.AccountType, &u.IsActive); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load users")
			return
		}
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
	defer tx.Rollback(ctx)
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
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to create user")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": user, "meta": gin.H{"request_id": c.GetString("request_id")}})
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
