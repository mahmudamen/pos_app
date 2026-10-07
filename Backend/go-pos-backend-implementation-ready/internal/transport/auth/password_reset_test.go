package auth

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/identity"
	"github.com/gin-gonic/gin"
)

// doPostFromIP is doPost with a caller-chosen client address, so the per-IP
// reset limiter can be exercised without gin's proxy-header configuration
// getting a vote.
func doPostFromIP(t *testing.T, router http.Handler, path, body, ip string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":41000"
	router.ServeHTTP(rec, req)
	return rec
}

func resetUnitRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, authConfig()).Register(router.Group("/v1"))
	return router
}

func TestForgotPasswordRejectsInvalidBody(t *testing.T) {
	router := resetUnitRouter()
	for name, body := range map[string]string{
		"not json":     `{`,
		"empty":        `{}`,
		"missing mail": `{"tenant_id":"demo"}`,
		"bad email":    `{"tenant_id":"demo","email":"not-an-email"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := doPost(t, router, "/v1/auth/forgot-password", body, "")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "validation_error") {
				t.Fatalf("expected validation_error, got %s", rec.Body.String())
			}
		})
	}
}

func TestForgotPasswordUnavailableWithoutDatabase(t *testing.T) {
	router := resetUnitRouter()
	rec := doPost(t, router, "/v1/auth/forgot-password",
		`{"tenant_id":"demo","email":"owner@example.com"}`, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// The limiter counts every request, including the ones that never reach the
// database, so a caller cannot use an unreachable pool as an excuse to hammer
// the route — and the counter is per IP, so one client's abuse does not lock
// out the rest of the network.
func TestForgotPasswordRateLimitsPerIP(t *testing.T) {
	router := resetUnitRouter()
	body := `{"tenant_id":"demo","email":"owner@example.com"}`
	for i := 1; i <= identity.PasswordResetRateLimit; i++ {
		rec := doPostFromIP(t, router, "/v1/auth/forgot-password", body, "198.51.100.9")
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("attempt %d: expected 503 (no database), got %d", i, rec.Code)
		}
	}
	rec := doPostFromIP(t, router, "/v1/auth/forgot-password", body, "198.51.100.9")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after %d attempts, got %d body=%s",
			identity.PasswordResetRateLimit, rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "rate_limited") {
		t.Fatalf("expected rate_limited, got %s", rec.Body.String())
	}
	other := doPostFromIP(t, router, "/v1/auth/forgot-password", body, "203.0.113.4")
	if other.Code != http.StatusServiceUnavailable {
		t.Fatalf("a different client must not be throttled: got %d", other.Code)
	}
}

func TestResetPasswordRejectsInvalidBody(t *testing.T) {
	router := resetUnitRouter()
	for name, body := range map[string]string{
		"not json":      `{`,
		"empty":         `{}`,
		"missing token": `{"password":"longenough"}`,
		"blank token":   `{"token":"   ","password":"longenough"}`,
		"short":         `{"token":"abc","password":"short"}`,
	} {
		t.Run(name, func(t *testing.T) {
			rec := doPost(t, router, "/v1/auth/reset-password", body, "")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestResetPasswordUnavailableWithoutDatabase(t *testing.T) {
	router := resetUnitRouter()
	rec := doPost(t, router, "/v1/auth/reset-password",
		`{"token":"deadbeef","password":"longenough"}`, "")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rec.Code, rec.Body.String())
	}
}

// The rendered mail says "30 minutes"; identity.PasswordResetTTL is what the
// database enforces. If the two ever drift the user is told a deadline the
// server does not honour.
func TestPasswordResetExpiryLabelMatchesPolicy(t *testing.T) {
	if identity.PasswordResetTTL != 30*time.Minute {
		t.Fatalf("PasswordResetTTL = %s, want 30m", identity.PasswordResetTTL)
	}
	if passwordResetExpiryLabel != "30 minutes" {
		t.Fatalf("expiry label = %q, want %q", passwordResetExpiryLabel, "30 minutes")
	}
	if resetLinkTTL() != identity.PasswordResetTTL {
		t.Fatalf("resetLinkTTL() = %s, want %s", resetLinkTTL(), identity.PasswordResetTTL)
	}
}

func TestRequestOrigin(t *testing.T) {
	for _, tc := range []struct {
		name   string
		host   string
		proto  string
		tls    bool
		expect string
	}{
		{name: "behind proxy", host: "api.xamltech.com", proto: "https", expect: "https://api.xamltech.com"},
		{name: "plain http", host: "localhost:8080", expect: "http://localhost:8080"},
		{name: "terminated tls", host: "api.example.com", tls: true, expect: "https://api.example.com"},
		{name: "bogus proto falls back", host: "api.example.com", proto: "ftp", expect: "http://api.example.com"},
		{name: "no host", host: "", expect: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			req := httptest.NewRequest(http.MethodPost, "/v1/auth/forgot-password", nil)
			req.Host = tc.host
			if tc.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tc.proto)
			}
			if tc.tls {
				req.TLS = &tls.ConnectionState{}
			}
			c.Request = req
			if got := requestOrigin(c); got != tc.expect {
				t.Fatalf("requestOrigin = %q, want %q", got, tc.expect)
			}
		})
	}
}
