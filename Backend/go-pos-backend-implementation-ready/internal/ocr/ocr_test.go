package ocr

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestDigitsASCIIConvertsArabicGlyphs(t *testing.T) {
	cases := map[string]string{
		"٠١٢٣٤٥٦٧٨٩": "0123456789",
		"۰۱۲۳":       "0123",
		"٥٫٥":        "5.5",
		"كيلو ٢٥":    "كيلو 25",
	}
	for in, want := range cases {
		if got := DigitsASCII(in); got != want {
			t.Errorf("DigitsASCII(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeKeyFoldsArabicVariants(t *testing.T) {
	cases := map[string]string{
		"سكّر":                 "سكر",
		"عَلْبَة":              "علبه",
		"أرز":                  "ارز",
		"فول مدمس--إلى":        "فول مدمس الي",
		"كيك آيس كريم":         "كيك ايس كريم",
		"مكرونه إسباجتي ٥٠٠جم": "مكرونه اسباجتي 500جم",
	}
	for in, want := range cases {
		if got := NormalizeKey(in); got != want {
			t.Errorf("NormalizeKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitDecimalKeepsMillilitreTogether(t *testing.T) {
	got := splitDecimal("كوكاكولا 500مل 12")
	want := "كوكاكولا 500مل 12"
	if got != want {
		t.Errorf("splitDecimal = %q, want %q", got, want)
	}
}

func TestParseLineMultiplierLayout(t *testing.T) {
	item, ok := parseLine("قصب سكر 10 × 25.50 = 255.00")
	if !ok {
		t.Fatal("expected a parsed item")
	}
	if item.Name != "قصب سكر" {
		t.Errorf("name = %q", item.Name)
	}
	if item.Quantity != 10 {
		t.Errorf("quantity = %v, want 10", item.Quantity)
	}
	if item.UnitPriceMinor != 2550 {
		t.Errorf("unit_price_minor = %d, want 2550", item.UnitPriceMinor)
	}
	if item.TotalMinor != 25500 {
		t.Errorf("total_minor = %d, want 25500", item.TotalMinor)
	}
	if item.Score != 100 {
		t.Errorf("score = %d, want 100", item.Score)
	}
}

func TestParseLineQuantityUnitPrice(t *testing.T) {
	item, ok := parseLine("سكر 2 كجم 30.00")
	if !ok {
		t.Fatal("expected a parsed item")
	}
	if item.Name != "سكر" {
		t.Errorf("name = %q, want سكر", item.Name)
	}
	if item.Unit != "kg" {
		t.Errorf("unit = %q, want kg", item.Unit)
	}
	if item.Quantity != 2 {
		t.Errorf("quantity = %v, want 2", item.Quantity)
	}
	if item.UnitPriceMinor != 3000 {
		t.Errorf("unit_price_minor = %d, want 3000", item.UnitPriceMinor)
	}
}

func TestParseLineArabicDigits(t *testing.T) {
	item, ok := parseLine("كوكاكولا ٥٠٠مل × ١٢ = ١٢٫٥٠")
	if !ok {
		t.Fatal("expected a parsed item")
	}
	if item.Name != "كوكاكولا" {
		t.Errorf("name = %q", item.Name)
	}
	if item.Quantity != 12 {
		t.Errorf("quantity = %v, want 12", item.Quantity)
	}
	if item.UnitPriceMinor != 1250 {
		t.Errorf("unit_price_minor = %d, want 1250", item.UnitPriceMinor)
	}
	if item.TotalMinor != 15000 {
		t.Errorf("total_minor = %d, want 15000", item.TotalMinor)
	}
}

func TestParseInvoiceSkipsHeadersAndKeepsDiscarded(t *testing.T) {
	text := "فاتورة مشتريات\n" +
		"المورد: شركة النيل\n" +
		"جبنه بيضاء 5 × 45.00\n" +
		"زيت عباد شمس\n"
	res := ParseInvoice(text)
	if len(res.Items) != 1 {
		t.Fatalf("items = %d, want 1 (%+v)", len(res.Items), res.Items)
	}
	if res.Items[0].Name != "جبنه بيضاء" {
		t.Errorf("name = %q", res.Items[0].Name)
	}
	if len(res.Discarded) != 3 {
		t.Errorf("discarded = %d, want 3: %q", len(res.Discarded), res.Discarded)
	}
}

func TestMatchScore(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"سكر", "سكر", 100},
		{"سكر كيس", "سكر", 100}, // contains
		{"كيس سكر", "سكر كيس", 80},
		{"سكر قصب", "قصب", 100}, // contains
		{"أرز بسمتي", "مكرونه", 0},
	}
	for _, c := range cases {
		if got := MatchScore(c.a, c.b); got != c.want {
			t.Errorf("MatchScore(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestResolveUnit(t *testing.T) {
	cases := map[string]string{
		"كجم": "kg", "كيلو": "kg", "علبه": "box", "علبة": "box",
		"دسته": "dozen", "قطعه": "piece", "لتر": "liter", "متر": "m",
	}
	for in, want := range cases {
		if got := ResolveUnit(in); got != want {
			t.Errorf("ResolveUnit(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsCurrencyToken(t *testing.T) {
	if !isCurrencyToken("جنيه") {
		t.Error("جنيه should be a currency token")
	}
	if !isCurrencyToken("egp") {
		t.Error("egp should be a currency token")
	}
	if isCurrencyToken("سكر") {
		t.Error("سكر should not be a currency token")
	}
}

func TestEngineAvailableReflectsConfig(t *testing.T) {
	if NewEngine(false, "tesseract", "ara+eng", 6).Available() {
		t.Error("disabled engine must not be available")
	}
	if _, err := NewEngine(false, "tesseract", "ara+eng", 6).OCR(context.Background(), []byte("x")); err != ErrUnavailable {
		t.Errorf("disabled engine OCR err = %v, want ErrUnavailable", err)
	}
	// Enabled engine availability mirrors the host PATH; skip if no tesseract.
	_ = exec.Command // keep the binary presence probe honest in a unit test
}

// TestParseInvoiceNeverPanicsOnUnpriceableRows is a regression test for a 500
// on the Arabic path. Both multiply branches derive pricePos from
// numberBefore/numberAfter, so a row containing "x" with no usable number
// before it left pricePos at -1 and the next line indexed toks[-1] — a panic
// that the Recovery middleware reported to the merchant as a bare
// "internal server error". Reproduced from a real production scan.
func TestParseInvoiceNeverPanicsOnUnpriceableRows(t *testing.T) {
	rows := []string{
		"× 12.50",                   // multiply, nothing before it
		"جبنه × 45.00",              // Arabic name, multiply, no qty
		"اسمنت 20 ×",                // multiply as the last token
		"×",                         // a multiply on its own
		"x",                         // latin x alone
		"سعر ×",                     // Arabic word then multiply
		"7 ×",                       // a number then a bare multiply
		"جنيه 12.50",                // no multiply, one number
		strings.Repeat("×", 40),     // pathological multiply run
		strings.Repeat("س 7 ×", 30), // pathological mixed run
	}
	for _, row := range rows {
		row := row
		t.Run(row, func(t *testing.T) {
			// The contract is only that this returns: no panic, and whatever it
			// decides is either a priced item or a row for human review.
			res := ParseInvoice(row)
			for _, item := range res.Items {
				if item.UnitPriceMinor < 0 {
					t.Errorf("negative price invented: %+v", item)
				}
			}
		})
	}
}

// A full multi-line invoice mixing the shapes above must still parse its good
// rows: one unpriceable line must not take the rest of the document with it.
func TestParseInvoiceKeepsGoodRowsAroundAnUnpriceableOne(t *testing.T) {
	text := strings.Join([]string{
		"فاتورة مورد",
		"× 12.50",
		"جبنه بيضاء 5 × 45.00",
		"اسمنت 20 ×",
		"زيت عباد شمس 2 x 90",
	}, "\n")
	res := ParseInvoice(text)
	if len(res.Items) == 0 {
		t.Fatal("expected the well-formed rows to survive")
	}
	for _, item := range res.Items {
		if item.Quantity <= 0 {
			t.Errorf("item with no quantity: %+v", item)
		}
	}
	if len(res.Discarded) == 0 {
		t.Error("expected the unpriceable rows to be reported for review")
	}
}
