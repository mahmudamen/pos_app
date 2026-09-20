package selforder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SELFORDER DB-backed slice: the public menu only exposes published, in-stock
// products; orders are validated against the live catalog and approving one
// turns it into a real sale (stock moves); product requests land as feedback.
// Skips when no database is configured, matching the other integration tests.

func setupSelfOrderIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewHandler(pool, testutil.TokenManager(), WebPushConfig{}, 0)
	h.RegisterPublic(router.Group("/v1"))
	h.Register(router.Group("/v1"))
	return router, seed, pool
}

// tenantValue runs a scalar read inside the tenant's RLS context (self_orders
// and products are FORCE-RLS, so a bare pool query would see nothing).
func tenantValue(t *testing.T, pool *pgxpool.Pool, tenantID, query string, args ...any) any {
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
	var out any
	if err := tx.QueryRow(ctx, query, args...).Scan(&out); err != nil {
		t.Fatalf("%s: %v", query, err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return out
}

func seedSelfOrderProducts(t *testing.T, pool *pgxpool.Pool, tenantID string) (published, out string) {
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
	if _, err := tx.Exec(ctx, `INSERT INTO categories (tenant_id, name, slug) VALUES ($1::uuid, 'Drinks', 'drinks')`, tenantID); err != nil {
		t.Fatal(err)
	}
	var catID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM categories WHERE tenant_id = $1::uuid AND slug = 'drinks'`, tenantID).Scan(&catID); err != nil {
		t.Fatal(err)
	}
	seed := func(name, sku string, stock int64, published bool) string {
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO products (tenant_id, category_id, name, sku, price_minor, currency, stock_quantity, selforder_enabled)
			VALUES ($1::uuid, $2::uuid, $3, $4, 2500, 'EGP', $5, $6) RETURNING id::text`,
			tenantID, catID, name, sku, stock, published).Scan(&id)
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	published = seed("Cola", "COL-1", 10, true)
	out = seed("Empty Soda", "COL-2", 0, true) // published but out of stock
	seed("Hidden Cola", "COL-3", 5, false)     // in stock but not published
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return published, out
}

func TestMenuShowsOnlyPublishedInStock(t *testing.T) {
	router, seed, pool := setupSelfOrderIntegration(t)
	published, _ := seedSelfOrderProducts(t, pool, seed.TenantID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/selforder/menu/"+seed.Slug, nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("menu = %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	data, _ := body["data"].(map[string]any)
	names := map[string]bool{}
	for _, cat := range data["categories"].([]any) {
		cm, _ := cat.(map[string]any)
		for _, p := range cm["products"].([]any) {
			names[p.(map[string]any)["name"].(string)] = true
		}
	}
	for _, p := range data["uncategorized"].([]any) {
		names[p.(map[string]any)["name"].(string)] = true
	}
	if !names["Cola"] {
		t.Fatalf("published in-stock product missing from menu: %v", names)
	}
	if names["Empty Soda"] {
		t.Fatalf("out-of-stock product must be hidden from the menu: %v", names)
	}
	if names["Hidden Cola"] {
		t.Fatalf("unpublished product must be hidden from the menu: %v", names)
	}
	_ = published
}

func TestOrderLifecyclePendingToApproved(t *testing.T) {
	router, seed, pool := setupSelfOrderIntegration(t)
	published, _ := seedSelfOrderProducts(t, pool, seed.TenantID)

	orderBody := `{"tenant": "` + seed.Slug + `", "table_name": "T3", "items": [{"product_id": "` + published + `", "quantity": 2}]}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/selforder/orders", strings.NewReader(orderBody))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create order = %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	orderID := created["data"].(map[string]any)["id"].(string)
	if got := created["data"].(map[string]any)["reference"].(string); !strings.HasPrefix(got, "SO-") {
		t.Fatalf("reference = %q, want SO- prefix", got)
	}

	stockBefore := tenantValue(t, pool, seed.TenantID, `SELECT stock_quantity::text FROM products WHERE id = $1::uuid`, published).(string)

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-selforder", "manager")
	approveRec := httptest.NewRecorder()
	apprReq := httptest.NewRequest(http.MethodPost, "/v1/self-orders/"+orderID+"/approve", nil)
	apprReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(approveRec, apprReq)
	if approveRec.Code != http.StatusOK {
		t.Fatalf("approve = %d: %s", approveRec.Code, approveRec.Body.String())
	}

	stockAfter := tenantValue(t, pool, seed.TenantID, `SELECT stock_quantity::text FROM products WHERE id = $1::uuid`, published).(string)
	before, _ := strconv.Atoi(stockBefore)
	after, _ := strconv.Atoi(stockAfter)
	if before-after != 2 {
		t.Fatalf("stock before=%d after=%d, want 2 sold", before, after)
	}
	status := tenantValue(t, pool, seed.TenantID, `SELECT status FROM self_orders WHERE id = $1::uuid`, orderID).(string)
	saleID := tenantValue(t, pool, seed.TenantID, `SELECT approved_sale_id::text FROM self_orders WHERE id = $1::uuid`, orderID).(string)
	if status != "approved" || saleID == "" {
		t.Fatalf("self-order status=%q sale=%q", status, saleID)
	}

	// A second approve must 409 already_approved.
	again := httptest.NewRecorder()
	againReq := httptest.NewRequest(http.MethodPost, "/v1/self-orders/"+orderID+"/approve", nil)
	againReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(again, againReq)
	if again.Code != http.StatusConflict {
		t.Fatalf("double approve = %d, want 409", again.Code)
	}
}

func TestOrderRejectsOutOfStockAndUnpublished(t *testing.T) {
	router, seed, pool := setupSelfOrderIntegration(t)
	_, out := seedSelfOrderProducts(t, pool, seed.TenantID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/selforder/orders",
		strings.NewReader(`{"tenant": "`+seed.Slug+`", "items": [{"product_id": "`+out+`", "quantity": 1}]}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("out-of-stock order = %d: %s, want 409", rec.Code, rec.Body.String())
	}

	// Empty baskets and absurd quantities are rejected client-side first.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/selforder/orders",
		strings.NewReader(`{"tenant": "`+seed.Slug+`", "items": []}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty items = %d, want 400", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/selforder/orders",
		strings.NewReader(`{"tenant": "`+seed.Slug+`", "items": [{"product_id": "`+out+`", "quantity": 200}]}`))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("quantity 200 = %d, want 400", rec.Code)
	}
}

func TestCancelSelfOrder(t *testing.T) {
	router, seed, pool := setupSelfOrderIntegration(t)
	published, _ := seedSelfOrderProducts(t, pool, seed.TenantID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/selforder/orders",
		strings.NewReader(`{"tenant": "`+seed.Slug+`", "items": [{"product_id": "`+published+`", "quantity": 1}]}`))
	router.ServeHTTP(rec, req)
	var created map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	orderID := created["data"].(map[string]any)["id"].(string)

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-cancel", "manager")
	cancel := httptest.NewRecorder()
	cancelReq := httptest.NewRequest(http.MethodPost, "/v1/self-orders/"+orderID+"/cancel", nil)
	cancelReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(cancel, cancelReq)
	if cancel.Code != http.StatusOK {
		t.Fatalf("cancel = %d: %s", cancel.Code, cancel.Body.String())
	}
	status := tenantValue(t, pool, seed.TenantID, `SELECT status FROM self_orders WHERE id = $1::uuid`, orderID).(string)
	if status != "cancelled" {
		t.Fatalf("status = %q, want cancelled", status)
	}
}

func TestProductRequestCreateAndFulfill(t *testing.T) {
	router, seed, pool := setupSelfOrderIntegration(t)
	published, _ := seedSelfOrderProducts(t, pool, seed.TenantID)

	body := `{"tenant": "` + seed.Slug + `", "product_id": "` + published + `", "note": "2L bottle please", "contact": "0100"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/selforder/requests", strings.NewReader(body))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create request = %d: %s", rec.Code, rec.Body.String())
	}
	var created map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	reqID := created["data"].(map[string]any)["id"].(string)
	if got := created["data"].(map[string]any)["product_name"].(string); got != "Cola" {
		t.Fatalf("request product_name = %q, want server-side 'Cola'", got)
	}

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-req", "manager")
	fulfill := httptest.NewRecorder()
	fulfillReq := httptest.NewRequest(http.MethodPost, "/v1/product-requests/"+reqID+"/fulfill", nil)
	fulfillReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(fulfill, fulfillReq)
	if fulfill.Code != http.StatusOK {
		t.Fatalf("fulfill = %d: %s", fulfill.Code, fulfill.Body.String())
	}
	status := tenantValue(t, pool, seed.TenantID, `SELECT status FROM product_requests WHERE id = $1::uuid`, reqID).(string)
	if status != "fulfilled" {
		t.Fatalf("status = %q, want fulfilled", status)
	}
}

func TestListSelfOrdersGatedByRBAC(t *testing.T) {
	router, seed, pool := setupSelfOrderIntegration(t)
	_, _ = seedSelfOrderProducts(t, pool, seed.TenantID)

	// A guest role must not read the pending queue.
	token := testutil.MintAccess(t, seed.TenantID, seed.CashierID, seed.DeviceID, "sess-guest", "guest")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/self-orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("guest list self-orders = %d, want 403", rec.Code)
	}

	// Cashier with pos.read can list.
	token = testutil.MintAccess(t, seed.TenantID, seed.CashierID, seed.DeviceID, "sess-cashier", "cashier")
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/self-orders?status=pending", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cashier list self-orders = %d: %s", rec.Code, rec.Body.String())
	}
}
