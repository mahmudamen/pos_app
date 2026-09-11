package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/infrastructure/database"
	redisinfra "github.com/example/pos-api/internal/infrastructure/redis"
	authtransport "github.com/example/pos-api/internal/transport/auth"
	catalogtransport "github.com/example/pos-api/internal/transport/catalog"
	customertransport "github.com/example/pos-api/internal/transport/customers"
	dashboardtransport "github.com/example/pos-api/internal/transport/dashboard"
	httptransport "github.com/example/pos-api/internal/transport/http"
	inventorytransport "github.com/example/pos-api/internal/transport/inventory"
	metatransport "github.com/example/pos-api/internal/transport/meta"
	registerstransport "github.com/example/pos-api/internal/transport/registers"
	saastransport "github.com/example/pos-api/internal/transport/saas"
	salestransport "github.com/example/pos-api/internal/transport/sales"
	settingsTransport "github.com/example/pos-api/internal/transport/settings"
	synctransport "github.com/example/pos-api/internal/transport/sync"
	usertransport "github.com/example/pos-api/internal/transport/users"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	var pool *pgxpool.Pool
	var redisClient *redis.Client
	connectContext, connectCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer connectCancel()
	if cfg.DatabaseURL != "" {
		pool, err = database.NewPool(connectContext, cfg)
		if err != nil {
			logger.Warn("database unavailable; readiness will fail", "error", err)
		}
	}
	if cfg.RedisAddr != "" {
		redisClient, err = redisinfra.NewClient(connectContext, cfg)
		if err != nil {
			logger.Warn("redis unavailable; readiness will fail", "error", err)
		}
	}

	router := gin.New()
	router.Use(
		httptransport.CORS(),
		httptransport.SecurityHeaders(),
		httptransport.RequestID(),
		httptransport.RequestLogger(logger),
		httptransport.Recovery(logger),
		httptransport.MaxBodySize(cfg.HTTPMaxBodyBytes),
	)

	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	router.GET("/health/ready", func(c *gin.Context) {
		ready := pool != nil && redisClient != nil
		status := http.StatusOK
		if !ready {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"status": map[bool]string{true: "ready", false: "not_ready"}[ready]})
	})
	api := router.Group("/v1")
	authHandler := authtransport.NewHandler(pool, cfg)
	// Register auth routes with rate limiting on login
	authGroup := api.Group("/auth")
	authGroup.POST("/login", httptransport.LoginRateLimit(5, 5*time.Minute), authHandler.Login())
	authGroup.POST("/refresh", authHandler.Refresh())
	authGroup.POST("/logout", authHandler.Logout())
	catalogtransport.NewHandler(pool, authHandler.Tokens()).Register(api)
	customertransport.NewHandler(pool, authHandler.Tokens()).Register(api)
	dashboardtransport.NewHandler(pool, authHandler.Tokens()).Register(api)
	salestransport.NewHandlerWithDiscountLimit(pool, authHandler.Tokens(), cfg.CashierDiscountPct).Register(api)
	registerstransport.NewHandler(pool, authHandler.Tokens()).Register(api)
	inventorytransport.NewHandler(pool, authHandler.Tokens()).Register(api)
	usertransport.NewHandler(pool, authHandler.Tokens()).Register(api)
	settingsTransport.NewHandler(pool, authHandler.Tokens()).Register(api)
	synctransport.NewHandler(pool, authHandler.Tokens()).Register(api)
	metatransport.NewHandler(pool).Register(api)
	saastransport.NewHandler(pool, authHandler.Tokens()).Register(api)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	go func() {
		slog.Info("starting HTTP server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTPShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown failed", "error", err)
		os.Exit(1)
	}

	if pool != nil {
		pool.Close()
	}
	if redisClient != nil {
		_ = redisClient.Close()
	}
	slog.Info("HTTP server stopped")
}
