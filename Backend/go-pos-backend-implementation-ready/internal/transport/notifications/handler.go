// Package notifications powers the tenant-level inbox for manager staff plus
// RFC 8030 web-push delivery to subscribed browser endpoints. Business events
// (low stock, OCR window limit reached, refund applied, manager user created)
// call emitNotification on a handler *after* the authoritative write commits —
// the notification is advisory and never blocks or rolls back the business
// transaction. The inbox is read through GET /v1/notifications, a badge count
// via GET /v1/notifications/count, and rows are marked read with
// PATCH /v1/notifications/:id/read. Rows are tenant-scoped through RLS FORCE.
package notifications

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pushTTLSeconds = 60 * 60 * 24 // 24h offline window

// Severity levels carried on a notification.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// EventType values emitted by the business modules.
const (
	TypeLowStock   = "low_stock"
	TypeOCRLimit   = "ocr_limit"
	TypeRefund     = "refund"
	TypeUserCreate = "user_created"
	TypeInfo       = "info"
)

// Notification is one inbox row surfaced to the client.
type Notification struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Key       string         `json:"key"`
	Severity  string         `json:"severity"`
	Title     string         `json:"title"`
	Body      string         `json:"body"`
	Payload   map[string]any `json:"payload"`
	Read      bool           `json:"read"`
	CreatedAt string         `json:"created_at"`
}

// Emit is the outbound contract business modules call after their writes
// commit. They must not hold a pool transaction when invoking it (it opens its
// own); a nil emitter disables notifications for that module.
type Emit func(ctx context.Context, tenantID, userID, ntype, key, severity, title, body string, payload map[string]any)

// Handler is the notifications HTTP surface + emitter implementation.
type Handler struct {
	pool  *pgxpool.Pool
	token security.TokenManager
	vapid WebPushConfig
}

// WebPushConfig is the VAPID identity used for manager browser notifications.
type WebPushConfig struct {
	PublicKey  string
	PrivateKey string
	Subject    string
}

// NewHandler wires the module. pool may be nil for the OpenAPI generator.
func NewHandler(pool *pgxpool.Pool, token security.TokenManager, vapid WebPushConfig) *Handler {
	return &Handler{pool: pool, token: token, vapid: vapid}
}

// Register attaches the /v1/notifications routes.
func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/notifications", h.list)
	router.GET("/notifications/count", h.count)
	router.PATCH("/notifications/:id/read", h.markRead)
	router.POST("/notifications/push-subscribe", h.subscribe)
}

// Emitter returns the Emit function for business modules. It is safe for
// modules to call with a nil pool (also a nil handler) — the call is dropped.
func (h *Handler) Emitter() Emit {
	if h == nil || h.pool == nil {
		return func(context.Context, string, string, string, string, string, string,
			string, map[string]any) {
		}
	}
	return h.emit
}

// emit writes one inbox row in its own transaction. Failure is logged and
// swallowed: the caller's business write already committed. When a manager
// browser subscription exists and VAPID is configured, the notification is
// forwarded as an RFC 8030 push on a background goroutine so a slow push
// endpoint never blocks the inbox write.
func (h *Handler) emit(ctx context.Context, tenantID, userID, ntype, key, severity,
	title, body string, payload map[string]any) {
	tenant, err := uuid.Parse(tenantID)
	if err != nil {
		return
	}
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		slog.Warn("notifications: begin tx", "err", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenant.String()); err != nil {
		slog.Warn("notifications: set tenant context", "err", err)
		return
	}
	payloadJSON, _ := json.Marshal(payload)
	if payload == nil {
		payloadJSON = []byte("{}")
	}
	uid, _ := uuid.Parse(userID)
	var id uuid.UUID
	if err := tx.QueryRow(ctx, `
		INSERT INTO notifications (tenant_id, user_id, type, key, severity, title, body, payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		tenant, nullableUUID(uid), ntype, key, severity, title, body, payloadJSON).Scan(&id); err != nil {
		slog.Warn("notifications: insert", "err", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		slog.Warn("notifications: commit", "err", err)
		return
	}

	// Best-effort web push to manager browser endpoints.
	if h.vapid.PrivateKey != "" {
		go func() {
			background, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			defer cancel()
			h.push(background, tenant, payload, title, body, ntype)
		}()
	}
}

func (h *Handler) push(ctx context.Context, tenant uuid.UUID, payload map[string]any, title, body, ntype string) {
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenant.String()); err != nil {
		return
	}
	rows, err := tx.Query(ctx, `SELECT sub FROM notification_push_sub`)
	if err != nil {
		slog.Warn("notifications: load push subs", "err", err)
		return
	}
	type sub struct {
		s *webpush.Subscription
	}
	subs := make([]sub, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		if s, ok := decodeSub(raw); ok {
			subs = append(subs, sub{s: s})
		}
	}
	rows.Close()
	if len(subs) == 0 {
		_ = tx.Commit(ctx)
		return
	}

	msg, _ := json.Marshal(map[string]any{
		"title": title,
		"body":  body,
		"tag":   ntype,
		"url":   "/admin/notifications",
	})
	options := &webpush.Options{
		Subscriber:      h.vapid.Subject,
		VAPIDPublicKey:  h.vapid.PublicKey,
		VAPIDPrivateKey: h.vapid.PrivateKey,
		TTL:             pushTTLSeconds,
	}
	for _, s := range subs {
		res, err := webpush.SendNotificationWithContext(ctx, msg, s.s, options)
		if err != nil || res == nil {
			slog.Debug("notifications: push failed", "err", err)
			continue
		}
		_ = res.Body.Close()
		if res.StatusCode == 404 || res.StatusCode == 410 {
			_, _ = tx.Exec(ctx, `DELETE FROM notification_push_sub WHERE sub = $1::jsonb`, string(mustJSON(s.s)))
		}
	}
	if err := tx.Commit(ctx); err != nil {
		slog.Warn("notifications: push commit", "err", err)
	}
}

func (h *Handler) list(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tenant, ok := h.tenantID(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenant.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	rows, err := tx.Query(ctx, `
		SELECT id::text, type, key, severity, title, body, payload, read_at IS NOT NULL, created_at::text
		FROM notifications
		ORDER BY created_at DESC
		LIMIT 100`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load notifications")
		return
	}
	items := make([]Notification, 0, 32)
	for rows.Next() {
		var n Notification
		var raw []byte
		if err := rows.Scan(&n.ID, &n.Type, &n.Key, &n.Severity, &n.Title, &n.Body, &raw, &n.Read, &n.CreatedAt); err != nil {
			rows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load notifications")
			return
		}
		_ = json.Unmarshal(raw, &n.Payload)
		items = append(items, n)
	}
	rows.Close()
	var unread int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE read_at IS NULL`).Scan(&unread); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count notifications")
		return
	}
	_ = tx.Commit(ctx)
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"items": items, "unread": unread},
	})
}

func (h *Handler) count(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tenant, ok := h.tenantID(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenant.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var unread int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE read_at IS NULL`).Scan(&unread); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count notifications")
		return
	}
	_ = tx.Commit(ctx)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"unread": unread}})
}

func (h *Handler) markRead(c *gin.Context) {
	if !h.authorize(c) {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_id", "notification id is invalid")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tenant, ok := h.tenantID(c)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenant.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	tag, err := tx.Exec(ctx,
		`UPDATE notifications SET read_at = NOW() WHERE tenant_id = $1 AND id = $2 AND read_at IS NULL`,
		tenant, id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update notification")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "not_found", "notification was not found or is already read")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update notification")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": id.String(), "read": true}})
}

// subWire is the PushSubscription shape sent by the browser.
type subWire struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}

func (h *Handler) subscribe(c *gin.Context) {
	claims, ok := h.auth(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "notifications", "read") {
		writeError(c, http.StatusForbidden, "permission_denied", "manager required")
		return
	}
	var wire subWire
	if err := c.ShouldBindJSON(&wire); err != nil || wire.Endpoint == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "a PushSubscription endpoint is required")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	raw, _ := json.Marshal(wire)
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	uid, _ := uuid.Parse(claims.UserID)
	_, err = tx.Exec(ctx, `
		INSERT INTO notification_push_sub (tenant_id, user_id, endpoint, sub)
		VALUES ($1, $2, $3, $4::jsonb)
		ON CONFLICT (user_id, endpoint)
		DO UPDATE SET sub = EXCLUDED.sub, created_at = NOW()`,
		claims.TenantID, uid, wire.Endpoint, string(raw))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save push subscription")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save push subscription")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"subscribed": true}})
}

// ---- shared helpers -------------------------------------------------------

func (h *Handler) authorize(c *gin.Context) bool {
	claims, ok := h.auth(c)
	if !ok {
		return false
	}
	if !httptransport.HasPermission(claims.Role, "notifications", "read") {
		writeError(c, http.StatusForbidden, "permission_denied", "manager required")
		return false
	}
	return true
}

func (h *Handler) auth(c *gin.Context) (security.Claims, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is required")
		return security.Claims{}, false
	}
	claims, err := h.token.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), security.AccessToken)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return security.Claims{}, false
	}
	return claims, true
}

func (h *Handler) tenantID(c *gin.Context) (uuid.UUID, bool) {
	claims, ok := h.auth(c)
	if !ok {
		return uuid.Nil, false
	}
	id, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return uuid.Nil, false
	}
	return id, true
}

func nullableUUID(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func decodeSub(raw []byte) (*webpush.Subscription, bool) {
	var wire subWire
	if json.Unmarshal(raw, &wire) != nil || wire.Endpoint == "" {
		return nil, false
	}
	return &webpush.Subscription{
		Endpoint: wire.Endpoint,
		Keys: webpush.Keys{
			P256dh: wire.Keys.P256dh,
			Auth:   wire.Keys.Auth,
		},
	}, true
}

func mustJSON(s *webpush.Subscription) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"code": code, "message": message, "request_id": c.GetString("request_id"),
		},
	})
}
