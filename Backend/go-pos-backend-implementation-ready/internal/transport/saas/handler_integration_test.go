package saas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	prod2 := fmt.Sprintf(`{"name":"Espresso","sku":"ESP-1","price_minor":300,"currency":"EGP","stock_quantity":5}`)
	rec = postJSON(t, router, "/v1/products", mgr, prod2)
	if rec.Code != http.StatusConflict {
		t.Fatalf("over-limit product: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("plan_limit_exceeded")) {
		t.Fatalf("over-limit product: expected plan_limit_exceeded, got %s", rec.Body.String())
	}
}
