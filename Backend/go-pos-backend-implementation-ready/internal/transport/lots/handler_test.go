package lots

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

func tokenForRole(t *testing.T, role string) string {
	t.Helper()
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004",
		role)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func setup() (*gin.Engine, *gin.RouterGroup) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-request")
		c.Next()
	})
	return router, router.Group("/v1")
}

func authedRequest(router *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if body == "" {
		req.Body = http.NoBody
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestRoutesRegistered(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	routes := map[string]bool{}
	for _, r := range router.Routes() {
		routes[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		"GET /v1/products/:id/lots",
		"POST /v1/products/:id/lots",
		"PATCH /v1/lots/:id",
	} {
		if !routes[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}

func TestRoutesRequireAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/v1/products/00000000-0000-0000-0000-000000000001/lots", ""},
		{http.MethodPost, "/v1/products/00000000-0000-0000-0000-000000000001/lots", `{"lot_number":"L1","quantity":10}`},
		{http.MethodPatch, "/v1/lots/00000000-0000-0000-0000-000000000002", `{"quantity":5}`},
	} {
		rec := authedRequest(router, tc.method, tc.path, tc.body, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d", tc.method, tc.path, rec.Code)
		}
	}
}

func TestUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/v1/products/00000000-0000-0000-0000-000000000001/lots", ""},
		{http.MethodPost, "/v1/products/00000000-0000-0000-0000-000000000001/lots", `{"lot_number":"L1","quantity":10}`},
		{http.MethodPatch, "/v1/lots/00000000-0000-0000-0000-000000000002", `{"quantity":5}`},
	} {
		rec := authedRequest(router, tc.method, tc.path, tc.body, tokenForRole(t, "manager"))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: expected 503, got %d", tc.method, tc.path, rec.Code)
		}
	}
}

func TestCreateLotRequiresPermission(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	rec := authedRequest(router, http.MethodPost, "/v1/products/00000000-0000-0000-0000-000000000001/lots", `{"lot_number":"L1","quantity":10}`, tokenForRole(t, "cashier"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier, got %d", rec.Code)
	}
}

func TestCreateLotValidatesBadge(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	rec := authedRequest(router, http.MethodPost, "/v1/products/00000000-0000-0000-0000-000000000001/lots", `{"lot_number":"","quantity":10}`, tokenForRole(t, "manager"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank lot number, got %d", rec.Code)
	}
}

func TestParseIncludeEmpty(t *testing.T) {
	for _, tc := range []struct {
		in      string
		want    bool
		wantErr bool
	}{
		{"", false, false},
		{"true", true, false},
		{"false", false, false},
		{"TRUE", true, false},
		{"maybe", false, true},
	} {
		got, err := parseIncludeEmpty(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("expected error for %q", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tc.in, err)
		}
		if got != tc.want {
			t.Errorf("parseIncludeEmpty(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestExpiryDateOrNull(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want any
	}{
		{"", nil},
		{"   ", nil},
		{"2026-12-31", "2026-12-31"},
		{" 2026-12-31 ", "2026-12-31"},
	} {
		if got := expiryDateOrNull(tc.in); got != tc.want {
			t.Errorf("expiryDateOrNull(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestIsExpired(t *testing.T) {
	day := func(offset int) string {
		return time.Now().AddDate(0, 0, offset).Format("2006-01-02")
	}
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"", false},
		{day(-1), true},
		{day(0), false},
		{day(1), false},
		{"not-a-date", false},
		{"2026-13-40", false},
	} {
		if got := isExpired(tc.in); got != tc.want {
			t.Errorf("isExpired(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestIncludeEmptySQL(t *testing.T) {
	if got := includeEmptySQL(true); got != "" {
		t.Errorf("includeEmptySQL(true) = %q, want empty", got)
	}
	if got := includeEmptySQL(false); got != " AND l.quantity > 0" {
		t.Errorf("includeEmptySQL(false) = %q, want quantity filter", got)
	}
}

func TestCreateLotValidatesPositiveQuantity(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	rec := authedRequest(router, http.MethodPost, "/v1/products/00000000-0000-0000-0000-000000000001/lots", `{"lot_number":"L1","quantity":0}`, tokenForRole(t, "manager"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for zero quantity, got %d", rec.Code)
	}
}
