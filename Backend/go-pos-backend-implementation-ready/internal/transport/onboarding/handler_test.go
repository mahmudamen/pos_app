package onboarding

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

func testTokens() security.TokenManager {
	return security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
}

func tokenWithRole(t *testing.T, role string) string {
	t.Helper()
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", role)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	return router
}

func do(router *gin.Engine, method, path, token string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRoutesRegistered(t *testing.T) {
	router := setupRouter()
	routes := map[string]bool{}
	for _, r := range router.Routes() {
		routes[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		"GET /v1/onboarding",
		"GET /v1/onboarding/tours",
		"GET /v1/onboarding/readiness",
		"POST /v1/onboarding/tours/:id/start",
		"PATCH /v1/onboarding/tours/:id",
		"POST /v1/onboarding/tours/:id/pause",
		"POST /v1/onboarding/tours/:id/resume",
		"POST /v1/onboarding/tours/:id/skip",
		"POST /v1/onboarding/tours/:id/complete",
		"POST /v1/onboarding/events",
		"POST /v1/onboarding/reward",
		"GET /v1/onboarding/summary",
	} {
		if !routes[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}

func TestOnboardingRequiresAccessToken(t *testing.T) {
	router := setupRouter()
	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/v1/onboarding"},
		{http.MethodGet, "/v1/onboarding/readiness"},
		{http.MethodPost, "/v1/onboarding/tours/cash_session/start"},
		{http.MethodPost, "/v1/onboarding/events"},
		{http.MethodPost, "/v1/onboarding/reward"},
		{http.MethodGet, "/v1/onboarding/summary"},
	} {
		if rec := do(router, tc.method, tc.path, "", ""); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without token: expected 401, got %d", tc.method, tc.path, rec.Code)
		}
	}
}

func TestOnboardingRejectsGarbageToken(t *testing.T) {
	router := setupRouter()
	rec := do(router, http.MethodGet, "/v1/onboarding", "not-a-jwt", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestOnboardingForbiddenForUnknownRole(t *testing.T) {
	router := setupRouter()
	// "auditor" is deliberately absent from the static RBAC matrix.
	rec := do(router, http.MethodGet, "/v1/onboarding", tokenWithRole(t, "auditor"), "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOnboardingUnavailableWithoutDatabase(t *testing.T) {
	router := setupRouter()
	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/v1/onboarding", ""},
		{http.MethodGet, "/v1/onboarding/tours", ""},
		{http.MethodGet, "/v1/onboarding/readiness", ""},
		{http.MethodPost, "/v1/onboarding/tours/first_sale/start", ""},
		{http.MethodPatch, "/v1/onboarding/tours/first_sale", `{"current_step":"grid"}`},
		{http.MethodPost, "/v1/onboarding/events", `{"event_id":"7f6b6d1e-6a6e-4b7a-8a9b-1c2d3e4f5a6b","event":"sale.completed"}`},
		{http.MethodPost, "/v1/onboarding/reward", ""},
		{http.MethodGet, "/v1/onboarding/summary", ""},
	} {
		rec := do(router, tc.method, tc.path, tokenWithRole(t, "cashier"), tc.body)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s without pool: expected 503, got %d: %s",
				tc.method, tc.path, rec.Code, rec.Body.String())
		}
	}
}

func TestApplyTransition(t *testing.T) {
	cases := []struct {
		status  string
		action  string
		want    string
		wantErr bool
	}{
		{"", "start", "running", false},
		{"not_started", "start", "running", false},
		{"skipped", "start", "running", false},
		{"paused", "start", "running", false},
		{"running", "start", "running", false},
		{"completed", "start", "completed", true},
		{"running", "pause", "paused", false},
		{"not_started", "pause", "not_started", true},
		{"paused", "pause", "paused", true},
		{"completed", "pause", "completed", true},
		{"paused", "resume", "running", false},
		{"running", "resume", "running", false},
		{"not_started", "resume", "not_started", true},
		{"completed", "resume", "completed", true},
		{"running", "skip", "skipped", false},
		{"not_started", "skip", "skipped", false},
		{"completed", "skip", "completed", true},
		{"running", "complete", "completed", false},
		{"not_started", "complete", "completed", false},
		{"completed", "complete", "completed", false},
		{"running", "explode", "running", true},
	}
	for _, tc := range cases {
		got, err := applyTransition(tc.status, tc.action)
		if tc.wantErr != (err != nil) {
			t.Fatalf("applyTransition(%q,%q) error=%v, wantErr=%v", tc.status, tc.action, err, tc.wantErr)
		}
		if !tc.wantErr && got != tc.want {
			t.Fatalf("applyTransition(%q,%q)=%q, want %q", tc.status, tc.action, got, tc.want)
		}
	}
}

func TestValidateStep(t *testing.T) {
	for _, good := range []string{"intro", "grid", "step-1", "a.b_c-1"} {
		if err := validateStep(good); err != nil {
			t.Fatalf("validateStep(%q) = %v, want nil", good, err)
		}
	}
	for _, bad := range []string{"", "-intro", "Intro", "has space", "step/one", strings.Repeat("a", 65)} {
		if err := validateStep(bad); err == nil {
			t.Fatalf("validateStep(%q) = nil, want error", bad)
		}
	}
}

func TestValidateEvent(t *testing.T) {
	okID := "7f6b6d1e-6a6e-4b7a-8a9b-1c2d3e4f5a6b"
	if err := validateEvent(okID, "sale.completed", json.RawMessage(`{"total_minor":500}`)); err != nil {
		t.Fatalf("valid event rejected: %v", err)
	}
	if err := validateEvent(okID, "sale.completed", json.RawMessage(`{}`)); err != nil {
		t.Fatalf("empty payload rejected: %v", err)
	}
	if err := validateEvent("not-a-uuid", "sale.completed", nil); err == nil {
		t.Fatal("bad event_id accepted")
	}
	if err := validateEvent(okID, "Sale.Completed", nil); err == nil {
		t.Fatal("uppercase event accepted")
	}
	if err := validateEvent(okID, "9sale", nil); err == nil {
		t.Fatal("digit-leading event accepted")
	}
	if err := validateEvent(okID, "sale.completed", json.RawMessage(`"a string"`)); err == nil {
		t.Fatal("non-object payload accepted")
	}
	big := json.RawMessage(`{"blob":"` + strings.Repeat("x", maxEventBytes) + `"}`)
	if err := validateEvent(okID, "sale.completed", big); err == nil {
		t.Fatal("oversized payload accepted")
	}
	if err := validateEvent(okID, "sale.completed", json.RawMessage(`{bad json`)); err == nil {
		t.Fatal("malformed payload accepted")
	}
}

func TestRecommendTour(t *testing.T) {
	allFresh := map[string]string{}
	if got := recommendTour(allFresh); got != "store_setup" {
		t.Fatalf("fresh tenant: recommended %q, want store_setup", got)
	}
	inFlight := map[string]string{"cash_session": "running"}
	if got := recommendTour(inFlight); got != "cash_session" {
		t.Fatalf("running tour: recommended %q, want cash_session", got)
	}
	paused := map[string]string{"store_setup": "completed", "first_product": "completed",
		"terminal_setup": "completed", "cash_session": "paused"}
	if got := recommendTour(paused); got != "cash_session" {
		t.Fatalf("paused tour: recommended %q, want cash_session", got)
	}
	skippedFirst := map[string]string{"store_setup": "skipped"}
	if got := recommendTour(skippedFirst); got != "first_product" {
		t.Fatalf("skipped first: recommended %q, want first_product", got)
	}
	done := map[string]string{
		"store_setup": "completed", "first_product": "completed",
		"terminal_setup": "completed", "cash_session": "completed", "first_sale": "completed",
	}
	if got := recommendTour(done); got != "" {
		t.Fatalf("all complete: recommended %q, want empty", got)
	}
	allSkipped := map[string]string{
		"store_setup": "skipped", "first_product": "skipped", "terminal_setup": "skipped",
		"cash_session": "skipped", "first_sale": "skipped",
	}
	if got := recommendTour(allSkipped); got != "" {
		t.Fatalf("all skipped: recommended %q, want empty", got)
	}
}

func TestCatalogInvariants(t *testing.T) {
	seen := map[string]bool{}
	for _, def := range catalog {
		if seen[def.ID] {
			t.Fatalf("duplicate tour id %q", def.ID)
		}
		seen[def.ID] = true
		if def.Version < 1 || def.StepCount < 1 {
			t.Fatalf("tour %q needs version>=1 and step_count>=1", def.ID)
		}
		found := false
		for _, goal := range []string{GoalStoreConfigured, GoalProductReady,
			GoalTerminalConfigured, GoalCashSessionOpened, GoalFirstSaleCompleted} {
			if def.Goal == goal {
				found = true
			}
		}
		if !found {
			t.Fatalf("tour %q has unknown goal %q", def.ID, def.Goal)
		}
	}
}
