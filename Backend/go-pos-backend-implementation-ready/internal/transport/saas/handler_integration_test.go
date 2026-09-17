package saas

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	catalogtransport "github.com/example/pos-api/internal/transport/catalog"
	usertransport "github.com/example/pos-api/internal/transport/users"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB-backed slice of D2 (SaaS admin polish): per-tenant analytics drill-down
// and plan-limit enforcement against a real PostgreSQL. Skips without
// TEST_DATABASE_URL / DATABASE_URL like the other integration suites.

func setupSaaSIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	tokens := testutil.TokenManager()
	api := router.Group("/v1")
	NewHandler(pool, tokens).Register(api.Group("/saas"))
	usertransport.NewHandler(pool, tokens).Register(api)
	catalogtransport.NewHandler(pool, tokens).Register(api)
	return router, seed, pool
}

func saasToken(t *testing.T, seed testutil.Seed) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-saas", "saas_admin")
}

func managerToken(t *testing.T, seed testutil.Seed) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-mgr", "manager")
}

func get(t *testing.T, router http.Handler, path, token string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(rec, req)
	body := map[string]any{}
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec, body
}

func postJSON(t *testing.T, router http.Handler, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

func setTenantPlan(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool, maxUsers, maxProducts int) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx,
		`UPDATE tenants SET plan = 'test', max_users = $1, max_products = $2 WHERE id = $3::uuid`,
		maxUsers, maxProducts, seed.TenantID); err != nil {
		t.Fatal(err)
	}
}

// seedSale inserts a bare sales row for the tenant inside its RLS context so
// the SaaS aggregates (which run the same tenant-scoped queries) pick it up.
func seedSale(t *testing.T, pool *pgxpool.Pool, tenantID, idempotencyKey, status string, amountMinor int) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO sales (tenant_id, idempotency_key, status, subtotal_minor, total_minor, currency)
		VALUES ($1::uuid, $2, $3, $4, $4, 'EGP')`,
		tenantID, idempotencyKey, status, amountMinor); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

// seedPlatformTenant ensures a slug-'saas' internal platform tenant exists and
// returns its id. Only used to prove it is excluded from merchant aggregates.
func seedPlatformTenant(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO tenants (name, slug, business_type)
		VALUES ('POS Go SaaS Platform', 'saas', 'general')
		ON CONFLICT (slug) DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM tenants WHERE slug = 'saas'`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// TestSaaSIntegration_SummaryAndList covers the enriched summary (per-type
// users/revenue, internal-tenant exclusion) and the filtered tenant list
// (revenue/sales/created_at, q/business_type/status filters, include_internal).
func TestSaaSIntegration_SummaryAndList(t *testing.T) {
	router, seed, pool := setupSaaSIntegration(t)
	token := saasToken(t, seed)

	// Give the seeded tenant a real business type and a completed + refunded
	// sale: revenue must only count the completed one.
	if _, err := pool.Exec(context.Background(),
		`UPDATE tenants SET business_type = 'restaurant', owner_user_id = $2::uuid WHERE id = $1::uuid`,
		seed.TenantID, seed.ManagerID); err != nil {
		t.Fatal(err)
	}
	seedSale(t, pool, seed.TenantID, "saas-int-completed", "completed", 1000)
	seedSale(t, pool, seed.TenantID, "saas-int-refunded", "refunded", 2000)
	platformTenantID := seedPlatformTenant(t, pool)

	var totalTenants int64
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM tenants WHERE slug <> 'saas'`).Scan(&totalTenants); err != nil {
		t.Fatal(err)
	}

	// Summary: the platform tenant is excluded and type breakdown carries
	// users + revenue alongside the tenant count.
	rec, body := get(t, router, "/v1/saas/summary", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("summary: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	data := body["data"].(map[string]any)
	if int64(data["total_tenants"].(float64)) != totalTenants {
		t.Fatalf("summary should exclude the saas platform tenant: got %v, want %d",
			data["total_tenants"], totalTenants)
	}
	if data["revenue_minor"].(float64) < 1000 {
		t.Fatalf("summary should count the completed sale: %v", data["revenue_minor"])
	}
	if data["total_sales"].(float64) < 1 {
		t.Fatalf("summary should count the completed sale: %v", data["total_sales"])
	}
	byBusiness := data["by_business"].([]any)
	var restaurant *map[string]any
	for _, entry := range byBusiness {
		e := entry.(map[string]any)
		if e["business_type"] == "restaurant" {
			restaurant = &e
			break
		}
	}
	if restaurant == nil {
		t.Fatalf("summary should carry a restaurant breakdown: %v", byBusiness)
	}
	if (*restaurant)["tenants"].(float64) < 1 || (*restaurant)["users"].(float64) < 2 {
		t.Fatalf("summary restaurant breakdown should carry tenants+users: %v", *restaurant)
	}
	if (*restaurant)["revenue_minor"].(float64) < 1000 {
		t.Fatalf("summary restaurant breakdown should carry revenue: %v", *restaurant)
	}

	// Tenant list: default view hides the platform tenant and each row carries
	// born-on date, owner, and completed-only revenue.
	rec, body = get(t, router, "/v1/saas/tenants?q="+seed.Slug, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list tenants: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	list := body["data"].([]any)
	if len(list) != 1 {
		t.Fatalf("q filter should find exactly the seeded tenant: %v", body["data"])
	}
	row := list[0].(map[string]any)
	if row["id"] != seed.TenantID {
		t.Fatalf("q filter matched the wrong tenant: %v", row)
	}
	if row["revenue_minor"].(float64) != 1000 || row["total_sales"].(float64) != 1 {
		t.Fatalf("tenant revenue must exclude the refunded sale: %v", row)
	}
	if row["created_at"] == "" || row["owner_user_id"] == "" || row["status"] != "active" {
		t.Fatalf("tenant list should expose created_at/owner/status: %v", row)
	}

	// business_type + status filters narrow the page.
	rec, body = get(t, router, "/v1/saas/tenants?business_type=restaurant&status=active", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("filtered list: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(body["data"].([]any)) < 1 {
		t.Fatalf("business_type+status filter should still match the tenant: %v", body["data"])
	}
	rec, body = get(t, router, "/v1/saas/tenants?status=suspended", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("suspended list: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if len(body["data"].([]any)) != 0 {
		t.Fatalf("no tenant should be suspended yet: %v", body["data"])
	}

	// The platform tenant only appears when explicitly requested.
	rec, body = get(t, router, "/v1/saas/tenants?include_internal=1&q=saas", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("internal list: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	seenPlatform := false
	for _, item := range body["data"].([]any) {
		if item.(map[string]any)["id"] == platformTenantID {
			seenPlatform = true
		}
	}
	if !seenPlatform {
		t.Fatalf("include_internal should expose the platform tenant: %v", body["data"])
	}
}

func TestSaaSIntegration_TenantAnalytics(t *testing.T) {
	router, seed, pool := setupSaaSIntegration(t)
	_ = pool
	token := saasToken(t, seed)

	// Unknown tenant → 404, malformed id → 400.
	rec, _ := get(t, router, "/v1/saas/tenants/00000000-0000-0000-0000-0000000000ff/analytics", token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown tenant: expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
	rec, _ = get(t, router, "/v1/saas/tenants/not-a-uuid/analytics", token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad tenant id: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}

	rec, body := get(t, router, "/v1/saas/tenants/"+seed.TenantID+"/analytics", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("analytics: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	data := body["data"].(map[string]any)
	tenant := data["tenant"].(map[string]any)
	if tenant["id"] != seed.TenantID || tenant["plan"] != "standard" || tenant["business_type"] == "" {
		t.Fatalf("analytics tenant facts wrong: %v", tenant)
	}
	counts := data["counts"].(map[string]any)
	if counts["users"].(float64) < 2 {
		t.Fatalf("analytics should see the seeded users: %v", counts)
	}
	if _, ok := data["today"].(map[string]any)["revenue_minor"]; !ok {
		t.Fatalf("analytics missing today block: %v", data)
	}
	if _, ok := data["revenue_trend"].([]any); !ok {
		t.Fatalf("analytics missing revenue_trend: %v", data)
	}
	if _, ok := data["top_products"].([]any); !ok {
		t.Fatalf("analytics missing top_products: %v", data)
	}
	if _, ok := data["recent_sales"].([]any); !ok {
		t.Fatalf("analytics missing recent_sales: %v", data)
	}

	// The tenant list exposes plan fields too.
	rec, body = get(t, router, "/v1/saas/tenants", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("list tenants: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	list, err := json.Marshal(body["data"])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(list, []byte(`"plan":"standard"`)) {
		t.Fatalf("tenant list should include plan: %s", list)
	}
}

func TestSaaSIntegration_PlanLimitEnforcedOnUsers(t *testing.T) {
	router, seed, pool := setupSaaSIntegration(t)
	mgr := managerToken(t, seed)
	setTenantPlan(t, seed, pool, 2, 0) // seeded manager + cashier already exist

	createBody := `{"email":"new@example.com","display_name":"New","password":"password123","role":"cashier"}`
	rec := postJSON(t, router, "/v1/users", mgr, createBody)
	if rec.Code != http.StatusConflict {
		t.Fatalf("over-limit user: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("plan_limit_exceeded")) {
		t.Fatalf("over-limit user: expected plan_limit_exceeded, got %s", rec.Body.String())
	}

	setTenantPlan(t, seed, pool, 3, 0)
	rec = postJSON(t, router, "/v1/users", mgr, createBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("under-limit user: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSaaSIntegration_PlanLimitEnforcedOnProducts(t *testing.T) {
	router, seed, pool := setupSaaSIntegration(t)
	mgr := managerToken(t, seed)
	setTenantPlan(t, seed, pool, 0, 1)

	prod := `{"name":"Latte","sku":"LAT-1","price_minor":400,"currency":"EGP","stock_quantity":5}`
	rec := postJSON(t, router, "/v1/products", mgr, prod)
	if rec.Code != http.StatusCreated {
		t.Fatalf("first product: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	prod2 := `{"name":"Espresso","sku":"ESP-1","price_minor":300,"currency":"EGP","stock_quantity":5}`
	rec = postJSON(t, router, "/v1/products", mgr, prod2)
	if rec.Code != http.StatusConflict {
		t.Fatalf("over-limit product: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("plan_limit_exceeded")) {
		t.Fatalf("over-limit product: expected plan_limit_exceeded, got %s", rec.Body.String())
	}
}
