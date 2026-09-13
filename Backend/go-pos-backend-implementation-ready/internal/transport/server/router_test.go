package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/infrastructure/metrics"
	"github.com/example/pos-api/internal/infrastructure/ratelimit"
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

func TestSaasGroupRateLimitedWhenApiLimiterProvided(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{
		Config:     config.Config{},
		ApiLimiter: ratelimit.NewMemory(1, time.Minute),
	})
	// First request passes the limiter and reaches auth (401 without a token).
	first := httptest.NewRecorder()
	engine.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/v1/saas/summary", nil))
	if first.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated first request, got %d", first.Code)
	}
	// Second request is throttled before auth.
	second := httptest.NewRecorder()
	engine.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/v1/saas/summary", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for throttled request, got %d", second.Code)
	}
}

func TestOtherRoutesNotRateLimitedWithoutApiLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})
	for i := 0; i < 5; i++ {
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
	reg.Register(isolated, isolated)

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
