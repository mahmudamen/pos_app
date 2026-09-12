package sales

import (
	"strings"
	"testing"
)

func TestNormalizePaymentsDefaultsToCashForFullTotal(t *testing.T) {
	payments, tips, err := normalizePayments(nil, 1250)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payments) != 1 {
		t.Fatalf("expected one payment line, got %d", len(payments))
	}
	if payments[0].Method != "cash" || payments[0].AmountMinor != 1250 {
		t.Errorf("expected cash/1250, got %v", payments[0])
	}
	if tips != 0 {
		t.Errorf("expected zero tips, got %d", tips)
	}
}

func TestNormalizePaymentsAcceptsSingleCard(t *testing.T) {
	payments, _, err := normalizePayments([]paymentRequest{{Method: "card", AmountMinor: 999}}, 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payments) != 1 || payments[0].Method != "card" {
		t.Errorf("expected single card line, got %v", payments)
	}
}

func TestNormalizePaymentsAcceptsSplitTender(t *testing.T) {
	payments, _, err := normalizePayments([]paymentRequest{
		{Method: "cash", AmountMinor: 500},
		{Method: "card", AmountMinor: 400},
		{Method: "mobile", AmountMinor: 350},
	}, 1250)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(payments) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(payments))
	}
}

func TestNormalizePaymentsRejectsUnknownMethod(t *testing.T) {
	_, _, err := normalizePayments([]paymentRequest{{Method: "cheque", AmountMinor: 1250}}, 1250)
	if err == nil {
		t.Fatal("expected error for unknown payment method")
	}
	if !strings.Contains(err.Error(), "cheque") {
		t.Errorf("expected method in error message, got %q", err.Error())
	}
}

func TestNormalizePaymentsRejectsZeroOrNegativeAmount(t *testing.T) {
	if _, _, err := normalizePayments([]paymentRequest{{Method: "cash", AmountMinor: 0}}, 1250); err == nil {
		t.Error("expected error for zero amount")
	}
	if _, _, err := normalizePayments([]paymentRequest{{Method: "cash", AmountMinor: -100}}, 1250); err == nil {
		t.Error("expected error for negative amount")
	}
}

func TestNormalizePaymentsRejectsSumMismatch(t *testing.T) {
	_, _, err := normalizePayments([]paymentRequest{
		{Method: "cash", AmountMinor: 500},
		{Method: "card", AmountMinor: 300},
	}, 1250)
	if err == nil {
		t.Fatal("expected error for payments that do not sum to the total")
	}
}

func TestNormalizePaymentsAggregatesTips(t *testing.T) {
	payments, tips, err := normalizePayments([]paymentRequest{
		{Method: "cash", AmountMinor: 1000, TipMinor: 50},
		{Method: "card", AmountMinor: 250, TipMinor: 25},
	}, 1250)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tips != 75 {
		t.Errorf("expected aggregate tips 75, got %d", tips)
	}
	if payments[0].TipMinor != 50 || payments[1].TipMinor != 25 {
		t.Errorf("expected per-line tips preserved, got %v", payments)
	}
}

func TestNormalizePaymentsRejectsNegativeTip(t *testing.T) {
	_, _, err := normalizePayments([]paymentRequest{{Method: "cash", AmountMinor: 1250, TipMinor: -10}}, 1250)
	if err == nil {
		t.Fatal("expected error for negative tip")
	}
}

func TestPrimaryPaymentMethodReturnsLargestLine(t *testing.T) {
	got := primaryPaymentMethod([]paymentRequest{
		{Method: "cash", AmountMinor: 400},
		{Method: "card", AmountMinor: 650},
		{Method: "mobile", AmountMinor: 200},
	})
	if got != "card" {
		t.Errorf("expected card, got %q", got)
	}
}

func TestPrimaryPaymentMethodIsStableOnTies(t *testing.T) {
	got := primaryPaymentMethod([]paymentRequest{
		{Method: "cash", AmountMinor: 625},
		{Method: "card", AmountMinor: 625},
	})
	if got != "cash" {
		t.Errorf("expected first line on tie, got %q", got)
	}
}

func TestPrimaryPaymentMethodDefaultsToCash(t *testing.T) {
	if got := primaryPaymentMethod(nil); got != "cash" {
		t.Errorf("expected cash, got %q", got)
	}
}
