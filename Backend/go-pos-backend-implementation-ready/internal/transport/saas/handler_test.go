package saas

import (
	"net/http"
	"net/http/httptest"
	"regexp"
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

func TestSlugSubdomain(t *testing.T) {
	cases := []struct {
		slug, want string
	}{
		{"demo-restaurant", "demo-restaurant.xamltech.com"},
		{"nile-cafe", "nile-cafe.xamltech.com"},
		{"", ".xamltech.com"},
		{"store42", "store42.xamltech.com"},
	}
	for _, tc := range cases {
		if got := slugSubdomain(tc.slug); got != tc.want {
			t.Errorf("slugSubdomain(%q) = %q, want %q", tc.slug, got, tc.want)
		}
	}
}

func TestJsonFeatures(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
		nil_ bool
	}{
		{"empty string", "", nil, false},
		{"empty array", "[]", nil, false},
		{"null literal", "null", nil, false},
		{"valid array", `["pos.basic","restaurant","pos.subdomain"]`, []string{"pos.basic", "restaurant", "pos.subdomain"}, false},
		{"single feature", `["pos.subdomain"]`, []string{"pos.subdomain"}, false},
		{"malformed", `["oops`, nil, true},
		{"not an array", `"pos.subdomain"`, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := jsonFeatures(tc.raw)
			if tc.nil_ {
				if got != nil {
					t.Errorf("jsonFeatures(%q) = %v, want nil", tc.raw, got)
				}
				return
			}
			if len(got) != len(tc.want) {
				t.Fatalf("jsonFeatures(%q) = %v, want %v", tc.raw, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("jsonFeatures(%q) = %v, want %v", tc.raw, got, tc.want)
				}
			}
		})
	}
}

func TestTenantDefaults(t *testing.T) {
	t.Run("filled defaults when empty", func(t *testing.T) {
		country, currency, language := tenantDefaults("", "", "")
		if country != "EG" || currency != "EGP" || language != "ar" {
			t.Fatalf("got %q/%q/%q, want EG/EGP/ar", country, currency, language)
		}
	})
	t.Run("preserves provided values", func(t *testing.T) {
		country, currency, language := tenantDefaults("US", "USD", "en")
		if country != "US" || currency != "USD" || language != "en" {
			t.Fatalf("got %q/%q/%q, want US/USD/en", country, currency, language)
		}
	})
	t.Run("whitespace treated as empty", func(t *testing.T) {
		country, currency, language := tenantDefaults("  ", "  ", "   ")
		if country != "EG" || currency != "EGP" || language != "ar" {
			t.Fatalf("got %q/%q/%q, want EG/EGP/ar", country, currency, language)
		}
	})
}

func TestTenantSlug(t *testing.T) {
	hex := regexp.MustCompile(`^[0-9a-f]{8}$`)

	cases := []struct{ name, prefix string }{
		{"Cafe Nile", "cafe-nile"},
		{"café & NILE", "caf-nile"},
		{"   ", "store"},
		{"!!!", "store"},
	}
	for _, tc := range cases {
		slug := tenantSlug(tc.name)
		if !strings.HasPrefix(slug, tc.prefix+"-") {
			t.Errorf("tenantSlug(%q) = %q, want prefix %q-", tc.name, slug, tc.prefix)
		}
		suffix := strings.TrimPrefix(slug, tc.prefix+"-")
		if len(suffix) != 8 || !hex.MatchString(suffix) {
			t.Errorf("tenantSlug(%q) suffix %q, want 8 lowercase hex chars", tc.name, suffix)
		}
	}
}
