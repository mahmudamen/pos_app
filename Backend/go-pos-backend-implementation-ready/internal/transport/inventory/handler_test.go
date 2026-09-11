package inventory

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

func validToken(t *testing.T) string {
	t.Helper()
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004",
		"manager")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func setup() (*gin.Engine, *gin.RouterGroup) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router, router.Group("/v1")
}

func TestRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	routes := map[string]bool{}
	for _, r := range router.Routes() {
		routes[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		"POST /v1/inventory/adjustments",
		"GET /v1/inventory/adjustments",
	} {
		if !routes[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}

func TestAdjustmentRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/inventory/adjustments",
		strings.NewReader(`{"product_id":"00000000-0000-0000-0000-000000000001","reason":"restock","quantity_delta":10}`)))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestListAdjustmentsRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/inventory/adjustments", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestAdjustmentUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/inventory/adjustments",
		strings.NewReader(`{"product_id":"00000000-0000-0000-0000-000000000001","reason":"restock","quantity_delta":10}`))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestAdjustmentRejectsInvalidReason(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, body := range []string{
		`{"product_id":"00000000-0000-0000-0000-000000000001","reason":"stolen","quantity_delta":2}`,
		`{"product_id":"00000000-0000-0000-0000-000000000001","reason":"","quantity_delta":2}`,
	} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/v1/inventory/adjustments", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+validToken(t))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", body, recorder.Code)
		}
	}
}

func TestAdjustmentRejectsZeroDelta(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/inventory/adjustments",
		strings.NewReader(`{"product_id":"00000000-0000-0000-0000-000000000001","reason":"count","quantity_delta":0}`))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdjustmentRejectsInvalidProduct(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/inventory/adjustments",
		strings.NewReader(`{"product_id":"not-a-uuid","reason":"count","quantity_delta":5}`))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdjustmentRejectsLongNote(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	long := strings.Repeat("a", 256)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/inventory/adjustments",
		strings.NewReader(`{"product_id":"00000000-0000-0000-0000-000000000001","reason":"count","quantity_delta":5,"note":"`+long+`"}`))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestAdjustmentsUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/inventory/adjustments", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestAdjustmentRequiresManagerPermission(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004",
		"cashier")
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/inventory/adjustments",
		strings.NewReader(`{"product_id":"00000000-0000-0000-0000-000000000001","reason":"restock","quantity_delta":10}`))
	request.Header.Set("Authorization", "Bearer "+raw)
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestValidReason(t *testing.T) {
	for _, reason := range []string{"damaged", "restock", "count"} {
		if !validReason(reason) {
			t.Fatalf("expected %s to be valid", reason)
		}
	}
	for _, reason := range []string{"", "stolen", "transfer", "sale"} {
		if validReason(reason) {
			t.Fatalf("expected %s to be invalid", reason)
		}
	}
}
