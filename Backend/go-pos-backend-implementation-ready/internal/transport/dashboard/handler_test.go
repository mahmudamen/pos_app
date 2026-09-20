package dashboard

import (
	"net/http"
	"net/http/httptest"
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
	raw, err := testTokens().Issue(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004")
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
	found := false
	for _, r := range router.Routes() {
		if r.Method == "GET" && r.Path == "/v1/dashboard/summary" {
			found = true
		}
	}
	if !found {
		t.Fatal("GET /v1/dashboard/summary not registered")
	}
}

func TestSummaryRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/dashboard/summary", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestSummaryUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/dashboard/summary", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestParseVATMode(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"", "exclusive"},
		{"exclusive", "exclusive"},
		{"inclusive", "inclusive"},
		{"INCLUSIVE", "inclusive"},
		{"  inclusive  ", "inclusive"},
		{"garbage", "exclusive"},
		{"none", "exclusive"},
	}
	for _, c := range cases {
		if got := parseVATMode(c.raw); got != c.want {
			t.Errorf("parseVATMode(%q): got %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestProfitFor(t *testing.T) {
	const revenue, tax, cogs int64 = 1140, 140, 700
	if got := profitFor("exclusive", revenue, tax, cogs); got != 300 {
		t.Errorf("exclusive profit: got %d, want 300", got)
	}
	if got := profitFor("inclusive", revenue, tax, cogs); got != 440 {
		t.Errorf("inclusive profit: got %d, want 440", got)
	}
	if got := profitFor("inclusive", revenue, 0, cogs); got != 440 {
		t.Errorf("inclusive profit without residual tax: got %d, want 440", got)
	}
	if got := profitFor("exclusive", revenue, 0, cogs); got != 440 {
		t.Errorf("exclusive profit without residual tax: got %d, want 440", got)
	}
	if got := profitFor("exclusive", 0, 0, 0); got != 0 {
		t.Errorf("zero profit: got %d, want 0", got)
	}
	if got := profitFor("exclusive", revenue, tax, 0); got != 1000 {
		t.Errorf("zero cogs: got %d, want 1000", got)
	}
}
