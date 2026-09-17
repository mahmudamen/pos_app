package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
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
