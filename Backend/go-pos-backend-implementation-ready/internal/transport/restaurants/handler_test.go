package restaurants

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
	session := struct {
		requestID string
	}{}
	router.Use(func(c *gin.Context) {
		session.requestID = "test-request"
		c.Set("request_id", session.requestID)
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
		"GET /v1/floors",
		"POST /v1/floors",
		"GET /v1/tables",
		"POST /v1/tables",
		"PATCH /v1/tables/:id",
		"POST /v1/sales/:id/split",
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
		{http.MethodGet, "/v1/floors", ""},
		{http.MethodPost, "/v1/floors", `{"name":"Ground"}`},
		{http.MethodGet, "/v1/tables", ""},
		{http.MethodPost, "/v1/tables", `{"floor_id":"00000000-0000-0000-0000-000000000001","name":"1"}`},
		{http.MethodPatch, "/v1/tables/00000000-0000-0000-0000-000000000001", `{"name":"2"}`},
		{http.MethodPost, "/v1/sales/00000000-0000-0000-0000-000000000001/split", `{"items":[{"product_id":"00000000-0000-0000-0000-000000000002","quantity":1}]}`},
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
		{http.MethodGet, "/v1/floors", ""},
		{http.MethodPost, "/v1/floors", `{"name":"Ground"}`},
		{http.MethodGet, "/v1/tables", ""},
		{http.MethodPost, "/v1/tables", `{"floor_id":"00000000-0000-0000-0000-000000000001","name":"1"}`},
		{http.MethodPost, "/v1/sales/00000000-0000-0000-0000-000000000001/split", `{"children":[{"lines":[{"sale_item_id":"00000000-0000-0000-0000-000000000003","quantity":1}]},{"lines":[{"sale_item_id":"00000000-0000-0000-0000-000000000004","quantity":1}]}]}`},
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
	rec := authedRequest(router, http.MethodPost, "/v1/floors", `{"name":"Ground"}`, tokenForRole(t, "cashier"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier write, got %d", rec.Code)
	}
}

func TestCreateFloorValidatesName(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	rec := authedRequest(router, http.MethodPost, "/v1/floors", `{"name":""}`, tokenForRole(t, "manager"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank floor name, got %d", rec.Code)
	}
}

func TestCreateTableValidatesFloor(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	rec := authedRequest(router, http.MethodPost, "/v1/tables", `{"floor_id":"not-a-uuid","name":"1"}`, tokenForRole(t, "manager"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid floor id, got %d", rec.Code)
	}
}

func TestSplitValidatesItems(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	rec := authedRequest(router, http.MethodPost, "/v1/sales/00000000-0000-0000-0000-000000000001/split", `{"items":[]}`, tokenForRole(t, "cashier"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty split items, got %d", rec.Code)
	}
}

func TestSplitRejectsInvalidSaleID(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	rec := authedRequest(router, http.MethodPost, "/v1/sales/not-a-uuid/split", `{"items":[{"product_id":"00000000-0000-0000-0000-000000000002","quantity":1}]}`, tokenForRole(t, "cashier"))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid sale id, got %d", rec.Code)
	}
}
