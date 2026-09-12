// Package testutil provides shared helpers for PostgreSQL-backed integration
// tests. Tests are skipped when no TEST_DATABASE_URL / DATABASE_URL is set so
// that `go test ./...` and CI without a database still pass.
package testutil

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DatabaseURL returns the connection string for integration tests, skipping
// the test when neither TEST_DATABASE_URL nor DATABASE_URL is configured.
func DatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = os.Getenv("DATABASE_URL")
	}
	if url == "" {
		t.Skip("integration test requires TEST_DATABASE_URL or DATABASE_URL")
	}
	return url
}

// Pool returns a pgx pool for integration tests and closes it on cleanup.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	poolConfig, err := pgxpool.ParseConfig(DatabaseURL(t))
	if err != nil {
		t.Fatalf("parse database url: %v", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Fatalf("ping test database: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Migrate applies the goose migrations to the test database once per test
// process. Goose is idempotent, so repeated calls are safe.
func Migrate(t *testing.T, connString string) {
	t.Helper()
	migrateOnce.Do(func() {
		migrateErr = runGoose(connString)
	})
	if migrateErr != nil {
		t.Fatalf("migrate test database: %v", migrateErr)
	}
}

var migrateOnce sync.Once
var migrateErr error

func runGoose(connString string) error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	migrations := filepath.Join(root, "internal", "infrastructure", "database", "migrations")
	cmd := exec.Command("go", "run", "github.com/pressly/goose/v3/cmd/goose@v3.21.1",
		"-dir", migrations, "postgres", connString, "up")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("goose up: %w\n%s", err, output)
	}
	return nil
}

// moduleRoot walks up from the working directory to the directory containing
// go.mod. Tests run with their package directory as the working directory.
func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found above %s", dir)
		}
		dir = parent
	}
}

// Seed is the tenant fixture used across auth, catalog, sales, and sync tests.
type Seed struct {
	TenantID   string
	Slug       string
	ManagerID  string
	CashierID  string
	DeviceID   string
	Password   string
	DeviceCode string
}

// SeedTenant inserts a tenant, an active manager and cashier, and one device.
func SeedTenant(t *testing.T, pool *pgxpool.Pool) Seed {
	t.Helper()
	ctx := context.Background()
	seed := Seed{
		Slug:       "store-" + shortID(),
		Password:   "StrongPass2026",
		DeviceCode: "terminal-" + shortID(),
	}
	hash, err := security.HashPassword(seed.Password, 10)
	if err != nil {
		t.Fatalf("hash seed password: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin seed transaction: %v", err)
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx,
		`INSERT INTO tenants (name, slug) VALUES ($1, $2) RETURNING id::text`,
		"Test Store", seed.Slug).Scan(&seed.TenantID); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	// RLS is FORCE-enabled on users/devices, so all writes by the seeded
	// tenant must run inside the transaction-local tenant context.
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", seed.TenantID); err != nil {
		t.Fatalf("set seed tenant context: %v", err)
	}
	for _, role := range []string{"manager", "cashier"} {
		target := &seed.ManagerID
		if role == "cashier" {
			target = &seed.CashierID
		}
		email := role + "-" + shortID() + "@example.com"
		if err := tx.QueryRow(ctx,
			`INSERT INTO users (tenant_id, email, password_hash, display_name, role)
			 VALUES ($1::uuid, $2, $3, $4, $5) RETURNING id::text`,
			seed.TenantID, email, hash, role, role).Scan(target); err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	if err := tx.QueryRow(ctx,
		`INSERT INTO devices (tenant_id, client_device_id, name)
		 VALUES ($1::uuid, $2, 'Counter 1') RETURNING id::text`,
		seed.TenantID, seed.DeviceCode).Scan(&seed.DeviceID); err != nil {
		t.Fatalf("seed device: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit seed: %v", err)
	}
	return seed
}

// TokenManager returns a TokenManager backed by fixed test secrets.
func TokenManager() security.TokenManager {
	return security.TokenManager{
		Issuer:        "pos-api-test",
		AccessSecret:  []byte("test-access-secret-0123456789abcdef"),
		RefreshSecret: []byte("test-refresh-secret-0123456789abcdef"),
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    7 * 24 * time.Hour,
	}
}

// MintAccess issues an access token carrying the given claims without touching
// the database, for handler-level integration tests.
func MintAccess(t *testing.T, tenantID, userID, deviceID, sessionID, role string) string {
	t.Helper()
	token, err := TokenManager().IssueWithRole(time.Now(), security.AccessToken,
		tenantID, userID, deviceID, sessionID, role)
	if err != nil {
		t.Fatalf("mint access token: %v", err)
	}
	return token
}

func shortID() string {
	var value [8]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(value[:])
}
