package posgo

import (
	"strings"
	"testing"
	"time"
)

func TestFormatMinorRendersCurrencyFromMinorUnits(t *testing.T) {
	cases := []struct {
		minor    int64
		currency string
		want     string
	}{
		{0, "EGP", "E£0.00"},
		{5, "EGP", "E£0.05"},
		{50, "EGP", "E£0.50"},
		{1250, "EGP", "E£12.50"},
		{97860, "EGP", "E£978.60"},
		{986030, "EGP", "E£9,860.30"},
		{123456789, "EGP", "E£1,234,567.89"},
		{-500, "EGP", "-E£5.00"},
		{100, "EGP", "E£1.00"},
		// A 1000s boundary is where minor-unit formatting usually breaks: the
		// integer and the fraction must be computed separately.
		{100000, "EGP", "E£1,000.00"},
		{100000000, "EGP", "E£1,000,000.00"},
		{999, "EGP", "E£9.99"},
		{1000, "EGP", "E£10.00"},
		// Other currencies keep their own glyph; unknown ones keep the code.
		{1250, "USD", "$12.50"},
		{1250, "eur", "€12.50"},
		{1250, "ZZZ", "ZZZ 12.50"},
		// An empty currency falls back to the platform's supported currency
		// rather than rendering a bare number.
		{1250, "", "E£12.50"},
		{1250, "   ", "E£12.50"},
	}
	for _, tc := range cases {
		if got := FormatMinor(tc.minor, tc.currency); got != tc.want {
			t.Errorf("FormatMinor(%d, %q) = %q, want %q", tc.minor, tc.currency, got, tc.want)
		}
	}
}

func TestFormatMinorAlwaysShowsTwoDecimals(t *testing.T) {
	for minor := int64(0); minor < 200; minor++ {
		got := FormatMinor(minor, "EGP")
		dot := strings.LastIndex(got, ".")
		if dot < 0 || len(got)-dot != 3 {
			t.Fatalf("FormatMinor(%d) = %q, want exactly 2 decimals", minor, got)
		}
	}
}

func TestFormatCountCompactsLargeNumbers(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{4329, "4329"},
		{9999, "9999"},
		// Below 10k the exact digits are no wider than the compacted form
		// ("7735" and "7.7k" are both four cells), so precision wins. The
		// threshold matches the Flutter control panel so both surfaces read
		// the same number the same way.
		{7735, "7735"},
		{10000, "10.0k"},
		{1500000, "1.5M"},
		{4329000000, "4329.0M"},
	}
	for _, tc := range cases {
		if got := FormatCount(tc.in); got != tc.want {
			t.Errorf("FormatCount(%d) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestStatusPillNormalisesStatuses(t *testing.T) {
	cases := map[string]string{
		"active":    "ACTIVE",
		"suspended": "SUSPENDED",
		"disabled":  "STOPPED",
		"closed":    "CLOSED",
		"trial":     "TRIAL",
		"":          "UNKNOWN",
		"weird":     "WEIRD",
	}
	for in, want := range cases {
		if got := StatusPill(in); got != want {
			t.Errorf("StatusPill(%q) = %q, want %q", in, got, want)
		}
	}
}

// The disabled pill must read STOPPED, not DISABLED: that is the word the
// control panel and the Flutter client both use for the same state, and an ops
// console that invents a third word trains operators to distrust it.
func TestStatusPillUsesProductWording(t *testing.T) {
	if got := StatusPill(StatusDisabled); got != "STOPPED" {
		t.Fatalf("disabled pill = %q, want STOPPED", got)
	}
}

func TestShortIDTrimsUUID(t *testing.T) {
	full := "f11c6eea-16f1-41c0-ad64-40ed40683528"
	if got := ShortID(full); got != "f11c6eea" {
		t.Fatalf("ShortID = %q", got)
	}
	if got := ShortID("nohyphens"); got != "nohyphens" {
		t.Fatalf("ShortID on a bare id = %q", got)
	}
	if got := ShortID(""); got != "" {
		t.Fatalf("ShortID on empty = %q", got)
	}
}

func TestFormatDateFallsBackOnBadInput(t *testing.T) {
	if got := FormatDate("2026-09-09T17:24:45Z"); len(got) != 10 || got[4] != '-' {
		t.Fatalf("FormatDate = %q", got)
	}
	if got := FormatDate(""); got != "-" {
		t.Fatalf("empty timestamp = %q, want -", got)
	}
	if got := FormatDate("not-a-date"); got != "not-a-date" {
		t.Fatalf("unparseable timestamp = %q, want the raw string", got)
	}
}

func TestRelativeAgeBuckets(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		ago  time.Duration
		want string
	}{
		{30 * time.Second, "just now"},
		{5 * time.Minute, "5m ago"},
		{3 * time.Hour, "3h ago"},
		{50 * time.Hour, "2d ago"},
	}
	for _, tc := range cases {
		ts := now.Add(-tc.ago).Format(time.RFC3339)
		if got := RelativeAge(ts, now); got != tc.want {
			t.Errorf("RelativeAge(-%s) = %q, want %q", tc.ago, got, tc.want)
		}
	}
	if got := RelativeAge("", now); got != "-" {
		t.Fatalf("empty = %q", got)
	}
	if got := RelativeAge("garbage", now); got != "-" {
		t.Fatalf("garbage = %q", got)
	}
}

func TestFiltersQueryOmitsUnsetFields(t *testing.T) {
	got := Filters{}.Query()
	for _, unwanted := range []string{"q=", "business_type=", "status=", "plan="} {
		if strings.Contains(got, unwanted) {
			t.Fatalf("zero Filters should omit %q, got %q", unwanted, got)
		}
	}
	// Pagination is always sent, and always clamped into the API's range.
	if !strings.Contains(got, "page=1") || !strings.Contains(got, "limit=25") {
		t.Fatalf("defaults not applied: %q", got)
	}
	if !strings.HasPrefix(got, "?") {
		t.Fatalf("query must start with ? so it can be concatenated: %q", got)
	}
}

func TestFiltersQueryClampsPagination(t *testing.T) {
	if got := (Filters{Page: -3}).Query(); !strings.Contains(got, "page=1") {
		t.Fatalf("negative page not clamped: %q", got)
	}
	if got := (Filters{Limit: 0}).Query(); !strings.Contains(got, "limit=25") {
		t.Fatalf("zero limit not defaulted: %q", got)
	}
	if got := (Filters{Limit: 5000}).Query(); !strings.Contains(got, "limit=25") {
		t.Fatalf("oversized limit not clamped: %q", got)
	}
	if got := (Filters{Limit: 100}).Query(); !strings.Contains(got, "limit=100") {
		t.Fatalf("limit 100 is in range and must survive: %q", got)
	}
}

func TestFiltersQueryTrimsSearch(t *testing.T) {
	if got := (Filters{Search: "   "}).Query(); strings.Contains(got, "q=") {
		t.Fatalf("whitespace search should be dropped: %q", got)
	}
	got := (Filters{Search: "  cafe  "}).Query()
	if !strings.Contains(got, "q=+cafe+") && !strings.Contains(got, "q=cafe") {
		t.Fatalf("search not sent: %q", got)
	}
}

func TestTotalPagesRoundsUpAndNeverZero(t *testing.T) {
	cases := []struct {
		total int64
		limit int
		want  int
	}{
		{0, 25, 1},
		{1, 25, 1},
		{25, 25, 1},
		{26, 25, 2},
		{4329, 25, 174},
		{4329, 0, 1},
		{-5, 25, 1},
	}
	for _, tc := range cases {
		if got := TotalPages(tc.total, tc.limit); got != tc.want {
			t.Errorf("TotalPages(%d, %d) = %d, want %d", tc.total, tc.limit, got, tc.want)
		}
	}
}

func TestTenantFeatureAndStatusHelpers(t *testing.T) {
	tenant := Tenant{
		Status:       StatusActive,
		PlanFeatures: []string{"pos.basic", "pos.subdomain", "ocr_capture"},
	}
	if !tenant.HasFeature("pos.subdomain") {
		t.Fatal("HasFeature should find a listed entitlement")
	}
	if tenant.HasFeature("billing.advanced") {
		t.Fatal("HasFeature must not match an absent entitlement")
	}
	if tenant.HasFeature("") {
		t.Fatal("HasFeature must not match the empty string")
	}
	if tenant.IsStopped() {
		t.Fatal("an active tenant is not stopped")
	}
	if !(Tenant{Status: StatusDisabled}).IsStopped() {
		t.Fatal("a disabled tenant is stopped")
	}
}

func TestSessionIsAdminOnlyForSaaSAdmin(t *testing.T) {
	if (Session{Role: "saas_admin"}).IsAdmin() != true {
		t.Fatal("saas_admin must be admin")
	}
	for _, role := range []string{"owner", "manager", "cashier", "guest", "", "admin"} {
		if (Session{Role: role}).IsAdmin() {
			t.Fatalf("role %q must not pass the saas_admin check", role)
		}
	}
}

func TestBusinessTypeAndStatusOptionsAreFilterable(t *testing.T) {
	// Both option lists lead with "" so the UI can offer an "All" entry that
	// serialises to no query parameter at all.
	if BusinessTypeOptions[0] != "" || StatusOptions[0] != "" {
		t.Fatal("option lists must start with the empty (All) entry")
	}
	for _, opt := range BusinessTypeOptions[1:] {
		if strings.ContainsAny(opt, " ") {
			t.Fatalf("business type %q must be a wire value, not a label", opt)
		}
	}
}
