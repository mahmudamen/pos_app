package billing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/example/pos-api/internal/email"
	"github.com/example/pos-api/internal/notify"
)

// invoiceEmailData is everything the invoice template renders.
func invoiceEmailData(tenantName, invoiceID string, amountMinor int64, currency, status string, dueAt *time.Time) map[string]string {
	due := ""
	if dueAt != nil {
		due = dueAt.Format("2006-01-02")
	}
	return map[string]string{
		"store":       tenantName,
		"number":      invoiceID,
		"amount":      formatMinor(amountMinor, currency),
		"status":      status,
		"due_date":    due,
		"invoice_url": "https://api.xamltech.com/admin/",
	}
}

// mailInvoice sends the "invoice issued" notice. Best-effort: a mail failure
// must never fail the billing action that produced the invoice.
func (h *Handler) mailInvoice(ctx context.Context, tenantID, tenantName, invoiceID string, amountMinor int64, currency string, dueAt *time.Time) {
	if h.mail == nil {
		return
	}
	to := notify.ContactEmail(ctx, h.pool, tenantID)
	if to == "" {
		slog.Warn("billing: no contact email for tenant", "tenant_id", tenantID)
		return
	}
	data := invoiceEmailData(tenantName, invoiceID, amountMinor, currency, "open", dueAt)
	if err := h.mail.SendTemplate(ctx, email.TemplateInvoice, to, data); err != nil {
		slog.Warn("billing: invoice email failed", "tenant_id", tenantID, "invoice_id", invoiceID, "err", err)
	}
}

// mailInvoiceStatus updates the merchant when an invoice is paid/voided/refunded.
func (h *Handler) mailInvoiceStatus(ctx context.Context, tenantID, tenantName, invoiceID string, amountMinor int64, currency, status string) {
	if h.mail == nil {
		return
	}
	to := notify.ContactEmail(ctx, h.pool, tenantID)
	if to == "" {
		return
	}
	title, body := invoiceStatusCopy(status, formatMinor(amountMinor, currency), invoiceID)
	data := map[string]string{"name": tenantName, "title": title, "body": body}
	if err := h.mail.SendTemplate(ctx, email.TemplateNotification, to, data); err != nil {
		slog.Warn("billing: invoice status email failed", "tenant_id", tenantID, "invoice_id", invoiceID, "err", err)
	}
}

// mailSubscriptionEnded tells a merchant their subscription was cancelled.
// Best-effort like the invoice mails; a delivery failure never rolls back the
// status change that triggered it.
func (h *Handler) mailSubscriptionEnded(ctx context.Context, tenantID, subscriptionID, tenantName, reason string) {
	if h.mail == nil || tenantID == "" {
		return
	}
	to := notify.ContactEmail(ctx, h.pool, tenantID)
	if to == "" {
		slog.Warn("billing: no contact email for subscription end", "tenant_id", tenantID)
		return
	}
	name := tenantName
	if name == "" {
		name = "there"
	}
	if err := h.mail.SendTemplate(ctx, email.TemplateSubscriptionEnd, to, map[string]string{
		"name": name, "reason": reason,
	}); err != nil {
		slog.Warn("billing: subscription-ended email failed", "tenant_id", tenantID, "subscription_id", subscriptionID, "err", err)
	}
}

func invoiceStatusCopy(status, amount, invoiceID string) (string, string) {
	switch status {
	case "paid":
		return "تم استلام الدفعة / Payment received",
			"تم سداد الفاتورة " + invoiceID + " بمبلغ " + amount + ".\nInvoice " + invoiceID + " for " + amount + " has been paid."
	case "void":
		return "تم إلغاء الفاتورة / Invoice voided",
			"تم إلغاء الفاتورة " + invoiceID + ".\nInvoice " + invoiceID + " has been voided."
	case "refunded":
		return "تم رد المبلغ / Invoice refunded",
			"تم رد مبلغ الفاتورة " + invoiceID + " (" + amount + ").\nInvoice " + invoiceID + " (" + amount + ") has been refunded."
	default:
		return "تحديث الفاتورة / Invoice update",
			"تم تحديث حالة الفاتورة " + invoiceID + " إلى " + status + ".\nInvoice " + invoiceID + " is now " + status + "."
	}
}

// formatMinor renders integer minor units as a currency-prefixed decimal,
// e.g. 12345 + "EGP" -> "EGP 123.45".
func formatMinor(amountMinor int64, currency string) string {
	sign := ""
	if amountMinor < 0 {
		sign = "-"
		amountMinor = -amountMinor
	}
	if currency == "" {
		currency = defaultCurrency
	}
	return fmt.Sprintf("%s%s %d.%02d", sign, currency, amountMinor/100, amountMinor%100)
}
