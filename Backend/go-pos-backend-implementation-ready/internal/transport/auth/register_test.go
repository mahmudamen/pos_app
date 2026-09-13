package auth

import (
	"strings"
	"testing"
)

func TestValidBusinessTypesIncludeSignupVerticals(t *testing.T) {
	for _, vt := range []string{"coffee_shop", "restaurant", "retail", "book_store", "mobile_shop", "computer_shop", "grocery"} {
		if !validBusinessTypes[vt] {
			t.Fatalf("expected %q to be a valid signup business type", vt)
		}
	}
}

func TestSeedCatalogsCoversEverySignupVertical(t *testing.T) {
	catalogs := seedCatalogs()
	for vt := range validBusinessTypes {
		vertical, ok := catalogs[vt]
		if !ok {
			t.Fatalf("business type %q has no demo catalog", vt)
		}
		if len(vertical.categories) == 0 {
			t.Fatalf("business type %q has no categories", vt)
		}
		if len(vertical.products) == 0 {
			t.Fatalf("business type %q has no products", vt)
		}
		for _, p := range vertical.products {
			if p.name == "" || p.sku == "" || p.barcode == "" || p.priceMinor <= 0 || p.stock < 0 {
				t.Fatalf("%s %q: incomplete demo product", vt, p.name)
			}
			if !strings.HasPrefix(p.imageURL, "photo-") {
				t.Fatalf("%s %q: image must be an Unsplash photo id, got %q", vt, p.name, p.imageURL)
			}
			if p.description == "" {
				t.Fatalf("%s %q: missing product info", vt, p.name)
			}
			if len(p.barcode) != 13 {
				t.Fatalf("%s %q: barcode %q is not EAN-13", vt, p.name, p.barcode)
			}
			for _, d := range p.barcode {
				if d < '0' || d > '9' {
					t.Fatalf("%s %q: barcode %q has non-digit", vt, p.name, p.barcode)
				}
			}
		}
	}
}

func TestSlugFor(t *testing.T) {
	slug := slugFor("  Café Nile  ")
	if !strings.HasPrefix(slug, "caf-nile-") {
		t.Fatalf("slug %q should start with caf-nile-", slug)
	}
	if len(slug) != len("caf-nile-")+8 {
		t.Fatalf("slug %q should have an 8-char suffix", slug)
	}
	if got := slugFor(""); !strings.HasPrefix(got, "store-") {
		t.Fatalf("empty name slug %q should default to store-", got)
	}
}

func TestDefaults(t *testing.T) {
	cc, cur, lang := defaults("", "", "")
	if cc != "EG" || cur != "EGP" || lang != "ar" {
		t.Fatalf("expected EG/EGP/ar defaults, got %s/%s/%s", cc, cur, lang)
	}
	cc, cur, lang = defaults("US", "USD", "en")
	if cc != "US" || cur != "USD" || lang != "en" {
		t.Fatalf("expected provided values to win, got %s/%s/%s", cc, cur, lang)
	}
}
