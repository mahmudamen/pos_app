package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pos-saas/internal/config"
	"pos-saas/internal/httpapi"
	"pos-saas/internal/infra/postgres"
	"pos-saas/internal/infra/redisx"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rdb, err := redisx.Open(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatal(err)
	}
	defer rdb.Close()

	router := httpapi.New(cfg, db, rdb)

	go func() {
		if err := router.Run(cfg.HTTPAddr); err != nil {
			log.Printf("http server stopped: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = shutdownCtx
}
