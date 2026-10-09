package platform

import (
	"context"
	"log/slog"

	"github.com/example/pos-api/internal/email"
	"github.com/example/pos-api/internal/notify"
)

// mailAccountNotice sends an admin-triggered lifecycle notice (suspend,
// activate, stop) to the tenant's contact address. Best-effort: the state
// change is already committed, so a mail failure only logs.
func (h *Handler) mailAccountNotice(ctx context.Context, tenantID, title, body string) {
	if h.mail == nil || tenantID == "" {
		return
	}
	to := notify.ContactEmail(ctx, h.pool, tenantID)
	if to == "" {
		slog.Warn("platform: no contact email for tenant notice", "tenant_id", tenantID)
		return
	}
	name := "there"
	if n, err := h.tenantName(ctx, tenantID); err == nil && n != "" {
		name = n
	}
	if err := h.mail.SendTemplate(ctx, email.TemplateAccountNotice, to, map[string]string{
		"name": name, "title": title, "body": body,
	}); err != nil {
		slog.Warn("platform: account notice email failed", "tenant_id", tenantID, "err", err)
	}
}

func (h *Handler) tenantName(ctx context.Context, tenantID string) (string, error) {
	var name string
	err := h.pool.QueryRow(ctx, `SELECT name FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&name)
	return name, err
}

// noticeBody appends an optional admin reason to the localized body.
func noticeBody(bodyAr, bodyEn, reason string) string {
	out := bodyAr + "\n" + bodyEn
	if reason != "" {
		out += "\n\nالسبب / Reason: " + reason
	}
	return out
}
