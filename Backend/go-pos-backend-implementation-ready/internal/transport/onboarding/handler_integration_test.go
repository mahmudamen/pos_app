package onboarding

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupOnboarding(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, testutil.TokenManager()).Register(router.Group("/v1"))
	return router, seed, pool
}

// onbCall performs one request and splits the response into data or error.
func onbCall(t *testing.T, router http.Handler, method, path, token, body string) (int, map[string]any, string) {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var envelope struct {
		Data  map[string]any `json:"data"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("%s %s: parse %d body %q: %v", method, path, rec.Code, rec.Body.String(), err)
	}
	return rec.Code, envelope.Data, envelope.Error.Code
}

func onbToken(t *testing.T, seed testutil.Seed, userID, role string) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, userID, seed.DeviceID, "sess-onb", role)
}

// execTenant runs one statement inside the tenant's RLS context (FORCE RLS
// hides every row from the test connection until the context is set).
func execTenant(t *testing.T, pool *pgxpool.Pool, tenantID, sql string, args ...any) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("exec %s: %v", sql, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func tourStatus(data map[string]any, tourID string) map[string]any {
	tours, _ := data["tours"].([]any)
	for _, raw := range tours {
		row, _ := raw.(map[string]any)
		if row["id"] == tourID {
			return row
		}
	}
	return nil
}

func TestOnboardingIntegration_LifecycleWithServerVerifiedGoal(t *testing.T) {
	router, seed, pool := setupOnboarding(t)
	token := onbToken(t, seed, seed.ManagerID, "manager")

	code, data, errCode := onbCall(t, router, http.MethodGet, "/v1/onboarding", token, "")
	if code != http.StatusOK || errCode != "" {
		t.Fatalf("state: expected 200, got %d (%s)", code, errCode)
	}
	tours, _ := data["tours"].([]any)
	if len(tours) != len(catalog) {
		t.Fatalf("tours: expected %d, got %d", len(catalog), len(tours))
	}
	if rec, _ := data["recommended"].(string); rec != "store_setup" {
		t.Fatalf("recommended: expected store_setup, got %q", rec)
	}
	ready, _ := data["readiness"].(map[string]any)
	if ready["store_configured"] != false || ready["cash_session_opened"] != false {
		t.Fatalf("fresh tenant should start unconfigured: %v", ready)
	}
	if row := tourStatus(data, "cash_session"); row["status"] != "not_started" || row["goal_met"] != false {
		t.Fatalf("cash_session initial: %v", row)
	}

	code, data, errCode = onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/cash_session/start", token, `{"tour_version":1}`)
	if code != http.StatusOK || data["status"] != "running" {
		t.Fatalf("start: expected 200 running, got %d %v (%s)", code, data, errCode)
	}

	code, data, errCode = onbCall(t, router, http.MethodPatch, "/v1/onboarding/tours/cash_session",
		token, `{"current_step":"opening_dialog"}`)
	if code != http.StatusOK || data["current_step"] != "opening_dialog" {
		t.Fatalf("step: expected 200 opening_dialog, got %d %v (%s)", code, data, errCode)
	}

	code, data, errCode = onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/cash_session/pause", token, "")
	if code != http.StatusOK || data["status"] != "paused" {
		t.Fatalf("pause: expected 200 paused, got %d %v (%s)", code, data, errCode)
	}
	// A pause keeps the user's place so resume lands on the same step.
	_, data, _ = onbCall(t, router, http.MethodGet, "/v1/onboarding", token, "")
	if row := tourStatus(data, "cash_session"); row["status"] != "paused" || row["current_step"] != "opening_dialog" {
		t.Fatalf("paused state: %v", row)
	}

	if code, _, errCode = onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/cash_session/complete", token, ""); code != http.StatusConflict || errCode != "goal_not_met" {
		t.Fatalf("complete before any session: expected 409 goal_not_met, got %d (%s)", code, errCode)
	}

	execTenant(t, pool, seed.TenantID, `
		INSERT INTO register_sessions (tenant_id, device_id, user_id, status)
		VALUES ($1::uuid, $2::uuid, $3::uuid, 'open')`,
		seed.TenantID, seed.DeviceID, seed.CashierID)

	code, data, errCode = onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/cash_session/complete", token, "")
	if code != http.StatusOK || data["status"] != "completed" {
		t.Fatalf("complete after session: expected 200 completed, got %d %v (%s)", code, data, errCode)
	}

	// A completed tour is retry-safe: the goal was verified once and a later
	// repeat never re-opens (nor re-checks) it.
	if code, data, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/cash_session/complete", token, ""); code != http.StatusOK || data["status"] != "completed" {
		t.Fatalf("repeat complete: expected 200 completed, got %d %v (%s)", code, data, errCode)
	}
	if code, _, errCode = onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/cash_session/start", token, ""); code != http.StatusConflict || errCode != "tour_completed" {
		t.Fatalf("restart of completed tour: expected 409 tour_completed, got %d (%s)", code, errCode)
	}

	_, data, _ = onbCall(t, router, http.MethodGet, "/v1/onboarding", token, "")
	row := tourStatus(data, "cash_session")
	if row["status"] != "completed" || row["goal_met"] != true {
		t.Fatalf("completed state: %v", row)
	}
	if rec, _ := data["recommended"].(string); rec != "store_setup" {
		t.Fatalf("recommended after completing cash_session: %q", rec)
	}
	// The open session counts for terminal_setup too, but the session row's
	// own fact is what cash_session measured.
	if ready, _ := data["readiness"].(map[string]any); ready["cash_session_opened"] != true {
		t.Fatalf("readiness cash_session_opened: %v", ready)
	}
}

func TestOnboardingIntegration_SkipVersionAndValidation(t *testing.T) {
	router, seed, _ := setupOnboarding(t)
	token := onbToken(t, seed, seed.ManagerID, "manager")

	if code, _, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/first_sale/start",
		token, `{"tour_version":99}`); code != http.StatusConflict || errCode != "tour_version_mismatch" {
		t.Fatalf("stale version: expected 409 tour_version_mismatch, got %d (%s)", code, errCode)
	}
	if code, _, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/nope/start", token, ""); code != http.StatusNotFound || errCode != "tour_not_found" {
		t.Fatalf("unknown tour start: expected 404 tour_not_found, got %d (%s)", code, errCode)
	}
	if code, _, errCode := onbCall(t, router, http.MethodPatch, "/v1/onboarding/tours/nope", token,
		`{"current_step":"x"}`); code != http.StatusNotFound || errCode != "tour_not_found" {
		t.Fatalf("unknown tour patch: expected 404 tour_not_found, got %d (%s)", code, errCode)
	}

	if code, _, errCode := onbCall(t, router, http.MethodPatch, "/v1/onboarding/tours/first_sale", token,
		`{"current_step":"grid"}`); code != http.StatusConflict || errCode != "tour_not_running" {
		t.Fatalf("step before start: expected 409 tour_not_running, got %d (%s)", code, errCode)
	}

	if code, _, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/first_sale/start", token, ""); code != http.StatusOK {
		t.Fatalf("start: expected 200, got %d (%s)", code, errCode)
	}
	if code, _, errCode := onbCall(t, router, http.MethodPatch, "/v1/onboarding/tours/first_sale", token,
		`{"current_step":"Not A Step"}`); code != http.StatusBadRequest || errCode != "invalid_step" {
		t.Fatalf("bad step: expected 400 invalid_step, got %d (%s)", code, errCode)
	}
	if code, _, errCode := onbCall(t, router, http.MethodPatch, "/v1/onboarding/tours/first_sale", token,
		`{}`); code != http.StatusBadRequest || errCode != "invalid_step" {
		t.Fatalf("empty step: expected 400 invalid_step, got %d (%s)", code, errCode)
	}
	if code, _, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/first_sale/skip", token, ""); code != http.StatusOK {
		t.Fatalf("skip: expected 200, got %d (%s)", code, errCode)
	}
	_, data, _ := onbCall(t, router, http.MethodGet, "/v1/onboarding", token, "")
	if row := tourStatus(data, "first_sale"); row["status"] != "skipped" {
		t.Fatalf("skipped state: %v", row)
	}
	// A declined tour is not the recommendation, but it stays restartable.
	if rec, _ := data["recommended"].(string); rec != "store_setup" {
		t.Fatalf("recommended with skipped tour: %q", rec)
	}
	// Skipped is not running: pausing it is an invalid transition.
	if code, _, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/first_sale/pause", token, ""); code != http.StatusConflict || errCode != "invalid_status" {
		t.Fatalf("pause a skipped tour: expected 409 invalid_status, got %d (%s)", code, errCode)
	}
	if code, _, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/first_sale/start", token, ""); code != http.StatusOK {
		t.Fatalf("restart skipped tour: expected 200, got %d (%s)", code, errCode)
	}
}

func TestOnboardingIntegration_ReadinessFollowsBusinessFacts(t *testing.T) {
	router, seed, pool := setupOnboarding(t)
	managerToken := onbToken(t, seed, seed.ManagerID, "manager")

	code, data, errCode := onbCall(t, router, http.MethodGet, "/v1/onboarding/readiness", managerToken, "")
	if code != http.StatusOK || errCode != "" {
		t.Fatalf("readiness: expected 200, got %d (%s)", code, errCode)
	}
	if data["product_ready"] != false || data["first_sale_completed"] != false || data["activated"] != false {
		t.Fatalf("fresh readiness should be empty: %v", data)
	}

	execTenant(t, pool, seed.TenantID, `UPDATE tenants SET address = '12 Test Street' WHERE id = $1::uuid`, seed.TenantID)
	execTenant(t, pool, seed.TenantID, `
		INSERT INTO products (tenant_id, name, sku, price_minor)
		VALUES ($1::uuid, 'Espresso', 'SKU-ONB-1', 500)`, seed.TenantID)

	code, data, _ = onbCall(t, router, http.MethodGet, "/v1/onboarding/readiness", managerToken, "")
	if code != http.StatusOK {
		t.Fatalf("readiness: expected 200, got %d", code)
	}
	if data["store_configured"] != true || data["product_ready"] != true {
		t.Fatalf("configured facts not visible: %v", data)
	}
	if data["terminal_configured"] != false || data["activated"] != false {
		t.Fatalf("premature facts: %v", data)
	}

	var saleID string
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", seed.TenantID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `
		INSERT INTO sales (tenant_id, idempotency_key, status, subtotal_minor, total_minor, currency)
		VALUES ($1::uuid, 'onb-first-sale', 'completed', 500, 500, 'EGP')
		RETURNING id`, seed.TenantID).Scan(&saleID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO sale_payments (tenant_id, sale_id, method, amount_minor)
		VALUES ($1::uuid, $2::uuid, 'cash', 500)`, seed.TenantID, saleID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	code, data, _ = onbCall(t, router, http.MethodGet, "/v1/onboarding/readiness", managerToken, "")
	if code != http.StatusOK || data["first_sale_completed"] != true || data["activated"] != true {
		t.Fatalf("after a paid completed sale: %d %v", code, data)
	}

	// The goal being met lets first_sale complete without trusting the client.
	if code, _, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/first_sale/start", managerToken, ""); code != http.StatusOK {
		t.Fatalf("start first_sale: %d (%s)", code, errCode)
	}
	code, data, errCode = onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/first_sale/complete", managerToken, "")
	if code != http.StatusOK || data["status"] != "completed" {
		t.Fatalf("complete first_sale: expected 200 completed, got %d %v (%s)", code, data, errCode)
	}
}

func TestOnboardingIntegration_EventsAreIdempotent(t *testing.T) {
	router, seed, pool := setupOnboarding(t)
	token := onbToken(t, seed, seed.CashierID, "cashier")
	eventID := "0a5f4d0c-1f4c-4e9f-9a2d-8b1c2d3e4f50"

	body := `{"event_id":"` + eventID + `","event":"sale.completed","payload":{"total_minor":500}}`
	code, data, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/events", token, body)
	if code != http.StatusCreated || data["replayed"] != false {
		t.Fatalf("first event: expected 201 replayed=false, got %d %v (%s)", code, data, errCode)
	}

	code, data, errCode = onbCall(t, router, http.MethodPost, "/v1/onboarding/events", token, body)
	if code != http.StatusOK || data["replayed"] != true {
		t.Fatalf("replay: expected 200 replayed=true, got %d %v (%s)", code, data, errCode)
	}

	var count int
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", seed.TenantID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM onboarding_events WHERE event_id = $1::uuid`, eventID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("event rows: expected exactly 1, got %d", count)
	}

	if code, _, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/events", token,
		`{"event_id":"bogus","event":"sale.completed"}`); code != http.StatusBadRequest || errCode != "invalid_event" {
		t.Fatalf("bad event id: expected 400 invalid_event, got %d (%s)", code, errCode)
	}
	if code, _, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/events", token,
		`{"event_id":"0a5f4d0c-1f4c-4e9f-9a2d-8b1c2d3e4f51","event":"Sale.Completed"}`); code != http.StatusBadRequest || errCode != "invalid_event" {
		t.Fatalf("bad event name: expected 400 invalid_event, got %d (%s)", code, errCode)
	}
}

func TestOnboardingIntegration_TenantIsolation(t *testing.T) {
	router, seedA, pool := setupOnboarding(t)
	tokenA := onbToken(t, seedA, seedA.ManagerID, "manager")
	seedB := testutil.SeedTenant(t, pool)
	tokenB := onbToken(t, seedB, seedB.ManagerID, "manager")

	if code, _, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/cash_session/start", tokenA, ""); code != http.StatusOK {
		t.Fatalf("tenant A start: %d (%s)", code, errCode)
	}
	execTenant(t, pool, seedA.TenantID, `UPDATE tenants SET address = 'Cairo' WHERE id = $1::uuid`, seedA.TenantID)

	_, dataB, _ := onbCall(t, router, http.MethodGet, "/v1/onboarding", tokenB, "")
	if row := tourStatus(dataB, "cash_session"); row["status"] != "not_started" {
		t.Fatalf("tenant B sees A's progress: %v", row)
	}
	if ready, _ := dataB["readiness"].(map[string]any); ready["store_configured"] != false {
		t.Fatalf("tenant B sees A's readiness: %v", ready)
	}

	// Each tenant keeps its own progress rows for the same user identity.
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, tenantID := range []string{seedA.TenantID, seedB.TenantID} {
		var n int
		if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
			t.Fatal(err)
		}
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM onboarding_progress WHERE tenant_id = $1::uuid`, tenantID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		want := 1
		if tenantID == seedB.TenantID {
			want = 0
		}
		if n != want {
			t.Fatalf("tenant %s progress rows: expected %d, got %d", tenantID, want, n)
		}
	}
}
