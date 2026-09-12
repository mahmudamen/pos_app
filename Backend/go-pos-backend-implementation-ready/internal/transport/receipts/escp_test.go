package receipts

import (
	"bytes"
	"strings"
	"testing"
)

func sampleReceipt() Receipt {
	return Receipt{
		TenantName:    "Demo Store",
		TenantAddress: "Cairo, Egypt",
		SaleID:        "sale-123",
		Status:        "completed",
		CreatedAt:     "2026-09-12 12:00:00",
		Cashier:       "Demo Manager",
		Device:        "register-1",
		TableName:     "T1",
		FloorName:     "Ground",
		Currency:      "EGP",
		SubtotalMinor: 2500,
		DiscountMinor: 500,
		TipsMinor:     0,
		TotalMinor:    2000,
		Lines: []ReceiptLine{
			{Name: "Flat White", Sku: "FW", Qty: 2, Price: 1000, Total: 2000},
		},
		Payments: []ReceiptPayment{{Method: "card", Amount: 2000}},
	}
}

func TestBuildBytesStartsWithInit(t *testing.T) {
	out := BuildBytes(sampleReceipt())
	if len(out) == 0 || out[0] != esc || out[1] != 0x40 {
		t.Fatalf("expected ESC @ init prefix, got %v", out[:2])
	}
}

func TestBuildBytesEndsWithCut(t *testing.T) {
	out := BuildBytes(sampleReceipt())
	if !bytes.HasSuffix(out, []byte{gs, 0x56, 0x00}) {
		t.Fatal("expected GS V 0 cut at the end")
	}
}

func TestBuildBytesEmbedsQRForSale(t *testing.T) {
	out := BuildBytes(sampleReceipt())
	if !bytes.Contains(out, []byte("posgo:sale:sale-123")) {
		t.Fatal("expected QR payload with the sale reference")
	}
}

func TestBuildBytesEmbedsTotalsAndLines(t *testing.T) {
	out := BuildBytes(sampleReceipt())
	text := string(out)
	for _, want := range []string{
		"Demo Store",
		"TOTAL E£20.00",
		"Subtotal",
		"E£20.00",
		"Flat White",
		"2 x E£10.00",
		"Paid(card)",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in rendered receipt", want)
		}
	}
}

func TestCenterPadsWithinWidth(t *testing.T) {
	got := center("X", 32)
	if len([]rune(got)) != 32 {
		t.Fatalf("expected 32 runes, got %d", len([]rune(got)))
	}
	if !strings.HasPrefix(got, " ") || !strings.Contains(strings.TrimSpace(got), "X") {
		t.Errorf("expected X centered between spaces, got %q", got)
	}
}

func TestMoneyFormatsEGP(t *testing.T) {
	if got := money(2000, "EGP"); got != "E£20.00" {
		t.Errorf("money(2000, EGP) = %q, want E£20.00", got)
	}
	if got := money(-1234, "USD"); got != "-USD 12.34" {
		t.Errorf("money(-1234, USD) = %q, want -USD 12.34", got)
	}
}

func TestPairPadsLabel(t *testing.T) {
	got := pair("Subtotal", "E£20.00")
	if !strings.HasPrefix(got, "Subtotal") || !strings.Contains(got, "E£20.00") {
		t.Errorf("unexpected pair layout: %q", got)
	}
}

func TestBuildBytesEmptyPaymentsRendersTotal(t *testing.T) {
	r := sampleReceipt()
	r.Payments = nil
	out := BuildBytes(r)
	if !strings.Contains(string(out), "TOTAL E£20.00") {
		t.Fatal("expected total even without payment lines")
	}
}
