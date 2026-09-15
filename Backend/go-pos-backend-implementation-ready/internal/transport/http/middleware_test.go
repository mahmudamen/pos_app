package http

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/ratelimit"
	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

func TestCORSAllowsPreflightRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS([]string{"*"}))
	router.POST("/v1/sales", func(c *gin.Context) { c.Status(http.StatusCreated) })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/v1/sales", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected wildcard CORS origin, got %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSAllowsNormalRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS([]string{"*"}))
	router.GET("/v1/products", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/products", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("expected wildcard CORS origin header")
	}
}

func TestCORSEchoesAllowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS([]string{"https://app.example.com", "https://admin.example.com"}))
	router.GET("/v1/products", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/products", nil)
	req.Header.Set("Origin", "https://app.example.com")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("expected echoed allow-origin, got %q", got)
	}
}

func TestCORSOmitsHeaderForDisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS([]string{"https://app.example.com"}))
	router.GET("/v1/products", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/products", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header for disallowed origin, got %q", got)
	}
}

func TestCORSOmitsHeaderForAllowlistWithoutOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS([]string{"https://app.example.com"}))
	router.GET("/v1/products", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/products", nil))
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS header, got %q", got)
	}
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Cache-Control":          "no-store",
	}
	for key, expected := range headers {
		if got := recorder.Header().Get(key); got != expected {
			t.Errorf("header %s: expected %q, got %q", key, expected, got)
		}
	}
	if h := recorder.Header().Get("Strict-Transport-Security"); h == "" {
		t.Error("expected Strict-Transport-Security header")
	}
}

func TestLoginRateLimitAllowsRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LoginRateLimit(3, time.Minute))
	router.POST("/login", func(c *gin.Context) { c.Status(http.StatusOK) })
	for i := 0; i < 3; i++ {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/login", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, recorder.Code)
		}
	}
}

func TestLoginRateLimitWithCustomLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LoginRateLimitWith(ratelimit.NewMemory(1, time.Minute)))
	router.POST("/login", func(c *gin.Context) { c.Status(http.StatusOK) })
	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodPost, "/login", nil))
	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodPost, "/login", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("expected 200 for first request, got %d", first.Code)
	}
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 for blocked request, got %d", second.Code)
	}
}

func TestApiRateLimitBlocksAfterLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ApiRateLimitWith(ratelimit.NewMemory(2, time.Minute)))
	router.GET("/v1/saas/summary", func(c *gin.Context) { c.Status(http.StatusOK) })
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/saas/summary", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, recorder.Code)
		}
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/saas/summary", nil))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}
}

func TestApiRateLimitResetsAfterWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ApiRateLimitWith(ratelimit.NewMemory(1, 50*time.Millisecond)))
	router.GET("/v1/saas/summary", func(c *gin.Context) { c.Status(http.StatusOK) })
	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/v1/saas/summary", nil))
	time.Sleep(60 * time.Millisecond)
	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/v1/saas/summary", nil))
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("expected both 200 after window reset, got %d then %d", first.Code, second.Code)
	}
}

func TestLoginRateLimitBlocksAfterLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LoginRateLimit(2, time.Minute))
	router.POST("/login", func(c *gin.Context) { c.Status(http.StatusOK) })
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/login", nil))
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/login", nil))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", recorder.Code)
	}
}

func TestLoginRateLimitResetsAfterWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LoginRateLimit(2, 50*time.Millisecond))
	router.POST("/login", func(c *gin.Context) { c.Status(http.StatusOK) })
	for i := 0; i < 2; i++ {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/login", nil))
	}
	time.Sleep(60 * time.Millisecond)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/login", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 after window reset, got %d", recorder.Code)
	}
}

func TestRequestIDGeneratesIDWhenMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		id := c.GetString("request_id")
		if id == "" {
			t.Fatal("expected request_id to be set")
		}
		c.Status(http.StatusOK)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected X-Request-ID header")
	}
}

func TestRequestIDPreservesExistingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/test", nil)
	request.Header.Set("X-Request-ID", "my-custom-id")
	router.ServeHTTP(recorder, request)
	if recorder.Header().Get("X-Request-ID") != "my-custom-id" {
		t.Fatalf("expected preserved request ID, got %q", recorder.Header().Get("X-Request-ID"))
	}
}

func TestRecoveryMiddlewareCatchesPanics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(&dummyHandler{})
	router := gin.New()
	router.Use(Recovery(logger))
	router.GET("/panic", func(c *gin.Context) { panic("test panic") })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/panic", nil))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "internal server error") {
		t.Errorf("expected error message in body: %s", recorder.Body.String())
	}
}

func TestMaxBodySizeRejectsOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MaxBodySize(10))
	router.POST("/test", func(c *gin.Context) {
		_, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})
	body := strings.Repeat("x", 11)
	request := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", recorder.Code)
	}
}

func TestMaxBodySizeAllowsSmallBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MaxBodySize(100))
	router.POST("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
	body := "small"
	request := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
}

func TestRandomIDReturnsHex(t *testing.T) {
	id := randomID()
	if id == "" {
		t.Fatal("expected non-empty random ID")
	}
	if len(id) != 32 {
		t.Fatalf("expected 32 hex chars, got %d", len(id))
	}
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Fatalf("expected hex character, got %q", string(c))
		}
	}
}

func TestLoginRateLimitReturnsRateLimitedErrorJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(LoginRateLimit(1, time.Minute))
	router.POST("/login", func(c *gin.Context) { c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/login", nil))
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/login", nil))
	if !strings.Contains(recorder.Body.String(), "rate_limited") {
		t.Errorf("expected rate_limited error code: %s", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "too many login attempts") {
		t.Errorf("expected rate limit message: %s", recorder.Body.String())
	}
}

type dummyHandler struct{}

func (d *dummyHandler) Enabled(_ context.Context, _ slog.Level) bool  { return true }
func (d *dummyHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (d *dummyHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return d }
func (d *dummyHandler) WithGroup(_ string) slog.Handler               { return d }

func TestRequestLoggerEmitsRequestFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestLogger(logger))
	router.GET("/v1/health/live", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/health/live", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
	out := buf.String()
	if !strings.Contains(out, `msg="http request"`) {
		t.Fatalf("expected request log line, got %q", out)
	}
	for _, want := range []string{"method=GET", "path=/v1/health/live", "status=204"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in log output: %s", want, out)
		}
	}
}

func TestRequestLoggerIncludesClaimsWhenPresent(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestLogger(logger))
	router.Use(func(c *gin.Context) {
		c.Set(claimsKey, security.Claims{
			TenantID: "tenant-1", UserID: "user-1", DeviceID: "dev-1", Role: "manager",
		})
		c.Next()
	})
	router.GET("/v1/sales", func(c *gin.Context) { c.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/sales", nil))
	out := buf.String()
	if !strings.Contains(out, `msg="http request"`) {
		t.Fatalf("expected request log line, got %q", out)
	}
	for _, want := range []string{"tenant_id=tenant-1", "user_id=user-1", "device_id=dev-1", "role=manager"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in log output: %s", want, out)
		}
	}
}
