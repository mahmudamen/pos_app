package notifications

import (
	"context"
	"encoding/json"
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
		Issuer: "pos-api-test", AccessSecret: []byte("test-access-secret-0123456789abcdef"),
		RefreshSecret: []byte("test-refresh-secret-0123456789abcdef"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
}

func roleToken(t *testing.T, role string) string {
	t.Helper()
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", role)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestRegisterMountsNotificationRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens(), WebPushConfig{}).Register(router.Group("/v1"))
	for _, tc := range []struct {
		method, path string
	}{
		{http.MethodGet, "/v1/notifications"},
		{http.MethodGet, "/v1/notifications/count"},
		{http.MethodPost, "/v1/notifications/push-subscribe"},
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(tc.method, tc.path, nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401 without auth, got %d", tc.method, tc.path, recorder.Code)
		}
	}
}

func TestListReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens(), WebPushConfig{}).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/notifications", nil)
	request.Header.Set("Authorization", "Bearer "+roleToken(t, "manager"))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestCountReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens(), WebPushConfig{}).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/notifications/count", nil)
	request.Header.Set("Authorization", "Bearer "+roleToken(t, "manager"))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestNotificationsRequireManagerRBAC(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens(), WebPushConfig{}).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/notifications", nil)
	request.Header.Set("Authorization", "Bearer "+roleToken(t, "cashier"))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cashier expected 403, got %d", recorder.Code)
	}
}

func TestMarkReadValidatesID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens(), WebPushConfig{}).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/v1/notifications/not-a-uuid/read", nil)
	request.Header.Set("Authorization", "Bearer "+roleToken(t, "manager"))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestSubscribeValidatesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens(), WebPushConfig{}).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/notifications/push-subscribe",
		strings.NewReader(`{"endpoint":""}`))
	request.Header.Set("Authorization", "Bearer "+roleToken(t, "manager"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty endpoint, got %d", recorder.Code)
	}
}

func TestEmitterNoopWithoutPool(t *testing.T) {
	h := NewHandler(nil, testTokens(), WebPushConfig{})
	emit := h.Emitter()
	emit(context.Background(), "00000000-0000-0000-0000-000000000001", "", "info",
		"k", SeverityInfo, "t", "b", map[string]any{"a": "b"}) // must not panic / block
}

func TestWireShape(t *testing.T) {
	// subWire round-trips the PushSubscription JSONB the browser sends and the
	// push bridge decodes into a webpush Subscription.
	raw := `{"endpoint":"https://push.example/p1","keys":{"p256dh":"dHg=","auth":"YXV0aA=="}}`
	var wire subWire
	if err := json.Unmarshal([]byte(raw), &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Endpoint == "" || wire.Keys.P256dh == "" || wire.Keys.Auth == "" {
		t.Fatalf("push subscription decode lost fields: %+v", wire)
	}
	s, ok := decodeSub([]byte(raw))
	if !ok || s.Endpoint != "https://push.example/p1" {
		t.Fatalf("decodeSub failed: %+v", s)
	}
}
