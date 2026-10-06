package registers

import (
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

func validToken(t *testing.T) string {
	t.Helper()
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004",
		"cashier")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func setup() (*gin.Engine, *gin.RouterGroup) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router, router.Group("/v1")
}

func TestRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	routes := map[string]bool{}
	for _, r := range router.Routes() {
		routes[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		"GET /v1/registers/current",
		"GET /v1/registers/summary",
		"POST /v1/registers/open",
		"POST /v1/registers/:id/close",
		"GET /v1/registers",
		"GET /v1/registers/:id",
	} {
		if !routes[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}

func TestCurrentSessionRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/registers/current", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestCurrentSessionUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/registers/current", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestOpenSessionRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/registers/open", strings.NewReader(`{"opening_cash_minor":5000}`)))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestOpenSessionUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/registers/open", strings.NewReader(`{"opening_cash_minor":5000}`))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestCloseSessionRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		"/v1/registers/00000000-0000-0000-0000-000000000001/close", strings.NewReader(`{}`)))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestCloseSessionUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost,
		"/v1/registers/00000000-0000-0000-0000-000000000001/close", strings.NewReader(`{"closing_cash_minor":0}`))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestListSessionsRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/registers", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestGetSessionRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet,
		"/v1/registers/00000000-0000-0000-0000-000000000001", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestSessionsUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, path := range []string{"/v1/registers", "/v1/registers/list"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.Header.Set("Authorization", "Bearer "+validToken(t))
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s: expected 503, got %d", path, recorder.Code)
		}
	}
}

func TestSummaryRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/registers/summary", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestSummaryUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	request := httptest.NewRequest(http.MethodGet, "/v1/registers/summary", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestSummaryRejectsInvalidBounds(t *testing.T) {
	for _, query := range []string{"from=not-a-time", "to=not-a-time"} {
		router, group := setup()
		NewHandler(nil, testTokens()).Register(group)
		request := httptest.NewRequest(http.MethodGet, "/v1/registers/summary?"+query, nil)
		request.Header.Set("Authorization", "Bearer "+validToken(t))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d: %s", query, recorder.Code, recorder.Body.String())
		}
	}
}

func TestSummaryValidBoundsPassValidation(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	query := "from=2026-10-01&to=2026-10-06"
	request := httptest.NewRequest(http.MethodGet, "/v1/registers/summary?"+query, nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("well-formed bounds should reach the database: got %d: %s",
			recorder.Code, recorder.Body.String())
	}
}

func deniedToken(t *testing.T) string {
	t.Helper()
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004",
		"guest")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestSessionsDeniedWithoutPermission(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	cases := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/v1/registers", ""},
		{http.MethodGet, "/v1/registers/list", ""},
		{http.MethodGet, "/v1/registers/current", ""},
		{http.MethodGet, "/v1/registers/summary", ""},
		{http.MethodGet, "/v1/registers/00000000-0000-0000-0000-000000000001", ""},
		{http.MethodPost, "/v1/registers/open", `{"opening_cash_minor":0}`},
		{http.MethodPost, "/v1/registers/00000000-0000-0000-0000-000000000001/close", `{"closing_cash_minor":0}`},
	}
	for _, tc := range cases {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		request.Header.Set("Authorization", "Bearer "+deniedToken(t))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden {
			t.Fatalf("%s %s: expected 403, got %d", tc.method, tc.path, recorder.Code)
		}
	}
}

func TestListSessionsRejectsInvalidFilters(t *testing.T) {
	for _, query := range []string{
		"status=suspended",
		"device_id=not-a-uuid",
		"from=not-a-time",
		"to=not-a-time",
	} {
		router, group := setup()
		NewHandler(nil, testTokens()).Register(group)
		request := httptest.NewRequest(http.MethodGet, "/v1/registers?"+query, nil)
		request.Header.Set("Authorization", "Bearer "+validToken(t))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d: %s", query, recorder.Code, recorder.Body.String())
		}
	}
}

func TestListSessionsValidFiltersPassValidation(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	query := "status=open&device_id=00000000-0000-0000-0000-000000000042&from=2026-10-01&to=2026-10-06"
	request := httptest.NewRequest(http.MethodGet, "/v1/registers?"+query, nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("well-formed filters should reach the database: got %d: %s",
			recorder.Code, recorder.Body.String())
	}
}

func TestParseTimeBound(t *testing.T) {
	if from, err := parseTimeBound("", false); err != nil || from != nil {
		t.Fatalf("empty bound: expected nil, nil, got %v, %v", from, err)
	}
	instant, err := parseTimeBound("2026-10-06T12:00:00Z", false)
	if err != nil || instant == nil || instant.UTC().Hour() != 12 {
		t.Fatalf("rfc3339: expected 12:00Z, got %v, %v", instant, err)
	}
	dayFrom, err := parseTimeBound("2026-10-06", false)
	if err != nil || dayFrom == nil || dayFrom.UTC().Hour() != 0 {
		t.Fatalf("date-only from: expected midnight, got %v, %v", dayFrom, err)
	}
	dayTo, err := parseTimeBound("2026-10-06", true)
	if err != nil || dayTo == nil ||
		dayTo.UTC().Year() != 2026 || dayTo.UTC().Month() != time.October || dayTo.UTC().Day() != 7 {
		t.Fatalf("date-only to: expected next midnight, got %v, %v", dayTo, err)
	}
	if _, err := parseTimeBound("yesterday", false); err == nil {
		t.Fatal("garbage bound: expected an error")
	}
}

func TestWhereClause(t *testing.T) {
	if clause, args := whereClause(nil, 1); clause != "" || args != nil {
		t.Fatalf("no filters: expected empty clause, got %q, %v", clause, args)
	}
	filters := []sessionFilter{
		{column: "rs.status = ?", value: "open"},
		{column: "rs.device_id = ?", value: "device-1"},
	}
	clause, args := whereClause(filters, 3)
	want := "WHERE rs.status = $3 AND rs.device_id = $4"
	if clause != want {
		t.Fatalf("clause: expected %q, got %q", want, clause)
	}
	if len(args) != 2 || args[0] != "open" || args[1] != "device-1" {
		t.Fatalf("args: unexpected %v", args)
	}
}
