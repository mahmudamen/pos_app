package sales

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/example/pos-api/internal/transport/settings"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SALE-008 DB-backed slice: idempotent duplicate requests and concurrent
// duplicate requests against a real PostgreSQL. Skips when no database is
// configured, matching the other integration tests.

func setupSalesIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, testutil.TokenManager()).Register(router.Group("/v1"))
	return router, seed, pool
}

func salesToken(t *testing.T, seed testutil.Seed) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-sales", "manager")
}

func insertSaleProduct(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool) string {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := tx.QueryRow(ctx,
		`INSERT INTO products (tenant_id, name, sku, price_minor, currency, stock_quantity)
		 VALUES ($1::uuid, 'Mocha', 'MOC-SALE', 500, 'EGP', 10) RETURNING id::text`,
		seed.TenantID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return id
}

type saleResponse struct {
	Data struct {
		ID         string `json:"id"`
		TotalMinor int64  `json:"total_minor"`
	} `json:"data"`
}

func postSale(t *testing.T, router http.Handler, token, key, body string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sales", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	router.ServeHTTP(rec, req)
	var s saleResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	return rec.Code, s.Data.ID
}

func countSales(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool, key string) int {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM sales WHERE idempotency_key = $1`, key).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return n
}

func checkStock(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool, productID string, want int64) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
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
	if q != want {
		t.Fatalf("stock = %d, want %d", q, want)
	}
}

func TestSalesIntegration_DuplicateRequestIsIdempotent(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	token := salesToken(t, seed)
	prodID := insertSaleProduct(t, seed, pool)
	body := fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":2}]}`, prodID)
	key := "sale-key-duplicate"

	firstCode, firstID := postSale(t, router, token, key, body)
	if firstCode != http.StatusCreated {
		t.Fatalf("first sale: expected 201, got %d", firstCode)
	}
	if firstID == "" {
		t.Fatal("first sale: empty id")
	}
	checkStock(t, seed, pool, prodID, 8)

	secondCode, secondID := postSale(t, router, token, key, body)
	if secondCode != http.StatusCreated {
		t.Fatalf("duplicate sale: expected 201, got %d", secondCode)
	}
	if secondID != firstID {
		t.Fatalf("duplicate sale: expected same sale %s, got %s", firstID, secondID)
	}
	checkStock(t, seed, pool, prodID, 8)
	if got := countSales(t, seed, pool, key); got != 1 {
		t.Fatalf("idempotency: expected exactly 1 sale row, got %d", got)
	}
}

func TestSalesIntegration_RejectsBackorderWithoutSetting(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	token := salesToken(t, seed)
	prodID := insertSaleProduct(t, seed, pool)
	body := fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":12}]}`, prodID)

	code, _ := postSale(t, router, token, "sale-key-no-backorder", body)
	if code != http.StatusConflict {
		t.Fatalf("expected 409 insufficient_stock, got %d", code)
	}
	checkStock(t, seed, pool, prodID, 10)
}

func TestSalesIntegration_BackorderSellsIntoNegative(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	token := salesToken(t, seed)
	prodID := insertSaleProduct(t, seed, pool)
	testutil.SetTenantSetting(t, pool, seed.TenantID, settings.KeyAllowNegativeStock, "true")
	body := fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":12}]}`, prodID)

	code, saleID := postSale(t, router, token, "sale-key-backorder", body)
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", code)
	}
	if saleID == "" {
		t.Fatal("backorder sale: empty id")
	}
	checkStock(t, seed, pool, prodID, -2)
}

func TestSalesIntegration_ConcurrentDuplicateRequests(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	token := salesToken(t, seed)
	prodID := insertSaleProduct(t, seed, pool)
	body := fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":1}]}`, prodID)
	const key = "sale-key-concurrent"
	const workers = 5

	var wg sync.WaitGroup
	codes := make([]int, workers)
	ids := make([]string, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i], ids[i] = postSale(t, router, token, key, body)
		}(i)
	}
	wg.Wait()

	applied := 0
	conflicted := 0
	for i := 0; i < workers; i++ {
		switch codes[i] {
		case http.StatusCreated:
			applied++
			if ids[i] == "" {
				t.Fatalf("concurrent sale %d: applied but empty id", i)
			}
		case http.StatusConflict:
			conflicted++
		default:
			t.Fatalf("concurrent sale %d: unexpected status %d", i, codes[i])
		}
	}
	if applied < 1 || applied+conflicted != workers {
		t.Fatalf("concurrent: applied=%d conflicted=%d workers=%d", applied, conflicted, workers)
	}
	checkStock(t, seed, pool, prodID, 9)
	if got := countSales(t, seed, pool, key); got != 1 {
		t.Fatalf("concurrent: expected exactly 1 sale row, got %d", got)
	}
}
