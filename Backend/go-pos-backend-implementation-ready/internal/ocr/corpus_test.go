package ocr

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestTemplateCorpusValidatesImages treats OCR-IMG-TEMPLETE as the reference
// invoice corpus for the Arabic OCR pipeline. The images are the live fixtures
// a deployment renders with tesseract; this test keeps the corpus present and
// well-formed (non-empty, decodable by content magic) so vendor-shipped samples
// cannot silently rot the OCR path.
func TestTemplateCorpusValidatesImages(t *testing.T) {
	const corpus = "../../OCR-IMG-TEMPLETE"
	entries, err := os.ReadDir(corpus)
	if err != nil {
		t.Skipf("OCR template corpus not present (%v); skipping", err)
	}
	if len(entries) < 10 {
		t.Fatalf("OCR template corpus must hold at least 10 sample invoices, got %d", len(entries))
	}
	imgs := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		imgs++
		data, err := os.ReadFile(filepath.Join(corpus, e.Name()))
		if err != nil || len(data) < 16 {
			t.Errorf("corpus file %s: unreadable or empty (%v)", e.Name(), err)
			continue
		}
		if !isKnownImage(data) {
			t.Errorf("corpus file %s: unrecognized image magic", e.Name())
		}
	}
	if imgs < 10 {
		t.Fatalf("corpus contains %d image files, want >= 10", imgs)
	}
}

func isKnownImage(b []byte) bool {
	switch {
	case len(b) >= 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff: // JPEG
		return true
	case len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}): // PNG
		return true
	case len(b) >= 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")): // WebP
		return true
	}
	return false
}

// TestParseInvoiceTemplateLayouts drives the parser over the row layouts
// present on the corpus invoices (tax-invoice ribbons, multi-line grocery and
// food-supplier rows, Arabic-Indic digits, decimal commas).
func TestParseInvoiceTemplateLayouts(t *testing.T) {
	cases := map[string]struct {
		text       string
		wantItems  int
		wantName   string
		wantQty    float64
		wantPrice  int64
		wantScore  int
		wantDiscrd int
		wantLast   string
	}{
		"grocery invoice": {
			text:      "فاتورة بيع مواد غذائية\nسكر 10 × 25.50 = 255.00\nزيت عباد شمس 4 كجم 65.00",
			wantItems: 2, wantName: "سكر", wantQty: 10, wantPrice: 2550, wantScore: 100, wantDiscrd: 1,
			wantLast: "زيت عباد شمس",
		},
		"tax invoice arabic digits": {
			text:      "نموذج فاتورة ضريبية\nكوكاكولا ٥٠٠مل × ٦ = ١٢٫٥٠\nالمورد: شركة الإسكندرية",
			wantItems: 1, wantName: "كوكاكولا", wantQty: 6, wantPrice: 1250, wantScore: 100, wantDiscrd: 2,
		},
		"supplier invoice padded row": {
			text:      "فاتورة شركات عصري\nجبنه بيضاء 5 × 45.00 = 225.00\nماء معدني 48 قطعه 3.50",
			wantItems: 2, wantName: "جبنه بيضاء", wantQty: 5, wantPrice: 4500, wantScore: 100, wantDiscrd: 1,
			wantLast: "ماء معدني",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			res := ParseInvoice(c.text)
			if len(res.Items) != c.wantItems {
				t.Fatalf("items = %d, want %d (%+v)", len(res.Items), c.wantItems, res.Items)
			}
			last := res.Items[len(res.Items)-1]
			if c.wantName != "" && res.Items[0].Name != c.wantName {
				t.Errorf("first item name = %q, want %q", res.Items[0].Name, c.wantName)
			}
			if c.wantLast != "" && last.Name != c.wantLast {
				t.Errorf("last item name = %q, want %q", last.Name, c.wantLast)
			}
			if c.wantQty > 0 && res.Items[0].Quantity != c.wantQty {
				t.Errorf("first item quantity = %v, want %v", res.Items[0].Quantity, c.wantQty)
			}
			if c.wantPrice > 0 && res.Items[0].UnitPriceMinor != c.wantPrice {
				t.Errorf("first item unit_price_minor = %d, want %d", res.Items[0].UnitPriceMinor, c.wantPrice)
			}
			if c.wantScore > 0 && res.Items[0].Score < c.wantScore {
				t.Errorf("first item score = %d, want >= %d", res.Items[0].Score, c.wantScore)
			}
			if len(res.Discarded) != c.wantDiscrd {
				t.Errorf("discarded = %d, want %d: %q", len(res.Discarded), c.wantDiscrd, res.Discarded)
			}
		})
	}
}
