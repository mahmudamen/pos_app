package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/config"
	"github.com/gin-gonic/gin"
)

func testRouting() config.SurfaceRouting {
	return config.SurfaceRouting{
		Enabled:       true,
		CompanyDomain: config.DefaultCompanyDomain,
		SaaSDomain:    config.DefaultSaaSDomain,
		POSDomain:     config.DefaultPOSDomain,
	}
}

func TestRequiredSurfaceClassification(t *testing.T) {
	cases := map[string]config.Surface{
		// Platform control plane: console host only.
		"/v1/saas/summary":                config.SurfaceSaaS,
		"/v1/saas/tenants":                config.SurfaceSaaS,
		"/v1/saas/billing/summary":        config.SurfaceSaaS,
		"/v1/saas":                        config.SurfaceSaaS,
		"/v1/platform/tenants/:id/backup": config.SurfaceSaaS,
		// Session bootstrap and reference data: every surface.
		"/v1/auth/login":     config.SurfaceUnknown,
		"/v1/auth/refresh":   config.SurfaceUnknown,
		"/v1/auth/logout":    config.SurfaceUnknown,
		"/v1/meta/countries": config.SurfaceUnknown,
		// Store surface.
		"/v1/sales":          config.SurfacePOS,
		"/v1/categories":     config.SurfacePOS,
		"/v1/auth/register":  config.SurfacePOS,
		"/v1/auth/set-pin":   config.SurfacePOS,
		"/v1/sync/pull":      config.SurfacePOS,
		"/v1/selforder/menu": config.SurfacePOS,
		"/v1/community/jobs": config.SurfacePOS,
		// Outside the API: not gated.
		"/":                 config.SurfaceUnknown,
		"/private":          config.SurfaceUnknown,
		"/pricing":          config.SurfaceUnknown,
		"/health/live":      config.SurfaceUnknown,
		"/selforder":        config.SurfaceUnknown,
		"/screenshots/hero": config.SurfaceUnknown,
		// A near-miss prefix is not the platform group.
		"/v1/saasx/summary": config.SurfacePOS,
		// /v1/sales is the POS group even for nested paths.
		"/v1/sales/s1/receipt/print": config.SurfacePOS,
	}
	for path, want := range cases {
		if got := RequiredSurface(path); got != want {
			t.Errorf("RequiredSurface(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestAllowsRolePerSurface(t *testing.T) {
	cases := []struct {
		surface config.Surface
		role    string
		want    bool
	}{
		{config.SurfaceSaaS, "saas_admin", true},
		{config.SurfaceSaaS, "owner", false},
		{config.SurfaceSaaS, "manager", false},
		{config.SurfaceSaaS, "cashier", false},
		{config.SurfacePOS, "saas_admin", false},
		{config.SurfacePOS, "owner", true},
		{config.SurfacePOS, "manager", true},
		{config.SurfacePOS, "cashier", true},
		{config.SurfacePOS, "", false},
		{config.SurfaceUnknown, "saas_admin", true},
		{config.SurfaceUnknown, "cashier", true},
		{config.SurfaceCompany, "saas_admin", true},
	}
	for _, c := range cases {
		if got := AllowsRole(c.surface, c.role); got != c.want {
			t.Errorf("AllowsRole(%q, %q) = %v, want %v", c.surface, c.role, got, c.want)
		}
	}
}

func surfaceEngine(t *testing.T, routing config.SurfaceRouting) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(SurfaceRouting(routing))
	echo := func(c *gin.Context) { c.String(http.StatusOK, string(Surface(c))) }
	engine.GET("/v1/sales", echo)
	engine.POST("/v1/saas/summary", echo)
	engine.GET("/v1/auth/login", echo)
	engine.GET("/v1/meta/countries", echo)
	engine.GET("/health/live", echo)
	engine.GET("/", echo)
	engine.GET("/pricing", echo)
	engine.GET("/private", echo)
	engine.GET("/selforder", echo)
	engine.GET("/screenshots/hero", echo)
	engine.GET(googleVerifyPath, echo)
	return engine
}

func call(t *testing.T, engine *gin.Engine, host, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Host = host
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	return rec
}

func TestSurfaceRoutingSeparatesTheThreeDomains(t *testing.T) {
	engine := surfaceEngine(t, testRouting())

	// POS host serves the store API and refuses the control plane.
	if rec := call(t, engine, config.DefaultPOSDomain, http.MethodGet, "/v1/sales"); rec.Code != http.StatusOK ||
		rec.Body.String() != "pos" {
		t.Errorf("POS /v1/sales = %d %q, want 200 pos", rec.Code, rec.Body.String())
	}
	if rec := call(t, engine, config.DefaultPOSDomain, http.MethodPost, "/v1/saas/summary"); rec.Code != http.StatusNotFound {
		t.Errorf("POS /v1/saas/summary = %d, want 404", rec.Code)
	}

	// Console host serves the control plane and refuses the store API.
	if rec := call(t, engine, config.DefaultSaaSDomain, http.MethodPost, "/v1/saas/summary"); rec.Code != http.StatusOK ||
		rec.Body.String() != "saas" {
		t.Errorf("SaaS /v1/saas/summary = %d %q, want 200 saas", rec.Code, rec.Body.String())
	}
	if rec := call(t, engine, config.DefaultSaaSDomain, http.MethodGet, "/v1/sales"); rec.Code != http.StatusNotFound {
		t.Errorf("SaaS /v1/sales = %d, want 404", rec.Code)
	}

	// The company host serves no API at all — not even sign-in, so the brand
	// page cannot be used to mint a token.
	for _, path := range []string{"/v1/sales", "/v1/saas/summary", "/v1/auth/login", "/v1/meta/countries"} {
		rec := call(t, engine, config.DefaultCompanyDomain, http.MethodGet, path)
		if rec.Code != http.StatusNotFound {
			t.Errorf("company %s = %d, want 404", path, rec.Code)
		} else if !strings.Contains(rec.Body.String(), "not_available_on_host") {
			t.Errorf("company %s: body = %s, want not_available_on_host", path, rec.Body.String())
		}
	}

	// Session bootstrap and reference data answer on both API domains.
	for _, host := range []string{config.DefaultPOSDomain, config.DefaultSaaSDomain} {
		if rec := call(t, engine, host, http.MethodGet, "/v1/auth/login"); rec.Code != http.StatusOK {
			t.Errorf("%s /v1/auth/login = %d, want 200", host, rec.Code)
		}
		if rec := call(t, engine, host, http.MethodGet, "/v1/meta/countries"); rec.Code != http.StatusOK {
			t.Errorf("%s /v1/meta/countries = %d, want 200", host, rec.Code)
		}
	}
	if rec := call(t, engine, config.DefaultPOSDomain, http.MethodGet, "/health/live"); rec.Code != http.StatusOK {
		t.Errorf("company /health/live = %d, want 200 (probes stay shared)", rec.Code)
	}
}

func TestSurfaceRoutingPublishesOnlyTheRightPages(t *testing.T) {
	engine := surfaceEngine(t, testRouting())
	cases := []struct {
		host string
		path string
		want int
	}{
		// Company: brand page, pricing and the legal pages.
		{config.DefaultCompanyDomain, "/", http.StatusOK},
		{config.DefaultCompanyDomain, "/pricing", http.StatusOK},
		{config.DefaultCompanyDomain, "/private", http.StatusOK},
		{config.DefaultCompanyDomain, "/screenshots/hero", http.StatusOK},
		{config.DefaultCompanyDomain, "/selforder", http.StatusNotFound},
		{config.DefaultCompanyDomain, "/health/live", http.StatusOK},
		// POS: the install site plus self-ordering.
		{config.DefaultPOSDomain, "/", http.StatusOK},
		{config.DefaultPOSDomain, "/pricing", http.StatusOK},
		{config.DefaultPOSDomain, "/private", http.StatusOK},
		{config.DefaultPOSDomain, "/selforder", http.StatusOK},
		{config.DefaultPOSDomain, "/health/live", http.StatusOK},
		// Console: the redirect to the admin app, nothing else.
		{config.DefaultSaaSDomain, "/", http.StatusOK},
		{config.DefaultSaaSDomain, "/pricing", http.StatusNotFound},
		{config.DefaultSaaSDomain, "/private", http.StatusNotFound},
		{config.DefaultSaaSDomain, "/selforder", http.StatusNotFound},
		{config.DefaultSaaSDomain, "/screenshots/hero", http.StatusNotFound},
		{config.DefaultSaaSDomain, "/health/live", http.StatusOK},
	}
	for _, c := range cases {
		if rec := call(t, engine, c.host, http.MethodGet, c.path); rec.Code != c.want {
			t.Errorf("%s %s = %d, want %d", c.host, c.path, rec.Code, c.want)
		}
	}
}

func TestServesPageAndAPI(t *testing.T) {
	if ServesAPI(config.SurfaceCompany, "/v1/auth/login") {
		t.Error("the company domain must not publish sign-in")
	}
	if !ServesAPI(config.SurfaceSaaS, "/v1/auth/login") {
		t.Error("the console domain must publish sign-in")
	}
	if ServesAPI(config.SurfaceSaaS, "/v1/sales") {
		t.Error("the console domain must not publish the store API")
	}
	if ServesPage(config.SurfaceSaaS, "/") != true || ServesPage(config.SurfaceSaaS, "/private") != false {
		t.Error("the console domain publishes only its root redirect")
	}
	if !ServesPage(config.SurfacePOS, "/sw.js") || !ServesPage(config.SurfacePOS, "/apk/pos_go.apk") {
		t.Error("the POS domain publishes the service worker and the APK path")
	}
	if ServesPage(config.SurfacePOS, "/v1/sales") {
		t.Error("API paths are not pages")
	}
}

func TestSurfaceRoutingRejectsUnknownHosts(t *testing.T) {
	engine := surfaceEngine(t, testRouting())
	for _, host := range []string{"evil.com", "api.xamltech.com.evil.io", "posgo.xamltech.computer", "staging.xamltech.com"} {
		rec := call(t, engine, host, http.MethodGet, "/health/live")
		if rec.Code != http.StatusNotFound {
			t.Errorf("unknown host %q: got %d, want 404", host, rec.Code)
		}
	}
}

func TestSurfaceRoutingAllowsInternalHosts(t *testing.T) {
	engine := surfaceEngine(t, testRouting())
	for _, host := range []string{"localhost", "127.0.0.1:8080", "api", "caddy", "host.docker.internal"} {
		rec := call(t, engine, host, http.MethodGet, "/v1/sales")
		if rec.Code != http.StatusOK {
			t.Errorf("internal host %q: got %d, want 200", host, rec.Code)
		}
		if got := rec.Body.String(); got != "" {
			t.Errorf("internal host %q: surface = %q, want unknown (ungated)", host, got)
		}
	}
}

func TestSurfaceRoutingDisabledIsInert(t *testing.T) {
	engine := surfaceEngine(t, config.SurfaceRouting{})
	for _, host := range []string{config.DefaultPOSDomain, config.DefaultSaaSDomain, "anything.example"} {
		if rec := call(t, engine, host, http.MethodGet, "/v1/sales"); rec.Code != http.StatusOK {
			t.Errorf("disabled routing %q /v1/sales = %d, want 200", host, rec.Code)
		}
		if rec := call(t, engine, host, http.MethodPost, "/v1/saas/summary"); rec.Code != http.StatusOK {
			t.Errorf("disabled routing %q /v1/saas/summary = %d, want 200", host, rec.Code)
		}
	}
}

func TestSurfaceIsUnknownWithoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/", func(c *gin.Context) {
		if Surface(c) != config.SurfaceUnknown {
			t.Errorf("Surface = %q, want SurfaceUnknown", Surface(c))
		}
		c.Status(http.StatusOK)
	})
	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestAbortRoleNotOnSurfaceMessagesPerHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, c := range []struct {
		surface config.Surface
		want    string
	}{
		{config.SurfaceSaaS, "sign in with a platform administrator account"},
		{config.SurfacePOS, "platform administrators sign in on the console domain"},
		{config.SurfaceUnknown, "this account cannot sign in on this domain"},
	} {
		engine := gin.New()
		engine.Use(func(ctx *gin.Context) { ctx.Set(surfaceKey, c.surface) })
		engine.GET("/", func(ctx *gin.Context) { AbortRoleNotOnSurface(ctx) })
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusForbidden {
			t.Errorf("surface %q: status = %d, want 403", c.surface, rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "wrong_surface") || !strings.Contains(body, c.want) {
			t.Errorf("surface %q: body = %s, want code wrong_surface and %q", c.surface, body, c.want)
		}
	}
}

func TestGoogleVerificationPublishedOnEverySurface(t *testing.T) {
	engine := surfaceEngine(t, testRouting())
	for _, host := range []string{
		config.DefaultCompanyDomain, config.DefaultPOSDomain, config.DefaultSaaSDomain,
	} {
		rec := call(t, engine, host, http.MethodGet, googleVerifyPath)
		if rec.Code != http.StatusOK {
			t.Errorf("%s %s = %d, want 200", host, googleVerifyPath, rec.Code)
		}
	}
	// A near-miss token must not be published — the route is the exact file.
	if rec := call(t, engine, config.DefaultPOSDomain, http.MethodGet, "/google.html"); rec.Code != http.StatusNotFound {
		t.Errorf("/google.html = %d, want 404", rec.Code)
	}
}
