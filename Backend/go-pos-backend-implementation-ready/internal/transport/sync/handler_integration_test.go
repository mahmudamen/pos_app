package sync

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SYNC-007 DB-backed slice: the push endpoint apply → replay → conflict →
// dedupe lifecycle against real PostgreSQL. Skips when no database is
// configured, like the other DB-backed integration tests.

func setupSyncIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, testutil.TokenManager()).Register(router.Group("/v1"))
	return router, seed, pool
}

func pushToken(t *testing.T, seed testutil.Seed) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-push", "manager")
}

// insertProduct seeds a product directly in the tenant context so the push
// handler can reference it. Returns the product id.
func insertProduct(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := tx.QueryRow(ctx,
		`INSERT INTO products (tenant_id, name, sku, price_minor, currency, stock_quantity)
		 VALUES ($1::uuid, 'Cappuccino', 'CAP-SYNC', 350, 'EGP', 10) RETURNING id::text`,
		seed.TenantID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return id
}

func pushJSON(t *testing.T, router http.Handler, token string, body string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sync/push", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	data := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &data)
	return rec, data
}

func commandPayload(prodID, qty, commandID string) string {
	return `{"commands":[{"command_id":"` + commandID + `","operation":"sale.create","payload":{"items":[{"product_id":"` + prodID + `","quantity":` + qty + `}]}}]}`
}

// freshCommandID returns a random command id so the lifecycle steps (apply →
// replay → conflict → dedupe) are independent of any earlier run against a
// shared test database.
func freshCommandID(t *testing.T) string {
	t.Helper()
	return uuid.NewString()
}

func resultsOf(t *testing.T, resp map[string]any) []map[string]any {
	t.Helper()
	inner, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("push: no data envelope in %v", resp)
	}
	list, ok := inner["results"].([]any)
	if !ok {
		t.Fatalf("push: no results list in %v", resp)
	}
	out := make([]map[string]any, 0, len(list))
	for _, r := range list {
		out = append(out, r.(map[string]any))
	}
	return out
}

func TestSyncIntegration_PushApplyReplayConflict(t *testing.T) {
	router, seed, pool := setupSyncIntegration(t)
	token := pushToken(t, seed)
	prodID := insertProduct(t, seed, pool)
	cmdA := freshCommandID(t)

	// 1) First push applies the sale and consumes stock.
	rec, resp := pushJSON(t, router, token, commandPayload(prodID, "2", cmdA))
	if rec.Code != http.StatusOK {
		t.Fatalf("push apply: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	first := resultsOf(t, resp)[0]
	if first["status"] != "applied" {
		t.Fatalf("push apply: expected status applied, got %v", first["status"])
	}
	if first["replayed"] != false {
		t.Fatalf("push apply: expected replayed=false, got %v", first["replayed"])
	}
	if got, want := quantity(t, seed, pool, prodID), int64(8); got != want {
		t.Fatalf("stock after apply = %d, want %d", got, want)
	}

	// 2) Same command_id + same payload replays the stored outcome, no re-apply.
	rec2, resp2 := pushJSON(t, router, token, commandPayload(prodID, "2", cmdA))
	if rec2.Code != http.StatusOK {
		t.Fatalf("push replay: expected 200, got %d", rec2.Code)
	}
	second := resultsOf(t, resp2)[0]
	if second["status"] != "applied" || second["replayed"] != true {
		t.Fatalf("push replay: expected applied+replayed, got %v", second)
	}
	if got, want := quantity(t, seed, pool, prodID), int64(8); got != want {
		t.Fatalf("stock after replay = %d, want %d (no re-apply)", got, want)
	}

	// 3) Same command_id + DIFFERENT payload is a command conflict.
	rec3, resp3 := pushJSON(t, router, token, commandPayload(prodID, "3", cmdA))
	if rec3.Code != http.StatusOK {
		t.Fatalf("push conflict: expected 200, got %d", rec3.Code)
	}
	third := resultsOf(t, resp3)[0]
	if third["status"] != "conflict" || third["error_code"] != "command_conflict" {
		t.Fatalf("push conflict: expected conflict/command_conflict, got %v", third)
	}

	// 4) New command_id with the SAME device+operation+payload is deduped: the
	// sale is NOT applied twice (stock stays at 8).
	cmdB := freshCommandID(t)
	dedupeBody := commandPayload(prodID, "2", cmdB)
	rec4, resp4 := pushJSON(t, router, token, dedupeBody)
	if rec4.Code != http.StatusOK {
		t.Fatalf("push dedupe: expected 200, got %d", rec4.Code)
	}
	fourth := resultsOf(t, resp4)[0]
	if fourth["status"] != "applied" || fourth["replayed"] != true {
		t.Fatalf("push dedupe: expected applied+replayed, got %v", fourth)
	}
	if got, want := quantity(t, seed, pool, prodID), int64(8); got != want {
		t.Fatalf("stock after dedupe = %d, want %d (deduped)", got, want)
	}

	// 5) Unknown operation is rejected with a typed error code.
	rec5, resp5 := pushJSON(t, router, token,
		`{"commands":[{"command_id":"`+freshCommandID(t)+`","operation":"nope","payload":{}}]}`)
	if rec5.Code != http.StatusOK {
		t.Fatalf("push unknown op: expected 200, got %d", rec5.Code)
	}
	fifth := resultsOf(t, resp5)[0]
	if fifth["status"] != "rejected" || fifth["error_code"] != "unknown_command" {
		t.Fatalf("push unknown op: expected rejected/unknown_command, got %v", fifth)
	}
}

func quantity(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool, productID string) int64 {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	var q int64
	if err := tx.QueryRow(ctx, `SELECT stock_quantity FROM products WHERE id = $1::uuid`, productID).Scan(&q); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return q
}
