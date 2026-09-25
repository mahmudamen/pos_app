package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/infrastructure/ratelimit"
	"github.com/gin-gonic/gin"
)

func engineWithTrust(t *testing.T, trusted []string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	if err := ConfigureTrustedProxies(engine, trusted); err != nil {
		t.Fatalf("ConfigureTrustedProxies: %v", err)
	}
	engine.GET("/ip", func(c *gin.Context) {
		c.String(http.StatusOK, c.ClientIP())
	})
	return engine
}

func getWithForwardedFor(t *testing.T, engine *gin.Engine, peer, forwardedFor string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/ip", nil)
	req.RemoteAddr = peer
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status %d", recorder.Code)
	}
	return recorder.Body.String()
}

func TestClientIPIgnoresASpoofedLeftmostEntry(t *testing.T) {
	engine := engineWithTrust(t, config.DefaultTrustedProxies())

	// Cloudflare appends the address it observed to whatever the client sent,
	// so the real client sits second-from-the-right and the leftmost entry is
	// attacker-controlled. The old default (trust everything) returned it.
	got := getWithForwardedFor(t, engine, "127.0.0.1:5555", "203.0.113.9, 198.51.100.7, 104.16.0.1")
	if got != "198.51.100.7" {
		t.Errorf("ClientIP() = %q, want the Cloudflare-observed client 198.51.100.7", got)
	}
}

func TestClientIPWalksPastTrustedCloudflareHops(t *testing.T) {
	engine := engineWithTrust(t, config.DefaultTrustedProxies())

	// Only Cloudflare edge hops may sit to the right of the client.
	for _, edge := range []string{"104.16.0.1", "172.64.5.5", "198.41.128.7", "2400:cb00::1"} {
		got := getWithForwardedFor(t, engine, "127.0.0.1:5555", "1.2.3.4, 203.0.113.9, "+edge)
		if got != "203.0.113.9" {
			t.Errorf("edge %s: ClientIP() = %q, want 203.0.113.9", edge, got)
		}
	}
}

func TestClientIPFallsBackToThePeerWithoutAHeader(t *testing.T) {
	engine := engineWithTrust(t, config.DefaultTrustedProxies())

	got := getWithForwardedFor(t, engine, "203.0.113.55:4321", "")
	if got != "203.0.113.55" {
		t.Errorf("ClientIP() = %q, want the peer address 203.0.113.55", got)
	}
}

func TestClientIPDoesNotTrustAnUnlistedProxy(t *testing.T) {
	engine := engineWithTrust(t, config.DefaultTrustedProxies())

	// A header arriving from a peer that is not a trusted proxy is ignored
	// outright, because the peer is the only thing that can be believed.
	got := getWithForwardedFor(t, engine, "203.0.113.55:4321", "9.9.9.9")
	if got != "203.0.113.55" {
		t.Errorf("ClientIP() = %q, want the peer address 203.0.113.55", got)
	}
}

func TestClientIPStopsAtTheFirstUntrustedHop(t *testing.T) {
	engine := engineWithTrust(t, config.DefaultTrustedProxies())

	// Everything left of a private-range hop is discarded, so a merchant behind
	// a corporate NAT still gets its own bucket.
	got := getWithForwardedFor(t, engine, "127.0.0.1:5555", "1.1.1.1, 203.0.113.9, 10.0.0.7")
	if got != "203.0.113.9" {
		t.Errorf("ClientIP() = %q, want 203.0.113.9", got)
	}
}

func TestLoginRateLimitCannotBeResetByASpoofedHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	if err := ConfigureTrustedProxies(engine, config.DefaultTrustedProxies()); err != nil {
		t.Fatalf("ConfigureTrustedProxies: %v", err)
	}
	limiter := ratelimit.NewMemory(3, time.Minute)
	engine.POST("/v1/auth/login", LoginRateLimitWith(limiter), func(c *gin.Context) {
		c.Status(http.StatusUnauthorized)
	})

	attempt := func(forwardedFor string) int {
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/login", nil)
		req.RemoteAddr = "127.0.0.1:5555"
		req.Header.Set("X-Forwarded-For", forwardedFor)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, req)
		return recorder.Code
	}

	// Four attempts claiming four different identities. The bucket is keyed on
	// the one address the edge actually observed, so the fourth is throttled.
	for i, claimed := range []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"} {
		if code := attempt(claimed + ", 203.0.113.9, 104.16.0.1"); code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status %d, want 401", i+1, code)
		}
	}
	if code := attempt("4.4.4.4, 203.0.113.9, 104.16.0.1"); code != http.StatusTooManyRequests {
		t.Fatalf("status %d, want 429 once the observed client exhausts its budget", code)
	}
}
