// Command site serves the XamlTech company website (xamltech.com) — a
// self-contained, bilingual (English / Arabic RTL) replacement for the Odoo
// website that previously occupied port 8069 on the production box.
//
// It needs no database and no other services: everything (copy, styles and the
// favicon) is compiled into the binary. Run it behind nginx, which terminates
// TLS and proxies to 127.0.0.1:8069.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/example/pos-api/internal/mainsite"
)

func main() {
	addr := os.Getenv("SITE_ADDR")
	if addr == "" {
		addr = "0.0.0.0:8069"
	}
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	// Liveness/readiness for the systemd unit and any proxy checks.
	engine.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	mainsite.Register(engine)

	srv := &http.Server{
		Addr:              addr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("xamltech site listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down…")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}
	log.Println("stopped")
}
