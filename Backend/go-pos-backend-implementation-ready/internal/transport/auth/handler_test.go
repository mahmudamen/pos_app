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

func testConfig() config.Config {
	return config.Config{
		JWTIssuer: "pos-api", JWTAccessSecret: "access-secret-that-is-at-least-32-bytes",
		JWTRefreshSecret: "refresh-secret-that-is-at-least-32-bytes",
		JWTAccessTTL:     time.Minute, JWTRefreshTTL: time.Hour, BcryptCost: 4,
	}
}

func TestLoginReturnsUnavailableWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testConfig()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"cashier@example.com","password":"password","tenant_id":"tenant","device_id":"device","device_name":"Counter 1"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestLoginReturnsBadRequestForInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testConfig()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader("not json"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestLoginReturnsBadRequestForMissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testConfig()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"test@example.com"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestLoginReturnsBadRequestForEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testConfig()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRefreshReturnsUnauthorizedForInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testConfig()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"refresh_token":"invalid"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRefreshReturnsBadRequestForInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testConfig()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", strings.NewReader("not json"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRefreshReturnsBadRequestForMissingRefreshToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testConfig()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", strings.NewReader(`{}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRefreshReturnsUnavailableWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(nil, testConfig())
	tokens := handler.Tokens()
	refreshToken, err := tokens.Issue(time.Now(), security.RefreshToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004")
	if err != nil {
		t.Fatal(err)
	}
	handler.Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"refresh_token":"` + refreshToken + `"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestLogoutReturnsUnauthorizedWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testConfig()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestLogoutReturnsUnavailableWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewHandler(nil, testConfig())
	tokens := handler.Tokens()
	raw, err := tokens.Issue(time.Now(), security.AccessToken,
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

func TestLoginRejectsInvalidEmailFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testConfig()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"not-an-email","password":"password","tenant_id":"tenant","device_id":"device","device_name":"Counter 1"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid email, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestTokensAccessorReturnsSameTokens(t *testing.T) {
	handler := NewHandler(nil, testConfig())
	tokens := handler.Tokens()
	if tokens.Issuer != "pos-api" {
		t.Errorf("expected issuer 'pos-api', got %q", tokens.Issuer)
	}
}

func TestRegisterSetsUpRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testConfig()).Register(router.Group("/v1"))
	routes := router.Routes()
	paths := make(map[string]bool)
	for _, r := range routes {
		paths[r.Method+" "+r.Path] = true
	}
	for _, expected := range []string{"POST /v1/auth/login", "POST /v1/auth/refresh", "POST /v1/auth/logout"} {
		if !paths[expected] {
			t.Errorf("missing route: %s", expected)
		}
	}
}
