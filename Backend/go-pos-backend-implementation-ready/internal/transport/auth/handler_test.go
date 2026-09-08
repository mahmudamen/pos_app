package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

func TestLoginReturnsUnavailableWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := config.Config{
		JWTIssuer: "pos-api", JWTAccessSecret: "access-secret-that-is-at-least-32-bytes",
		JWTRefreshSecret: "refresh-secret-that-is-at-least-32-bytes",
		JWTAccessTTL:     time.Minute, JWTRefreshTTL: time.Hour,
	}
	NewHandler(nil, cfg).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{"email":"cashier@example.com","password":"password","tenant_id":"tenant","device_id":"device","device_name":"Counter 1"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRefreshReturnsUnauthorizedForInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := config.Config{
		JWTIssuer: "pos-api", JWTAccessSecret: "access-secret-that-is-at-least-32-bytes",
		JWTRefreshSecret: "refresh-secret-that-is-at-least-32-bytes",
	}
	NewHandler(nil, cfg).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", strings.NewReader(`{"refresh_token":"invalid"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestLogoutReturnsUnavailableWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cfg := config.Config{
		JWTIssuer: "pos-api", JWTAccessSecret: "access-secret-that-is-at-least-32-bytes",
		JWTRefreshSecret: "refresh-secret-that-is-at-least-32-bytes",
		JWTAccessTTL:     time.Minute, JWTRefreshTTL: time.Hour,
	}
	handler := NewHandler(nil, cfg)
	raw, err := handler.Tokens().Issue(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004")
	if err != nil {
		t.Fatal(err)
	}
	handler.Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}
