package billing

import (
	"context"
	"errors"
	"testing"
)

func TestMockGatewayChargeSuccess(t *testing.T) {
	g := NewMockGateway()
	req := ChargeRequest{TenantID: "t1", InvoiceID: "inv1", AmountMinor: 29900, Currency: "EGP", IdempotencyKey: "pay-1"}
	res, err := g.Charge(context.Background(), req)
	if err != nil {
		t.Fatalf("charge failed: %v", err)
	}
	if res.Status != PaymentSucceeded || res.Provider != "mock" || res.AmountMinor != 29900 || res.Currency != "EGP" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.Reference == "" || res.PaidAt.IsZero() {
		t.Fatalf("reference/paid_at missing: %+v", res)
	}
	// Same idempotency key must replay the identical reference.
	again, err := g.Charge(context.Background(), req)
	if err != nil {
		t.Fatalf("replayed charge failed: %v", err)
	}
	if again.Reference != res.Reference {
		t.Fatalf("idempotent charge reference changed: %q vs %q", again.Reference, res.Reference)
	}

	// A different invoice gets a different reference.
	other, _ := g.Charge(context.Background(), ChargeRequest{InvoiceID: "inv2", AmountMinor: 100, Currency: "EGP", IdempotencyKey: "pay-2"})
	if other.Reference == res.Reference {
		t.Fatal("distinct charges should not share a reference")
	}
}

func TestMockGatewayChargeValidation(t *testing.T) {
	g := NewMockGateway()
	if _, err := g.Charge(context.Background(), ChargeRequest{AmountMinor: 0, Currency: "EGP"}); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("zero amount: want ErrInvalidAmount, got %v", err)
	}
	if _, err := g.Charge(context.Background(), ChargeRequest{AmountMinor: -5, Currency: "EGP"}); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("negative amount: want ErrInvalidAmount, got %v", err)
	}
}

func TestMockGatewayCurrencySupport(t *testing.T) {
	g := &MockGateway{SupportedCurrencies: []string{"EGP"}}
	if !g.Supports("EGP") {
		t.Fatal("EGP should be supported")
	}
	if g.Supports("USD") {
		t.Fatal("USD should be unsupported")
	}
	if _, err := g.Charge(context.Background(), ChargeRequest{AmountMinor: 100, Currency: "USD"}); !errors.Is(err, ErrUnsupportedCurrency) {
		t.Fatalf("want ErrUnsupportedCurrency, got %v", err)
	}
	// Empty list = every currency.
	all := NewMockGateway()
	if !all.Supports("USD") || !all.Supports("EGP") {
		t.Fatal("empty supported list should accept all currencies")
	}
}

func TestMockGatewayDecline(t *testing.T) {
	declining := &MockGateway{Decline: true}
	if _, err := declining.Charge(context.Background(), ChargeRequest{AmountMinor: 100, Currency: "EGP"}); !errors.Is(err, ErrDeclined) {
		t.Fatalf("want ErrDeclined, got %v", err)
	}
	byAmount := &MockGateway{DeclineAmountMinor: 4242}
	if _, err := byAmount.Charge(context.Background(), ChargeRequest{AmountMinor: 4242, Currency: "EGP"}); !errors.Is(err, ErrDeclined) {
		t.Fatalf("matching amount: want ErrDeclined, got %v", err)
	}
	if _, err := byAmount.Charge(context.Background(), ChargeRequest{AmountMinor: 100, Currency: "EGP"}); err != nil {
		t.Fatalf("non-matching amount should succeed: %v", err)
	}
}

func TestMockGatewayRefund(t *testing.T) {
	g := NewMockGateway()
	res, err := g.Refund(context.Background(), RefundRequest{Reference: "mock_abc", AmountMinor: 500, IdempotencyKey: "r1"})
	if err != nil {
		t.Fatalf("refund failed: %v", err)
	}
	if res.Status != PaymentSucceeded || res.AmountMinor != 500 || res.Provider != "mock" {
		t.Fatalf("unexpected refund: %+v", res)
	}
	if _, err := g.Refund(context.Background(), RefundRequest{Reference: "mock_abc", AmountMinor: 0}); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("zero refund: want ErrInvalidAmount, got %v", err)
	}
	if _, err := g.Refund(context.Background(), RefundRequest{AmountMinor: 100}); err == nil {
		t.Fatal("refund without reference should error")
	}
}

func TestManualGateway(t *testing.T) {
	g := ManualGateway{}
	if g.Name() != "manual" {
		t.Fatalf("name = %q", g.Name())
	}
	// Manual invoices may be zero (recorded, no charge).
	res, err := g.Charge(context.Background(), ChargeRequest{AmountMinor: 0, Currency: "EGP", InvoiceID: "inv1", IdempotencyKey: "m1"})
	if err != nil {
		t.Fatalf("manual zero charge failed: %v", err)
	}
	if res.Status != PaymentSucceeded || res.Message == "" {
		t.Fatalf("unexpected manual result: %+v", res)
	}
	if _, err := g.Charge(context.Background(), ChargeRequest{AmountMinor: -1}); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("negative manual charge: want ErrInvalidAmount, got %v", err)
	}
	if _, err := g.Refund(context.Background(), RefundRequest{Reference: "m1", AmountMinor: 100}); err != nil {
		t.Fatalf("manual refund failed: %v", err)
	}
}

func TestGatewayRegistry(t *testing.T) {
	g, ok := ResolveGateway("mock")
	if !ok || g.Name() != "mock" {
		t.Fatalf("mock should be registered: %v %v", g, ok)
	}
	if _, ok := ResolveGateway("manual"); !ok {
		t.Fatal("manual should be registered")
	}
	if _, ok := ResolveGateway("does-not-exist"); ok {
		t.Fatal("unknown provider should not resolve")
	}
	names := GatewayNames()
	if len(names) < 2 || names[0] != "manual" || names[1] != "mock" {
		t.Fatalf("gateway names not sorted/complete: %v", names)
	}
}

func TestPaymentReferenceDeterministic(t *testing.T) {
	a := paymentReference("mock", "key", "inv")
	b := paymentReference("mock", "key", "inv")
	if a != b {
		t.Fatalf("reference should be deterministic: %q vs %q", a, b)
	}
	if c := paymentReference("mock", "key", "other"); c == a {
		t.Fatal("different invoice should change the reference")
	}
	if len(a) == 0 || a[:5] != "mock_" {
		t.Fatalf("reference should carry the provider prefix: %q", a)
	}
}
