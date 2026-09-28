package opsconsole

import (
	"strings"
	"testing"
	"time"

	"posgo-ops/internal/posgo"
)

func TestTruncate(t *testing.T) {
	cases := []struct {
		in    string
		width int
		want  string
	}{
		{"short", 10, "short"},
		{"exactly-ten", 11, "exactly-ten"},
		{"truncates-here", 8, "truncat…"},
		{"", 4, ""},
		{"abc", 0, ""},
		{"ab", 1, "…"}, // a one-cell column can only hold the ellipsis
	}
	for _, tc := range cases {
		if got := truncate(tc.in, tc.width); got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.in, tc.width, got, tc.want)
		}
	}
}

// TestTruncateNeverOverflowsColumn guards the invariant the fixed-width table
// layout depends on: every rendered cell must fit the column it was sized for.
func TestTruncateNeverOverflowsColumn(t *testing.T) {
	long := strings.Repeat("x", 200)
	for _, col := range tenantColumns {
		got := truncate(long, col.Width)
		if len([]rune(got)) != col.Width {
			t.Errorf("column %q: rendered %d cells, want %d", col.Title, len([]rune(got)), col.Width)
		}
	}
}

func TestOptionsForLabelsBlankValues(t *testing.T) {
	opts := optionsFor(posgo.StatusOptions)
	if len(opts) != len(posgo.StatusOptions) {
		t.Fatalf("got %d options, want %d", len(opts), len(posgo.StatusOptions))
	}
	if opts[0].Label != "All" {
		t.Errorf("first (empty) option label = %q, want %q", opts[0].Label, "All")
	}
	if opts[0].Value != "" {
		t.Errorf("first option value = %q, want empty so no filter is sent", opts[0].Value)
	}
	// Every label must be non-empty: a blank row in a filter list is
	// indistinguishable from a rendering bug.
	for _, opt := range opts {
		if strings.TrimSpace(opt.Label) == "" {
			t.Errorf("option %v has a blank label", opt.Value)
		}
	}
}

func TestOptionsForPassesWireValuesThrough(t *testing.T) {
	want := map[string]bool{"": true, "active": true, "restaurant": true}
	for _, opt := range optionsFor([]string{"", "active", "restaurant"}) {
		value, ok := opt.Value.(string)
		if !ok || !want[value] {
			t.Errorf("unexpected wire value %v", opt.Value)
		}
	}
}

// TestTabIndexMapsEveryScreen covers the three authenticated screens. The login
// screen has no tab of its own once signed in: signOut swaps the whole strip
// back to a single pane, so tabIndex is never asked for it.
func TestTabIndexMapsEveryScreen(t *testing.T) {
	cases := map[screen]int{
		screenOverview: tabOverview,
		screenTenants:  tabTenants,
		screenDetail:   tabDetail,
	}
	for screen, want := range cases {
		if got := tabIndex(screen); got != want {
			t.Errorf("tabIndex(%v) = %d, want %d", screen, got, want)
		}
	}
	// The default arm is the tenants tab, so an unrecognised screen lands on the
	// list rather than on a detail pane with nothing in it.
	if got := tabIndex(screenLogin); got != tabTenants {
		t.Errorf("tabIndex(screenLogin) = %d, want the tenants tab", got)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := firstNonEmpty("", "  ", "EGP"); got != "EGP" {
		t.Errorf("got %q, want EGP", got)
	}
	if got := firstNonEmpty("", ""); got != "" {
		t.Errorf("got %q, want empty", got)
	}
	if got := firstNonEmpty("USD", "EGP"); got != "USD" {
		t.Errorf("got %q, want the first value", got)
	}
}

// analyticsFixture is a fully populated drill-down, so the layout test exercises
// every section rather than only the header.
func analyticsFixture() posgo.Analytics {
	return posgo.Analytics{
		Tenant: posgo.Tenant{
			ID:                 "11111111-1111-1111-1111-111111111111",
			Name:               "Demo Restaurant",
			Slug:               "demo-restaurant",
			Status:             posgo.StatusActive,
			Plan:               "starter",
			Subdomain:          "demo-restaurant.xamltech.com",
			SubscriptionStatus: "active",
			CurrencyCode:       "EGP",
			DefaultLanguage:    "ar",
			CreatedAt:          "2026-03-01T09:00:00Z",
			TrialEndsAt:        "2026-03-16T09:00:00Z",
		},
		Counts: posgo.AnalyticsCounts{Users: 3, Products: 10},
		Today: posgo.TodayStats{
			Date:         "2026-03-03",
			RevenueMinor: 123456, SalesCount: 12, AvgSaleMinor: 10288, ItemsSold: 34,
		},
		RevenueTrend: []posgo.TrendPoint{
			{Day: "2026-03-01", RevenueMinor: 1000},
			{Day: "2026-03-02", RevenueMinor: 2000},
		},
		TopProducts: []posgo.TopProduct{
			{ProductName: "Koshari", Quantity: 5, RevenueMinor: 50000},
		},
		RecentSales: []posgo.RecentSale{
			{Status: "completed", TotalMinor: 7500, PaymentMethod: "cash", Cashier: "Manager"},
		},
	}
}

func TestRenderAnalyticsShowsIdentityAndMetrics(t *testing.T) {
	nowFunc = func() time.Time { return time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC) }
	defer func() { nowFunc = func() time.Time { return time.Now() } }()

	got := renderAnalytics(analyticsFixture())

	for _, want := range []string{
		"Demo Restaurant",
		"demo-restaurant.xamltech.com",
		"starter",
		"active",
		"E£",       // revenue money, not raw minor units
		"Users",    // counts section
		"Products", // counts section
		"7-day revenue",
		"Koshari",
		"Recent sales",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("analytics layout is missing %q\n---\n%s", want, got)
		}
	}
	// Raw minor units must never reach the operator's screen.
	if strings.Contains(got, "123456") {
		t.Errorf("analytics layout leaked minor units\n---\n%s", got)
	}
}

func TestRenderAnalyticsOmitsAbsentSections(t *testing.T) {
	a := analyticsFixture()
	a.RevenueTrend = nil
	a.TopProducts = nil
	a.RecentSales = nil
	a.Tenant.SubscriptionStatus = ""
	a.Tenant.TrialEndsAt = ""
	a.Tenant.Subdomain = ""

	got := renderAnalytics(a)
	for _, unwanted := range []string{"7-day revenue", "Top products", "Recent sales", "Subscription", "Subdomain", "Trial ends"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("analytics layout should omit %q when the API returns nothing\n---\n%s", unwanted, got)
		}
	}
	if !strings.Contains(got, "Demo Restaurant") {
		t.Errorf("identity should still render\n---\n%s", got)
	}
}

func TestRenderAnalyticsUsesPerRowCurrencyFallback(t *testing.T) {
	a := analyticsFixture()
	// A sale with no currency of its own falls back to the tenant's, rather
	// than to the platform default, so a USD merchant is never shown EGP.
	a.Tenant.CurrencyCode = "USD"
	a.RecentSales[0].Currency = ""

	got := renderAnalytics(a)
	if !strings.Contains(got, "$") {
		t.Errorf("per-row currency fallback should render USD\n---\n%s", got)
	}
	if strings.Contains(got, "E£") {
		t.Errorf("tenant currency should win over the platform default\n---\n%s", got)
	}
}

// TestSortedBusinessIsTenantCountDescending pins the documented ordering: the
// overview leads with the vertical that has the most merchants, not the one with
// the most revenue, because tenant count is the platform's own growth metric.
func TestSortedBusinessIsTenantCountDescending(t *testing.T) {
	in := []posgo.BusinessSlice{
		{BusinessType: "grocery", Tenants: 1, RevenueMinor: 999_999},
		{BusinessType: "restaurant", Tenants: 3, RevenueMinor: 100},
		{BusinessType: "book_store", Tenants: 2, RevenueMinor: 500},
	}
	got := sortedBusiness(in)
	want := []string{"restaurant", "book_store", "grocery"}
	for i, name := range want {
		if got[i].BusinessType != name {
			t.Errorf("position %d = %q, want %q", i, got[i].BusinessType, name)
		}
	}
}

// TestSortedBusinessTieBreaksOnName keeps the overview stable across refreshes;
// an operator reading two consecutive screens should not see rows swap for no
// reason.
func TestSortedBusinessTieBreaksOnName(t *testing.T) {
	in := []posgo.BusinessSlice{
		{BusinessType: "zeta", RevenueMinor: 100},
		{BusinessType: "alpha", RevenueMinor: 100},
	}
	got := sortedBusiness(in)
	if got[0].BusinessType != "alpha" {
		t.Errorf("tie should break alphabetically, got %q first", got[0].BusinessType)
	}
}

func TestSortedBusinessDoesNotMutateInput(t *testing.T) {
	in := []posgo.BusinessSlice{
		{BusinessType: "grocery", RevenueMinor: 100},
		{BusinessType: "restaurant", RevenueMinor: 900},
	}
	sortedBusiness(in)
	if in[0].BusinessType != "grocery" {
		t.Errorf("sortedBusiness mutated its argument: %q", in[0].BusinessType)
	}
}

func TestStatusPill(t *testing.T) {
	for status, want := range map[string]string{
		posgo.StatusActive:    "ACTIVE",
		posgo.StatusSuspended: "SUSPENDED",
		posgo.StatusDisabled:  "STOPPED",
		posgo.StatusClosed:    "CLOSED",
	} {
		if got := statusPill(status); got != want {
			t.Errorf("statusPill(%q) = %q, want %q", status, got, want)
		}
	}
}
