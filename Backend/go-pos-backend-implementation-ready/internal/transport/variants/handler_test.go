package variants

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
		"GET /v1/products/:id/variants",
		"POST /v1/products/:id/variants",
		"PATCH /v1/variants/:id",
		"DELETE /v1/variants/:id",
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
		{http.MethodGet, "/v1/products/00000000-0000-0000-0000-000000000001/variants", ""},
		{http.MethodPost, "/v1/products/00000000-0000-0000-0000-000000000001/variants", `{"name":"M Blue","sku":"TS-BLUE-M","price_minor":1000}`},
		{http.MethodPatch, "/v1/variants/00000000-0000-0000-0000-000000000002", `{"price_minor":1200}`},
		{http.MethodDelete, "/v1/variants/00000000-0000-0000-0000-000000000002", ""},
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
		{http.MethodGet, "/v1/products/00000000-0000-0000-0000-000000000001/variants", ""},
		{http.MethodPost, "/v1/products/00000000-0000-0000-0000-000000000001/variants", `{"name":"M Blue","sku":"TS-BLUE-M","price_minor":1000}`},
		{http.MethodPatch, "/v1/variants/00000000-0000-0000-0000-000000000002", `{"price_minor":1200}`},
		{http.MethodDelete, "/v1/variants/00000000-0000-0000-0000-000000000002", ""},
	} {
		rec := authedRequest(router, tc.method, tc.path, tc.body, tokenForRole(t, "manager"))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: expected 503, got %d", tc.method, tc.path, rec.Code)
		}
	}
}

func TestWriteRequiresPermission(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	rec := authedRequest(router, http.MethodPost, "/v1/products/00000000-0000-0000-0000-000000000001/variants", `{"name":"M Blue","sku":"TS-BLUE-M","price_minor":1000}`, tokenForRole(t, "cashier"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier, got %d", rec.Code)
	}
}

func TestCreateVariantsValidatesFields(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, tc := range []struct{ body string }{
		{`{"name":"","sku":"TS","price_minor":1000}`},
		{`{"name":"M","sku":"","price_minor":1000}`},
		{`{"name":"M","sku":"TS","price_minor":-1}`},
	} {
		rec := authedRequest(router, http.MethodPost, "/v1/products/00000000-0000-0000-0000-000000000001/variants", tc.body, tokenForRole(t, "manager"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for %s, got %d", tc.body, rec.Code)
		}
	}
}

func TestPatchVariantsRejectsNegativePrice(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	rec := authedRequest(router, http.MethodPatch, "/v1/variants/00000000-0000-0000-0000-000000000002", `{"price_minor":-5}`, tokenForRole(t, "manager"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative price, got %d", rec.Code)
	}
}
