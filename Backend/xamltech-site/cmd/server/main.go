// Command server runs the XamlTech marketing website.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/xamltech/xamltech-site/internal/config"
	"github.com/xamltech/xamltech-site/internal/content"
	"github.com/xamltech/xamltech-site/internal/web"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	cfg := config.Load()
	store, err := content.Load()
	if err != nil {
		log.Error("content load failed", "err", err)
		os.Exit(1)
	}
	srv, err := web.New(cfg, store, log)
	if err != nil {
		log.Error("server init failed", "err", err)
		os.Exit(1)
	}

	httpSrv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      srv.Handler(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("listening", "addr", cfg.Addr, "env", cfg.Env, "base_url", cfg.BaseURL)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}
