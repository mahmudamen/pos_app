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
	"github.com/example/pos-api/internal/infrastructure/metrics"
	"github.com/example/pos-api/internal/infrastructure/ratelimit"
	redisinfra "github.com/example/pos-api/internal/infrastructure/redis"
	"github.com/example/pos-api/internal/transport/server"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
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

	// Login rate limiting: Redis-backed when Redis is up (multi-instance),
	// falling back to an in-memory limiter otherwise.
	var loginLimiter ratelimit.Limiter
	if redisClient != nil {
		loginLimiter = ratelimit.NewRedis(redisClient, cfg.LoginRateMax, cfg.LoginRateWindow)
		logger.Info("login rate limiting uses redis", "max", cfg.LoginRateMax, "window", cfg.LoginRateWindow)
	} else {
		loginLimiter = ratelimit.NewMemory(cfg.LoginRateMax, cfg.LoginRateWindow)
		logger.Info("login rate limiting uses in-memory limiter", "max", cfg.LoginRateMax, "window", cfg.LoginRateWindow)
	}

	// SaaS control-plane rate limit: Redis-backed when Redis is up, else
	// in-memory. Only wired when RATE_LIMIT_ENABLED=true.
	var apiLimiter ratelimit.Limiter
	if cfg.ApiRateLimitEnabled {
		if redisClient != nil {
			apiLimiter = ratelimit.NewRedis(redisClient, cfg.ApiRateLimitMax, cfg.ApiRateLimitWindow)
			logger.Info("api rate limiting uses redis", "max", cfg.ApiRateLimitMax, "window", cfg.ApiRateLimitWindow)
		} else {
			apiLimiter = ratelimit.NewMemory(cfg.ApiRateLimitMax, cfg.ApiRateLimitWindow)
			logger.Info("api rate limiting uses in-memory limiter", "max", cfg.ApiRateLimitMax, "window", cfg.ApiRateLimitWindow)
		}
	}

	var metricsRegistry *metrics.Registry
	if cfg.MetricsEnabled {
		metricsRegistry = metrics.NewScoped()
		if err := metricsRegistry.Register(prometheus.DefaultRegisterer, prometheus.DefaultGatherer); err != nil {
			// Already registered (duplicate main run) — keep serving; the vectors
			// middleware feeds are the same ones /metrics gathers either way.
			logger.Warn("metrics registry already registered", "err", err)
		}
	}
	router := gin.New()
	server.Register(router, server.Deps{
		Pool:         pool,
		Config:       cfg,
		LoginLimiter: loginLimiter,
		ApiLimiter:   apiLimiter,
		Metrics:      metricsRegistry,
		Logger:       logger,
		Readiness:    func() bool { return pool != nil && redisClient != nil },
	})

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
