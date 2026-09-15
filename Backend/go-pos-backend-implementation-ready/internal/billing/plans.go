// Package billing holds the platform billing domain: the plan catalog, the
// subscription state machine, invoice lifecycle rules, and the payment gateway
// abstraction. Everything here is pure (no database, no HTTP) so it can be
// unit-tested in isolation; the transport layer only wires it to Postgres.
package billing

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Billing periods a plan can be priced at.
const (
	PeriodMonthly = "monthly"
	PeriodYearly  = "yearly"
)

// Plan is a row of the plans catalog.
type Plan struct {
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	PriceMinor    int64    `json:"price_minor"`
	Currency      string   `json:"currency"`
	BillingPeriod string   `json:"billing_period"`
	Features      []string `json:"features"`
	MaxUsers      int      `json:"max_users"`
	MaxProducts   int      `json:"max_products"`
	IsActive      bool     `json:"is_active"`
}

// ValidBillingPeriod reports whether p is a supported period.
func ValidBillingPeriod(p string) bool {
	return p == PeriodMonthly || p == PeriodYearly
}

// Validate checks the invariants the database also enforces, so bad input is
// rejected with a clear message before it ever reaches Postgres.
func (p Plan) Validate() error {
	if strings.TrimSpace(p.Code) == "" {
		return fmt.Errorf("code is required")
	}
	if len(p.Code) > 32 {
		return fmt.Errorf("code must be at most 32 characters")
	}
	for _, r := range p.Code {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !valid {
			return fmt.Errorf("code may only contain lowercase letters, digits, '-' and '_'")
		}
	}
	if strings.TrimSpace(p.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if len(p.Name) > 100 {
		return fmt.Errorf("name must be at most 100 characters")
	}
	if p.PriceMinor < 0 {
		return fmt.Errorf("price must not be negative")
	}
	if len(p.Currency) != 3 {
		return fmt.Errorf("currency must be a 3-letter code")
	}
	if !ValidBillingPeriod(p.BillingPeriod) {
		return fmt.Errorf("billing period must be %q or %q", PeriodMonthly, PeriodYearly)
	}
	if p.MaxUsers < 0 || p.MaxProducts < 0 {
		return fmt.Errorf("limits must not be negative")
	}
	return nil
}

// MonthlyEquivalentMinor normalizes a plan price to a monthly figure so plans
// on different billing periods can be compared. Yearly plans are divided by 12.
func MonthlyEquivalentMinor(priceMinor int64, period string) int64 {
	if period == PeriodYearly {
		return int64(math.Round(float64(priceMinor) / 12))
	}
	return priceMinor
}

// ProrateMinor returns the value of the *remaining* part of a billing period
// starting at `at`. It is used when a plan changes mid-cycle: the credit (or
// charge) is the fraction of the period still unused. Full price before the
// period starts, zero at or after the period ends.
func ProrateMinor(priceMinor int64, periodStart, periodEnd, at time.Time) int64 {
	if !periodEnd.After(periodStart) {
		return priceMinor
	}
	if !at.After(periodStart) {
		return priceMinor
	}
	if !at.Before(periodEnd) {
		return 0
	}
	total := periodEnd.Sub(periodStart)
	remaining := periodEnd.Sub(at)
	return int64(math.Round(float64(priceMinor) * float64(remaining) / float64(total)))
}

// PeriodEnd computes the end of a billing period that starts at `start` for
// the given period. Monthly = one calendar month, yearly = one calendar year.
func PeriodEnd(start time.Time, period string) time.Time {
	if period == PeriodYearly {
		return start.AddDate(1, 0, 0)
	}
	return start.AddDate(0, 1, 0)
}
