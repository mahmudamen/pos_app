package receipts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	saleshandler "github.com/example/pos-api/internal/transport/sales"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// receiptResponse mirrors the GET /sales/:id/receipt envelope.
type receiptResponse struct {
	Data Receipt `json:"data"`
}

func setupReceiptsIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool, string) {
	t.Helper()
	pool := testutil.Pool(t)
	if pool == nil {
		t.Skip("needs TEST_DATABASE_URL or DATABASE_URL")
	}
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	// sales handler creates the sale over HTTP; receipts handler reads it back
	// through the exact same Gin route table the server exposes.
	saleshandler.NewHandler(pool, testutil.TokenManager()).Register(router.Group("/v1"))
	NewHandler(pool, testutil.TokenManager()).Register(router.Group("/v1"))
	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-receipts", "manager")
	return router, seed, pool, token
}

func insertReceiptProduct(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool) string {
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
		 VALUES ($1::uuid, 'Mocha', 'MOC-RCPT', 500, 'EGP', 10) RETURNING id::text`,
		seed.TenantID).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return id
}

func postReceiptSale(t *testing.T, router http.Handler, token, key, body string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sales", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	router.ServeHTTP(rec, req)
	var r struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	return rec.Code, r.Data.ID
}

func TestReceiptsIntegration_JSONAndPrintReflectTheSale(t *testing.T) {
	router, seed, pool, token := setupReceiptsIntegration(t)
	prodID := insertReceiptProduct(t, seed, pool)
	body := fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":2}]}`, prodID)
	code, saleID := postReceiptSale(t, router, token, "sale-key-receipt", body)
	if code != http.StatusCreated || saleID == "" {
		t.Fatalf("expected 201 sale, got %d id=%q", code, saleID)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sales/"+saleID+"/receipt", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("receipt JSON status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var r receiptResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("invalid receipt JSON: %v", err)
	}
	if r.Data.SaleID != saleID {
		t.Fatalf("receipt sale_id = %q, want %q", r.Data.SaleID, saleID)
	}
	if r.Data.Status != "completed" {
		t.Fatalf("receipt status = %q, want completed", r.Data.Status)
	}
	if r.Data.TotalMinor != 1000 || r.Data.SubtotalMinor != 1000 {
		t.Fatalf("receipt totals = %d/%d, want 1000/1000", r.Data.SubtotalMinor, r.Data.TotalMinor)
	}
	if len(r.Data.Lines) != 1 || r.Data.Lines[0].Name != "Mocha" || r.Data.Lines[0].Total != 1000 {
		t.Fatalf("receipt lines unexpected: %+v", r.Data.Lines)
	}
	if len(r.Data.Payments) == 0 || r.Data.Payments[0].Amount != 1000 {
		t.Fatalf("receipt payments unexpected: %+v", r.Data.Payments)
	}
	if r.Data.TenantName == "" {
		t.Fatal("receipt tenant_name should be populated from the tenant row")
	}
	if r.Data.Cashier == "" {
		t.Fatal("receipt cashier should be populated from the sale created_by")
	}

	// Re-read the sale through a fresh tx: the receipt handler must leave the DB intact.
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	var stock int64
	if err := tx.QueryRow(ctx, `SELECT stock_quantity FROM products WHERE id = $1::uuid`, prodID).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if stock != 8 {
		t.Fatalf("stock = %d, want 8 (receipt is read-only)", stock)
	}

	// Print bytes: 200, non-empty, starts with the ESC/POS init/reset sequence.
	printRec := httptest.NewRecorder()
	printReq := httptest.NewRequest(http.MethodGet, "/v1/sales/"+saleID+"/receipt/print", nil)
	printReq.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(printRec, printReq)
	if printRec.Code != http.StatusOK {
		t.Fatalf("receipt print status = %d, want 200", printRec.Code)
	}
	if printRec.Body.Len() == 0 || printRec.Body.Bytes()[0] != 0x1B {
		t.Fatalf("print output should start with ESC/POS ESC, got %d bytes first=%x",
			printRec.Body.Len(), printRec.Body.Bytes()[0])
	}
}

func TestReceiptsIntegration_MissingSaleGives404(t *testing.T) {
	router, _, _, token := setupReceiptsIntegration(t)
	for _, path := range []string{
		"/v1/sales/" + uuid.NewString() + "/receipt",
		"/v1/sales/" + uuid.NewString() + "/receipt/print",
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, rec.Code)
		}
	}
}