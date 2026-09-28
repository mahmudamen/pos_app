// Command posgo-ops is the POS.Go control plane as a terminal application.
//
// It talks to the same /v1/saas and /v1/platform endpoints as the React admin
// SPA, so an operator on the VPS gets the tenant list, the per-tenant
// drill-down, and the activate/suspend/stop/backup actions without a browser.
//
// Usage:
//
//	posgo-ops --api https://api.xamltech.com
//
// Credentials come from the environment rather than flags, so they never land
// in the shell history or in `ps` output for another user on the box:
//
//	POSGO_OPS_PASSWORD   SaaS staff password
//	POSGO_OPS_TOKEN      access token, skips the sign-in form
//	POSGO_OPS_STOP_REASON  justification recorded with a tenant stop
//	POSGO_OPS_BACKUP_DIR   where tenant exports are written (default $TMPDIR/posgo-backups)
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"posgo-ops/internal/opsconsole"
)

// version is stamped by the Makefile at build time.
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "posgo-ops: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		apiBase = flag.String("api", envOr("POSGO_OPS_API", "https://api.xamltech.com"),
			"base URL of the POS API")
		tenantID = flag.String("tenant", os.Getenv("POSGO_OPS_TENANT_ID"),
			"SaaS platform tenant id (required for password sign-in)")
		email = flag.String("email", envOr("POSGO_OPS_EMAIL", "admin@posgo.saas"),
			"SaaS staff email")
		deviceID = flag.String("device", envOr("POSGO_OPS_DEVICE", "posgo-ops-tui"),
			"device id recorded against the session")
		mcpAddr = flag.String("mcp", os.Getenv("POSGO_OPS_MCP"),
			"unix socket for agent control, e.g. /tmp/posgo-ops.sock (empty disables it)")
		showVersion = flag.Bool("version", false, "print the version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("posgo-ops %s\n", version)
		return nil
	}

	cfg := opsconsole.Config{
		BaseURL:     *apiBase,
		TenantID:    *tenantID,
		Email:       *email,
		Password:    os.Getenv("POSGO_OPS_PASSWORD"),
		StopReason:  os.Getenv("POSGO_OPS_STOP_REASON"),
		BackupDir:   os.Getenv("POSGO_OPS_BACKUP_DIR"),
		DeviceID:    *deviceID,
		DeviceName:  "posgo-ops terminal console",
		AccessToken: os.Getenv("POSGO_OPS_TOKEN"),
	}
	if cfg.Password == "" && cfg.AccessToken == "" && cfg.TenantID == "" {
		return fmt.Errorf("set POSGO_OPS_PASSWORD with POSGO_OPS_TENANT_ID, or POSGO_OPS_TOKEN")
	}

	// Ctrl-C cancels the context, which unwinds the app's run loop and lets the
	// in-flight fetches finish rather than killing the process mid-write.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	console, root, err := opsconsole.New(ctx, cfg)
	if err != nil {
		return err
	}
	if *mcpAddr != "" {
		// Reported rather than silently enabled: opening a control socket on a
		// privileged view is an operator decision that should be visible in
		// the terminal they are sitting at.
		fmt.Printf("agent control socket: %s\n", *mcpAddr)
	}
	return opsconsole.Run(ctx, console, root, *mcpAddr)
}

// envOr returns the environment value for key, or fallback when it is unset.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
