package receipts

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

func authedRequest(router *gin.Engine, method, path, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
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
		"GET /v1/sales/:id/receipt",
		"GET /v1/sales/:id/receipt/print",
	} {
		if !routes[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}

func TestRoutesRequireAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, path := range []string{
		"/v1/sales/00000000-0000-0000-0000-000000000001/receipt",
		"/v1/sales/00000000-0000-0000-0000-000000000001/receipt/print",
	} {
		rec := authedRequest(router, http.MethodGet, path, "")
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("GET %s: expected 401, got %d", path, rec.Code)
		}
	}
}

func TestUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, path := range []string{
		"/v1/sales/00000000-0000-0000-0000-000000000001/receipt",
		"/v1/sales/00000000-0000-0000-0000-000000000001/receipt/print",
	} {
		rec := authedRequest(router, http.MethodGet, path, tokenForRole(t, "cashier"))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("GET %s: expected 503, got %d", path, rec.Code)
		}
	}
}

func TestReceiptRejectsInvalidSaleID(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, path := range []string{
		"/v1/sales/not-a-uuid/receipt",
		"/v1/sales/not-a-uuid/receipt/print",
	} {
		rec := authedRequest(router, http.MethodGet, path, tokenForRole(t, "cashier"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("GET %s: expected 400, got %d", path, rec.Code)
		}
	}
}

func TestBuildBytesEmptyTenantName(t *testing.T) {
	r := sampleReceipt()
	r.TenantName = ""
	out := BuildBytes(r)
	if !strings.Contains(string(out), "Sale     sale-123") {
		t.Fatal("expected sale id block even without tenant name")
	}
}
