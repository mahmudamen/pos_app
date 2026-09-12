package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry owns the HTTP-observability vectors. Each Registry creates its own
// vectors (never the global registerer) so per-process tests can hang them on a
// private prometheus.NewRegistry() while main wires the same vectors onto the
// default registerer/gatherer.
type Registry struct {
	requests *prometheus.CounterVec
	latency  *prometheus.HistogramVec
	inflight prometheus.Gauge
	gather   prometheus.Gatherer
}

// NewScoped returns a Registry with fresh, unregistered vectors. main calls
// Register(DefaultRegisterer, DefaultGatherer); tests call
// Register(promReg, promReg) with a private registry so exposition and the
// private vectors always agree.
func NewScoped() *Registry {
	return &Registry{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "pos_api_http_requests_total",
			Help: "Total number of HTTP requests by method, route and status class.",
		}, []string{"method", "route", "status_class"}),
		latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "pos_api_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds by method and route.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
		inflight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "pos_api_http_requests_inflight",
			Help: "Number of HTTP requests currently being served.",
		}),
	}
}

// Register wires the vectors onto the given registerer and records the gatherer
// Handler() will expose — always the same vectors the middleware feeds, so
// /metrics never diverges from runtime.
func (r *Registry) Register(reg prometheus.Registerer, gather prometheus.Gatherer) error {
	if err := reg.Register(r.requests); err != nil {
		return err
	}
	if err := reg.Register(r.latency); err != nil {
		return err
	}
	if err := reg.Register(r.inflight); err != nil {
		return err
	}
	r.gather = gather
	return nil
}

// statusClass buckets a status code into its hundreds class for label
// cardinality: 200 → "2xx", 404 → "4xx", 503 → "5xx".
func statusClass(code int) string {
	return strconv.Itoa(code/100) + "xx"
}

// Middleware records per-request count, latency and the in-flight gauge. It is
// dependency-free of any tenant context and safe for /metrics itself.
func (r *Registry) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		r.inflight.Inc()
		c.Next()
		r.inflight.Dec()
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		r.requests.WithLabelValues(c.Request.Method, route, statusClass(c.Writer.Status())).Inc()
		r.latency.WithLabelValues(c.Request.Method, route).Observe(time.Since(start).Seconds())
	}
}

// Handler returns a Gin handler serving Prometheus text exposition for
// /metrics. It reflects the gatherer captured at Register() time so exposition
// and middleware always agree.
func (r *Registry) Handler() gin.HandlerFunc {
	gather := r.gather
	if gather == nil {
		// Fall back to the default gatherer so exposition is at least sensible
		// for a Registry that was never Register()ed, though /metrics in main
		// always Register()s so this branch is only exercised by sloppy tests.
		gather = prometheus.DefaultGatherer
	}
	return func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		promhttp.HandlerFor(gather, promhttp.HandlerOpts{}).ServeHTTP(c.Writer, c.Request)
	}
}

var _ = http.StatusOK
