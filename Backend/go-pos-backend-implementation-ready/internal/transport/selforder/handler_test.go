package selforder

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

var testVapid = WebPushConfig{PublicKey: "", PrivateKey: "", Subject: "mailto:test@example.com"}

func request(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body == "" {
		req.Body = http.NoBody
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func authed(router *gin.Engine, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/v1/self-orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestRoutesRegistered(t *testing.T) {
	router, group := setup()
	h := NewHandler(nil, testTokens(), testVapid, 5)
	h.RegisterPublic(group)
	h.Register(group)
	routes := map[string]bool{}
	for _, r := range router.Routes() {
		routes[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		"GET /v1/selforder/menu/:tenant",
		"POST /v1/selforder/orders",
		"POST /v1/selforder/requests",
		"GET /v1/self-orders",
		"POST /v1/self-orders/:id/approve",
		"POST /v1/self-orders/:id/cancel",
		"GET /v1/product-requests",
		"POST /v1/product-requests/:id/fulfill",
		"POST /v1/product-requests/:id/close",
	} {
		if !routes[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}

// Public self-order endpoints must answer WITHOUT a bearer token. With a nil
// pool they 503 database_unavailable (they never 401).
func TestPublicRoutesNeedNoToken(t *testing.T) {
	router, group := setup()
	h := NewHandler(nil, testTokens(), testVapid, 5)
	h.RegisterPublic(group)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/v1/selforder/menu/mystore", ""},
		{http.MethodPost, "/v1/selforder/orders", `{"tenant":"mystore","items":[{"product_id":"00000000-0000-0000-0000-000000000001","quantity":1}]}`},
		{http.MethodPost, "/v1/selforder/requests", `{"tenant":"mystore","product_name":"Soda"}`},
	} {
		rec := request(router, tc.method, tc.path, tc.body)
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s = %d, want 503 database_unavailable (nil pool)", tc.method, tc.path, rec.Code)
		}
	}
}

func TestPublicRoutesRejectBadOrderBody(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens(), testVapid, 5).RegisterPublic(group)
	// With a nil pool the handler answers 503 (never 401, never a panic).
	rec := request(router, http.MethodPost, "/v1/selforder/orders", `{"tenant":"mystore","items":[]}`)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("empty items = %d, want 503", rec.Code)
	}
	rec = request(router, http.MethodPost, "/v1/selforder/orders", `{"tenant":"mystore","items":[{"product_id":"x","quantity":200}]}`)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("quantity 200 = %d, want 503", rec.Code)
	}
}

func TestStaffRoutesRequireAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens(), testVapid, 5).Register(group)
	rec := authed(router, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token = %d, want 401", rec.Code)
	}
	rec = authed(router, "garbage-token")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad token = %d, want 401", rec.Code)
	}
}

func TestStaffRoutes503WithoutDB(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens(), testVapid, 5).Register(group)
	rec := authed(router, tokenForRole(t, "manager"))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("manager with nil pool = %d, want 503", rec.Code)
	}
}

func TestPageAndServiceWorker(t *testing.T) {
	router := gin.New()
	h := NewHandler(nil, testTokens(), WebPushConfig{PublicKey: "pubkey", PrivateKey: "priv", Subject: "s"}, 5)
	router.GET("/selforder", h.Page)
	router.GET("/sw.js", h.ServiceWorker)

	rec := request(router, http.MethodGet, "/selforder?tenant=mystore", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("/selforder = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "mystore") {
		t.Fatalf("page does not embed the tenant slug")
	}
	if !strings.Contains(body, "'pubkey'") {
		t.Fatalf("page does not embed the VAPID public key when configured")
	}

	rec = request(router, http.MethodGet, "/selforder", "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("/selforder without tenant = %d, want 400", rec.Code)
	}

	rec = request(router, http.MethodGet, "/sw.js", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "push") {
		t.Fatalf("/sw.js did not serve the service worker")
	}
}

func TestPageHidesNotifyWithoutVAPID(t *testing.T) {
	router := gin.New()
	h := NewHandler(nil, testTokens(), WebPushConfig{}, 5)
	router.GET("/selforder", h.Page)
	rec := request(router, http.MethodGet, "/selforder?tenant=mystore&lang=en", "")
	if !strings.Contains(rec.Body.String(), "VAPID_PUBLIC = ''") {
		t.Fatalf("page should embed an empty VAPID key when none is configured")
	}
}
