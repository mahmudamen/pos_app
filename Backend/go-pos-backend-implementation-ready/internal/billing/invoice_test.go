package billing

import "testing"

func TestInvoiceStatuses(t *testing.T) {
	for _, s := range InvoiceStatuses() {
		if !ValidInvoiceStatus(s) {
			t.Fatalf("%q should be valid", s)
		}
	}
	if ValidInvoiceStatus("draft") || ValidInvoiceStatus("") {
		t.Fatal("unknown invoice statuses must be invalid")
	}
}

func TestCanTransitionInvoice(t *testing.T) {
	allowed := [][2]string{
		{"open", "paid"},
		{"open", "void"},
		{"paid", "refunded"},
		{"open", "open"},
		{"paid", "paid"},
	}
	for _, e := range allowed {
		if !CanTransitionInvoice(e[0], e[1]) {
			t.Fatalf("invoice %s -> %s should be allowed", e[0], e[1])
		}
	}
	denied := [][2]string{
		{"open", "refunded"},
		{"paid", "open"},
		{"paid", "void"},
		{"void", "paid"},
		{"void", "open"},
		{"refunded", "paid"},
		{"refunded", "open"},
		{"void", "void"}, // same state but void is terminal; still "same state" is allowed by design
	}
	for _, e := range denied {
		want := e == [2]string{"void", "void"}
		if got := CanTransitionInvoice(e[0], e[1]); got != want {
			t.Fatalf("invoice %s -> %s = %v, want %v", e[0], e[1], got, want)
		}
	}
	if CanTransitionInvoice("bogus", "paid") || CanTransitionInvoice("open", "bogus") {
		t.Fatal("unknown states must never transition")
	}
}

func TestValidateInvoiceTransition(t *testing.T) {
	if err := ValidateInvoiceTransition("open", "paid"); err != nil {
		t.Fatalf("valid transition rejected: %v", err)
	}
	if err := ValidateInvoiceTransition("paid", "open"); err == nil {
		t.Fatal("paid -> open should error")
	}
	if err := ValidateInvoiceTransition("open", "bogus"); err == nil {
		t.Fatal("unknown target should error")
	}
}

func TestInvoiceValidate(t *testing.T) {
	ok := Invoice{AmountMinor: 0, Currency: "EGP", Description: "Starter monthly"}
	if err := ok.Validate(); err != nil {
		t.Fatalf("free invoice rejected: %v", err)
	}
	cases := []struct {
		name string
		inv  Invoice
	}{
		{"negative", Invoice{AmountMinor: -1, Currency: "EGP", Description: "x"}},
		{"bad currency", Invoice{AmountMinor: 1, Currency: "EG", Description: "x"}},
		{"empty description", Invoice{AmountMinor: 1, Currency: "EGP"}},
		{"long description", Invoice{AmountMinor: 1, Currency: "EGP", Description: string(make([]byte, 256))}},
	}
	for _, tc := range cases {
		if err := tc.inv.Validate(); err == nil {
			t.Fatalf("%s should fail validation", tc.name)
		}
	}
}

func TestInvoiceTotalMinor(t *testing.T) {
	invoices := []Invoice{
		{AmountMinor: 1000, Status: InvoiceOpen},
		{AmountMinor: 2500, Status: InvoicePaid},
		{AmountMinor: 9999, Status: InvoiceVoid},
		{AmountMinor: 500, Status: InvoiceRefunded},
	}
	if got := InvoiceTotalMinor(invoices); got != 4000 {
		t.Fatalf("total = %d, want 4000 (void ignored)", got)
	}
	if got := InvoiceTotalMinor(nil); got != 0 {
		t.Fatalf("empty total = %d, want 0", got)
	}
}
