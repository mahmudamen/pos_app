package telemetry

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

func handlerTokens() security.TokenManager {
	return security.TokenManager{
		Issuer: "pos-api-test", AccessSecret: []byte("test-access-secret-0123456789abcdef"),
		RefreshSecret: []byte("test-refresh-secret-0123456789abcdef"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
}

func handlerToken(t *testing.T) string {
	t.Helper()
	raw, err := handlerTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", "owner")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, handlerTokens()).Register(router.Group("/v1"))
	return router
}

func serve(t *testing.T, router *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	return body.Error.Code
}

func TestClientEventsRouteRegistered(t *testing.T) {
	router := newRouter()
	recorder := serve(t, router, http.MethodPost, "/v1/client/events", "", "")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected route to exist (401 without auth), got %d", recorder.Code)
	}
}

func TestPostEventRequiresAuth(t *testing.T) {
	router := newRouter()
	recorder := serve(t, router, http.MethodPost, "/v1/client/events",
		`{"event":"crash"}`, "")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestPostEventInvalidToken(t *testing.T) {
	router := newRouter()
	recorder := serve(t, router, http.MethodPost, "/v1/client/events",
		`{"event":"crash"}`, "not-a-token")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestPostEventRejectsBadJSON(t *testing.T) {
	router := newRouter()
	recorder := serve(t, router, http.MethodPost, "/v1/client/events",
		`{not json`, handlerToken(t))
	if recorder.Code != http.StatusBadRequest || errorCode(t, recorder) != "validation_error" {
		t.Fatalf("expected 400 validation_error, got %d %q", recorder.Code, recorder.Body.String())
	}
}

func TestPostEventRejectsMissingEvent(t *testing.T) {
	router := newRouter()
	recorder := serve(t, router, http.MethodPost, "/v1/client/events",
		`{"screen":"pos"}`, handlerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing event, got %d", recorder.Code)
	}
}

func TestPostEventRejectsBlankAndOversizedEvent(t *testing.T) {
	router := newRouter()
	for _, event := range []string{"   ", strings.Repeat("x", 65)} {
		recorder := serve(t, router, http.MethodPost, "/v1/client/events",
			`{"event":"`+event+`"}`, handlerToken(t))
		if recorder.Code != http.StatusBadRequest || errorCode(t, recorder) != "validation_error" {
			t.Fatalf("expected 400 validation_error for %q, got %d %q", event, recorder.Code, recorder.Body.String())
		}
	}
}

func TestPostEventRejectsOversizedFields(t *testing.T) {
	router := newRouter()
	recorder := serve(t, router, http.MethodPost, "/v1/client/events",
		`{"event":"crash","stack_trace":"`+strings.Repeat("s", 10001)+`"}`, handlerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized stack_trace, got %d", recorder.Code)
	}
}

func TestPostEventUnavailableWithoutDB(t *testing.T) {
	router := newRouter()
	recorder := serve(t, router, http.MethodPost, "/v1/client/events",
		`{"event":"crash","payload":{"item":1}}`, handlerToken(t))
	if recorder.Code != http.StatusServiceUnavailable || errorCode(t, recorder) != "database_unavailable" {
		t.Fatalf("expected 503 database_unavailable, got %d %q", recorder.Code, recorder.Body.String())
	}
}
