package billing

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

func allBillingRoutes() []struct{ method, path string } {
	return []struct {
		method, path string
	}{
		{"GET", "/v1/saas/plans"},
		{"POST", "/v1/saas/plans"},
		{"PATCH", "/v1/saas/plans/:id"},
		{"DELETE", "/v1/saas/plans/:id"},
		{"GET", "/v1/saas/subscriptions"},
		{"GET", "/v1/saas/tenants/:id/subscription"},
		{"POST", "/v1/saas/tenants/:id/subscription"},
		{"POST", "/v1/saas/subscriptions/:id/status"},
		{"POST", "/v1/saas/subscriptions/:id/change-plan"},
		{"GET", "/v1/saas/invoices"},
		{"POST", "/v1/saas/tenants/:id/invoices"},
		{"POST", "/v1/saas/invoices/:id/pay"},
		{"POST", "/v1/saas/invoices/:id/void"},
		{"POST", "/v1/saas/invoices/:id/refund"},
		{"GET", "/v1/saas/payment-providers"},
		{"GET", "/v1/saas/billing/summary"},
	}
}

func setupRouter() (*gin.Engine, *gin.RouterGroup) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router, router.Group("/v1/saas")
}

func TestBillingRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1/saas"))
	for _, r := range allBillingRoutes() {
		found := false
		for _, route := range router.Routes() {
			if route.Method == r.method && route.Path == r.path {
				found = true
			}
		}
		if !found {
			t.Fatalf("%s %s not registered", r.method, r.path)
		}
	}
}

func TestBillingEndpointsRequireAccessToken(t *testing.T) {
	for _, r := range allBillingRoutes() {
		router, group := setupRouter()
		NewHandler(nil, testTokens()).Register(group)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(r.method, r.path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d", r.method, r.path, rec.Code)
		}
	}
}

func TestBillingEndpointsForbidNonSaasAdmin(t *testing.T) {
	for _, r := range allBillingRoutes() {
		router, group := setupRouter()
		NewHandler(nil, testTokens()).Register(group)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(r.method, r.path, nil)
		req.Header.Set("Authorization", "Bearer "+mintToken(t, "manager"))
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("%s %s: expected 403, got %d", r.method, r.path, rec.Code)
		}
	}
}

func TestBillingEndpointsUnavailableWithoutDatabase(t *testing.T) {
	// GET endpoints return 503 without a DB. POST endpoints first try to
	// bind JSON; with an empty body they return 400 instead. This test only
	// covers the GET endpoints for clarity — integration tests cover the
	// full path for mutating routes.
	for _, path := range []string{
		"/v1/saas/plans",
		"/v1/saas/subscriptions",
		"/v1/saas/tenants/00000000-0000-0000-0000-000000000001/subscription",
		"/v1/saas/invoices",
		"/v1/saas/billing/summary",
	} {
		router, group := setupRouter()
		NewHandler(nil, testTokens()).Register(group)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+mintToken(t, "saas_admin"))
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s: expected 503, got %d", path, rec.Code)
		}
	}
}

func TestPaymentProvidersAvailableOffline(t *testing.T) {
	// The gateway registry is in-memory, so this route works without a DB.
	router, group := setupRouter()
	NewHandler(nil, testTokens()).Register(group)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/saas/payment-providers", nil)
	req.Header.Set("Authorization", "Bearer "+mintToken(t, "saas_admin"))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
