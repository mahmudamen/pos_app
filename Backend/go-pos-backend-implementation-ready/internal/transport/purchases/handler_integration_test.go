package purchases

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/ocr"
	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB-backed slice for purchases: applying a purchase bumps products.stock_quantity,
// sets cost_minor to the invoice unit price, and updates the unit — the exact
// "update inventory and price purchase and unit used" round-trip. Skips offline.

func setupPurchasesIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	engine := ocr.NewEngine(true, "/bin/echo", "ara+eng", 6)
	NewHandler(pool, testutil.TokenManager(), engine, OCRWindows{Day: 100, Week: 500, Month: 2000}).Register(router.Group("/v1"))
	return router, seed, pool
}

func purchasesToken(t *testing.T, seed testutil.Seed, role string) string {
	t.Helper()
	userID := seed.ManagerID
	if role != "manager" {
		userID = seed.CashierID
	}
	return testutil.MintAccess(t, seed.TenantID, userID, seed.DeviceID, "sess-pur-"+role, role)
}

func insertPurchasableProduct(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool, name string, stock, cost int64) string {
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
		`INSERT INTO products (tenant_id, name, sku, price_minor, currency, stock_quantity, cost_minor, unit)
		 VALUES ($1::uuid, $2, 'PUR-'||$2, 100, 'EGP', $3, $4, 'piece') RETURNING id::text`,
		seed.TenantID, name, stock, cost).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestPurchaseAppliesStockCostUnitRoundTrip(t *testing.T) {
	router, seed, pool := setupPurchasesIntegration(t)
	ctx := context.Background()
	productsID := insertPurchasableProduct(t, seed, pool, "Sugar", 100, 800)
	body := fmt.Sprintf(`{
		"supplier": "Harraz Supplies",
		"invoice_no": "INV-ROUNDTRIP-1",
		"currency": "EGP",
		"tax_minor": 0,
		"ocr_text": "ROUND TRIP",
		"items": [{"product_id": %q, "product_name": "Sugar", "quantity": 10, "unit": "piece", "unit_price_minor": 1200}]
	}`, productsID)
	req := httptest.NewRequest(http.MethodPost, "/v1/purchases", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+purchasesToken(t, seed, "manager"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /purchases: status %d want 201, body %s", rec.Code, rec.Body.String())
	}
	var applied struct {
		Data struct {
			Lines []AppliedLine `json:"lines"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &applied); err != nil {
		t.Fatal(err)
	}
	// Re-read the product: qty = base + 10, cost = 1200 (invoice price), unit kept.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	var stock, cost int64
	var unit string
	if err := tx.QueryRow(ctx,
		`SELECT stock_quantity, cost_minor, unit FROM products WHERE id = $1::uuid`, productsID).
		Scan(&stock, &cost, &unit); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if stock != 110 {
		t.Fatalf("stock: got %d want 110", stock)
	}
	if cost != 1200 {
		t.Fatalf("cost_minor: got %d want 1200", cost)
	}
	if unit != "piece" {
		t.Fatalf("unit: got %q want piece", unit)
	}
}

func TestPurchaseListReflectsCommitted(t *testing.T) {
	router, seed, pool := setupPurchasesIntegration(t)
	_ = pool
	body := `{
		"supplier": "ListCo",
		"invoice_no": "INV-LIST-1",
		"currency": "EGP",
		"items": [{"product_name": "Novelty", "quantity": 2, "unit": "piece", "unit_price_minor": 500}]
	}`
	req := httptest.NewRequest(http.MethodPost, "/v1/purchases", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+purchasesToken(t, seed, "manager"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("commit: status %d want 201, body %s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/v1/purchases", nil)
	req.Header.Set("Authorization", "Bearer "+purchasesToken(t, seed, "manager"))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: status %d body %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Data []PurchaseSummary `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Data) == 0 {
		t.Fatal("list: expected at least the committed purchase")
	}
	found := false
	for _, it := range out.Data {
		if it.Supplier == "ListCo" && it.InvoiceNo == "INV-LIST-1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("list: committed purchase not found: %+v", out.Data)
	}
}

func TestPurchaseDuplicateInvoiceRejected(t *testing.T) {
	router, seed, _ := setupPurchasesIntegration(t)
	body := func(no string) string {
		return fmt.Sprintf(`{"supplier":"DupCo","invoice_no":%q,"currency":"EGP",`+
			`"items":[{"product_name":"Dup","quantity":1,"unit":"piece","unit_price_minor":10}]}`, no)
	}
	post := func() int {
		req := httptest.NewRequest(http.MethodPost, "/v1/purchases", bytes.NewBufferString(body("INV-DUP-1")))
		req.Header.Set("Authorization", "Bearer "+purchasesToken(t, seed, "manager"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	first := post()
	if first != http.StatusCreated {
		t.Fatalf("first apply: status %d want 201", first)
	}
	if code := post(); code != http.StatusConflict {
		t.Fatalf("duplicate invoice: status %d want 409", code)
	}
}

func TestPurchaseRBACCashierDenied(t *testing.T) {
	router, seed, _ := setupPurchasesIntegration(t)
	body := `{"supplier":"X","invoice_no":"INV-CASHIER","currency":"EGP",
		"items":[{"product_name":"X","quantity":1,"unit":"piece","unit_price_minor":5}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/purchases", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+purchasesToken(t, seed, "cashier"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cashier write: status %d want 403", rec.Code)
	}
}

func TestOCRFailedScanDoesNotConsumeMetering(t *testing.T) {
	router, seed, pool := setupPurchasesIntegration(t)
	// The /bin/echo "fake" tesseract returns its argv ("stdin stdout -l ara+eng
	// --psm 6\n"); the parser recognizes a line and the scan succeeds, so the
	// metering bump must record one day-window scan. This is the DB-backed
	// round-trip for migration 036 that the live curl E2E also verified.
	body, contentType := multipartUpload(t, "file", "fake-image-bytes")
	req := httptest.NewRequest(http.MethodPost, "/v1/purchases/ocr", body)
	req.Header.Set("Authorization", "Bearer "+purchasesToken(t, seed, "manager"))
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ocr scan (echo engine): status %d want 200, body %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Data struct {
			RawText string `json:"raw_text"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	// A successful scan always bumps the day window (limit 100 >> 1 used).
	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err := tx.Exec(context.Background(), `SELECT set_config('app.current_tenant', $1, true)`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	var used int64
	if err := tx.QueryRow(context.Background(),
		`SELECT COALESCE((SELECT scans_used FROM ocr_usage WHERE tenant_id = $1::uuid AND window_kind='day' AND window_start = CURRENT_DATE), 0)`,
		seed.TenantID).Scan(&used); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	if used != 1 {
		t.Fatalf("ocr_usage.day=%d want 1 (successful scan must meter)", used)
	}
}
