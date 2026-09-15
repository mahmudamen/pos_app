// Package pricing refreshes demo tenant prices from the bundled local-Egyptian
// catalog. The catalog rows use real GS1-Egypt (622xxx) EAN-13 barcodes so the
// job can match seeded products on barcode alone; it only touches tenants whose
// catalog was produced by the signup/demo seeder (tenants.is_demo_seeded).
package pricing

import "math/rand"

// Item is a single bundled local-product price entry.
type Item struct {
	Barcode    string
	Name       string
	Category   string
	Unit       string
	PriceMinor int64
	CostMinor  int64
}

// EgyptCatalog is the bundled local-market catalog. It must stay in sync with
// the rows seeded by migration 030 and with the demo grocery vertical in the
// auth seeder, so that barcode matching always finds something to refresh.
func EgyptCatalog() []Item {
	return []Item{
		{"6221010000017", "Juhayna Fresh Milk 1L", "dairy", "liter", 4200, 3800},
		{"6221010000024", "Juhayna Plain Yogurt 500g", "dairy", "piece", 2600, 2300},
		{"6221010000031", "Domty Cheddar Cheese 100g", "dairy", "piece", 5500, 5000},
		{"6221010000048", "Panda Cheese Triangles", "dairy", "piece", 3400, 3000},
		{"6221010000062", "Coca-Cola 1L", "beverages", "liter", 2400, 2000},
		{"6221010000086", "Schweppes Lemon 1L", "beverages", "liter", 2600, 2200},
		{"6221010000093", "Mineral Water 1.5L", "beverages", "piece", 1500, 1100},
		{"6221010000109", "Syrup Mango Juice 1L", "beverages", "liter", 3800, 3200},
		{"6221010000116", "Lipton Yellow Label Tea 50g", "pantry", "piece", 6500, 5800},
		{"6221010000123", "Nescafe Classic 100g", "pantry", "piece", 14500, 13200},
		{"6221010000147", "El-Mashreq Sugar 1kg", "pantry", "kg", 3100, 2900},
		{"6221010000154", "El-Gomhoria Rice 1kg", "pantry", "kg", 4200, 3800},
		{"6221010000178", "Al-Shark Pasta 400g", "pantry", "piece", 1800, 1500},
		{"6221010000185", "Mazola Corn Oil 1L", "pantry", "liter", 13500, 12500},
		{"6221010000208", "El-Nasr Table Salt 1kg", "pantry", "kg", 800, 500},
		{"6221010000215", "Brown Lentils 1kg", "pantry", "kg", 5800, 5200},
		{"6221010000239", "Chipsy Chips 80g", "snacks", "piece", 1600, 1300},
		{"6221010000246", "Lays Chips 80g", "snacks", "piece", 1700, 1400},
		{"6221010000253", "Oman Chips King", "snacks", "piece", 1200, 950},
		{"6221010000307", "Stella Biscuits 400g", "snacks", "piece", 3500, 3000},
	}
}

// VariedPrice nudges a minor-unit price by a small random percentage so the
// refresh run does not look like a perfect copy. pct is the maximum swing
// (1..10); 0 means no variation. The result never goes below 0.
func VariedPrice(minor int64, pct int, rng *rand.Rand) int64 {
	if minor <= 0 {
		return 0
	}
	if pct <= 0 || rng == nil {
		return minor
	}
	delta := minor * int64(pct) / 100
	if delta <= 0 {
		delta = 1
	}
	if rng.Intn(2) == 0 {
		minor += delta
	} else {
		minor -= delta
	}
	if minor < 0 {
		return 0
	}
	return minor
}
