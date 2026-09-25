package server_test

import (
	"encoding/json"
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
		"POST /v1/saas/tenants",
		"PATCH /v1/saas/tenants/:id",
		"GET /v1/saas/tenants/:id/analytics",
		"GET /v1/saas/users",
		"POST /v1/saas/tenants/:id/users",
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
	for _, want := range []string{"POS.Go", "XAMLtech", "Sign in", "/admin/", "SaaS admin"} {
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

func TestLandingGalleryServesExternalizedScreenshots(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()

	// The gallery card row, phone-mockup frames, dots and arrows are present...
	for _, want := range []string{`class="gal-track"`, `class="gal-dots"`, `class="gal-btn prev"`, `class="gal-btn next"`, `class="phone"`, `class="phone-screen"`} {
		if !strings.Contains(body, want) {
			t.Errorf("index body missing %q", want)
		}
	}
	// ...each screenshot is served from /screenshots/:name (not a data URI)...
	for i, want := range []string{
		`"/screenshots/checkout"`, `"/screenshots/cart"`, `"/screenshots/dashboard"`, `"/screenshots/sales"`,
		`"/screenshots/customers"`, `"/screenshots/sessions"`, `"/screenshots/community"`, `"/screenshots/national"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("gallery image %d: missing %q", i, want)
		}
	}
	if strings.Contains(body, "data:image/png;base64,") {
		t.Error("index still inlines screenshots as base64 data URIs")
	}
	// ...captions render...
	for _, cap := range []string{"Checkout", "Cart &amp; payment", "Today&#39;s dashboard", "Sales history", "Customers &amp; loyalty", "Cash sessions", "Community hub", "Egypt community"} {
		if !strings.Contains(body, cap) {
			t.Errorf("gallery caption %q not rendered", cap)
		}
	}
	// The screenshot route serves the embedded PNG with long-lived caching.
	for _, name := range []string{"checkout", "national"} {
		srec := httptest.NewRecorder()
		engine.ServeHTTP(srec, httptest.NewRequest(http.MethodGet, "/screenshots/"+name, nil))
		if srec.Code != http.StatusOK {
			t.Errorf("/screenshots/%s: expected 200, got %d", name, srec.Code)
		}
		if ct := srec.Header().Get("Content-Type"); ct != "image/png" {
			t.Errorf("/screenshots/%s: expected image/png, got %q", name, ct)
		}
		if cc := srec.Header().Get("Cache-Control"); !strings.Contains(cc, "max-age=31536000") {
			t.Errorf("/screenshots/%s: missing long-lived Cache-Control", name)
		}
	}
	srec := httptest.NewRecorder()
	engine.ServeHTTP(srec, httptest.NewRequest(http.MethodGet, "/screenshots/missing", nil))
	if srec.Code != http.StatusNotFound {
		t.Errorf("/screenshots/missing: expected 404, got %d", srec.Code)
	}
	// The old phone-mockup stage is gone.
	for _, gone := range []string{`class="stage-tabs"`, `class="slip"`} {
		if strings.Contains(body, gone) {
			t.Errorf("index still contains removed %q markup", gone)
		}
	}
}

func TestLandingRendersAllFeaturesModules(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`id="all"`, `class="all-grid"`, `class="all-card"`, "Everything included", "Every module, in one app."} {
		if !strings.Contains(body, want) {
			t.Errorf("index body missing %q", want)
		}
	}
	for _, title := range []string{
		"Sell fast", "Payments &amp; splitting", "Registers &amp; cash", "Inventory &amp; stock",
		"Lots &amp; variants", "Discounts &amp; approvals", "Reports &amp; analytics",
		"Customers &amp; loyalty", "Receipts &amp; refunds", "Multi-device sync",
		"Team &amp; community", "Owner&#39;s control plane",
	} {
		if !strings.Contains(body, title) {
			t.Errorf("module title %q not rendered", title)
		}
	}
	if !strings.Contains(body, `href="/#all"`) {
		t.Error("nav missing the all-features link")
	}
	if !strings.Contains(body, `href="/#gallery"`) {
		t.Error("nav missing the gallery link")
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

func TestIndexServesStatsStepsAndFAQ(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"How it works", "Open the register", "Close with a Z-report", "day free trial", "detail", "</details>"} {
		if !strings.Contains(body, want) {
			t.Errorf("index body missing %q", want)
		}
	}
	if !strings.Contains(body, `href="/pricing#compare"`) {
		t.Error("index plans link should point to the pricing comparison anchor")
	}
}

func TestPricingServesCompareTableAndFAQ(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/pricing", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`<table class="compare">`, `<th scope="col">Starter</th>`, `<th scope="col">Business</th>`, `<th scope="col">Enterprise</th>`, `class="match"`, `<span class="dash"`} {
		if !strings.Contains(body, want) {
			t.Errorf("pricing body missing %q", want)
		}
	}
}

// surfaceEngine registers the real route table with the three public domains
// enabled, i.e. how production is configured.
func surfaceEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{Surfaces: config.SurfaceRouting{
		Enabled:       true,
		CompanyDomain: config.DefaultCompanyDomain,
		SaaSDomain:    config.DefaultSaaSDomain,
		POSDomain:     config.DefaultPOSDomain,
	}}})
	return engine
}

func hostGet(t *testing.T, engine *gin.Engine, host, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Host = host
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestSurfaceRootServesTheRightPagePerDomain(t *testing.T) {
	engine := surfaceEngine(t)

	// xamltech.com — the studio's own home page.
	rec := hostGet(t, engine, config.DefaultCompanyDomain, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("company root: got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"XAMLtech", "Software studio", "What we build", "How we work", "Start here",
		"https://" + config.DefaultPOSDomain, "https://" + config.DefaultSaaSDomain + "/admin/",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("company home missing %q", want)
		}
	}
	// The product landing is not the company home.
	for _, unwanted := range []string{"Point of sale that keeps selling", "Everything included", "class=\"gal-track\""} {
		if strings.Contains(body, unwanted) {
			t.Errorf("company home should not contain product-page copy %q", unwanted)
		}
	}

	// posgo.xamltech.com — the POS.Go product landing.
	rec = hostGet(t, engine, config.DefaultPOSDomain, "/")
	if rec.Code != http.StatusOK {
		t.Fatalf("POS root: got %d, want 200", rec.Code)
	}
	body = rec.Body.String()
	for _, want := range []string{"POS.Go", "Point of sale that keeps selling", `class="gal-track"`, "Starter", "Business", "Enterprise"} {
		if !strings.Contains(body, want) {
			t.Errorf("POS landing missing %q", want)
		}
	}

	// api.xamltech.com — straight to the console.
	rec = hostGet(t, engine, config.DefaultSaaSDomain, "/")
	if rec.Code != http.StatusFound {
		t.Fatalf("SaaS root: got %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/admin/" {
		t.Errorf("SaaS root Location = %q, want /admin/", loc)
	}
}

func TestSurfaceKeepsPricingAndPrivacyOnTheProductAndCompanyDomains(t *testing.T) {
	engine := surfaceEngine(t)
	for _, host := range []string{config.DefaultCompanyDomain, config.DefaultPOSDomain} {
		for path, want := range map[string][]string{
			"/pricing": {"Simple pricing", "Starter", "/private"},
			"/private": {"Privacy Policy", "support@xamltech.com"},
		} {
			rec := hostGet(t, engine, host, path)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s%s: got %d, want 200", host, path, rec.Code)
			}
			for _, fragment := range want {
				if !strings.Contains(rec.Body.String(), fragment) {
					t.Errorf("%s%s missing %q", host, path, fragment)
				}
			}
		}
	}
}

func TestSurfaceGatesTheAPIByDomain(t *testing.T) {
	engine := surfaceEngine(t)

	// The console host only carries the control plane.
	for _, path := range []string{"/v1/saas/summary", "/v1/saas/tenants", "/v1/platform/audit"} {
		if rec := hostGet(t, engine, config.DefaultSaaSDomain, path); rec.Code != http.StatusUnauthorized {
			t.Errorf("SaaS %s = %d, want 401 (reaches the saas_admin guard)", path, rec.Code)
		}
	}
	for _, path := range []string{"/v1/sales", "/v1/products", "/v1/sync/pull", "/v1/registers/current"} {
		rec := hostGet(t, engine, config.DefaultSaaSDomain, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("SaaS %s = %d, want 404", path, rec.Code)
		}
		if code := errorCode(t, rec); code != "not_available_on_host" {
			t.Errorf("SaaS %s error code = %q, want not_available_on_host", path, code)
		}
	}

	// The POS host only carries the store API.
	for _, path := range []string{"/v1/sales", "/v1/products", "/v1/sync/pull", "/v1/registers/current", "/v1/settings"} {
		if rec := hostGet(t, engine, config.DefaultPOSDomain, path); rec.Code != http.StatusUnauthorized {
			t.Errorf("POS %s = %d, want 401 (reaches the access-token guard)", path, rec.Code)
		}
	}
	for _, path := range []string{"/v1/saas/summary", "/v1/platform/audit"} {
		if rec := hostGet(t, engine, config.DefaultPOSDomain, path); rec.Code != http.StatusNotFound {
			t.Errorf("POS %s = %d, want 404", path, rec.Code)
		}
	}

	// The company host serves no API.
	if rec := hostGet(t, engine, config.DefaultCompanyDomain, "/v1/sales"); rec.Code != http.StatusNotFound {
		t.Errorf("company /v1/sales = %d, want 404", rec.Code)
	}

	// Probes and reference data stay reachable everywhere the API is.
	for _, host := range []string{config.DefaultSaaSDomain, config.DefaultPOSDomain} {
		if rec := hostGet(t, engine, host, "/health/live"); rec.Code != http.StatusOK {
			t.Errorf("%s /health/live = %d, want 200", host, rec.Code)
		}
		if rec := hostGet(t, engine, host, "/v1/meta/countries"); rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s /v1/meta/countries = %d, want 503 (no pool)", host, rec.Code)
		}
	}
}

func TestSurfaceRejectsUnconfiguredHosts(t *testing.T) {
	engine := surfaceEngine(t)
	for _, host := range []string{"evil.com", "xamltech.com.attacker.io", "shop.xamltech.com"} {
		rec := hostGet(t, engine, host, "/health/live")
		if rec.Code != http.StatusNotFound {
			t.Errorf("host %q: got %d, want 404", host, rec.Code)
		}
		if code := errorCode(t, rec); code != "unknown_host" {
			t.Errorf("host %q error code = %q, want unknown_host", host, code)
		}
	}
}

func TestSurfaceDisabledKeepsSingleDomainBehaviour(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})
	// With routing off every host still serves the POS.Go landing and the whole
	// API — the pre-split behaviour.
	for _, host := range []string{config.DefaultSaaSDomain, config.DefaultPOSDomain, "localhost"} {
		if rec := hostGet(t, engine, host, "/"); rec.Code != http.StatusOK ||
			!strings.Contains(rec.Body.String(), "Point of sale that keeps selling") {
			t.Errorf("%s / = %d, want the POS.Go landing", host, rec.Code)
		}
		if rec := hostGet(t, engine, host, "/v1/sales"); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s /v1/sales = %d, want 401", host, rec.Code)
		}
	}
}

func TestSurfaceCompanyHomeServesArabicRTL(t *testing.T) {
	engine := surfaceEngine(t)
	rec := hostGet(t, engine, config.DefaultCompanyDomain, "/?lang=ar")
	if rec.Code != http.StatusOK {
		t.Fatalf("company home (ar): got %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{`lang="ar"`, `dir="rtl"`, "ما نبنيه", "كيف نعمل", "ابدأ من هنا", "English"} {
		if !strings.Contains(body, want) {
			t.Errorf("company home (ar) missing %q", want)
		}
	}
	if v := rec.Header().Get("Vary"); !strings.Contains(v, "Accept-Language") {
		t.Errorf("company home missing Vary: Accept-Language")
	}
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error body: %v (body=%s)", err, rec.Body.String())
	}
	return payload.Error.Code
}

// Register is the only place the engine's trusted-proxy list is configured,
// and the OpenAPI generator and every handler test build their engine through
// it, so this is the regression guard for the X-Forwarded-For spoofing hole.
func TestRegisterRestrictsTheTrustedProxyList(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})

	if err := engine.SetTrustedProxies([]string{"0.0.0.0/0", "::/0"}); err != nil {
		t.Fatalf("SetTrustedProxies: %v", err)
	}
	engine.GET("/ip", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if got := recorder.Body.String(); got != "203.0.113.9" {
		t.Fatalf("ClientIP() = %q, want the spoofed entry to be ignored as 127.0.0.1", got)
	}
}

func TestRegisterHonoursATrustedProxyOverride(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{
		Config: config.Config{TrustedProxyCIDRs: []string{"127.0.0.1/32"}},
	})
	engine.GET("/ip", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})

	// Cloudflare is no longer trusted, so the edge hop itself is the answer.
	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = "127.0.0.1:5555"
	req.Header.Set("X-Forwarded-For", "203.0.113.9, 104.16.0.1")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if got := recorder.Body.String(); got != "104.16.0.1" {
		t.Fatalf("ClientIP() = %q, want the untrusted edge hop 104.16.0.1", got)
	}
}
