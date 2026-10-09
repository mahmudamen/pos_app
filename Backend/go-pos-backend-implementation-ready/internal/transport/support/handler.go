// Package support exposes the platform support inbox to the web admin console:
// the cross-tenant message queue fed by the IMAP poller, plus reply and compose
// actions. Every route sits behind the exact saas_admin role and reads/writes
// the non-RLS `support_messages` platform table.
package support

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/example/pos-api/internal/email"
	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/infrastructure/security"
	core "github.com/example/pos-api/internal/support"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultMailbox is the mailbox label used when the poller is not configured.
const DefaultMailbox = "support"

// Handler serves the support-inbox API.
type Handler struct {
	pool    *pgxpool.Pool
	tokens  security.TokenManager
	mail    *email.Service
	mailbox string
}

// NewHandler wires the inbox API. mailbox labels outbound rows so a reply
// composes into the same conversation as an inbound message from that mailbox.
func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager, mailbox string, from string, mailer identity.Mailer) *Handler {
	if strings.TrimSpace(mailbox) == "" {
		mailbox = DefaultMailbox
	}
	h := &Handler{pool: pool, tokens: tokens, mailbox: mailbox}
	if mailer != nil {
		h.mail = email.NewService(mailer, from)
	}
	return h
}

// SetMailer overrides the outbound transport (tests capture sends here).
func (h *Handler) SetMailer(m identity.Mailer, from string) {
	if m != nil {
		h.mail = email.NewService(m, from)
	}
}

// Register mounts the inbox under /support on the supplied group. The caller
// registers that group behind RequireSaasAdmin (the saas group).
func (h *Handler) Register(router *gin.RouterGroup) {
	admin := router.Group("/support", httptransport.RequireSaasAdmin(h.tokens))
	admin.GET("/messages", h.listMessages)
	admin.GET("/summary", h.summary)
	admin.POST("/messages", h.sendMessage)
	admin.GET("/messages/:id", h.getMessage)
	admin.POST("/messages/:id/read", h.markRead)
	admin.POST("/messages/:id/archive", h.archiveMessage)
	admin.POST("/messages/:id/unarchive", h.unarchiveMessage)
	admin.POST("/messages/:id/reply", h.replyMessage)
}

type messageRow struct {
	ID         string     `json:"id"`
	ThreadID   string     `json:"thread_id"`
	Direction  string     `json:"direction"`
	Mailbox    string     `json:"mailbox"`
	TenantID   *string    `json:"tenant_id"`
	TenantName *string    `json:"tenant_name"`
	MessageID  *string    `json:"message_id"`
	InReplyTo  *string    `json:"in_reply_to"`
	FromEmail  string     `json:"from_email"`
	FromName   string     `json:"from_name"`
	ToEmail    string     `json:"to_email"`
	Subject    string     `json:"subject"`
	BodyText   string     `json:"body_text"`
	BodyHTML   string     `json:"body_html"`
	Status     string     `json:"status"`
	Error      *string    `json:"error"`
	ReceivedAt *time.Time `json:"received_at"`
	SentAt     *time.Time `json:"sent_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

const messageColumns = `
	m.id::text, m.thread_id::text, m.direction, m.mailbox,
	m.tenant_id::text, t.name, m.message_id, m.in_reply_to,
	m.from_email, m.from_name, m.to_email, m.subject,
	m.body_text, m.body_html, m.status, m.error,
	m.received_at, m.sent_at, m.created_at`

func scanMessage(row pgx.Row) (messageRow, error) {
	var m messageRow
	err := row.Scan(&m.ID, &m.ThreadID, &m.Direction, &m.Mailbox,
		&m.TenantID, &m.TenantName, &m.MessageID, &m.InReplyTo,
		&m.FromEmail, &m.FromName, &m.ToEmail, &m.Subject,
		&m.BodyText, &m.BodyHTML, &m.Status, &m.Error,
		&m.ReceivedAt, &m.SentAt, &m.CreatedAt)
	return m, err
}

func (h *Handler) listMessages(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	tenantID := strings.TrimSpace(c.Query("tenant_id"))
	direction := strings.TrimSpace(c.Query("direction"))
	threadID := strings.TrimSpace(c.Query("thread_id"))
	q := strings.TrimSpace(c.Query("q"))
	page, limit := pagination(c)

	ctx := c.Request.Context()
	var total int
	if err := h.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM support_messages m
		WHERE ($1 = '' OR m.status = $1)
		  AND (NULLIF($2, '') IS NULL OR m.tenant_id = NULLIF($2, '')::uuid)
		  AND ($3 = '' OR m.direction = $3)
		  AND (NULLIF($4, '') IS NULL OR m.thread_id = NULLIF($4, '')::uuid)
		  AND ($5 = '' OR m.subject ILIKE '%'||$5||'%' OR m.from_email ILIKE '%'||$5||'%' OR m.body_text ILIKE '%'||$5||'%')`,
		status, tenantID, direction, threadID, q).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count messages")
		return
	}

	rows, err := h.pool.Query(ctx, `
		SELECT`+messageColumns+`
		FROM support_messages m
		LEFT JOIN tenants t ON t.id = m.tenant_id
		WHERE ($1 = '' OR m.status = $1)
		  AND (NULLIF($2, '') IS NULL OR m.tenant_id = NULLIF($2, '')::uuid)
		  AND ($3 = '' OR m.direction = $3)
		  AND (NULLIF($4, '') IS NULL OR m.thread_id = NULLIF($4, '')::uuid)
		  AND ($5 = '' OR m.subject ILIKE '%'||$5||'%' OR m.from_email ILIKE '%'||$5||'%' OR m.body_text ILIKE '%'||$5||'%')
		ORDER BY m.created_at DESC
		LIMIT $6 OFFSET $7`,
		status, tenantID, direction, threadID, q, limit, (page-1)*limit)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load messages")
		return
	}
	defer rows.Close()

	list := make([]messageRow, 0)
	for rows.Next() {
		m, err := scanMessage(rows)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load messages")
			return
		}
		list = append(list, m)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load messages")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": list,
		"meta": gin.H{"page": page, "limit": limit, "total": total, "request_id": c.GetString("request_id")},
	})
}

func (h *Handler) summary(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	var unread, read, archived int
	if err := h.pool.QueryRow(c.Request.Context(), `
		SELECT
			COUNT(*) FILTER (WHERE status = 'unread'),
			COUNT(*) FILTER (WHERE status = 'read'),
			COUNT(*) FILTER (WHERE status = 'archived')
		FROM support_messages`).Scan(&unread, &read, &archived); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to summarize inbox")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"unread": unread, "read": read, "archived": archived, "total": unread + read + archived},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) getMessage(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	row := h.pool.QueryRow(c.Request.Context(), `
		SELECT`+messageColumns+`
		FROM support_messages m
		LEFT JOIN tenants t ON t.id = m.tenant_id
		WHERE m.id = $1::uuid`, c.Param("id"))
	m, err := scanMessage(row)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "message_not_found", "message not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load message")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": m, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) markRead(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(), `
		UPDATE support_messages SET status = 'read', updated_at = now()
		WHERE id = $1::uuid AND status = 'unread'`, c.Param("id"))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update message")
		return
	}
	if tag.RowsAffected() == 0 {
		// Either the message does not exist, or it was already read/archived.
		var exists bool
		if err := h.pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS (SELECT 1 FROM support_messages WHERE id = $1::uuid)`, c.Param("id")).Scan(&exists); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to update message")
			return
		}
		if !exists {
			writeError(c, http.StatusNotFound, "message_not_found", "message not found")
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": c.Param("id"), "status": core.StatusRead}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) archiveMessage(c *gin.Context) {
	h.setMessageStatus(c, core.StatusArchived)
}

// unarchiveMessage restores an archived message to the read queue.
func (h *Handler) unarchiveMessage(c *gin.Context) {
	h.setMessageStatus(c, core.StatusRead)
}

// setMessageStatus moves a message to a target status unless it is already
// there; an unknown id is a 404, a missing id is a no-op success (the queue is
// idempotent the same way markRead is).
func (h *Handler) setMessageStatus(c *gin.Context, want string) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tag, err := h.pool.Exec(c.Request.Context(), `
		UPDATE support_messages SET status = $2, updated_at = now()
		WHERE id = $1::uuid AND status <> $2`, c.Param("id"), want)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update message")
		return
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := h.pool.QueryRow(c.Request.Context(),
			`SELECT EXISTS (SELECT 1 FROM support_messages WHERE id = $1::uuid)`, c.Param("id")).Scan(&exists); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to update message")
			return
		}
		if !exists {
			writeError(c, http.StatusNotFound, "message_not_found", "message not found")
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": c.Param("id"), "status": want}, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

type replyRequest struct {
	BodyText string `json:"body_text" binding:"required"`
	BodyHTML string `json:"body_html"`
}

func (h *Handler) replyMessage(c *gin.Context) {
	var req replyRequest
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.BodyText) == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "body_text is required")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	original, err := scanMessage(h.pool.QueryRow(ctx, `
		SELECT`+messageColumns+`
		FROM support_messages m
		LEFT JOIN tenants t ON t.id = m.tenant_id
		WHERE m.id = $1::uuid`, c.Param("id")))
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "message_not_found", "message not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load message")
		return
	}

	to := original.FromEmail
	if to == "" {
		writeError(c, http.StatusConflict, "no_recipient", "the message has no reply address")
		return
	}
	subject := replySubject(original.Subject)
	sent, err := h.send(c, sendInput{
		To: to, Subject: subject, BodyText: req.BodyText, BodyHTML: req.BodyHTML,
		InReplyTo: deref(original.MessageID), ThreadID: original.ThreadID, TenantID: original.TenantID,
	})
	if err != nil {
		writeSendError(c, err)
		return
	}
	// Replying implies the message was handled.
	if _, err := h.pool.Exec(ctx, `
		UPDATE support_messages SET status = 'read', updated_at = now()
		WHERE id = $1::uuid AND status = 'unread'`, original.ID); err != nil {
		slog.Warn("support: failed to mark replied message read", "id", original.ID, "err", err)
	}
	c.JSON(http.StatusCreated, gin.H{"data": sent, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

type sendRequest struct {
	To       string `json:"to" binding:"required,email"`
	Subject  string `json:"subject" binding:"required"`
	BodyText string `json:"body_text" binding:"required"`
	BodyHTML string `json:"body_html"`
	TenantID string `json:"tenant_id"`
}

func (h *Handler) sendMessage(c *gin.Context) {
	var req sendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "to, subject and body_text are required")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	var tenantID *string
	if strings.TrimSpace(req.TenantID) != "" {
		if _, err := uuid.Parse(req.TenantID); err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "invalid tenant_id")
			return
		}
		tenantID = &req.TenantID
	}
	sent, err := h.send(c, sendInput{
		To: req.To, Subject: req.Subject, BodyText: req.BodyText, BodyHTML: req.BodyHTML,
		ThreadID: core.NewThreadID(), TenantID: tenantID,
	})
	if err != nil {
		writeSendError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": sent, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

type sendInput struct {
	To        string
	Subject   string
	BodyText  string
	BodyHTML  string
	InReplyTo string
	ThreadID  string
	TenantID  *string
}

func (h *Handler) send(c *gin.Context, in sendInput) (messageRow, error) {
	if h.mail == nil {
		return messageRow{}, errNoMailer
	}
	ctx := c.Request.Context()
	messageID := "<support-" + uuid.NewString() + "@xamltech.com>"
	from := h.mail.From()
	if err := h.mail.Send(ctx, email.Message{
		To: in.To, Subject: in.Subject, Text: in.BodyText, HTML: in.BodyHTML,
	}); err != nil {
		return messageRow{}, err
	}
	if in.ThreadID == "" {
		in.ThreadID = core.NewThreadID()
	}
	var id string
	if err := h.pool.QueryRow(ctx, `
		INSERT INTO support_messages (
			thread_id, thread_key, direction, mailbox, tenant_id, message_id, in_reply_to,
			from_email, from_name, to_email, subject, body_text, body_html, status, sent_at
		) VALUES (
			$1::uuid, $2, 'outbound', $3, $4, $5, $6,
			$7, '', $8, $9, $10, $11, 'read', now()
		) RETURNING id::text`,
		in.ThreadID, core.ThreadKey(in.Subject), h.mailbox, in.TenantID, messageID, nullIfEmpty(in.InReplyTo),
		from, in.To, in.Subject, in.BodyText, in.BodyHTML).Scan(&id); err != nil {
		return messageRow{}, err
	}
	return scanMessage(h.pool.QueryRow(ctx, `
		SELECT`+messageColumns+`
		FROM support_messages m
		LEFT JOIN tenants t ON t.id = m.tenant_id
		WHERE m.id = $1::uuid`, id))
}

var errNoMailer = errors.New("support: outbound mail is not configured")

func writeSendError(c *gin.Context, err error) {
	if errors.Is(err, errNoMailer) {
		writeError(c, http.StatusServiceUnavailable, "email_unavailable", "outbound email is not configured")
		return
	}
	writeError(c, http.StatusBadGateway, "email_send_failed", "unable to send email")
}

// replySubject prefixes "Re:" without stacking it on an already-replied thread.
func replySubject(subject string) string {
	trimmed := strings.TrimSpace(subject)
	if trimmed == "" {
		return "Re:"
	}
	if strings.HasPrefix(strings.ToLower(trimmed), "re:") {
		return trimmed
	}
	return "Re: " + trimmed
}

func pagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if limit < 1 || limit > 100 {
		limit = 25
	}
	return page, limit
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nullIfEmpty(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
