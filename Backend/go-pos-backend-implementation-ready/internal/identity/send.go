package identity

import (
	"context"
	"errors"
	"sync"

	"github.com/example/pos-api/internal/config"
)

// SupportEmail is the public support/from address printed on outbound mail and
// the default envelope sender for transactional mail. The email template layer
// aliases this constant so there is a single source of truth.
const SupportEmail = "support@xamltech.com"

// EmailMessage is what the mailer transport delivers. From is the envelope
// sender (empty falls back to the mailer's configured identity); HTML, when
// present, is sent as an alternative part alongside Text.
type EmailMessage struct {
	To      string
	From    string
	Subject string
	Text    string
	HTML    string
}

// MailerConfig configures the mail transport selected by NewMailer. Kind is
// "noop" (record to the dev outbox) or "smtp" (deliver over STARTTLS).
type MailerConfig struct {
	Kind     string
	Host     string
	Port     int
	Username string
	Password string
	From     string
	FromName string
}

// Mailer delivers verification emails. Only opaque carriers travel here; the
// plain email token is included in the body exactly once and is never stored
// in the database (only its sha256 hash is persisted).
type Mailer interface {
	Send(ctx context.Context, msg EmailMessage) error
}

// NewConfiguredMailer builds the process-wide outbound transport from the
// mail section of the config (MAILER / SMTP_*). Every module that sends mail —
// auth, billing, notifications, support — must build it the same way, so the
// envelope sender and the STARTTLS credentials cannot drift apart per caller.
// Nothing is configured → a noop transport writing to the dev outbox, which is
// what a developer without SMTP credentials expects.
func NewConfiguredMailer(t config.TrialConfig) Mailer {
	return NewMailer(MailerConfig{
		Kind: t.Mailer, Host: t.SMTPHost, Port: t.SMTPPort,
		Username: t.SMTPUsername, Password: t.SMTPPassword,
		From: t.MailFrom, FromName: t.MailFromName,
	}, NewDevOutbox())
}

// SMSMessage is what the SMS transport delivers.
type SMSMessage struct {
	To   string
	Text string
}

// SMSSender delivers OTP text messages.
type SMSSender interface {
	Send(ctx context.Context, msg SMSMessage) error
}

// ErrDeliveryUnconfigured is returned by sender backends that are declared but
// not wired (MAILER=smtp / SMS_SENDER=provider without a real provider). The
// verification machinery itself is complete; only delivery is the seam.
var ErrDeliveryUnconfigured = errors.New("delivery backend is not configured")

// OutboxItem is a dev-only copy of a delivered message (no secrets stored in
// production; the dev outbox only exists so a developer can test verification
// flows before a provider is chosen).
type OutboxItem struct {
	To      string
	From    string
	Subject string
	Text    string
	HTML    string
}

// DevOutbox is an in-memory, process-local transcript of NoopMailer/NoopSMS
// sends. Exposed ONLY by a dev-gated endpoint.
type DevOutbox struct {
	mu    sync.Mutex
	items []OutboxItem
}

func NewDevOutbox() *DevOutbox { return &DevOutbox{} }

func (o *DevOutbox) Add(item OutboxItem) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.items = append(o.items, item)
}

func (o *DevOutbox) List(n int) []OutboxItem {
	o.mu.Lock()
	defer o.mu.Unlock()
	if n <= 0 || n > len(o.items) {
		n = len(o.items)
	}
	out := make([]OutboxItem, len(o.items)-n)
	copy(out, o.items[len(o.items)-n:])
	return out
}

func (o *DevOutbox) Clear() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.items = o.items[:0]
}

// NoopMailer records the message in the dev outbox and delivers nowhere. If
// outbox is nil the send is a no-op (used when the backend runs non-dev).
type NoopMailer struct{ outbox *DevOutbox }

func NewNoopMailer(outbox *DevOutbox) *NoopMailer { return &NoopMailer{outbox: outbox} }

func (m *NoopMailer) Send(_ context.Context, msg EmailMessage) error {
	if m.outbox != nil {
		m.outbox.Add(OutboxItem{To: msg.To, From: msg.From, Subject: msg.Subject, Text: msg.Text, HTML: msg.HTML})
	}
	return nil
}

// NoopSMSSender mirrors NoopMailer for SMS.
type NoopSMSSender struct{ outbox *DevOutbox }

func NewNoopSMSSender(outbox *DevOutbox) *NoopSMSSender { return &NoopSMSSender{outbox: outbox} }

func (s *NoopSMSSender) Send(_ context.Context, msg SMSMessage) error {
	if s.outbox != nil {
		s.outbox.Add(OutboxItem{To: msg.To, Subject: "SMS", Text: msg.Text})
	}
	return nil
}

// SMTPMailerDeclared represents MAILER=smtp until SMTP host credentials are
// wired; it fails loudly rather than silently dropping verification mail.
type SMTPMailerDeclared struct{}

func (SMTPMailerDeclared) Send(context.Context, EmailMessage) error { return ErrDeliveryUnconfigured }

// ProviderSMSDeclared represents SMS_SENDER=provider until a provider client
// is wired.
type ProviderSMSDeclared struct{}

func (ProviderSMSDeclared) Send(context.Context, SMSMessage) error { return ErrDeliveryUnconfigured }

// NewMailer selects a mailer by config. SMTP with no host falls back to the
// declared-but-unconfigured mailer so a misconfigured deployment fails loudly
// rather than silently dropping mail.
func NewMailer(cfg MailerConfig, outbox *DevOutbox) Mailer {
	switch cfg.Kind {
	case "noop", "":
		return NewNoopMailer(outbox)
	case "smtp":
		if cfg.Host == "" {
			return SMTPMailerDeclared{}
		}
		return &SMTPMailer{
			host: cfg.Host, port: cfg.Port,
			username: cfg.Username, password: cfg.Password,
			from: cfg.From, fromName: cfg.FromName,
		}
	default:
		return SMTPMailerDeclared{}
	}
}

// NewSMSSender selects an SMS sender by config name.
func NewSMSSender(kind string, outbox *DevOutbox) SMSSender {
	if kind == "noop" {
		return NewNoopSMSSender(outbox)
	}
	return ProviderSMSDeclared{}
}
