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
	out := BuildBytes(sampleReceipt(), ReceiptOptions{Cols: cols80, Cut: true})
	if len(out) == 0 || out[0] != esc || out[1] != 0x40 {
		t.Fatalf("expected ESC @ init prefix, got %v", out[:2])
	}
}

func TestBuildBytesEndsWithCut(t *testing.T) {
	out := BuildBytes(sampleReceipt(), ReceiptOptions{Cols: cols80, Cut: true})
	if !bytes.HasSuffix(out, []byte{gs, 0x56, 0x00}) {
		t.Fatal("expected GS V 0 cut at the end")
	}
}

func TestBuildBytesOmitsCutWhenDisabled(t *testing.T) {
	out := BuildBytes(sampleReceipt(), ReceiptOptions{Cols: cols80})
	if bytes.HasSuffix(out, []byte{gs, 0x56, 0x00}) {
		t.Fatal("expected no GS V 0 cut when cut is disabled")
	}
}

func TestBuildBytesEmbedsQRForSale(t *testing.T) {
	out := BuildBytes(sampleReceipt(), ReceiptOptions{Cols: cols80, Cut: true})
	if !bytes.Contains(out, []byte("posgo:sale:sale-123")) {
		t.Fatal("expected QR payload with the sale reference")
	}
}

func TestBuildBytesEmbedsTotalsAndLines(t *testing.T) {
	out := BuildBytes(sampleReceipt(), ReceiptOptions{Cols: cols80, Cut: true})
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
	got := pair("Subtotal", "E£20.00", cols80)
	if !strings.HasPrefix(got, "Subtotal") || !strings.Contains(got, "E£20.00") {
		t.Errorf("unexpected pair layout: %q", got)
	}
}

func TestBuildBytesEmptyPaymentsRendersTotal(t *testing.T) {
	r := sampleReceipt()
	r.Payments = nil
	out := BuildBytes(r, ReceiptOptions{Cols: cols80, Cut: true})
	if !strings.Contains(string(out), "TOTAL E£20.00") {
		t.Fatal("expected total even without payment lines")
	}
}

func TestBuildBytesNarrow58mmLayout(t *testing.T) {
	out := BuildBytes(sampleReceipt(), ReceiptOptions{Cols: cols58, Cut: true})
	text := string(out)
	if strings.Contains(text, strings.Repeat("-", cols80)) {
		t.Fatal("58mm layout must not contain the 80mm dash separator")
	}
	if !strings.Contains(text, strings.Repeat("-", cols58)) {
		t.Fatal("expected 58mm dash separator")
	}
	for _, want := range []string{"Demo Store", "TOTAL E£20.00", "Flat White", "Paid(card)"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %q in 58mm receipt", want)
		}
	}
}

func TestBuildBytesClampsUnknownWidth(t *testing.T) {
	out := BuildBytes(sampleReceipt(), ReceiptOptions{Cols: 7, Cut: true})
	if !strings.Contains(string(out), strings.Repeat("-", cols80)) {
		t.Fatal("unknown widths must fall back to the 80mm layout")
	}
}

func TestReceiptColsMapping(t *testing.T) {
	for in, want := range map[int]int{cols58: cols58, cols80: cols80, 0: cols80, 40: cols80} {
		if got := receiptCols(in); got != want {
			t.Errorf("receiptCols(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestBuildBytesCompactSkipsInterlineFeeds(t *testing.T) {
	normal := BuildBytes(sampleReceipt(), ReceiptOptions{Cols: cols58, Cut: true})
	compact := BuildBytes(sampleReceipt(), ReceiptOptions{Cols: cols58, Cut: true, Compact: true})
	if len(compact) >= len(normal) {
		t.Fatalf("compact layout should be strictly smaller, got %d vs %d", len(compact), len(normal))
	}
	if !strings.Contains(string(compact), "TOTAL E£20.00") {
		t.Fatal("compact layout must still render totals")
	}
}
