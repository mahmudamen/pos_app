package inventory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/example/pos-api/internal/transport/settings"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB-backed slice for inventory adjustments: the negative-stock gate honors
// the tenant's inventory.allow_negative_stock setting. Skips offline.

func setupInventoryIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, testutil.TokenManager()).Register(router.Group("/v1"))
	return router, seed, pool
}

func inventoryToken(t *testing.T, seed testutil.Seed) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-inv", "manager")
}

func insertInventoryProduct(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool, stock int64) string {
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
		 VALUES ($1::uuid, 'Vendor', 'VEN-INV', 100, 'EGP', $2) RETURNING id::text`,
		seed.TenantID, stock).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return id
}

type adjustmentResponse struct {
	Data struct {
		ID string `json:"id"`
	} `json:"data"`
	Meta struct {
		StockQuantity int64 `json:"stock_quantity"`
	} `json:"meta"`
}

func postAdjustment(t *testing.T, router http.Handler, token, body string) (int, int64) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/inventory/adjustments", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	var resp adjustmentResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec.Code, resp.Meta.StockQuantity
}

func TestInventoryIntegration_RejectsNegativeWithoutSetting(t *testing.T) {
	router, seed, pool := setupInventoryIntegration(t)
	token := inventoryToken(t, seed)
	prodID := insertInventoryProduct(t, seed, pool, 10)
	body := fmt.Sprintf(`{"product_id":%q,"reason":"damaged","quantity_delta":-11}`, prodID)

	code, _ := postAdjustment(t, router, token, body)
	if code != http.StatusConflict {
		t.Fatalf("expected 409 insufficient_stock, got %d", code)
	}
}

func TestInventoryIntegration_AllowsNegativeWhenEnabled(t *testing.T) {
	router, seed, pool := setupInventoryIntegration(t)
	token := inventoryToken(t, seed)
	prodID := insertInventoryProduct(t, seed, pool, 10)
	testutil.SetTenantSetting(t, pool, seed.TenantID, settings.KeyAllowNegativeStock, "true")
	body := fmt.Sprintf(`{"product_id":%q,"reason":"damaged","quantity_delta":-11}`, prodID)

	code, stock := postAdjustment(t, router, token, body)
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", code)
	}
	if stock != -1 {
		t.Fatalf("stock = %d, want -1", stock)
	}
}

func TestInventoryIntegration_RefillRestoresToPositive(t *testing.T) {
	router, seed, pool := setupInventoryIntegration(t)
	token := inventoryToken(t, seed)
	prodID := insertInventoryProduct(t, seed, pool, 3)
	body := fmt.Sprintf(`{"product_id":%q,"reason":"restock","quantity_delta":7}`, prodID)

	code, stock := postAdjustment(t, router, token, body)
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", code)
	}
	if stock != 10 {
		t.Fatalf("stock = %d, want 10", stock)
	}
}

func TestInventoryIntegration_SetOnHandCount(t *testing.T) {
	router, seed, pool := setupInventoryIntegration(t)
	token := inventoryToken(t, seed)
	prodID := insertInventoryProduct(t, seed, pool, 8)
	body := fmt.Sprintf(`{"product_id":%q,"reason":"count","quantity_delta":-3}`, prodID)

	code, stock := postAdjustment(t, router, token, body)
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", code)
	}
	if stock != 5 {
		t.Fatalf("stock = %d, want 5", stock)
	}
}
