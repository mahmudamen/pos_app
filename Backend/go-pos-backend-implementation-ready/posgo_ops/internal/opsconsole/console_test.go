package opsconsole

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"posgo-ops/internal/posgo"

	"m31labs.dev/fluffyui/runtime"

	fluffytest "m31labs.dev/fluffyui/testing"
)

// stubAPI is a control-plane fake. Every request is recorded so a test can
// assert what the console actually sent, not just what it rendered.
type stubAPI struct {
	t *testing.T

	mu       sync.Mutex
	requests []recordedRequest

	summary     posgo.Summary
	tenants     posgo.TenantPage
	analytics   posgo.Analytics
	actionHits  map[string]int
	actionFails map[string]int
	// backupBody is the document the backup endpoint streams back.
	backupBody string

	// authHeader is what the last authenticated request carried, so a test can
	// prove the token reached the server.
	authHeader string
}

type recordedRequest struct {
	method string
	path   string
	query  string
	body   string
}

func newStubAPI(t *testing.T) *stubAPI {
	return &stubAPI{
		t:           t,
		backupBody:  `{"tenant_id":"t-1","tables":{}}`,
		actionHits:  map[string]int{},
		actionFails: map[string]int{},
		summary: posgo.Summary{
			TotalTenants: 7, TotalUsers: 24, TotalSales: 310,
			RevenueMinor: 987654, SupportedCurrency: "EGP",
			ByBusiness: []posgo.BusinessSlice{
				{BusinessType: "grocery", Tenants: 1, Users: 3, RevenueMinor: 10},
				{BusinessType: "restaurant", Tenants: 2, Users: 6, RevenueMinor: 20},
			},
		},
		tenants: posgo.TenantPage{
			Page: 1, Limit: 25, Total: 3,
			Tenants: []posgo.Tenant{
				{ID: "t-1", Name: "Demo Restaurant", Slug: "demo-restaurant", Status: posgo.StatusActive,
					BusinessType: "restaurant", Plan: "starter", CurrencyCode: "EGP",
					Users: 3, Products: 10, RevenueMinor: 500000},
				{ID: "t-2", Name: "Corner Grocery", Slug: "corner-grocery", Status: posgo.StatusSuspended,
					BusinessType: "grocery", Plan: "standard", CurrencyCode: "EGP",
					Users: 2, RevenueMinor: 12000},
				{ID: "t-3", Name: "Shut Shop", Slug: "shut-shop", Status: posgo.StatusDisabled,
					BusinessType: "grocery", Plan: "trial", CurrencyCode: "EGP", Users: 0},
			},
		},
		analytics: posgo.Analytics{
			Tenant: posgo.Tenant{ID: "t-1", Name: "Demo Restaurant", Slug: "demo-restaurant",
				Status: posgo.StatusActive, Plan: "starter", CurrencyCode: "EGP",
				CreatedAt: "2026-03-01T09:00:00Z"},
			Counts: posgo.AnalyticsCounts{Users: 3, Products: 10},
			Today:  posgo.TodayStats{RevenueMinor: 4321, SalesCount: 2, AvgSaleMinor: 2160, ItemsSold: 5},
		},
	}
}

func (s *stubAPI) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.requests = append(s.requests, recordedRequest{r.Method, r.URL.Path, r.URL.RawQuery, string(body)})
		s.authHeader = r.Header.Get("Authorization")
		s.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/auth/login":
			s.writeData(w, http.StatusOK, map[string]any{
				"access_token":  "tok-abc",
				"refresh_token": "ref-abc",
				"expires_in":    3600,
				"tenant": map[string]any{
					"id": "saas-1", "currency_code": "EGP", "country_code": "EG",
				},
				"user": map[string]any{
					"id": "u-1", "display_name": "Platform Admin", "role": "saas_admin",
				},
			})
		case r.URL.Path == "/v1/saas/summary":
			s.writeData(w, http.StatusOK, s.summary)
		case r.URL.Path == "/v1/saas/tenants":
			s.writeTenants(w, r)
		case strings.HasPrefix(r.URL.Path, "/v1/saas/tenants/") && strings.HasSuffix(r.URL.Path, "/analytics"):
			s.writeData(w, http.StatusOK, s.analytics)
		case strings.HasPrefix(r.URL.Path, "/v1/platform/tenants/"):
			verb := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
			s.mu.Lock()
			s.actionHits[verb]++
			fails := s.actionFails[verb]
			s.mu.Unlock()
			if fails > 0 {
				s.mu.Lock()
				s.actionFails[verb]--
				s.mu.Unlock()
				s.writeError(w, http.StatusConflict, "tenant_busy", "try again")
				return
			}
			if verb == "backup" {
				// The export is streamed raw, not wrapped in the data envelope.
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, s.backupBody)
				return
			}
			s.writeData(w, http.StatusOK, map[string]any{"ok": true})
		default:
			s.writeError(w, http.StatusNotFound, "not_found", r.URL.Path)
		}
	})
	return mux
}

// writeTenants applies the same filters the API does, so a test that counts
// disabled tenants through the status filter gets an honest answer.
func (s *stubAPI) writeTenants(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	query := r.URL.Query()
	rows := make([]posgo.Tenant, 0, len(s.tenants.Tenants))
	for _, tenant := range s.tenants.Tenants {
		if want := query.Get("status"); want != "" && tenant.Status != want {
			continue
		}
		if want := query.Get("q"); want != "" &&
			!strings.Contains(tenant.Name, want) && !strings.Contains(tenant.Slug, want) {
			continue
		}
		rows = append(rows, tenant)
	}
	limit := s.tenants.Limit
	if limit < 1 {
		limit = 25
	}
	if requested := query.Get("limit"); requested != "" {
		if parsed, err := strconv.Atoi(requested); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	// Only the total matters for the count request, but a real server still
	// honours the limit, so the console cannot be fooled by a full page.
	if len(rows) > limit {
		rows = rows[:limit]
	}
	s.writePage(w, http.StatusOK, rows, map[string]any{
		"page": 1, "limit": limit, "total": len(matchesOf(s.tenants.Tenants, query)),
	})
}

// matchesOf counts every tenant matching the filters, before the limit.
func matchesOf(all []posgo.Tenant, query url.Values) []posgo.Tenant {
	var out []posgo.Tenant
	for _, tenant := range all {
		if want := query.Get("status"); want != "" && tenant.Status != want {
			continue
		}
		if want := query.Get("q"); want != "" &&
			!strings.Contains(tenant.Name, want) && !strings.Contains(tenant.Slug, want) {
			continue
		}
		out = append(out, tenant)
	}
	return out
}

func (s *stubAPI) writeData(w http.ResponseWriter, status int, body any) {
	s.writePage(w, status, body, nil)
}

// writePage serves the list shape the API uses: a bare array under data with the
// paging counters in meta.
func (s *stubAPI) writePage(w http.ResponseWriter, status int, body any, meta any) {
	w.WriteHeader(status)
	payload := map[string]any{"data": body}
	if meta != nil {
		payload["meta"] = meta
	}
	// Encode errors are ignored: a test that finishes while a fetch is still in
	// flight writes to a server the cleanup has already closed, and that is
	// expected rather than a failure.
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *stubAPI) writeError(w http.ResponseWriter, status int, code, message string) {
	s.t.Helper()
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{"code": code, "message": message},
	})
}

func (s *stubAPI) hits(verb string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.actionHits[verb]
}

func (s *stubAPI) paths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.requests))
	for _, req := range s.requests {
		out = append(out, req.path)
	}
	return out
}

// sawQuery reports whether any recorded request to path carried a query
// containing want. The console hits /v1/saas/tenants twice - the visible page
// and the stopped-tenant count - so a "last query wins" check is not enough to
// prove the operator's filter was sent.
func (s *stubAPI) sawQuery(path, want string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, req := range s.requests {
		if req.path == path && strings.Contains(req.query, want) {
			return true
		}
	}
	return false
}

func (s *stubAPI) lastQuery(path string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.requests) - 1; i >= 0; i-- {
		if s.requests[i].path == path {
			return s.requests[i].query
		}
	}
	return ""
}

func (s *stubAPI) lastBody(path string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := len(s.requests) - 1; i >= 0; i-- {
		if s.requests[i].path == path {
			return s.requests[i].body
		}
	}
	return ""
}

// newTestConsole wires a console to a stub API and signs it in.
func newTestConsole(t *testing.T, stub *stubAPI, mutate func(*Config)) (*Console, *pumpScheduler) {
	t.Helper()
	server := httptest.NewServer(stub.handler())
	cfg := Config{
		BaseURL:    server.URL,
		TenantID:   "saas-1",
		Email:      "admin@posgo.saas",
		Password:   "admin",
		DeviceID:   "test-device",
		DeviceName: "test console",
		BackupDir:  t.TempDir(),
	}
	if mutate != nil {
		mutate(&cfg)
	}

	console, _, err := New(context.Background(), cfg)
	if err != nil {
		server.Close()
		t.Fatalf("build console: %v", err)
	}
	// Run hands the console the app's scheduler, which dispatches onto the
	// render thread. There is no render loop here, so the tests install a fake
	// one they pump by hand, and every assertion about widget state is made
	// between pumps.
	pump := &pumpScheduler{}
	console.bindScheduler(pump)

	t.Cleanup(func() {
		pump.drain()
		console.dispose()
		server.Close()
	})
	return console, pump
}

func signedInConsole(t *testing.T, stub *stubAPI, mutate func(*Config)) (*Console, *pumpScheduler) {
	t.Helper()
	console, pump := newTestConsole(t, stub, mutate)
	console.signIn()
	pump.drain()
	return console, pump
}

// waitForAnalytics blocks until the drill-down fetch for the selected tenant
// has settled. The resource is keyed on the selection, so it only starts once a
// tenant is opened.
func waitForAnalytics(t *testing.T, console *Console, pump *pumpScheduler) {
	t.Helper()
	waitFor(t, pump, func() bool {
		return console.analytics.Get().Data.TenantID != ""
	})
}

// waitForSummary blocks until the platform summary has settled.
func waitForSummary(t *testing.T, console *Console, pump *pumpScheduler) {
	t.Helper()
	waitFor(t, pump, func() bool {
		return len(console.summary.Get().Data.ByBusiness) > 0
	})
}

// waitForTenants blocks until the tenant fetch has settled and painted.
func waitForTenants(t *testing.T, console *Console, pump *pumpScheduler) {
	t.Helper()
	waitFor(t, pump, func() bool {
		return console.tenants.Get().Data.Total > 0 && console.table.RowCount() > 0
	})
}

func TestSignInSwapsInTheAuthenticatedTabs(t *testing.T) {
	stub := newStubAPI(t)
	console, _ := signedInConsole(t, stub, nil)

	if got := console.tabs.SelectedIndex(); got != tabOverview {
		t.Errorf("after sign-in the overview tab should be selected, got %d", got)
	}
	if !console.signedIn.Get() {
		t.Errorf("sign-in should reveal the authenticated panes")
	}
	if len(console.tabs.Tabs) != 3 {
		t.Errorf("expected the three authenticated panes, got %d", len(console.tabs.Tabs))
	}
	if msg := console.message.Get(); !strings.Contains(msg, "Platform Admin") {
		t.Errorf("status line should greet the operator, got %q", msg)
	}
	if console.session.Get().AccessToken != "tok-abc" {
		t.Errorf("session token was not stored")
	}
}

func TestSignInSendsCredentialsAndBearerTokenOnLaterCalls(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)

	body := stub.lastBody("/v1/auth/login")
	for _, want := range []string{"admin@posgo.saas", "admin", "saas-1", "test-device"} {
		if !strings.Contains(body, want) {
			t.Errorf("login body missing %q: %s", want, body)
		}
	}
	if got := stub.authHeader; got != "Bearer tok-abc" {
		t.Errorf("later request auth header = %q, want %q", got, "Bearer tok-abc")
	}
}

func TestSignInRequiresEmailAndTenant(t *testing.T) {
	stub := newStubAPI(t)
	console, _ := newTestConsole(t, stub, nil)

	console.emailInput.SetText("")
	console.tenantIn.SetText("saas-1")
	console.signIn()
	if got := console.message.Get(); !strings.Contains(got, "email is required") {
		t.Errorf("missing email should be refused, got %q", got)
	}
	if console.signedIn.Get() {
		t.Errorf("a refused sign-in must leave the login pane up")
	}

	console.emailInput.SetText("admin@posgo.saas")
	console.tenantIn.SetText("")
	console.signIn()
	if got := console.message.Get(); !strings.Contains(got, "tenant id is required") {
		t.Errorf("missing tenant id should be refused, got %q", got)
	}
}

func TestSuppliedTokenSkipsTheForm(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := newTestConsole(t, stub, func(c *Config) { c.AccessToken = "tok-env" })

	if !console.signedIn.Get() {
		t.Errorf("an access token in the config should go straight in")
	}
	if console.tabs.SelectedIndex() != tabOverview {
		t.Errorf("the strip should open on the overview, got %d", console.tabs.SelectedIndex())
	}
	waitForTenants(t, console, pump)
	if got := stub.authHeader; got != "Bearer tok-env" {
		t.Errorf("auth header = %q, want the configured token", got)
	}
	if body := stub.lastBody("/v1/auth/login"); body != "" {
		t.Errorf("a token sign-in should not also POST credentials: %s", body)
	}
}

// TestOverviewTilesShowPlatformTotals is the headline assertion: after sign-in
// the operator sees the real numbers, formatted as money rather than minor units.
func TestOverviewTilesShowPlatformTotals(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)

	waitFor(t, pump, func() bool { return console.summary.Get().Data.TotalTenants > 0 })

	if got := console.tiles[0].value; got != "7" {
		t.Errorf("tenants tile = %q, want 7", got)
	}
	if got := console.tiles[1].value; got != "24" {
		t.Errorf("users tile = %q, want 24", got)
	}
	if got := console.tiles[3].value; !strings.Contains(got, "9,876.54") {
		t.Errorf("revenue tile = %q, want a formatted amount", got)
	}
	if got := console.tiles[3].value; strings.Contains(got, "987654") {
		t.Errorf("revenue tile leaked minor units: %q", got)
	}
}

// TestStoppedTileFlagsDisabledTenants covers the one tile that carries urgency,
// which is the whole reason the console exists: knowing a merchant is locked out.
func TestStoppedTileFlagsDisabledTenants(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)

	waitFor(t, pump, func() bool { return console.stopped.Get().Data == 1 })
	if got := console.tiles[4].value; got != "1" {
		t.Errorf("stopped tile = %q, want 1", got)
	}
	if !console.tiles[4].urgent {
		t.Errorf("a stopped merchant should raise the tile's urgency flag")
	}
	for _, i := range []int{0, 1, 2, 3} {
		if console.tiles[i].urgent {
			t.Errorf("tile %d should not be urgent", i)
		}
	}
}

func TestTenantTableRendersEveryColumn(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)

	if got := console.table.RowCount(); got != 3 {
		t.Fatalf("row count = %d, want 3", got)
	}
	want := []string{"Demo Restaurant", "demo-restaurant", "restaurant", "ACTIVE", "starter", "3", "E£5,000.00"}
	for col, wantCell := range want {
		// Cells are padded to the column width so the grid stays aligned.
		got := strings.TrimSpace(console.table.GetCell(0, col))
		if got != wantCell {
			t.Errorf("row 0 column %d (%s) = %q, want %q", col, tenantColumns[col].Title, got, wantCell)
		}
	}
	if got := strings.TrimSpace(console.table.GetCell(2, 3)); got != "STOPPED" {
		t.Errorf("a disabled tenant should read STOPPED, got %q", got)
	}
}

func TestTenantRowsCarryTheirOwnIDs(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)

	console.table.SetSelected(1)
	if got := console.rowID(); got != "t-2" {
		t.Errorf("selected row id = %q, want t-2", got)
	}
	console.table.SetSelected(2)
	if got := console.rowID(); got != "t-3" {
		t.Errorf("selected row id = %q, want t-3", got)
	}
	// The table clamps an out-of-range selection to the last row, so this reads
	// the same id rather than nothing.
	console.table.SetSelected(99)
	if got := console.rowID(); got != "t-3" {
		t.Errorf("clamped selection = %q, want t-3", got)
	}
}

func TestFilterChangeRefetchesAndResetsToPageOne(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)

	console.filters.Update(func(f posgo.Filters) posgo.Filters { f.Page = 3; return f })
	console.updateFilters(func(f *posgo.Filters) { f.Search = "grocery" })

	if got := console.filters.Get(); got.Page != 1 || got.Search != "grocery" {
		t.Errorf("filter update should reset the page: %+v", got)
	}
	waitFor(t, pump, func() bool { return stub.sawQuery("/v1/saas/tenants", "q=grocery") })
	if q := stub.lastQuery("/v1/saas/tenants"); !strings.Contains(q, "page=1") {
		t.Errorf("refetch should carry page=1, query was %q", q)
	}
}

func TestSelectHandlersSendTheirWireValue(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)

	console.updateFilters(func(f *posgo.Filters) { f.BusinessType = "restaurant" })
	waitFor(t, pump, func() bool { return stub.sawQuery("/v1/saas/tenants", "business_type=restaurant") })

	console.updateFilters(func(f *posgo.Filters) { f.Status = posgo.StatusSuspended })
	waitFor(t, pump, func() bool { return stub.sawQuery("/v1/saas/tenants", "status=suspended") })
}

func TestPagerClampsToTheReportedRange(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)

	// Total 3 with limit 25 is a single page, so neither direction may move.
	console.setPage(1)
	if got := console.filters.Get().Page; got != 1 {
		t.Errorf("Next past the last page should stay put, page = %d", got)
	}
	console.setPage(-1)
	if got := console.filters.Get().Page; got != 1 {
		t.Errorf("Prev before the first page should stay put, page = %d", got)
	}

	// Grow the stub's tenant list so the reported total genuinely becomes 100,
	// the way the pager would see it with a bigger platform.
	grown := make([]posgo.Tenant, 0, 100)
	for i := range 100 {
		grown = append(grown, posgo.Tenant{ID: fmt.Sprintf("t-%d", i), Name: "Tenant", Status: posgo.StatusActive})
	}
	stub.mu.Lock()
	stub.tenants.Tenants = grown
	stub.mu.Unlock()
	console.tenants.Refetch()
	waitFor(t, pump, func() bool { return console.tenants.Get().Data.Total == 100 })
	console.setPage(1)
	if got := console.filters.Get().Page; got != 2 {
		t.Errorf("Next should advance to page 2, got %d", got)
	}
	console.setPage(-1)
	if got := console.filters.Get().Page; got != 1 {
		t.Errorf("Prev should return to page 1, got %d", got)
	}
}

func TestOpenSelectedFetchesAnalyticsAndSwitchesTab(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)

	console.table.SetSelected(0)
	console.openSelected()

	if got := console.tabs.SelectedIndex(); got != tabDetail {
		t.Errorf("opening a tenant should show the detail tab, got %d", got)
	}
	waitFor(t, pump, func() bool { return console.analytics.Get().Data.TenantID == "t-1" })
	if got := console.analytics.Get().Data.TenantID; got != "t-1" {
		t.Errorf("analytics bound to %q, want t-1", got)
	}
}

func TestOpenSelectedWithoutARowIsRefused(t *testing.T) {
	stub := newStubAPI(t)
	stub.tenants.Tenants = nil
	stub.tenants.Total = 0
	console, pump := signedInConsole(t, stub, nil)
	waitFor(t, pump, func() bool { return console.table.RowCount() == 1 })

	console.openSelected()
	if got := console.message.Get(); !strings.Contains(got, "select a tenant first") {
		t.Errorf("expected a refusal, got %q", got)
	}
	if got := console.tabs.SelectedIndex(); got == tabDetail {
		t.Errorf("the detail tab must not open without a tenant")
	}
}

func TestDetailTabRefusesWithoutASelection(t *testing.T) {
	stub := newStubAPI(t)
	console, _ := signedInConsole(t, stub, nil)

	console.showScreen(screenDetail)
	if got := console.tabs.SelectedIndex(); got != tabTenants {
		t.Errorf("detail without a selection should fall back to the list, tab = %d", got)
	}
	if got := console.message.Get(); !strings.Contains(got, "select a tenant") {
		t.Errorf("expected an explanation, got %q", got)
	}
}

func TestCloseDetailClearsTheSelection(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)
	console.table.SetSelected(0)
	console.openSelected()

	console.closeDetail()
	if got := console.selected.Get(); got != "" {
		t.Errorf("closing the detail pane should drop the selection, got %q", got)
	}
	if got := console.tabs.SelectedIndex(); got != tabTenants {
		t.Errorf("closing should return to the list, tab = %d", got)
	}
}

// TestStopRequiresConfirmation is the safety test: pressing Stop must not reach
// the API until the operator confirms.
func TestStopRequiresConfirmation(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)
	console.table.SetSelected(0)

	stop := console.stopAction()
	console.run(stop)

	if hits := stub.hits("stop"); hits != 0 {
		t.Fatalf("Stop reached the API %d time(s) before confirmation", hits)
	}
	if !console.confirming.Get() {
		t.Fatalf("Stop should have armed the confirmation strip")
	}
	if got := console.confirmID.Get(); got != "t-1" {
		t.Errorf("confirmation pinned to %q, want the tenant under the cursor", got)
	}

	console.confirmStop()
	if hits := stub.hits("stop"); hits != 1 {
		t.Errorf("confirming should send exactly one stop, got %d", hits)
	}
	if console.confirming.Get() {
		t.Errorf("the confirmation should stand down after acting")
	}
}

// TestStopConfirmationIsPinnedToTheTenantUnderTheCursor covers the case where
// the operator moves the cursor between arming and confirming.
func TestStopConfirmationIsPinnedToTheTenantUnderTheCursor(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, func(c *Config) { c.StopReason = "pinned" })
	waitForTenants(t, console, pump)

	console.table.SetSelected(0)
	console.run(console.stopAction())
	console.table.SetSelected(2) // cursor moves while the strip is up
	console.confirmStop()

	body := stub.lastBody("/v1/platform/tenants/t-1/stop")
	if body == "" {
		t.Fatalf("expected the pinned tenant t-1 to be stopped; requests: %v", stub.paths())
	}
	for _, path := range stub.paths() {
		if strings.Contains(path, "t-3/stop") {
			t.Errorf("the confirmation followed the cursor and stopped t-3: %v", path)
		}
	}
}

func TestCancellingConfirmationSendsNothing(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)
	console.table.SetSelected(0)

	console.run(console.stopAction())
	console.cancelConfirm()
	console.confirmStop()

	if hits := stub.hits("stop"); hits != 0 {
		t.Errorf("a cancelled confirmation must not call the API, got %d calls", hits)
	}
}

func TestStopReasonIsSentAndDefaultsToConfig(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, func(c *Config) { c.StopReason = "from-config" })
	waitForTenants(t, console, pump)
	console.table.SetSelected(0)

	console.run(console.stopAction())
	console.stopReasonIn.SetText("chargeback")
	console.confirmStop()

	if body := stub.lastBody("/v1/platform/tenants/t-1/stop"); !strings.Contains(body, "chargeback") {
		t.Errorf("typed reason should win, body was %s", body)
	}

	console.stopReasonIn.SetText("")
	console.run(console.stopAction())
	console.confirmStop()
	if body := stub.lastBody("/v1/platform/tenants/t-1/stop"); !strings.Contains(body, "from-config") {
		t.Errorf("empty input should fall back to the configured reason, body was %s", body)
	}
}

func TestNonDestructiveActionsRunImmediately(t *testing.T) {
	for _, verb := range []string{"activate", "suspend"} {
		t.Run(verb, func(t *testing.T) {
			stub := newStubAPI(t)
			console, pump := signedInConsole(t, stub, nil)
			waitForTenants(t, console, pump)
			console.table.SetSelected(1)

			var action platformAction
			for _, candidate := range console.platformActions() {
				if strings.EqualFold(candidate.verb, verb) {
					action = candidate
				}
			}
			console.run(action)

			if hits := stub.hits(verb); hits != 1 {
				t.Errorf("%s should run immediately, got %d calls", verb, hits)
			}
			if console.confirming.Get() {
				t.Errorf("%s must not arm a confirmation", verb)
			}
			if got := console.message.Get(); !strings.Contains(got, verb+" ok") {
				t.Errorf("status line should confirm the action, got %q", got)
			}
		})
	}
}

func TestBackupWritesTheExportToDisk(t *testing.T) {
	stub := newStubAPI(t)
	dir := t.TempDir()
	console, pump := signedInConsole(t, stub, func(c *Config) { c.BackupDir = dir })
	waitForTenants(t, console, pump)
	console.table.SetSelected(0)

	// A distinctive body, so the file can be matched against the response.
	payload := `{"tenant_id":"t-1","tables":{"users":[]}}`
	stub.mu.Lock()
	stub.backupBody = payload
	stub.mu.Unlock()

	backup, ok := console.actionByVerb("Backup")
	if !ok {
		t.Fatalf("expected a Backup action")
	}
	console.run(backup)

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read backup dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one backup file, got %d", len(entries))
	}
	path := filepath.Join(dir, entries[0].Name())
	if !strings.HasSuffix(path, "t-1.json") {
		t.Errorf("backup file should be named after the tenant, got %q", entries[0].Name())
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(got) != payload {
		t.Errorf("backup contents = %q, want the exported document", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat backup: %v", err)
	}
	// The export holds every row of the tenant; it must not be world readable.
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("backup permissions = %o, want 600", perm)
	}
	if msg := console.message.Get(); !strings.Contains(msg, path) {
		t.Errorf("status line should report where the backup landed, got %q", msg)
	}
}

func TestBackupFailureIsReportedAndWritesNothing(t *testing.T) {
	stub := newStubAPI(t)
	dir := t.TempDir()
	console, pump := signedInConsole(t, stub, func(c *Config) { c.BackupDir = dir })
	waitForTenants(t, console, pump)
	console.table.SetSelected(0)

	stub.mu.Lock()
	stub.actionFails["backup"] = 1
	stub.mu.Unlock()

	backup, ok := console.actionByVerb("Backup")
	if !ok {
		t.Fatalf("expected a Backup action")
	}
	console.run(backup)

	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("a failed export must not leave a file behind: %v", entries)
	}
	if got := console.message.Get(); !strings.Contains(got, "backup") {
		t.Errorf("a failed export should be surfaced, got %q", got)
	}
	if console.lastBackup.Get() != "" {
		t.Errorf("a failed export must not report a path: %q", console.lastBackup.Get())
	}
}

func TestActionFailureIsReportedWithoutChangingTheSelection(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)
	console.table.SetSelected(0)

	stub.mu.Lock()
	stub.actionFails["activate"] = 1
	stub.mu.Unlock()
	activate, _ := console.actionByVerb("Activate")
	console.run(activate)

	if got := console.message.Get(); !strings.Contains(got, "activate") || !strings.Contains(got, "tenant_busy") {
		t.Errorf("the failure should name the action and the API code, got %q", got)
	}
	if got := console.tabs.SelectedIndex(); got != tabOverview {
		t.Errorf("a failed action must not navigate away, tab = %d", got)
	}
	if got := console.selected.Get(); got != "" {
		t.Errorf("a failed action should not select a tenant, got %q", got)
	}
}

func TestExpiredSessionReturnsToTheLoginForm(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)

	console.handleAPIError("list tenants", &posgo.APIError{
		StatusCode: http.StatusUnauthorized, Code: "invalid_token", Message: "expired",
	})

	if console.signedIn.Get() {
		t.Errorf("an expired session should return to the login pane")
	}
	if console.session.Get().AccessToken != "" {
		t.Errorf("the stale token should be cleared from the session")
	}
	if got := console.message.Get(); !strings.Contains(got, "sign in again") {
		t.Errorf("the operator should be told to sign in again, got %q", got)
	}
}

func TestSignOutClearsEverythingAndReturnsToLogin(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)
	console.table.SetSelected(0)
	console.openSelected()

	// Arm a destructive confirmation first: signing out has to stand it down,
	// or the next sign-in would inherit a "yes, stop this tenant" prompt pinned
	// to a tenant id that is no longer in the session.
	console.run(console.stopAction())
	if !console.confirming.Get() {
		t.Fatalf("precondition: the stop confirmation should be armed")
	}

	console.signOut()

	if console.signedIn.Get() {
		t.Errorf("sign out should show the login pane")
	}
	if len(console.tabs.Tabs) != 3 {
		t.Errorf("the strip keeps its three panes across a sign-out, got %d", len(console.tabs.Tabs))
	}
	if console.confirming.Get() || console.confirmID.Get() != "" {
		t.Errorf("sign out must cancel a pending confirmation, got confirming=%v id=%q",
			console.confirming.Get(), console.confirmID.Get())
	}
	if console.session.Get().AccessToken != "" || console.selected.Get() != "" {
		t.Errorf("sign out must clear the session and the selection")
	}
	if got := console.message.Get(); !strings.Contains(got, "signed out") {
		t.Errorf("status line should confirm, got %q", got)
	}
}

func TestShowScreenRefusesBeforeSignIn(t *testing.T) {
	stub := newStubAPI(t)
	console, _ := newTestConsole(t, stub, nil)

	console.showScreen(screenOverview)
	if got := console.message.Get(); !strings.Contains(got, "sign in first") {
		t.Errorf("navigating before sign-in should be refused, got %q", got)
	}
	if console.signedIn.Get() {
		t.Errorf("the login pane must stay up")
	}
}

func TestNewRejectsAnEmptyBaseURL(t *testing.T) {
	if _, _, err := New(context.Background(), Config{}); err == nil {
		t.Fatalf("an empty base URL should be refused")
	}
}

// pumpScheduler is a fake UI thread. Schedule only queues, and drain - always
// called from the test goroutine - is what runs the work.
//
// Queuing rather than running inline is what makes the tests race-free. Paint
// callbacks arrive on whichever goroutine a fetch resolved on; if they ran there,
// they would write the tiles while the test goroutine reads them, which is
// exactly the race the app's real scheduler prevents. Here everything, producer
// and reader alike, ends up on the one goroutine that calls drain.
type pumpScheduler struct {
	mu    sync.Mutex
	queue []func()
}

func (s *pumpScheduler) Schedule(fn func()) {
	if fn == nil {
		return
	}
	s.mu.Lock()
	s.queue = append(s.queue, fn)
	s.mu.Unlock()
}

// drain runs everything queued so far, then returns.
func (s *pumpScheduler) drain() {
	for {
		s.mu.Lock()
		if len(s.queue) == 0 {
			s.mu.Unlock()
			return
		}
		fn := s.queue[0]
		s.queue = s.queue[1:]
		s.mu.Unlock()
		fn()
	}
}

// waitFor pumps the fake UI thread until a condition holds, so an async fetch's
// repaint is guaranteed to have landed before it is inspected.
func waitFor(t *testing.T, pump *pumpScheduler, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		pump.drain()
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("condition never became true")
}

// paint renders the console's root into a text screen.
//
// The other tests in this file assert widget state: which tab is selected, what
// rows the table holds, what the signals say. That is not the same as the
// operator seeing them, and the gap was not theoretical. A greedy gap widget in
// the nav bar made the root stack measure taller than the terminal, so the whole
// screen collapsed to zero height and painted nothing at all, while every state
// assertion in this file stayed green. These tests assert the painted screen so
// a layout regression fails here instead of in front of an operator.
//
// The tree is mounted around the render on purpose. widgets.SignalLabel reads
// its source when it mounts and then follows it by subscription, and the app
// only mounts the panes of the visible tab. Rendering without mounting left
// every reactive label frozen at the text it was constructed with, which made a
// correct drill-down pane look stale here while it was fine in the app.
func paint(t *testing.T, console *Console) string {
	t.Helper()
	runtime.MountTree(console.root)
	defer runtime.UnmountTree(console.root)
	return fluffytest.RenderToString(console.root, 120, 40)
}

func assertPainted(t *testing.T, screen string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(screen, w) {
			t.Errorf("screen is missing %q; painted:\n%s", w, screen)
		}
	}
}

// TestSignedOutScreenPaintsTheLoginForm covers the whole unauthenticated screen.
func TestSignedOutScreenPaintsTheLoginForm(t *testing.T) {
	stub := newStubAPI(t)
	console, _ := newTestConsole(t, stub, nil)

	assertPainted(t, paint(t, console),
		"POS.Go control plane",
		"Sign in",
		"Email",
		"admin@posgo.saas",
		"Platform tenant id",
		"Access token",
	)
	if screen := paint(t, console); strings.Contains(screen, "Business breakdown") {
		t.Errorf("the signed-out screen is showing the authenticated overview:\n%s", screen)
	}
}

// TestOverviewScreenPaintsTilesAndBreakdown covers the first authenticated tab.
func TestOverviewScreenPaintsTilesAndBreakdown(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)

	// The tile hint is the accessible description and deliberately stays off
	// screen, so the captions and values are what must be visible.
	assertPainted(t, paint(t, console),
		"Overview", "Tenants", "Tenant detail",
		"Tenants", "Users", "Sales", "Revenue", "Stopped",
		"Business breakdown",
	)
}

// TestTenantScreenPaintsTableAndActions covers the tenant list and its actions.
func TestTenantScreenPaintsTableAndActions(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)
	console.showScreen(screenTenants)
	pump.drain()

	assertPainted(t, paint(t, console),
		"Tenant", "Slug", "Status", "Plan",
		"demo-restaurant",
		"Actions on the selected tenant",
		"Stop selected",
	)
}

// TestDetailScreenPaintsTheSelectedTenant covers the drill-down pane.
func TestDetailScreenPaintsTheSelectedTenant(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)
	console.table.SetSelected(0)
	console.openSelected()
	waitForAnalytics(t, console, pump)
	console.showScreen(screenDetail)
	pump.drain()

	assertPainted(t, paint(t, console), "Demo Restaurant", "demo-restaurant")
}

// TestSignedOutScreenSurvivesATinyTerminal guards the layout against the class
// of bug above: the root stack must never measure taller than the space it is
// given, because a stack that overflows collapses to nothing rather than
// scrolling.
func TestSignedOutScreenSurvivesATinyTerminal(t *testing.T) {
	stub := newStubAPI(t)
	console, _ := newTestConsole(t, stub, nil)

	for _, size := range [][2]int{{120, 40}, {100, 24}, {80, 20}, {70, 14}} {
		screen := fluffytest.RenderToString(console.root, size[0], size[1])
		if !strings.Contains(screen, "Sign in") {
			t.Errorf("at %dx%d the login screen lost its form:\n%s", size[0], size[1], screen)
		}
	}
}

// TestSplitLinesDropsTrailingBlanks covers the text block's line splitting.
func TestSplitLinesDropsTrailingBlanks(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"one", []string{"one"}},
		{"a\nb\nc", []string{"a", "b", "c"}},
		{"a\nb\n", []string{"a", "b"}},
		{"a\n\n", []string{"a"}},
	} {
		got := splitLines(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("splitLines(%q) = %q, want %q", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitLines(%q) = %q, want %q", tc.in, got, tc.want)
				break
			}
		}
	}
}

// TestMultilineLabelMeasuresEveryLine is the regression test for the clipping
// bug: a text block must report one row per line, or the card around it sizes
// itself to a single row and drops the rest.
func TestMultilineLabelMeasuresEveryLine(t *testing.T) {
	for _, lines := range []int{1, 3, 6, 12} {
		text := strings.Repeat("row\n", lines)
		text = strings.TrimSuffix(text, "\n")
		label := multilineText(func() string { return text })
		got := fluffytest.MeasureWidget(label, 80, 40)
		if got.Height != lines {
			t.Errorf("%d lines measured %d rows tall, want %d", lines, got.Height, lines)
		}
	}
}

// TestMultilineBlockPaintsInsideItsCard proves the whole block is on screen, not
// just the first line: the card is sized from the block's measurement.
func TestMultilineBlockPaintsInsideItsCard(t *testing.T) {
	stub := newStubAPI(t)
	console, pump := signedInConsole(t, stub, nil)
	waitForTenants(t, console, pump)
	waitForSummary(t, console, pump)

	// Every business type the stub reports has to be on the overview, which is
	// what the single-row measurement used to swallow.
	screen := paint(t, console)
	for _, business := range []string{"restaurant", "grocery"} {
		if !strings.Contains(screen, business) {
			t.Errorf("business type %q is missing from the overview:\n%s", business, screen)
		}
	}
}
