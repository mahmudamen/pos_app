package email

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/identity"
)

// fakeSender records deliveries and can be told to fail.
type fakeSender struct {
	sent []identity.EmailMessage
	err  error
}

func (f *fakeSender) Send(_ context.Context, msg identity.EmailMessage) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, msg)
	return nil
}

func TestDefaultRegistryExposesEveryTemplate(t *testing.T) {
	r := DefaultRegistry()
	want := []string{
		"account_notice", "invoice", "notification", "password_reset",
		"subscription_ended", "verify_email", "welcome",
	}
	got := r.Names()
	if len(got) != len(want) {
		t.Fatalf("Names() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Names()[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
	for _, n := range want {
		if !r.Has(n) {
			t.Errorf("Has(%q) = false", n)
		}
	}
}

func TestRegistryRejectsUnknownTemplate(t *testing.T) {
	r := DefaultRegistry()
	if _, err := r.Render("nope", nil); err == nil {
		t.Fatal("expected error for unknown template")
	} else if !strings.Contains(err.Error(), "nope") {
		t.Fatalf("error should name the template: %v", err)
	}
	if r.Has("nope") {
		t.Error("Has must be false for unknown")
	}
}

func TestRegistryRegisterReplacesAndAdds(t *testing.T) {
	r := NewRegistry()
	r.Register("custom", func(d map[string]string) (Message, error) {
		return Message{Subject: d["s"]}, nil
	})
	msg, err := r.Render("custom", map[string]string{"s": "hi"})
	if err != nil || msg.Subject != "hi" {
		t.Fatalf("custom render = %+v, %v", msg, err)
	}
	// Replacing must not duplicate.
	r.Register("custom", func(map[string]string) (Message, error) {
		return Message{Subject: "replaced"}, nil
	})
	msg, _ = r.Render("custom", nil)
	if msg.Subject != "replaced" {
		t.Fatalf("replace failed: %+v", msg)
	}
	if names := r.Names(); len(names) != 1 || names[0] != "custom" {
		t.Fatalf("Names() = %v", names)
	}
}

func TestServiceSendFillsDefaultFrom(t *testing.T) {
	f := &fakeSender{}
	s := NewService(f, "custom@example.com")
	if err := s.Send(context.Background(), Message{To: "a@b.com", Subject: "x"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if f.sent[0].From != "custom@example.com" {
		t.Fatalf("From = %q", f.sent[0].From)
	}
}

func TestServiceSendUsesTemplateFrom(t *testing.T) {
	f := &fakeSender{}
	s := NewService(f, "fallback@example.com")
	if err := s.SendTemplate(context.Background(), TemplateWelcome, "a@b.com", nil); err != nil {
		t.Fatalf("SendTemplate: %v", err)
	}
	if f.sent[0].From != SupportEmail {
		t.Fatalf("From = %q, want template's %q", f.sent[0].From, SupportEmail)
	}
	if f.sent[0].To != "a@b.com" {
		t.Fatalf("To = %q", f.sent[0].To)
	}
}

func TestServiceSendRequiresRecipient(t *testing.T) {
	s := NewService(&fakeSender{}, "")
	if err := s.Send(context.Background(), Message{Subject: "x"}); err == nil {
		t.Fatal("missing recipient must error")
	}
}

func TestServiceSendTemplateUnknownTemplateDoesNotSend(t *testing.T) {
	f := &fakeSender{}
	s := NewService(f, "")
	if err := s.SendTemplate(context.Background(), "missing", "a@b.com", nil); err == nil {
		t.Fatal("expected error")
	}
	if len(f.sent) != 0 {
		t.Fatalf("no message should be sent, got %d", len(f.sent))
	}
}

func TestServicePropagatesTransportError(t *testing.T) {
	want := errors.New("smtp down")
	s := NewService(&fakeSender{err: want}, "")
	if err := s.SendTemplate(context.Background(), TemplateWelcome, "a@b.com", nil); !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestNewServiceDefaultsFromToSupport(t *testing.T) {
	f := &fakeSender{}
	s := NewService(f, "")
	if s.From() != SupportEmail {
		t.Fatalf("From() = %q", s.From())
	}
	if err := s.Send(context.Background(), Message{To: "a@b.com"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if f.sent[0].From != SupportEmail {
		t.Fatalf("From = %q", f.sent[0].From)
	}
}

func TestWelcomeDefaultsAndContent(t *testing.T) {
	msg, err := DefaultRegistry().Render(TemplateWelcome, map[string]string{
		"name": "Onboard Owner", "store": "Onboard Demo",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if msg.From != SupportEmail {
		t.Fatalf("From = %q", msg.From)
	}
	if !strings.Contains(msg.Subject, "Welcome to POS.Go") {
		t.Errorf("subject missing English greeting: %q", msg.Subject)
	}
	if !strings.Contains(msg.Subject, "مرحبًا") {
		t.Errorf("subject missing Arabic greeting: %q", msg.Subject)
	}
	for _, want := range []string{"Onboard Owner", "Onboard Demo", SupportEmail, "POS.Go"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("text missing %q", want)
		}
		if !strings.Contains(msg.HTML, want) {
			t.Errorf("html missing %q", want)
		}
	}
	if !strings.Contains(msg.HTML, `dir="rtl"`) {
		t.Error("html should be RTL for the Arabic-first body")
	}
}

func TestWelcomeEscapesHTML(t *testing.T) {
	msg, _ := DefaultRegistry().Render(TemplateWelcome, map[string]string{
		"name": "<b>Ann</b>", "store": `Store & "Co"`,
	})
	if strings.Contains(msg.HTML, "<b>Ann</b>") {
		t.Error("display name must be HTML-escaped")
	}
	if !strings.Contains(msg.HTML, "&lt;b&gt;Ann&lt;/b&gt;") {
		t.Errorf("escaped display name missing: %s", msg.HTML)
	}
	if !strings.Contains(msg.HTML, "Store &amp;") {
		t.Error("store name must be HTML-escaped")
	}
}

func TestWelcomeBlankNameFallbacks(t *testing.T) {
	msg, _ := DefaultRegistry().Render(TemplateWelcome, map[string]string{"name": "   ", "store": ""})
	if !strings.Contains(msg.Text, "there") || !strings.Contains(msg.Text, "your store") {
		t.Fatalf("blank name/store should fall back, got: %s", msg.Text)
	}
}

func TestPasswordResetRendersLinkCodeAndExpiry(t *testing.T) {
	msg, err := DefaultRegistry().Render(TemplatePasswordReset, map[string]string{
		"name": "Sara", "reset_url": "https://api.xamltech.com/reset?token=abc123",
		"code": "482913", "expires": "15 minutes",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"Sara", "https://api.xamltech.com/reset?token=abc123", "482913", "15 minutes", SupportEmail} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("text missing %q", want)
		}
		if !strings.Contains(msg.HTML, want) {
			t.Errorf("html missing %q", want)
		}
	}
	if !strings.Contains(msg.HTML, `href="https://api.xamltech.com/reset?token=abc123"`) {
		t.Error("reset link must be a button href")
	}
	// Bilingual: both an Arabic and an English section.
	if !strings.Contains(msg.Text, "إعادة تعيين") || !strings.Contains(msg.Text, "POS.Go password") {
		t.Error("password reset must be bilingual")
	}
}

func TestPasswordResetToleratesMissingOptionalKeys(t *testing.T) {
	msg, err := DefaultRegistry().Render(TemplatePasswordReset, map[string]string{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(msg.HTML, "href=\"\"") {
		t.Error("empty reset_url must not produce an empty href")
	}
	if !strings.Contains(msg.Text, "30 minutes") {
		t.Error("default expiry should apply")
	}
}

func TestPasswordResetEscapesResetURL(t *testing.T) {
	msg, _ := DefaultRegistry().Render(TemplatePasswordReset, map[string]string{
		"reset_url": `https://x.example/"onclick="alert(1)`,
	})
	if strings.Contains(msg.HTML, `onclick="alert(1)"`) {
		t.Fatalf("reset url must be escaped: %s", msg.HTML)
	}
	if !strings.Contains(msg.HTML, "&#34;onclick=&#34;") {
		t.Errorf("expected escaped quotes: %s", msg.HTML)
	}
}

func TestVerifyEmailCodeAndExpiry(t *testing.T) {
	msg, err := DefaultRegistry().Render(TemplateVerifyEmail, map[string]string{
		"name": "Nour", "code": "135791", "expires": "24 hours",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"Nour", "135791", "24 hours"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("text missing %q", want)
		}
		if !strings.Contains(msg.HTML, want) {
			t.Errorf("html missing %q", want)
		}
	}
}

func TestInvoiceRendersTableFields(t *testing.T) {
	msg, err := DefaultRegistry().Render(TemplateInvoice, map[string]string{
		"store": "Demo Coffee", "number": "INV-2026-0042",
		"amount": "E£1,250.00", "status": "paid", "due_date": "2026-11-01",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"Demo Coffee", "INV-2026-0042", "E£1,250.00", "paid", "2026-11-01"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("text missing %q", want)
		}
		if !strings.Contains(msg.HTML, want) {
			t.Errorf("html missing %q", want)
		}
	}
	if !strings.Contains(msg.Subject, "INV-2026-0042") {
		t.Errorf("subject should carry the invoice number: %q", msg.Subject)
	}
}

func TestInvoiceToleratesMissingDueDate(t *testing.T) {
	msg, _ := DefaultRegistry().Render(TemplateInvoice, map[string]string{
		"number": "INV-1", "amount": "E£10.00",
	})
	if strings.Contains(msg.Text, "تاريخ الاستحقاق") {
		t.Error("missing due_date should drop the line from the text body")
	}
	if strings.Contains(msg.HTML, "تاريخ الاستحقاق") {
		t.Error("missing due_date should drop the row from the HTML table")
	}
	if !strings.Contains(msg.Subject, "INV-1") {
		t.Errorf("subject = %q", msg.Subject)
	}
}

func TestNotificationCarriesTitleAndBody(t *testing.T) {
	msg, err := DefaultRegistry().Render(TemplateNotification, map[string]string{
		"name": "Omar", "title": "تمت إعادة تفعيل حسابك", "body": "يمكنك الآن البيع مجددًا.",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"Omar", "تمت إعادة تفعيل حسابك", "يمكنك الآن البيع مجددًا."} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("text missing %q", want)
		}
		if !strings.Contains(msg.HTML, want) {
			t.Errorf("html missing %q", want)
		}
	}
	if msg.Subject != "تمت إعادة تفعيل حسابك" {
		t.Errorf("subject = %q", msg.Subject)
	}
}

func TestNotificationEscapesBody(t *testing.T) {
	msg, _ := DefaultRegistry().Render(TemplateNotification, map[string]string{
		"title": "<script>x</script>", "body": "<img src=x onerror=1>",
	})
	if strings.Contains(msg.HTML, "<script>") || strings.Contains(msg.HTML, "<img") {
		t.Fatalf("notification content must be escaped: %s", msg.HTML)
	}
	if strings.Contains(msg.Subject, "<") {
		t.Errorf("subject should be plain: %q", msg.Subject)
	}
}

func TestAccountNoticeDefaultsTitle(t *testing.T) {
	msg, err := DefaultRegistry().Render(TemplateAccountNotice, map[string]string{"body": "status changed"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(msg.Subject, "تحديث حالة حسابك") {
		t.Errorf("default title missing from subject: %q", msg.Subject)
	}
	if !strings.Contains(msg.Text, "status changed") {
		t.Error("body missing")
	}
}

func TestSubscriptionEndedRendersReasonAndRenewLink(t *testing.T) {
	msg, err := DefaultRegistry().Render(TemplateSubscriptionEnd, map[string]string{
		"name": "Hana", "reason": "trial expired",
		"renew_url": "https://admin.example.com/renew",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, want := range []string{"Hana", "trial expired", "https://admin.example.com/renew"} {
		if !strings.Contains(msg.Text, want) {
			t.Errorf("text missing %q", want)
		}
		if !strings.Contains(msg.HTML, want) {
			t.Errorf("html missing %q", want)
		}
	}
	if !strings.Contains(msg.HTML, `href="https://admin.example.com/renew"`) {
		t.Error("renew button missing")
	}
}

func TestAllTemplatesFillSubjectAndUseSupportSender(t *testing.T) {
	r := DefaultRegistry()
	for _, name := range r.Names() {
		msg, err := r.Render(name, map[string]string{
			"name": "A", "store": "S", "title": "T", "body": "B",
			"number": "N", "amount": "E£1", "status": "ok",
			"code": "123456", "reset_url": "https://x/y", "verify_url": "https://x/v",
			"renew_url": "https://x/r", "reason": "r",
		})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.TrimSpace(msg.Subject) == "" {
			t.Errorf("%s: empty subject", name)
		}
		if msg.From != SupportEmail {
			t.Errorf("%s: From = %q", name, msg.From)
		}
		if strings.TrimSpace(msg.Text) == "" || strings.TrimSpace(msg.HTML) == "" {
			t.Errorf("%s: empty body", name)
		}
		if !strings.Contains(msg.HTML, "<!DOCTYPE html>") {
			t.Errorf("%s: html not wrapped in the branded shell", name)
		}
	}
}

func TestEnvelopeConversion(t *testing.T) {
	m := Message{To: "a@b.com", From: "f@x.com", Subject: "S", Text: "T", HTML: "H"}
	e := m.Envelope()
	if e.To != m.To || e.From != m.From || e.Subject != m.Subject || e.Text != m.Text || e.HTML != m.HTML {
		t.Fatalf("Envelope mismatch: %+v", e)
	}
}
