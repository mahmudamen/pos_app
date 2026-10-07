package onboarding

import (
	"context"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// --- unit tests: aggregation helpers (EPIC 13) ---

func TestAggregateToursCatalogOrder(t *testing.T) {
	counts := map[string]map[string]int{
		"cash_session": {"completed": 2, "skipped": 1},
		"first_sale":   {"running": 1},
		"store_setup":  {"not_started": 1},
	}
	tours := aggregateTours(counts)
	if len(tours) != len(catalog) {
		t.Fatalf("want %d tours, got %d", len(catalog), len(tours))
	}
	if tours[0].ID != "store_setup" {
		t.Fatalf("catalog order violated, first = %q", tours[0].ID)
	}
	byID := map[string]tourAggregate{}
	for _, a := range tours {
		byID[a.ID] = a
	}
	cs := byID["cash_session"]
	if cs.Started != 3 || cs.Completed != 2 || cs.Skipped != 1 || cs.InProgress != 0 {
		t.Fatalf("cash_session aggregate: %+v", cs)
	}
	fs := byID["first_sale"]
	if fs.Started != 1 || fs.InProgress != 1 {
		t.Fatalf("first_sale aggregate: %+v", fs)
	}
	if ss := byID["store_setup"]; ss.Started != 0 {
		t.Fatalf("not_started row must not count as started: %+v", ss)
	}
}

func TestAggregateTotalsAndRates(t *testing.T) {
	tours := []tourAggregate{
		{ID: "a", Started: 4, Completed: 2, Skipped: 1, InProgress: 1},
		{ID: "b", Started: 2, Completed: 1},
	}
	tt := aggregateTotals(tours)
	if tt.Started != 6 || tt.Completed != 3 || tt.Skipped != 1 || tt.InProgress != 1 {
		t.Fatalf("totals: %+v", tt)
	}
	completion, skip := completionAndSkipRate(tt.Started, tt.Completed, tt.Skipped)
	if completion != 0.5 || skip != 0.167 {
		t.Fatalf("rates: completion %v skip %v", completion, skip)
	}
	if c, s := completionAndSkipRate(0, 0, 0); c != 0 || s != 0 {
		t.Fatalf("zero-rate: %v %v", c, s)
	}
	if c, s := completionAndSkipRate(1, 0, 0); c != 0 || s != 0 {
		t.Fatalf("incomplete-zero-rate: %v %v", c, s)
	}
}

func TestEventCountsSortedKeys(t *testing.T) {
	e := eventCounts{"onboarding.completed": 1, "onboarding.started": 2}
	keys := e.sortedKeys()
	if keys[0] != "onboarding.completed" || keys[1] != "onboarding.started" {
		t.Fatalf("keys not sorted: %v", keys)
	}
}

func TestRewardRouteRegistered(t *testing.T) {
	router := setupRouter()
	routes := map[string]bool{}
	for _, r := range router.Routes() {
		routes[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		"GET /v1/onboarding/summary",
		"POST /v1/onboarding/reward",
	} {
		if !routes[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}

// The recorded EPIC-11 decision is tested: the constant stays
// ONBOARDING_ACTIVATION and is NOT the spec's 30-day extension.
func TestDecisionMirrorsConstants(t *testing.T) {
	if rewardType == "ONBOARDING_30_DAY_EXTENSION" {
		t.Fatal("30-day extension must stay unimplemented by decision")
	}
	if rewardType != "ONBOARDING_ACTIVATION" {
		t.Fatalf("reward type changed unexpectedly: %q", rewardType)
	}
}

// --- DB-backed integration: reward (EPIC 11) + summary (EPIC 13) ---

// activateTenant writes a completed, paid sale so readiness.activated flips.
func activateTenant(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
		t.Fatal(err)
	}
	var saleID string
	if err = tx.QueryRow(ctx, `
		INSERT INTO sales (tenant_id, idempotency_key, status, subtotal_minor, total_minor, currency)
		VALUES ($1::uuid, 'onb-reward-' || gen_random_uuid()::text, 'completed', 500, 500, 'EGP')
		RETURNING id`, tenantID).Scan(&saleID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `
		INSERT INTO sale_payments (tenant_id, sale_id, method, amount_minor)
		VALUES ($1::uuid, $2::uuid, 'cash', 500)`, tenantID, saleID); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestOnboardingIntegration_RewardAuthorityAndIdempotency(t *testing.T) {
	router, seed, pool := setupOnboarding(t)
	token := onbToken(t, seed, seed.ManagerID, "manager")

	// Before activation the server refuses: the client can never grant itself.
	code, data, errCode := onbCall(t, router, http.MethodGet, "/v1/onboarding", token, "")
	if code != http.StatusOK {
		t.Fatalf("state: %d (%s)", code, errCode)
	}
	reward, _ := data["reward"].(map[string]any)
	if reward["eligible"] != false || reward["granted"] != false {
		t.Fatalf("ineligible reward before activation: %v", reward)
	}
	if code, data, errCode := onbCall(t, router, http.MethodPost, "/v1/onboarding/reward", token, ""); code != http.StatusOK || data["eligible"] != false || data["granted"] != false {
		t.Fatalf("claim before activation must be a no-op state read: %d %v (%s)", code, data, errCode)
	}

	activateTenant(t, pool, seed.TenantID)

	// Now the reward is eligible and claimable exactly once.
	code, data, errCode = onbCall(t, router, http.MethodGet, "/v1/onboarding", token, "")
	if code != http.StatusOK {
		t.Fatalf("state after activation: %d (%s)", code, errCode)
	}
	reward, _ = data["reward"].(map[string]any)
	if reward["eligible"] != true || reward["granted"] != false {
		t.Fatalf("eligible state: %v", reward)
	}

	code, data, errCode = onbCall(t, router, http.MethodPost, "/v1/onboarding/reward", token, "")
	if code != http.StatusOK || data["granted"] != true || data["replayed"] != false {
		t.Fatalf("first claim: expected fresh grant, got %d %v (%s)", code, data, errCode)
	}
	if rt, _ := data["reward_type"].(string); rt != rewardType {
		t.Fatalf("reward_type echo: got %q want %q", rt, rewardType)
	}

	// Idempotent: a second claim replays, does not double-apply.
	code, data, errCode = onbCall(t, router, http.MethodPost, "/v1/onboarding/reward", token, "")
	if code != http.StatusOK || data["granted"] != true || data["replayed"] != true {
		t.Fatalf("second claim: expected replayed grant, got %d %v (%s)", code, data, errCode)
	}
	var grants int
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", seed.TenantID); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx,
		`SELECT count(*) FROM onboarding_rewards WHERE user_id = $1::uuid`, seed.ManagerID).Scan(&grants); err != nil {
		t.Fatal(err)
	}
	if grants != 1 {
		t.Fatalf("reward rows: expected exactly 1, got %d", grants)
	}
	// The grant writes a server-side analytics event exactly once.
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM onboarding_events
		WHERE event = 'trial.reward_granted'`).Scan(&grants); err != nil {
		t.Fatal(err)
	}
	if grants != 1 {
		t.Fatalf("reward event rows: expected exactly 1, got %d", grants)
	}

	// Another user of the same tenant is separately rewarded (per-user).
	cashierToken := onbToken(t, seed, seed.CashierID, "cashier")
	if code, data, _ := onbCall(t, router, http.MethodPost, "/v1/onboarding/reward", cashierToken, ""); code != http.StatusOK || data["granted"] != true || data["replayed"] != false {
		t.Fatalf("cashier claim: expected fresh grant, got %d %v", code, data)
	}
}

func TestOnboardingIntegration_SummaryAggregates(t *testing.T) {
	router, seed, pool := setupOnboarding(t)
	token := onbToken(t, seed, seed.ManagerID, "manager")

	// Guest is outside dashboard.read: summary must 403 for them.
	guestToken := onbToken(t, seed, seed.CashierID, "guest")
	if code, _, errCode := onbCall(t, router, http.MethodGet, "/v1/onboarding/summary", guestToken, ""); code != http.StatusForbidden || errCode != "permission_denied" {
		t.Fatalf("guest summary: expected 403 permission_denied, got %d (%s)", code, errCode)
	}

	// Seed progress + events, then activate.
	for _, tour := range []string{"store_setup", "first_product"} {
		if code, _, _ := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/"+tour+"/start", token, "{}"); code != http.StatusOK {
			t.Fatalf("start %s: %d", tour, code)
		}
	}
	if code, _, _ := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/store_setup/complete", token, ""); code != http.StatusConflict {
		t.Fatalf("complete with no address must 409, got %d", code)
	}
	execTenant(t, pool, seed.TenantID, `UPDATE tenants SET address = 'Giza' WHERE id = $1::uuid`, seed.TenantID)
	if code, _, _ := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/store_setup/complete", token, ""); code != http.StatusOK {
		t.Fatalf("complete store_setup after address: %d", code)
	}
	if code, _, _ := onbCall(t, router, http.MethodPost, "/v1/onboarding/tours/first_product/skip", token, ""); code != http.StatusOK {
		t.Fatalf("skip first_product: %d", code)
	}
	onbCall(t, router, http.MethodPost, "/v1/onboarding/events", token, `{"event_id":"`+testUUID()+`","event":"onboarding.started","payload":{"tour":"store_setup"}}`)
	onbCall(t, router, http.MethodPost, "/v1/onboarding/events", token, `{"event_id":"`+secondUUID()+`","event":"onboarding.viewed","payload":{}}`)

	activateTenant(t, pool, seed.TenantID)
	if code, _, _ := onbCall(t, router, http.MethodPost, "/v1/onboarding/reward", token, ""); code != http.StatusOK {
		t.Fatalf("reward: %d", code)
	}

	code, data, errCode := onbCall(t, router, http.MethodGet, "/v1/onboarding/summary", token, "")
	if code != http.StatusOK || errCode != "" {
		t.Fatalf("summary: expected 200, got %d (%s)", code, errCode)
	}
	events, _ := data["events"].(map[string]any)
	if events["onboarding.started"] != float64(1) || events["onboarding.viewed"] != float64(1) {
		t.Fatalf("event counts: %v", events)
	}
	tours, _ := data["tours"].([]any)
	if len(tours) != len(catalog) {
		t.Fatalf("tours: expected %d, got %d", len(catalog), len(tours))
	}
	totals, _ := data["totals"].(map[string]any)
	if totals["started"] != float64(2) || totals["completed"] != float64(1) || totals["skipped"] != float64(1) {
		t.Fatalf("totals: %v", totals)
	}
	rates, _ := data["rates"].(map[string]any)
	if rates["completion_rate"] != float64(0.5) || rates["skip_rate"] != float64(0.5) {
		t.Fatalf("rates: %v", rates)
	}
	if grants, _ := data["reward_grants"].(float64); grants != 1 {
		t.Fatalf("reward_grants: %v", data["reward_grants"])
	}
	if data["time_to_first_sale_seconds"] == nil {
		t.Fatalf("time_to_first_sale_seconds should be numeric after activation: %v", data)
	}
	if activated, _ := data["activated"].(bool); !activated {
		t.Fatal("activated should be true")
	}
}

func testUUID() string { return "9f6a4c11-9d2b-4e3f-8a5c-1d2e3f4a5b6c" }

func secondUUID() string { return "9f6a4c11-9d2b-4e3f-8a5c-1d2e3f4a5b6d" }
