package platform

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

// Unit coverage for the SaaS control-panel platform surface: route wiring and
// auth/RBAC/database-unavailable behavior with a nil pool (no DB needed).
// DB-backed behavior is covered in handler_integration_test.go.

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

func setup(t *testing.T) (*gin.Engine, http.Handler) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/v1")
	NewHandler(nil, testTokens(), config.Config{}).Register(group)
	return router, router
}

type routeCase struct {
	method, path, body string
}

func platformRoutes() []routeCase {
	return []routeCase{
		{http.MethodGet, "/v1/platform/trial/entitlements", ""},
		{http.MethodGet, "/v1/platform/trial/entitlements/:id", ""},
		{http.MethodPost, "/v1/platform/trial/entitlements/:id/extend", `{"extra_days":7}`},
		{http.MethodPost, "/v1/platform/trial/entitlements/:id/revoke", ""},
		{http.MethodPost, "/v1/platform/trial/entitlements/:id/convert", ""},
		{http.MethodPost, "/v1/platform/trial/grant", `{"account_id":"a","organization_id":"b","days":30}`},
		{http.MethodGet, "/v1/platform/trial/settings", ""},
		{http.MethodPut, "/v1/platform/trial/settings", `{}`},
		{http.MethodGet, "/v1/platform/audit", ""},
		{http.MethodPost, "/v1/platform/accounts/:id/suspend", `{"reason":"x"}`},
		{http.MethodPost, "/v1/platform/tenants/:id/suspend", `{"reason":"x"}`},
		{http.MethodPost, "/v1/platform/tenants/:id/activate", ""},
		{http.MethodPost, "/v1/platform/tenants/:id/stop", `{"reason":"x"}`},
		{http.MethodPost, "/v1/platform/tenants/:id/backup", ""},
	}
}

func TestRoutesRegistered(t *testing.T) {
	router, _ := setup(t)
	for _, path := range platformRoutes() {
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
	_, handler := setup(t)
	for _, path := range platformRoutes() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(path.method, path.path, strings.NewReader(path.body))
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d", path.method, path.path, rec.Code)
		}
	}
}

func TestEndpointsForbidNonSaasAdmin(t *testing.T) {
	_, handler := setup(t)
	for _, path := range platformRoutes() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(path.method, path.path, strings.NewReader(path.body))
		req.Header.Set("Authorization", "Bearer "+mintToken(t, "manager"))
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s %s: expected 403, got %d", path.method, path.path, rec.Code)
		}
	}
}

func TestEndpointsUnavailableWithoutDatabase(t *testing.T) {
	_, handler := setup(t)
	for _, path := range platformRoutes() {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(path.method, path.path, strings.NewReader(path.body))
		req.Header.Set("Authorization", "Bearer "+mintToken(t, "saas_admin"))
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: expected 503, got %d", path.method, path.path, rec.Code)
		}
	}
}
