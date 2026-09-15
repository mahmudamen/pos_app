package billing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Charge/refund outcomes a gateway can report.
const (
	PaymentSucceeded = "succeeded"
	PaymentPending   = "pending"
	PaymentFailed    = "failed"
)

var (
	// ErrDeclined is returned when a provider refuses a charge.
	ErrDeclined = errors.New("payment declined")
	// ErrInvalidAmount is returned for a non-positive charge amount.
	ErrInvalidAmount = errors.New("charge amount must be greater than zero")
	// ErrUnsupportedCurrency is returned when a provider cannot bill a currency.
	ErrUnsupportedCurrency = errors.New("unsupported currency")
)

// ChargeRequest describes one attempt to collect an invoice.
type ChargeRequest struct {
	TenantID       string
	InvoiceID      string
	AmountMinor    int64
	Currency       string
	Description    string
	IdempotencyKey string
	Metadata       map[string]string
}

// ChargeResult is the normalized outcome of a charge attempt.
type ChargeResult struct {
	Provider    string    `json:"provider"`
	Reference   string    `json:"reference"`
	Status      string    `json:"status"`
	AmountMinor int64     `json:"amount_minor"`
	Currency    string    `json:"currency"`
	Message     string    `json:"message,omitempty"`
	PaidAt      time.Time `json:"paid_at"`
}

// RefundRequest describes a (partial) refund against a previous charge.
type RefundRequest struct {
	Reference      string
	AmountMinor    int64
	Currency       string
	IdempotencyKey string
}

// RefundResult is the normalized outcome of a refund.
type RefundResult struct {
	Provider    string `json:"provider"`
	Reference   string `json:"reference"`
	Status      string `json:"status"`
	AmountMinor int64  `json:"amount_minor"`
}

// Gateway is the provider-agnostic payment interface. The SaaS core depends
// only on this, never on a concrete provider SDK, so a real gateway (Paymob,
// Stripe, ...) can be added behind it without touching billing logic.
type Gateway interface {
	Name() string
	Charge(ctx context.Context, req ChargeRequest) (ChargeResult, error)
	Refund(ctx context.Context, req RefundRequest) (RefundResult, error)
	Supports(currency string) bool
}

// MockGateway is a deterministic in-memory provider for development and tests.
// It records charges by idempotency key so a retried charge returns the same
// reference, and it can be told to decline a specific amount or everything.
type MockGateway struct {
	mu sync.Mutex

	// Decline, when true, makes every charge fail.
	Decline bool
	// DeclineAmountMinor, when > 0, makes charges of exactly that amount fail.
	DeclineAmountMinor int64
	// SupportedCurrencies limits the currencies accepted; empty = all.
	SupportedCurrencies []string

	byKey map[string]ChargeResult
}

// NewMockGateway returns a MockGateway that accepts every currency and
// succeeds for every charge.
func NewMockGateway() *MockGateway {
	return &MockGateway{byKey: map[string]ChargeResult{}}
}

func (g *MockGateway) Name() string { return "mock" }

func (g *MockGateway) Supports(currency string) bool {
	if len(g.SupportedCurrencies) == 0 {
		return true
	}
	for _, c := range g.SupportedCurrencies {
		if c == currency {
			return true
		}
	}
	return false
}

func (g *MockGateway) Charge(_ context.Context, req ChargeRequest) (ChargeResult, error) {
	if req.AmountMinor <= 0 {
		return ChargeResult{}, ErrInvalidAmount
	}
	if !g.Supports(req.Currency) {
		return ChargeResult{}, ErrUnsupportedCurrency
	}
	if key := req.IdempotencyKey; key != "" {
		g.mu.Lock()
		if prior, ok := g.byKey[key]; ok {
			g.mu.Unlock()
			return prior, nil
		}
		g.mu.Unlock()
	}
	if g.Decline || (g.DeclineAmountMinor > 0 && req.AmountMinor == g.DeclineAmountMinor) {
		return ChargeResult{}, ErrDeclined
	}
	result := ChargeResult{
		Provider:    g.Name(),
		Reference:   paymentReference("mock", req.IdempotencyKey, req.InvoiceID),
		Status:      PaymentSucceeded,
		AmountMinor: req.AmountMinor,
		Currency:    req.Currency,
		PaidAt:      time.Now().UTC(),
	}
	if req.IdempotencyKey != "" {
		g.mu.Lock()
		g.byKey[req.IdempotencyKey] = result
		g.mu.Unlock()
	}
	return result, nil
}

func (g *MockGateway) Refund(_ context.Context, req RefundRequest) (RefundResult, error) {
	if req.AmountMinor <= 0 {
		return RefundResult{}, ErrInvalidAmount
	}
	if req.Reference == "" {
		return RefundResult{}, fmt.Errorf("reference is required")
	}
	return RefundResult{
		Provider:    g.Name(),
		Reference:   paymentReference("mockrefund", req.IdempotencyKey, req.Reference),
		Status:      PaymentSucceeded,
		AmountMinor: req.AmountMinor,
	}, nil
}

// ManualGateway models an offline/manual invoice: an administrator records the
// payment out-of-band (bank transfer, cash) and marks it settled. It never
// performs an external call.
type ManualGateway struct{}

func (ManualGateway) Name() string { return "manual" }

func (ManualGateway) Supports(string) bool { return true }

func (ManualGateway) Charge(_ context.Context, req ChargeRequest) (ChargeResult, error) {
	if req.AmountMinor < 0 {
		return ChargeResult{}, ErrInvalidAmount
	}
	return ChargeResult{
		Provider:    "manual",
		Reference:   paymentReference("manual", req.IdempotencyKey, req.InvoiceID),
		Status:      PaymentSucceeded,
		AmountMinor: req.AmountMinor,
		Currency:    req.Currency,
		Message:     "recorded manually",
		PaidAt:      time.Now().UTC(),
	}, nil
}

func (ManualGateway) Refund(_ context.Context, req RefundRequest) (RefundResult, error) {
	if req.AmountMinor <= 0 {
		return RefundResult{}, ErrInvalidAmount
	}
	return RefundResult{
		Provider:    "manual",
		Reference:   paymentReference("manualrefund", req.IdempotencyKey, req.Reference),
		Status:      PaymentSucceeded,
		AmountMinor: req.AmountMinor,
	}, nil
}

// DefaultGateway is the provider-free gateway the server registers by default.
func DefaultGateway() Gateway { return NewMockGateway() }

// registry is the process-wide provider lookup the transport uses to resolve a
// provider name from a request. It is seeded with the built-in providers.
var (
	registryMu sync.RWMutex
	registry   = map[string]Gateway{
		"mock":   NewMockGateway(),
		"manual": ManualGateway{},
	}
)

// RegisterGateway makes a provider available by name. Intended for tests and
// for wiring a real provider at startup.
func RegisterGateway(g Gateway) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[g.Name()] = g
}

// ResolveGateway returns the provider registered under name.
func ResolveGateway(name string) (Gateway, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	g, ok := registry[name]
	return g, ok
}

// GatewayNames returns the registered provider names, sorted.
func GatewayNames() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// paymentReference builds a stable, provider-prefixed reference from the
// idempotency key and invoice id. Deterministic so the same logical payment
// always yields the same reference.
func paymentReference(prefix, idempotencyKey, invoiceID string) string {
	sum := sha256.Sum256([]byte(idempotencyKey + "|" + invoiceID))
	return prefix + "_" + hex.EncodeToString(sum[:12])
}
