package settings

import (
	"net/http"

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
	router.GET("/settings", h.getSettings)
	router.PUT("/settings", h.updateSettings)
}

type updateSettingsRequest struct {
	Settings map[string]string `json:"settings" binding:"required"`
}

func (h *Handler) getSettings(c *gin.Context) {
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

	settings := make(map[string]string, len(settingDefinitions))
	for key := range settingDefinitions {
		settings[key] = defaultSettingValue(key)
	}

	rows, err := tx.Query(ctx, `SELECT key, value FROM tenant_settings WHERE key = ANY($1)`, settingKeys())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load settings")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load settings")
			return
		}
		settings[key] = value
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load settings")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load settings")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": settings, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) updateSettings(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !canManageSettings(claims.Role) {
		writeError(c, http.StatusForbidden, "forbidden", "owner or manager role is required")
		return
	}
	var request updateSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid settings request")
		return
	}
	if len(request.Settings) == 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "settings must not be empty")
		return
	}
	for key, value := range request.Settings {
		if err := validateSetting(key, value); err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "invalid setting: "+key+": "+err.Error())
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
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	for key, value := range request.Settings {
		if _, err = tx.Exec(ctx, `
			INSERT INTO tenant_settings (tenant_id, key, value, updated_by) VALUES ($1, $2, $3, $4)
			ON CONFLICT (tenant_id, key)
			DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
			tenantID.String(), key, value, claims.UserID); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to save settings")
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save settings")
		return
	}

	settings := make(map[string]string, len(settingDefinitions))
	for key := range settingDefinitions {
		settings[key] = defaultSettingValue(key)
	}
	for key, value := range request.Settings {
		settings[key] = value
	}
	c.JSON(http.StatusOK, gin.H{"data": settings, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) authenticate(c *gin.Context) (security.Claims, bool) {
	header := c.GetHeader("Authorization")
	if !hasBearer(header) {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is required")
		return security.Claims{}, false
	}
	claims, err := h.tokens.Parse(trimBearer(header), security.AccessToken)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return security.Claims{}, false
	}
	return claims, true
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
