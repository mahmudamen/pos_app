// Package email is the transactional mail layer: a registry of named
// templates plus a service that renders one and hands it to a transport. Every
// workflow (welcome, password reset, verification, invoices, notifications)
// sends through here so wording and branding stay in one place, and the
// transport remains swappable (SMTP in prod, noop/outbox in dev, a fake in tests).
package email

import (
	"context"
	"fmt"
	"sort"

	"github.com/example/pos-api/internal/identity"
)

// SupportEmail is the public support/from address printed on outbound mail and
// the default envelope sender. It is shared with the identity layer (single
// definition, no drift).
const SupportEmail = identity.SupportEmail

// Built-in template names for Service.SendTemplate.
const (
	TemplateWelcome         = "welcome"
	TemplateVerifyEmail     = "verify_email"
	TemplatePasswordReset   = "password_reset"
	TemplateInvoice         = "invoice"
	TemplateNotification    = "notification"
	TemplateAccountNotice   = "account_notice"
	TemplateSubscriptionEnd = "subscription_ended"
)

// Sender delivers a message. identity.Mailer satisfies it, so a transport built
// by identity.NewMailer can be passed straight in.
type Sender interface {
	Send(ctx context.Context, msg identity.EmailMessage) error
}

// Renderer turns template data into a concrete message. Templates must
// tolerate missing optional keys (they fall back to sensible copy); validating
// required keys belongs to the workflow that owns them.
type Renderer func(data map[string]string) (Message, error)

// Registry maps template names to renderers. The zero value is not usable;
// build one with NewRegistry or DefaultRegistry.
type Registry struct {
	renderers map[string]Renderer
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{renderers: map[string]Renderer{}}
}

// DefaultRegistry returns a registry pre-loaded with every built-in template.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(TemplateWelcome, renderWelcome)
	r.Register(TemplateVerifyEmail, renderVerifyEmail)
	r.Register(TemplatePasswordReset, renderPasswordReset)
	r.Register(TemplateInvoice, renderInvoice)
	r.Register(TemplateNotification, renderNotification)
	r.Register(TemplateAccountNotice, renderAccountNotice)
	r.Register(TemplateSubscriptionEnd, renderSubscriptionEnded)
	return r
}

// Register adds or replaces a template.
func (r *Registry) Register(name string, fn Renderer) {
	if r.renderers == nil {
		r.renderers = map[string]Renderer{}
	}
	r.renderers[name] = fn
}

// Has reports whether a template is registered.
func (r *Registry) Has(name string) bool {
	_, ok := r.renderers[name]
	return ok
}

// Render builds a message for a registered template.
func (r *Registry) Render(name string, data map[string]string) (Message, error) {
	fn, ok := r.renderers[name]
	if !ok {
		return Message{}, fmt.Errorf("email: unknown template %q", name)
	}
	return fn(data)
}

// Names lists registered template names, sorted.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.renderers))
	for n := range r.renderers {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Message is a rendered email awaiting delivery.
type Message struct {
	To      string
	From    string
	Subject string
	Text    string
	HTML    string
}

// Envelope converts to the transport wire type.
func (m Message) Envelope() identity.EmailMessage {
	return identity.EmailMessage{
		To:      m.To,
		From:    m.From,
		Subject: m.Subject,
		Text:    m.Text,
		HTML:    m.HTML,
	}
}

// Service renders named templates and delivers them through a Sender, stamping
// the configured default From when a template leaves it empty.
type Service struct {
	sender   Sender
	registry *Registry
	from     string
}

// NewService wires a transport to the default template registry. from is the
// fallback sender stamped on messages that do not set one.
func NewService(sender Sender, from string) *Service {
	if from == "" {
		from = SupportEmail
	}
	return &Service{sender: sender, registry: DefaultRegistry(), from: from}
}

// Registry exposes the template registry (read-only use intended).
func (s *Service) Registry() *Registry { return s.registry }

// From returns the default sender address.
func (s *Service) From() string { return s.from }

// Send delivers an already-built message, filling in the default From.
func (s *Service) Send(ctx context.Context, msg Message) error {
	if msg.From == "" {
		msg.From = s.from
	}
	if msg.To == "" {
		return fmt.Errorf("email: missing recipient")
	}
	return s.sender.Send(ctx, msg.Envelope())
}

// SendTemplate renders a named template and delivers it to `to`.
func (s *Service) SendTemplate(ctx context.Context, name, to string, data map[string]string) error {
	msg, err := s.registry.Render(name, data)
	if err != nil {
		return err
	}
	msg.To = to
	return s.Send(ctx, msg)
}
