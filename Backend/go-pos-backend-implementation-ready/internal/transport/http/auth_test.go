package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

func TestRequireAccessTokenStoresClaims(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tokens := security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
	raw, err := tokens.Issue(time.Now(), security.AccessToken, "tenant", "user", "device", "session")
	if err != nil {
		t.Fatal(err)
	}
	router := gin.New()
	router.Use(RequestID(), RequireAccessToken(tokens))
	router.GET("/protected", func(c *gin.Context) {
		claims, ok := Claims(c)
		if !ok || claims.TenantID != "tenant" {
			t.Fatal("claims were not stored")
		}
		c.Status(http.StatusNoContent)
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
}

func TestRequireAccessTokenRejectsMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequireAccessToken(security.TokenManager{}))
	router.GET("/protected", func(c *gin.Context) { t.Fatal("protected handler ran") })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestRequireAccessTokenRejectsMalformedToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	tokens := security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
	router.Use(RequireAccessToken(tokens))
	router.GET("/protected", func(c *gin.Context) { t.Fatal("protected handler ran") })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer not-a-real-jwt")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestRequireAccessTokenRejectsWrongType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	tokens := security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
	refreshToken, err := tokens.Issue(time.Now(), security.RefreshToken, "tenant", "user", "device", "session")
	if err != nil {
		t.Fatal(err)
	}
	router.Use(RequireAccessToken(tokens))
	router.GET("/protected", func(c *gin.Context) { t.Fatal("protected handler ran") })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+refreshToken)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestRequireAccessTokenRejectsExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	tokens := security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
	raw, err := tokens.Issue(time.Now().Add(-2*time.Minute), security.AccessToken, "tenant", "user", "device", "session")
	if err != nil {
		t.Fatal(err)
	}
	router.Use(RequireAccessToken(tokens))
	router.GET("/protected", func(c *gin.Context) { t.Fatal("protected handler ran") })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestRequireAccessTokenRejectsBearerLowercase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	tokens := security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
	raw, err := tokens.Issue(time.Now(), security.AccessToken, "tenant", "user", "device", "session")
	if err != nil {
		t.Fatal(err)
	}
	router.Use(RequireAccessToken(tokens))
	router.GET("/protected", func(c *gin.Context) { t.Fatal("protected handler ran") })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "bearer "+raw)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for lowercase 'bearer', got %d", recorder.Code)
	}
}

func TestClaimsReturnsFalseWhenNotSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c := &gin.Context{}
	claims, ok := Claims(c)
	if ok {
		t.Fatal("Claims should return false when key not set")
	}
	if claims.TenantID != "" {
		t.Fatal("claims should be zero value")
	}
}
