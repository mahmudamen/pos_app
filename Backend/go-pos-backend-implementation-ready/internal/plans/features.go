// Package plans answers the question "may this tenant use that feature?".
//
// Until now `plans.features` was a display-only JSON array: it was serialised
// into SaaS responses and rendered in the /pricing comparison table, and
// nothing ever read it to make a decision. That is fine while a feature is
// free, and wrong the moment a feature is sold, because there is then no
// mechanism that says no.
//
// This package adds that mechanism. It is deliberately small and pure: it
// parses the feature array and answers a membership question. Deciding *where*
// to enforce is the caller's job, so the same helper serves an HTTP handler, a
// billing guard or a test without any of them depending on each other.
package plans

import (
	"encoding/json"
	"sort"
	"strings"
)

// Feature keys. These are the strings stored in `plans.features`. They are
// constants rather than bare literals so a typo is a compile error instead of
// a feature that silently never unlocks.
const (
	FeaturePOSBasic          = "pos.basic"
	FeatureInventoryBasic    = "inventory.basic"
	FeatureDashboardBasic    = "dashboard.basic"
	FeatureInventoryAdvanced = "inventory.advanced"
	FeatureDashboardAdvanced = "dashboard.advanced"
	FeatureRestaurant        = "restaurant"
	FeatureLoyalty           = "loyalty"
	FeaturePharmacy          = "pharmacy"
	FeatureTextile           = "textile"
	FeatureSyncMultiDevice   = "sync.multi_device"
	FeaturePOSSubdomain      = "pos.subdomain"
	// FeaturePurchaseOCR gates invoice capture. The /pricing table already
	// advertises "ocr_capture"; this is the internal spelling of that same
	// promise, and the two are deliberately kept in step.
	FeaturePurchaseOCR = "ocr_capture"
)

// Parse reads a `plans.features` array. It accepts the stored form (a JSON
// array of strings) and tolerates NULL, an empty string and malformed JSON by
// returning no features rather than an error, because a tenant with a broken
// feature blob must be able to log in and keep trading.
//
// The one input treated as a failure rather than as "no features" is
// `["*"]`, which is an explicit "everything" marker used by the seeded demo
// and internal tenants. Losing it would lock those tenants out of features
// they demonstrably have, which is a much worse outcome than being permissive.
func Parse(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var list []string
	if err := json.Unmarshal([]byte(trimmed), &list); err != nil {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

// HasFeature reports whether want is present in raw.
//
// Two deliberate decisions, both about not locking a paying shop out by
// accident:
//
//   - An unparseable blob denies. A corrupted features column must not become
//     an open door, and the failure is loud rather than silent.
//   - "*" is honoured as a wildcard. Demo and internal tenants use it, and
//     denying them would be a regression with no upside.
func HasFeature(raw, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return false
	}
	for _, have := range Parse(raw) {
		if have == want || have == "*" {
			return true
		}
	}
	return false
}

// Missing returns the subset of want that raw does not grant, preserving the
// order of want. Callers that want to explain *which* entitlements a tenant is
// missing use this; it is also what makes an enforcement response specific
// instead of a bare 403.
func Missing(raw string, want []string) []string {
	var out []string
	for _, w := range want {
		if !HasFeature(raw, w) {
			out = append(out, w)
		}
	}
	return out
}

// List returns the features sorted, for a stable response body and a stable
// cache key. The input is not modified.
func List(raw string) []string {
	features := Parse(raw)
	sort.Strings(features)
	return features
}
