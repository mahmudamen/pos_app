package server_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/infrastructure/metrics"
	"github.com/example/pos-api/internal/transport/server"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// registerWith nil pool + bare config must assemble the full route table with
// no database — that is exactly how the OpenAPI generator runs offline.
func TestRegisterBuildsRouteTableWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{
		Config: config.Config{},
	})

	routes := map[string]bool{}
	for _, r := range engine.Routes() {
		routes[r.Method+" "+r.Path] = true
	}
	required := []string{
		"GET /health/live",
		"GET /health/ready",
		"POST /v1/auth/login",
		"POST /v1/auth/refresh",
		"POST /v1/auth/logout",
		"POST /v1/auth/set-pin",
		"POST /v1/auth/verify-pin",
		"GET /v1/categories",
		"GET /v1/products",
		"GET /v1/products/barcode/:barcode",
		"POST /v1/sales",
		"GET /v1/sales",
		"GET /v1/sales/:id",
		"POST /v1/sales/:id/refund",
		"POST /v1/sales/:id/split",
		"GET /v1/sales/:id/receipt",
		"GET /v1/sales/:id/receipt/print",
		"GET /v1/customers",
		"GET /v1/dashboard/summary",
		"GET /v1/inventory/adjustments",
		"POST /v1/inventory/adjustments",
		"GET /v1/products/:id/lots",
		"GET /v1/products/:id/variants",
		"GET /v1/registers/current",
		"POST /v1/registers/open",
		"GET /v1/floors",
		"GET /v1/tables",
		"GET /v1/meta/countries",
		"GET /v1/meta/currencies",
		"GET /v1/saas/summary",
		"GET /v1/saas/tenants",
		"GET /v1/settings",
		"GET /v1/sync/pull",
		"POST /v1/sync/push",
		"GET /v1/users",
	}
	for _, want := range required {
		if !routes[want] {
			t.Errorf("route %s not registered", want)
		}
	}
}

func TestHealthLiveAnswersWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSaasGroupNotRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})
	// The SaaS control plane is saas_admin role-gated on every route and must
	// NOT be subject to the API limiter (the admin panel issues many parallel
	// /v1/saas/* calls per page), so rapid unauthenticated hits are plain 401s.
	for i := 0; i < 8; i++ {
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/saas/summary", nil))
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("request %d: expected 401, got %d", i, rec.Code)
		}
	}
}

func TestMetricsRouteOnlyWhenRegistryProvided(t *testing.T) {
	gin.SetMode(gin.TestMode)

	without := gin.New()
	server.Register(without, server.Deps{Config: config.Config{}})
	for _, r := range without.Routes() {
		if r.Path == "/metrics" {
			t.Fatal("expected no /metrics route when registry is nil")
		}
	}

	isolated := prometheus.NewRegistry()
	reg := metrics.NewScoped()
	_ = reg.Register(isolated, isolated)

	with := gin.New()
	server.Register(with, server.Deps{Config: config.Config{}, Metrics: reg})
	found := false
	for _, r := range with.Routes() {
		if r.Path == "/metrics" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected /metrics route when registry is provided")
	}
}

func TestIndexServesBrandedLandingWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}
	body := rec.Body.String()
	for _, want := range []string{"POS.Go", "XAMLtech", "Admin sign in", "/admin/"} {
		if !strings.Contains(body, want) {
			t.Errorf("index body missing %q", want)
		}
	}
}

func TestPrivacyServesPublicPolicyPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/private", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q, want text/html", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	for _, want := range []string{"POS.Go", "Privacy Policy", "support@xamltech.com", "/admin/"} {
		if !strings.Contains(body, want) {
			t.Errorf("privacy body missing %q", want)
		}
	}
}

func TestPricingServesFallbackPlansWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/pricing", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Simple pricing", "Starter", "Business", "Enterprise", "/admin/", "/private"} {
		if !strings.Contains(body, want) {
			t.Errorf("pricing body missing %q", want)
		}
	}
	// The landing preview must also render plan cards without a DB.
	rec2 := httptest.NewRecorder()
	engine.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected index 200, got %d", rec2.Code)
	}
	for _, want := range []string{"Starter", "Business", "Enterprise", "/pricing"} {
		if !strings.Contains(rec2.Body.String(), want) {
			t.Errorf("index body missing plan preview %q", want)
		}
	}
}

func TestSitePagesServeArabicRTL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})

	for _, path := range []string{"/", "/pricing", "/private"} {
		req := httptest.NewRequest(http.MethodGet, path+"?lang=ar", nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", path, rec.Code)
		}
		body := rec.Body.String()
		for _, want := range []string{`lang="ar"`, `dir="rtl"`, "English", "نظام نقاط بيع"} {
			if !strings.Contains(body, want) {
				t.Errorf("%s?lang=ar body missing %q", path, want)
			}
		}
		if v := rec.Header().Get("Vary"); !strings.Contains(v, "Accept-Language") {
			t.Errorf("%s missing Vary: Accept-Language", path)
		}
	}
}

func TestSitePagesHonorAcceptLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "ar,en;q=0.8")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `dir="rtl"`) {
		t.Fatal("expected Arabic RTL page from Accept-Language")
	}
}
