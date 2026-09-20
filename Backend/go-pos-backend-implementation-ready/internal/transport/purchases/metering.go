package purchases

// Metering is the pure heart of OCR metering (migration 036_ocr_usage). It
// turns three rolling-window counters (day / week / month, `ocr_usage`) plus
// the tenant's scan-borrowing credit balance (`tenants.ocr_credits_remaining`)
// into one unit-tested "may this tenant scan one more invoice image right now?"
// answer. No DB, no engine, no money — just a decision matrix, so the ocrScan
// handler stays thin and the policy is provable.

import "errors"

var (
	// ErrOCRWindowLimit is returned by ShouldAdmit when the tenant is past the
	// limit on every active window AND has no scan-borrowing credits left; the
	// client should tell the shop to top up OCR points.
	ErrOCRWindowLimit = errors.New("ocr_window_limit_reached")

	// ErrOCRNoEngine is returned when metering is demanded but no limits are
	// configured at all (limits == Unlimited) — that is the plan being slightly
	// misconfigured, never a spend decision.
	ErrOCRNoEngine = errors.New("ocr_metering_unconfigured")
)

// OCRWindows are limits for the three rolling windows the contract meters.
// Zero on a window means "unlimited" for that window.
type OCRWindows struct {
	Day   int64 `json:"day"`
	Week  int64 `json:"week"`
	Month int64 `json:"month"`
}

// Unlimited is the zero value: every window unlimited. Metering is opt-in —
// a deployment that never sets OCR_DAY_LIMIT etc. keeps scanning silently.
var Unlimited OCRWindows

// MeterState is the per-tenant ledger snapshot the metering decision runs on.
type MeterState struct {
	Used      OCRWindows `json:"used"`
	Limits    OCRWindows `json:"limits"`
	Credits   int64      `json:"credits_remaining"`
	HaveLimit bool
}

// MeterState.Zero returns the default empty ledger.
func (m MeterState) Zero() MeterState { return MeterState{} }

// ShouldAdmit answers "may the tenant scan one more invoice image now?".
//
// A scan is ALWAYS admitted unless it would push the tenant past ANY active
// window (window limit reached) AND the credit balance is empty. When a window
// would be exceeded but credits are left, one credit is borrowed and the scan
// is still admitted (the shop top-up flow). Windows whose limit is 0 are
// inactive and never counted.
//
// metered contains the used counters; limits the configured caps. meteringEn
// gates enforcement: when the config has no limits at all (OCR limits all
// zero), metering stays on but enforcement cannot reject — the handler still
// bumps the window counters so a later limit setting has history.
func ShouldAdmit(metered, limits OCRWindows, credits int64) (admit bool, borrow bool) {
	over := 0
	for _, u := range []struct {
		used, limit int64
	}{{metered.Day, limits.Day}, {metered.Week, limits.Week}, {metered.Month, limits.Month}} {
		if u.limit > 0 && u.used >= u.limit {
			over++
		}
	}
	if over == 0 {
		return true, false
	}
	if credits > 0 {
		return true, true
	}
	return false, false
}
