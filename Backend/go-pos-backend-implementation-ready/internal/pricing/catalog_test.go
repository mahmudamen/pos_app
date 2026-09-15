package pricing

import (
	"math/rand"
	"testing"
)

func TestEgyptCatalogWellFormed(t *testing.T) {
	items := EgyptCatalog()
	if len(items) < 15 {
		t.Fatalf("bundled catalog too small: %d items", len(items))
	}
	seen := map[string]bool{}
	for _, it := range items {
		if len(it.Barcode) != 13 {
			t.Fatalf("%s: barcode must be EAN-13, got %q", it.Name, it.Barcode)
		}
		for _, d := range it.Barcode {
			if d < '0' || d > '9' {
				t.Fatalf("%s: barcode %q has non-digit", it.Name, it.Barcode)
			}
		}
		if seen[it.Barcode] {
			t.Fatalf("duplicate barcode %s", it.Barcode)
		}
		seen[it.Barcode] = true
		if it.Name == "" || it.Unit == "" || it.PriceMinor <= 0 || it.CostMinor <= 0 {
			t.Fatalf("incomplete catalog item %v", it)
		}
	}
}

func TestVariedPrice(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	if got := VariedPrice(1000, 0, rng); got != 1000 {
		t.Fatalf("pct 0 must be identity, got %d", got)
	}
	if got := VariedPrice(0, 10, rng); got != 0 {
		t.Fatalf("zero price must stay zero, got %d", got)
	}
	if got := VariedPrice(1000, 10, nil); got != 1000 {
		t.Fatalf("nil rng must be identity, got %d", got)
	}
	for i := 0; i < 200; i++ {
		got := VariedPrice(10_000_000, 5, rng)
		if got < 9_500_000 || got > 10_500_000 {
			t.Fatalf("varied price %d out of 5%% band", got)
		}
	}
}
