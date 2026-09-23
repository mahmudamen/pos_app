package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB-backed slice of the platform admin surface: tenant suspend → activate is
// an audited round-trip that also folds into the subscription row.

func setupPlatformIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	tokens := testutil.TokenManager()
	api := router.Group("/v1")
	NewHandler(pool, tokens, config.Config{}).Register(api)
	return router, seed, pool
}

func tenantStatus(t *testing.T, pool *pgxpool.Pool, tenantID string) string {
	t.Helper()
	var status string
	if err := pool.QueryRow(context.Background(),
		`SELECT status FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	return status
}

func TestPlatformIntegration_TenantSuspendActivate(t *testing.T) {
	router, seed, pool := setupPlatformIntegration(t)
	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-saas", "saas_admin")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/v1/platform/tenants/"+seed.TenantID+"/suspend",
		bytes.NewBufferString(`{"reason":"policy review"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("suspend: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := tenantStatus(t, pool, seed.TenantID); got != "suspended" {
		t.Fatalf("tenant status after suspend = %q, want suspended", got)
	}
	// The audit trail captured the suspension.
	var auditCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_log WHERE action = 'organization.suspended' AND tenant_id = $1::uuid`,
		seed.TenantID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("suspension audit count = %d, want 1", auditCount)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost,
		"/v1/platform/tenants/"+seed.TenantID+"/activate", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("activate: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := tenantStatus(t, pool, seed.TenantID); got != "active" {
		t.Fatalf("tenant status after activate = %q, want active", got)
	}
	body := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body["data"].(map[string]any)["activated"] != true {
		t.Fatalf("activate should echo activated: %v", body)
	}
	var activated int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_log WHERE action = 'organization.activated' AND tenant_id = $1::uuid`,
		seed.TenantID).Scan(&activated); err != nil {
		t.Fatal(err)
	}
	if activated != 1 {
		t.Fatalf("activation audit count = %d, want 1", activated)
	}

	// Reactivating an already-active tenant is a no-op 404 (must be suspended).
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost,
		"/v1/platform/tenants/"+seed.TenantID+"/activate", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("double activate: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestPlatformIntegration_StopTenant covers the permanent stop action: status
// flips to 'disabled', live subscriptions are cancelled, an organization.stopped
// audit row is written, and a second stop is a no-op 404.
func TestPlatformIntegration_StopTenant(t *testing.T) {
	router, seed, pool := setupPlatformIntegration(t)
	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-saas", "saas_admin")

	// Give the tenant a live subscription to cancel.
	var planID string
	if err := pool.QueryRow(context.Background(), `SELECT id::text FROM plans ORDER BY created_at LIMIT 1`).Scan(&planID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO subscriptions (tenant_id, plan_id, status)
		VALUES ($1::uuid, $2::uuid, 'active')`, seed.TenantID, planID); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/v1/platform/tenants/"+seed.TenantID+"/stop",
		bytes.NewBufferString(`{"reason":"tenant asked to shut down"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("stop: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := tenantStatus(t, pool, seed.TenantID); got != "disabled" {
		t.Fatalf("tenant status after stop = %q, want disabled", got)
	}
	var subStatus string
	errQ := pool.QueryRow(context.Background(),
		`SELECT status FROM subscriptions WHERE tenant_id = $1::uuid AND status <> 'cancelled'`,
		seed.TenantID).Scan(&subStatus)
	if errQ == nil {
		t.Fatalf("expected no active subscription after stop; got %q", subStatus)
	} else if !errors.Is(errQ, pgx.ErrNoRows) {
		t.Fatalf("querying subscriptions after stop: %v", errQ)
	}
	var cancelled int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM subscriptions WHERE tenant_id = $1::uuid AND status = 'cancelled'`,
		seed.TenantID).Scan(&cancelled); err != nil {
		t.Fatal(err)
	}
	if cancelled != 1 {
		t.Fatalf("expected exactly one cancelled subscription, got %d", cancelled)
	}
	var auditCount int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM audit_log WHERE action = 'organization.stopped' AND tenant_id = $1::uuid`,
		seed.TenantID).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("stop audit count = %d, want 1", auditCount)
	}

	// Stopping again is a no-op (already disabled).
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost,
		"/v1/platform/tenants/"+seed.TenantID+"/stop", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("double stop: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestPlatformIntegration_BackupTenant exercises the RLS-scoped export: rows
// from another tenant must never appear in the backup, and the JSON document
// must include the tenant's slug and tenant-scoped tables.
func TestPlatformIntegration_BackupTenant(t *testing.T) {
	router, seed, pool := setupPlatformIntegration(t)
	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-saas", "saas_admin")

	ctx := context.Background()
	// Seed one product inside the tenant's RLS context.
	insertProduct := func(tenantID string) {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO products (tenant_id, name, sku, price_minor, currency, stock_quantity)
			VALUES ($1::uuid, 'BackupTest-'||substr($1::text,1,8), 'P-'||substr($1::text,1,8), 100, 'EGP', 3)`,
			tenantID); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	insertProduct(seed.TenantID)
	// A second, unrelated tenant—its product must not leak into our backup.
	otherTenantID := seedOtherTenant(t, pool)
	insertProduct(otherTenantID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost,
		"/v1/platform/tenants/"+seed.TenantID+"/backup", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("backup: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("backup content-type = %q, want application/json", ct)
	}

	var doc struct {
		TenantID  string                       `json:"tenant_id"`
		Slug      string                       `json:"slug"`
		Subdomain string                       `json:"subdomain"`
		Tables    map[string][]json.RawMessage `json:"tables"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("backup should be valid JSON: %v", err)
	}
	if doc.TenantID != seed.TenantID || doc.Slug != seed.Slug {
		t.Fatalf("backup doc headers wrong: %+v", doc)
	}
	if doc.Subdomain != seed.Slug+".xamltech.com" {
		t.Fatalf("backup should expose the store subdomain, got %q", doc.Subdomain)
	}
	if len(doc.Tables["products"]) != 1 {
		t.Fatalf("expected exactly the tenant's product in the backup, got %d: %s",
			len(doc.Tables["products"]), rec.Body.String())
	}
	var prod struct {
		Name string `json:"name"`
		SKU  string `json:"sku"`
	}
	if err := json.Unmarshal(doc.Tables["products"][0], &prod); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(prod.Name, otherTenantID[:8]) {
		t.Fatalf("cross-tenant leak: backup contains the other tenant's product: %s", rec.Body.String())
	}
	if _, ok := doc.Tables["users_pos_security"]; !ok {
		t.Fatalf("backup should carry users_pos_security: %s", rec.Body.String())
	}
	if _, ok := doc.Tables["plans"]; ok {
		t.Fatalf("backup must NOT enumerate platform tables: %s", rec.Body.String())
	}
}

// seedOtherTenant creates an unrelated tenant for cross-tenant isolation tests.
func seedOtherTenant(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO tenants (name, slug) VALUES ('Other Backup Tenant', 'other-backup-'||substr(gen_random_uuid()::text,1,8)) RETURNING id::text`).
		Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
