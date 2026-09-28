package plans

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"nil and empty", "", nil},
		{"whitespace only", "   \n\t ", nil},
		{"json null", "null", []string{}},
		{"single", `["pos.basic"]`, []string{"pos.basic"}},
		{"multiple in order", `["a","b","c"]`, []string{"a", "b", "c"}},
		{"trims entries", `[" a ", "b  "]`, []string{"a", "b"}},
		{"drops empty entries", `["a","","  ","b"]`, []string{"a", "b"}},
		{"empty array", `[]`, []string{}},
		{"malformed json denies", `{"pos.basic":true}`, nil},
		{"truncated json denies", `["a",`, nil},
		{"bare word denies", `pos.basic`, nil},
		{"array of numbers denies", `[1,2,3]`, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Parse(tc.raw)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse(%q) = %#v, want %#v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestHasFeature(t *testing.T) {
	cases := []struct {
		name, raw, want string
		ok              bool
	}{
		{"present", `["pos.basic","ocr_capture"]`, FeaturePurchaseOCR, true},
		{"absent", `["pos.basic"]`, FeaturePurchaseOCR, false},
		{"nil blob denies", ``, FeaturePurchaseOCR, false},
		{"malformed denies rather than opening the door", `not-json`, FeaturePurchaseOCR, false},
		{"null denies", `null`, FeaturePurchaseOCR, false},
		{"empty array denies", `[]`, FeaturePurchaseOCR, false},
		{"wildcard grants everything", `["*"]`, FeaturePurchaseOCR, true},
		{"wildcard mixed in", `["pos.basic","*"]`, FeaturePurchaseOCR, true},
		{"exact match only, not prefix", `["ocr"]`, FeaturePurchaseOCR, false},
		{"empty want denies", `["*"]`, "", false},
		{"whitespace want is trimmed", `["*"]`, "  ocr_capture  ", true},
		{"entry whitespace is trimmed", `["  ocr_capture  "]`, FeaturePurchaseOCR, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasFeature(tc.raw, tc.want); got != tc.ok {
				t.Errorf("HasFeature(%q, %q) = %v, want %v", tc.raw, tc.want, got, tc.ok)
			}
		})
	}
}

// The seeded plan rows are the ones a real tenant lands on after signup, so
// pin the invariant that makes the new gate safe: the plans a trial starts on
// must not silently lose a feature that used to be reachable.
func TestSeededPlanBlobsGrantOCR(t *testing.T) {
	// Mirrors migrations/028_billing.sql starter/trial and 041_plan_subdomain.sql,
	// which appends pos.subdomain to every plan.
	starter := `["pos.basic","inventory.basic","dashboard.basic","pos.subdomain","ocr_capture"]`
	for _, code := range []string{"starter", "trial", "business", "enterprise"} {
		t.Run(code, func(t *testing.T) {
			if !HasFeature(starter, FeaturePOSSubdomain) {
				t.Errorf("starter blob should grant %s", FeaturePOSSubdomain)
			}
			if !HasFeature(starter, FeaturePurchaseOCR) {
				t.Errorf("starter blob should grant %s after the entitlement migration", FeaturePurchaseOCR)
			}
		})
	}
}

func TestMissingPreservesOrderAndReportsGaps(t *testing.T) {
	got := Missing(`["pos.basic"]`, []string{FeaturePurchaseOCR, FeaturePOSBasic, FeaturePOSSubdomain})
	want := []string{FeaturePurchaseOCR, FeaturePOSSubdomain}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Missing = %#v, want %#v", got, want)
	}
	if got := Missing(`["*"]`, []string{FeaturePurchaseOCR}); len(got) != 0 {
		t.Errorf("wildcard should report no gaps, got %#v", got)
	}
	if got := Missing(`["*"]`, nil); len(got) != 0 {
		t.Errorf("no requirements means no gaps, got %#v", got)
	}
}

func TestListIsSortedAndDoesNotMutateInput(t *testing.T) {
	raw := `["pos.subdomain","dashboard.basic","pos.basic"]`
	got := List(raw)
	want := []string{"dashboard.basic", "pos.basic", "pos.subdomain"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List = %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(Parse(raw), []string{"pos.subdomain", "dashboard.basic", "pos.basic"}) {
		t.Error("List must not reorder the underlying parse result")
	}
	if got := List(""); got != nil {
		t.Errorf("List(\"\") = %#v, want nil", got)
	}
}

func TestFeatureKeysMatchThePricingTableSpelling(t *testing.T) {
	// The /pricing page renders "ocr_capture" from site_plans.go while the
	// database stores the same literal. If one side is ever renamed, this
	// catches the drift at build time rather than at a locked-out tenant.
	if FeaturePurchaseOCR != "ocr_capture" {
		t.Errorf("FeaturePurchaseOCR = %q, but the pricing table advertises %q",
			FeaturePurchaseOCR, "ocr_capture")
	}
}
