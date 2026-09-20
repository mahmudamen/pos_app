package saas

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

func mintToken(t *testing.T, role string) string {
	t.Helper()
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", role)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func setup() (*gin.Engine, *gin.RouterGroup) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router, router.Group("/v1/saas")
}

type routeCase struct {
	method, path, body string
}

func saasRoutes() []routeCase {
	return []routeCase{
		{http.MethodGet, "/v1/saas/summary", ""},
		{http.MethodGet, "/v1/saas/tenants", ""},
		{http.MethodPost, "/v1/saas/tenants", `{"name":"Cafe Nile","business_type":"coffee_shop"}`},
		{http.MethodPatch, "/v1/saas/tenants/:id", `{"plan":"starter"}`},
		{http.MethodGet, "/v1/saas/tenants/:id/analytics", ""},
		{http.MethodGet, "/v1/saas/users", ""},
		{http.MethodPost, "/v1/saas/tenants/:id/users",
			`{"email":"x@y.com","display_name":"X","password":"password","role":"cashier"}`},
	}
}

func TestRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1/saas"))
	want := saasRoutes()
	for _, path := range want {
		found := false
		for _, r := range router.Routes() {
			if r.Method == path.method && r.Path == path.path {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s %s not registered", path.method, path.path)
		}
	}
}

func TestEndpointsRequireAccessToken(t *testing.T) {
	for _, path := range saasRoutes() {
		router, group := setup()
		NewHandler(nil, testTokens()).Register(group)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(path.method, path.path, strings.NewReader(path.body))
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d", path.method, path.path, rec.Code)
		}
	}
}

func TestEndpointsForbidNonSaasAdmin(t *testing.T) {
	for _, path := range saasRoutes() {
		router, group := setup()
		NewHandler(nil, testTokens()).Register(group)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(path.method, path.path, strings.NewReader(path.body))
		req.Header.Set("Authorization", "Bearer "+mintToken(t, "manager"))
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s %s: expected 403, got %d", path.method, path.path, rec.Code)
		}
	}
}

func TestEndpointsUnavailableWithoutDatabase(t *testing.T) {
	for _, path := range saasRoutes() {
		router, group := setup()
		NewHandler(nil, testTokens()).Register(group)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(path.method, path.path, strings.NewReader(path.body))
		req.Header.Set("Authorization", "Bearer "+mintToken(t, "saas_admin"))
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: expected 503, got %d", path.method, path.path, rec.Code)
		}
	}
}
