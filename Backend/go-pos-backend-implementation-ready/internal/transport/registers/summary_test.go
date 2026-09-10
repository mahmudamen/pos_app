package registers

import "testing"

func TestExpectedCashMinor(t *testing.T) {
	tests := []struct {
		name        string
		opening     int64
		cashSales   int64
		wantOverall int64
	}{
		{"empty drawer sale", 5000, 0, 5000},
		{"float plus takings", 10000, 2750, 12750},
		{"no float", 0, 1234, 1234},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expectedCashMinor(tt.opening, tt.cashSales); got != tt.wantOverall {
				t.Fatalf("expected %d, got %d", tt.wantOverall, got)
			}
		})
	}
}

func TestBalanceDifference(t *testing.T) {
	tests := []struct {
		name     string
		closing  int64
		expected int64
		wantDiff int64
	}{
		{"balanced", 12750, 12750, 0},
		{"shortage", 12000, 12750, -750},
		{"overage", 13000, 12750, 250},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := balanceDifference(tt.closing, tt.expected); got != tt.wantDiff {
				t.Fatalf("expected %d, got %d", tt.wantDiff, got)
			}
		})
	}
}

func TestBuildSummary(t *testing.T) {
	totals := accountTotals{
		salesCount:    3,
		subtotalMinor: 900,
		discountMinor: 50,
		taxMinor:      20,
		totalMinor:    870,
		cashMinor:     600,
		cardMinor:     200,
		mobileMinor:   70,
	}
	summary := buildSummary(totals, 1000)
	if summary.SalesCount != 3 {
		t.Fatalf("sales count = %d", summary.SalesCount)
	}
	if summary.SubtotalMinor != 900 || summary.DiscountMinor != 50 || summary.TaxMinor != 20 {
		t.Fatalf("wrong sale aggregates: %+v", summary)
	}
	if summary.TotalMinor != 870 {
		t.Fatalf("total = %d", summary.TotalMinor)
	}
	if summary.CashMinor != 600 || summary.CardMinor != 200 || summary.MobileMinor != 70 {
		t.Fatalf("wrong tender split: %+v", summary)
	}
	if summary.ExpectedCashMinor != 1600 {
		t.Fatalf("expected cash = %d (want opening 1000 + cash 600)", summary.ExpectedCashMinor)
	}
}
