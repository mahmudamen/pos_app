// Package telemetry ingests client-observability events (crashes, errors,
// screen/action telemetry) reported by the Flutter app. Rows are tenant-scoped
// by the same app.current_tenant RLS mechanism as every other tenant table.
package telemetry

import (
	"encoding/json"
	"net/http"
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
	router.POST("/client/events", h.postEvent)
}

type eventRequest struct {
	Event      string                 `json:"event" binding:"required"`
	AppVersion string                 `json:"app_version"`
	Screen     string                 `json:"screen"`
	StackTrace string                 `json:"stack_trace"`
	Payload    map[string]interface{} `json:"payload"`
}

func (h *Handler) postEvent(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	var request eventRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid event request")
		return
	}
	request.Event = strings.TrimSpace(request.Event)
	if request.Event == "" || len(request.Event) > 64 {
		writeError(c, http.StatusBadRequest, "validation_error", "event must be 1-64 characters")
		return
	}
	if len(request.Screen) > 64 || len(request.AppVersion) > 32 || len(request.StackTrace) > 10000 {
		writeError(c, http.StatusBadRequest, "validation_error", "event fields exceed size limits")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	payload := "{}"
	if request.Payload != nil {
		raw, err := json.Marshal(request.Payload)
		if err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "invalid event payload")
			return
		}
		payload = string(raw)
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
	var id string
	if err = tx.QueryRow(ctx, `
		INSERT INTO client_events (tenant_id, user_id, device_id, event, app_version, screen, payload, stack_trace)
		VALUES ($1::uuid, NULLIF($2, '')::uuid, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7::jsonb, NULLIF($8, ''))
		RETURNING id::text`,
		tenantID.String(), claims.UserID, claims.DeviceID, request.Event, request.AppVersion, request.Screen,
		payload, request.StackTrace).Scan(&id); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to store event")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to store event")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{"id": id, "event": request.Event},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
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

func hasBearer(header string) bool {
	return len(header) > 7 && strings.EqualFold(header[:7], "Bearer ")
}

func trimBearer(header string) string {
	return strings.TrimSpace(header[7:])
}
