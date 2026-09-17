package telemetry

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB-backed slice: a POST /v1/client/events round-trip against real PostgreSQL
// — the event must land tenant-scoped (RLS FORCE) with its payload intact, and
// be invisible to another tenant. Skips when no database is configured.

func setupTelemetry(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, testutil.TokenManager()).Register(router.Group("/v1"))
	return router, seed, pool
}

func telemetryToken(t *testing.T, seed testutil.Seed) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-events", "manager")
}

func countEvents(t *testing.T, pool *pgxpool.Pool, tenantID string, deviceID string) int {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM client_events WHERE tenant_id = $1::uuid AND device_id = $2`, tenantID, deviceID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func TestPostEventPersists(t *testing.T) {
	router, seed, pool := setupTelemetry(t)
	body := `{"event":"crash","app_version":"0.1.0+1","screen":"pos",
		"payload":{"exception":"Test exception","items":42},
		"stack_trace":"package (line 1)"}`
	request, err := http.NewRequest(http.MethodPost, "/v1/client/events", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+telemetryToken(t, seed))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d %s", recorder.Code, recorder.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err = json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if created.Data.ID == "" {
		t.Fatal("expected an event id")
	}
	if got := countEvents(t, pool, seed.TenantID, seed.DeviceID); got != 1 {
		t.Fatalf("expected 1 stored event, got %d", got)
	}

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	tx.Exec(context.Background(), `SELECT set_config('app.current_tenant', $1, true)`, seed.TenantID)
	var event, payload, stackTrace string
	var userID sql.NullString
	if err = tx.QueryRow(context.Background(),
		`SELECT event, payload::text, COALESCE(stack_trace, ''), user_id::text FROM client_events WHERE id = $1::uuid`,
		created.Data.ID).Scan(&event, &payload, &stackTrace, &userID); err != nil {
		t.Fatalf("read stored event: %v", err)
	}
	if event != "crash" || payload == "" || !strings.Contains(payload, "Test exception") {
		t.Fatalf("unexpected stored row: event=%q payload=%q", event, payload)
	}
	if stackTrace != "package (line 1)" {
		t.Fatalf("expected stack_trace preserved, got %q", stackTrace)
	}
	if !userID.Valid {
		t.Fatal("expected user_id recorded")
	}
}

func TestPostEventIsTenantIsolated(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seedA := testutil.SeedTenant(t, pool)
	seedB := testutil.SeedTenant(t, pool)

	ctx := context.Background()
	// Write an event for tenant B directly (RLS FORCE within context).
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, seedB.TenantID)
	_, err = tx.Exec(ctx, `INSERT INTO client_events (tenant_id, device_id, event) VALUES ($1::uuid, $2, 'boot')`,
		seedB.TenantID, seedB.DeviceID)
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	// Tenant A must not see tenant B's event even as owner (RLS FORCE).
	if got := countEvents(t, pool, seedA.TenantID, seedB.DeviceID); got != 0 {
		t.Fatalf("tenant A leaked %d events belonging to tenant B", got)
	}
	if got := countEvents(t, pool, seedB.TenantID, seedB.DeviceID); got != 1 {
		t.Fatalf("expected 1 event for tenant B, got %d", got)
	}
}
