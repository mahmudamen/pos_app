package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// newTestRouter returns a Registry + gin.Engine wired onto a per-test private
// Prometheus registry (no global MustRegister), keeping the tests independent
// and the default registerer pristine. Register() captures the private
// gatherer so /metrics exposition reflects exactly the vectors seen in tests.
func newTestRouter() (*Registry, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	reg := NewScoped()
	promReg := prometheus.NewRegistry()
	if err := reg.Register(promReg, promReg); err != nil {
		panic(err)
	}
	router := gin.New()
	router.Use(reg.Middleware())
	router.GET("/v1/health/ready", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	router.GET("/metrics", reg.Handler())
	return reg, router
}

// TestExpositionServesCoreFamilyNames verifies the text exposition contains the
// core metric family names, the per-request labels wired via Middleware with
// route + status-class, and the histograms (with the DefBuckets series).
func TestMetricsExpositionServesCoreFamilyNames(t *testing.T) {
	_, router := newTestRouter()

	// Fire a request so the middleware observes non-zero vectors.
	r1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/v1/health/ready", nil)
	router.ServeHTTP(r1, req1)

	// Fetch the exposition and assert the core names + labels appear.
	r2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	router.ServeHTTP(r2, req2)
	if r2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", r2.Code, r2.Body.String())
	}
	body := r2.Body.String()
	for _, want := range []string{
		`pos_api_http_requests_total`,
		`pos_api_http_request_duration_seconds`,
		`pos_api_http_requests_inflight`,
		`route="/v1/health/ready"`,
		`status_class="2xx"`,
		`method="GET"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("metrics body missing %q", want)
		}
	}
}

// TestStatusClassUsesHundredsDigit checks that status codes are bucketed into
// their hundreds class with stable cardinality (never one label per code).
func TestStatusClassUsesHundredsDigit(t *testing.T) {
	for _, tt := range []struct {
		code int
		want string
	}{
		{200, "2xx"}, {201, "2xx"}, {299, "2xx"},
		{400, "4xx"}, {404, "4xx"}, {499, "4xx"},
		{500, "5xx"}, {503, "5xx"},
	} {
		if got := statusClass(tt.code); got != tt.want {
			t.Errorf("statusClass(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

// TestMetricsInflightReturnsToZero verifies the in-flight gauge is decremented
// after the request completes so /metrics never shows a stuck positive value.
func TestMetricsInflightReturnsToZero(t *testing.T) {
	reg, router := newTestRouter()
	r1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/v1/health/ready", nil)
	router.ServeHTTP(r1, req1)
	if got := testutil.ToFloat64(reg.inflight); got != 0 {
		t.Fatalf("inflight after request = %v, want 0", got)
	}
}
