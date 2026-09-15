package billing

import (
	"testing"
	"time"
)

func validPlan() Plan {
	return Plan{
		Code: "business", Name: "Business", PriceMinor: 29900, Currency: "EGP",
		BillingPeriod: PeriodMonthly, Features: []string{"pos.basic"}, MaxUsers: 10, MaxProducts: 1000,
	}
}

func TestValidBillingPeriod(t *testing.T) {
	if !ValidBillingPeriod(PeriodMonthly) || !ValidBillingPeriod(PeriodYearly) {
		t.Fatal("monthly/yearly should be valid")
	}
	for _, p := range []string{"", "weekly", "Monthly", "annual"} {
		if ValidBillingPeriod(p) {
			t.Fatalf("%q should be invalid", p)
		}
	}
}

func TestPlanValidate(t *testing.T) {
	if err := validPlan().Validate(); err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
	cases := []struct {
		name   string
		mutate func(*Plan)
	}{
		{"empty code", func(p *Plan) { p.Code = "" }},
		{"uppercase code", func(p *Plan) { p.Code = "Business" }},
		{"spaced code", func(p *Plan) { p.Code = "biz ness" }},
		{"code too long", func(p *Plan) { p.Code = string(make([]byte, 33)) }},
		{"empty name", func(p *Plan) { p.Name = "" }},
		{"negative price", func(p *Plan) { p.PriceMinor = -1 }},
		{"short currency", func(p *Plan) { p.Currency = "EG" }},
		{"bad period", func(p *Plan) { p.BillingPeriod = "weekly" }},
		{"negative user limit", func(p *Plan) { p.MaxUsers = -1 }},
		{"negative product limit", func(p *Plan) { p.MaxProducts = -1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := validPlan()
			tc.mutate(&p)
			if err := p.Validate(); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
	// 0 limits mean unlimited and must be accepted.
	p := validPlan()
	p.MaxUsers, p.MaxProducts = 0, 0
	if err := p.Validate(); err != nil {
		t.Fatalf("unlimited plan rejected: %v", err)
	}
}

func TestMonthlyEquivalentMinor(t *testing.T) {
	if got := MonthlyEquivalentMinor(29900, PeriodMonthly); got != 29900 {
		t.Fatalf("monthly = %d, want 29900", got)
	}
	if got := MonthlyEquivalentMinor(120000, PeriodYearly); got != 10000 {
		t.Fatalf("yearly/12 = %d, want 10000", got)
	}
	if got := MonthlyEquivalentMinor(100000, PeriodYearly); got != 8333 {
		t.Fatalf("rounded yearly/12 = %d, want 8333", got)
	}
}

func TestProrateMinor(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0) // 31 days

	if got := ProrateMinor(3100, start, end, start.AddDate(0, 0, -1)); got != 3100 {
		t.Fatalf("before start should be full price, got %d", got)
	}
	if got := ProrateMinor(3100, start, end, start); got != 3100 {
		t.Fatalf("at start should be full price, got %d", got)
	}
	if got := ProrateMinor(3100, start, end, end); got != 0 {
		t.Fatalf("at end should be zero, got %d", got)
	}
	if got := ProrateMinor(3100, start, end, end.AddDate(0, 0, 5)); got != 0 {
		t.Fatalf("after end should be zero, got %d", got)
	}
	// Exactly halfway through 31 days -> half of 3100 = 1550 (15.5 days rounds).
	mid := start.AddDate(0, 0, 15)
	if got := ProrateMinor(3100, start, end, mid); got != 1600 {
		t.Fatalf("mid-period proration = %d, want 1600", got)
	}
	// A degenerate period falls back to the full price.
	if got := ProrateMinor(500, start, start, start); got != 500 {
		t.Fatalf("degenerate period = %d, want 500", got)
	}
}

func TestPeriodEnd(t *testing.T) {
	start := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	if got := PeriodEnd(start, PeriodMonthly); !got.Equal(time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)) {
		// AddDate(0,1,0) on Jan 31 normalizes to Mar 3 (Feb 31 -> Mar 3).
		t.Fatalf("monthly period end = %s", got)
	}
	if got := PeriodEnd(start, PeriodYearly); !got.Equal(time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("yearly period end = %s", got)
	}
}
