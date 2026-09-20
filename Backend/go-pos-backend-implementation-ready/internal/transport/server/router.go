// Package server wires the full HTTP route table for the pos-api application.
//
// main (cmd/api) and the OpenAPI generator (cmd/openapi) both call Register so
// the shipped spec never drifts from the real routes. Handlers accept a nil
// pool and still register their routes, which is what lets the generator run
// with no database.
package server

import (
	"log/slog"
	"net/http"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/infrastructure/metrics"
	"github.com/example/pos-api/internal/infrastructure/ratelimit"
	ocrorcore "github.com/example/pos-api/internal/ocr"
	authtransport "github.com/example/pos-api/internal/transport/auth"
	billingtransport "github.com/example/pos-api/internal/transport/billing"
	catalogtransport "github.com/example/pos-api/internal/transport/catalog"
	customertransport "github.com/example/pos-api/internal/transport/customers"
	dashboardtransport "github.com/example/pos-api/internal/transport/dashboard"
	httptransport "github.com/example/pos-api/internal/transport/http"
	identitytransport "github.com/example/pos-api/internal/transport/identitytransport"
	inventorytransport "github.com/example/pos-api/internal/transport/inventory"
	lotstransport "github.com/example/pos-api/internal/transport/lots"
	metatransport "github.com/example/pos-api/internal/transport/meta"
	platformtransport "github.com/example/pos-api/internal/transport/platform"
	purchasetransport "github.com/example/pos-api/internal/transport/purchases"
	receiptstransport "github.com/example/pos-api/internal/transport/receipts"
	registerstransport "github.com/example/pos-api/internal/transport/registers"
	restauranttransport "github.com/example/pos-api/internal/transport/restaurants"
	saastransport "github.com/example/pos-api/internal/transport/saas"
	salestransport "github.com/example/pos-api/internal/transport/sales"
	selftransport "github.com/example/pos-api/internal/transport/selforder"
	settingsTransport "github.com/example/pos-api/internal/transport/settings"
	subscriptiontransport "github.com/example/pos-api/internal/transport/subscription"
	synctransport "github.com/example/pos-api/internal/transport/sync"
	telemetrytransport "github.com/example/pos-api/internal/transport/telemetry"
	usertransport "github.com/example/pos-api/internal/transport/users"
	variantstransport "github.com/example/pos-api/internal/transport/variants"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps is everything Register needs to assemble the engine. Every field that
// can be nil is optional, so the OpenAPI generator can pass just a bare Config,
// a limiter and nil pools.
type Deps struct {
	Pool         *pgxpool.Pool
	Config       config.Config
	LoginLimiter ratelimit.Limiter // nil → login is registered without rate limiting
	Metrics      *metrics.Registry // nil → /metrics middleware + route are omitted
	Logger       *slog.Logger      // nil → slog.Default()
	Readiness    func() bool       // health/ready probe; nil → pool != nil
}

// Register attaches middleware, /health, /metrics, and every /v1 route to the
// engine. It mirrors the route table that cmd/api serves in production.
func Register(engine *gin.Engine, d Deps) {
	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if d.Metrics != nil {
		engine.Use(d.Metrics.Middleware())
	}
	engine.Use(
		httptransport.CORS(d.Config.CORSAllowedOrigins),
		httptransport.SecurityHeaders(),
		httptransport.RequestID(),
		httptransport.RequestLogger(logger),
		httptransport.Recovery(logger),
		httptransport.MaxBodySize(d.Config.HTTPMaxBodyBytes),
	)

	if d.Metrics != nil {
		engine.GET("/metrics", d.Metrics.Handler())
	}

	ready := d.Readiness
	if ready == nil {
		ready = func() bool { return d.Pool != nil }
	}
	engine.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	engine.GET("/health/ready", func(c *gin.Context) {
		status := http.StatusOK
		if !ready() {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"status": map[bool]string{true: "ready", false: "not_ready"}[ready()]})
	})

	registerSitePages(engine, d.Pool)

	api := engine.Group("/v1")
	authHandler := authtransport.NewHandler(d.Pool, d.Config)
	authGroup := api.Group("/auth")
	authGroup.POST("/register", authHandler.Signup)
	if d.LoginLimiter != nil {
		authGroup.POST("/login", httptransport.LoginRateLimitWith(d.LoginLimiter), authHandler.Login())
	} else {
		authGroup.POST("/login", authHandler.Login())
	}
	authGroup.POST("/refresh", authHandler.Refresh())
	authGroup.POST("/logout", authHandler.Logout())
	authGroup.POST("/set-pin", authHandler.SetPin())
	authGroup.POST("/verify-pin", authHandler.VerifyPin())
	catalogHandler := catalogtransport.NewHandler(d.Pool, authHandler.Tokens())
	catalogHandler.Register(api)
	customertransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	dashboardtransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	salestransport.NewHandlerWithDiscountLimit(d.Pool, authHandler.Tokens(), d.Config.CashierDiscountPct).Register(api)
	registerstransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	inventorytransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	purchasetransport.NewHandler(d.Pool, authHandler.Tokens(),
		ocrorcore.NewEngine(d.Config.OCREnabled, d.Config.TesseractBin,
			d.Config.TesseractLangs, d.Config.TesseractPSM),
		purchasetransport.OCRWindows{
			Day:   d.Config.OCRDayLimit,
			Week:  d.Config.OCRWeekLimit,
			Month: d.Config.OCRMonthLimit,
		}).Register(api)
	usertransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	settingsTransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	synctransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	metatransport.NewHandler(d.Pool).Register(api)
	saasGroup := api.Group("/saas")
	// The SaaS control plane is gated by the saas_admin role on every route, so
	// it intentionally bypasses the API-wide rate limiter (the admin panel makes
	// several parallel /v1/saas/* calls per page).
	saastransport.NewHandler(d.Pool, authHandler.Tokens()).Register(saasGroup)
	billingtransport.NewHandler(d.Pool, authHandler.Tokens()).Register(saasGroup)
	restauranttransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	lotstransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	variantstransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	receiptstransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	telemetrytransport.NewHandler(d.Pool, authHandler.Tokens()).Register(api)
	subscriptiontransport.NewHandler(d.Pool, authHandler.Tokens(), d.Config).Register(api)
	identitytransport.NewHandler(d.Pool, authHandler.Tokens(), d.Config).Register(api)
	platformtransport.NewHandler(d.Pool, authHandler.Tokens(), d.Config).Register(api)

	// Self-ordering: public menu/order/request endpoints used by any browser
	// that scans a store's self-order QR, plus staff routes to list, approve
	// and cancel pending orders. The web page and its service worker are served
	// at /selforder and /sw.js. When a product transitions back to "available
	// online" the catalog handler fires NotifyBackInStock.
	selfOrderHandler := selftransport.NewHandler(d.Pool, authHandler.Tokens(), selftransport.WebPushConfig{
		PublicKey:  d.Config.VAPIDPublicKey,
		PrivateKey: d.Config.VAPIDPrivateKey,
		Subject:    d.Config.VAPIDSubject,
	}, d.Config.CashierDiscountPct)
	selfOrderHandler.RegisterPublic(api)
	selfOrderHandler.Register(api)
	catalogHandler.SetBackInStockNotifier(selfOrderHandler.NotifyBackInStock)
	engine.GET("/selforder", selfOrderHandler.Page)
	engine.GET("/sw.js", selfOrderHandler.ServiceWorker)
}
