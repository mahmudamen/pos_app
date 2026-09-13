package sales

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// refundResponse mirrors the sale_refunds envelope.
type refundResponse struct {
	Data struct {
		ID          string `json:"id"`
		SaleID      string `json:"sale_id"`
		RefundMinor int64  `json:"refund_minor"`
		Status      string `json:"status"`
	} `json:"data"`
}

func postRefund(t *testing.T, router http.Handler, token, saleID, key, body string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sales/"+saleID+"/refund", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	router.ServeHTTP(rec, req)
	var r refundResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &r)
	return rec.Code, r.Data.ID
}

func checkSaleStatus(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool, saleID, want string) {
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
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM sales WHERE id = $1::uuid`, saleID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if status != want {
		t.Fatalf("sale status = %q, want %q", status, want)
	}
}

func countRefunds(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool, key string) int {
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
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM sale_refunds WHERE idempotency_key = $1`, key).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRefundsIntegration_RefundRestoresStockAndIsIdempotent(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	token := salesToken(t, seed)
	prodID := insertSaleProduct(t, seed, pool)

	body := fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":2}]}`, prodID)
	firstCode, saleID := postSale(t, router, token, "sale-key-refund", body)
	if firstCode != http.StatusCreated || saleID == "" {
		t.Fatalf("expected 201 sale, got %d id=%q", firstCode, saleID)
	}
	checkStock(t, seed, pool, prodID, 8)
	checkSaleStatus(t, seed, pool, saleID, "completed")

	refundBody := `{"reason":"customer returned goods"}`
	refundStatus, refundID := postRefund(t, router, token, saleID, "refund-key-1", refundBody)
	if refundStatus != http.StatusCreated || refundID == "" {
		t.Fatalf("expected 201 refund, got %d id=%q", refundStatus, refundID)
	}
	checkStock(t, seed, pool, prodID, 10)
	checkSaleStatus(t, seed, pool, saleID, "refunded")

	// Idempotent duplicate: same refund, stock moved only once.
	dupStatus, dupID := postRefund(t, router, token, saleID, "refund-key-1", refundBody)
	if dupStatus != http.StatusCreated || dupID != refundID {
		t.Fatalf("duplicate refund: expected same refund %s (201), got %d id=%q", refundID, dupStatus, dupID)
	}
	checkStock(t, seed, pool, prodID, 10)
	if got := countRefunds(t, seed, pool, "refund-key-1"); got != 1 {
		t.Fatalf("idempotency: expected exactly 1 refund row, got %d", got)
	}
}

func TestRefundsIntegration_RefundRejectsSecondRefund(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	token := salesToken(t, seed)
	prodID := insertSaleProduct(t, seed, pool)

	saleCode, saleID := postSale(t, router, token, "sale-key-refund-2",
		fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":1}]}`, prodID))
	if saleCode != http.StatusCreated {
		t.Fatalf("expected 201 sale, got %d", saleCode)
	}
	if code, _ := postRefund(t, router, token, saleID, "refund-key-2", `{}`); code != http.StatusCreated {
		t.Fatalf("first refund: expected 201, got %d", code)
	}
	// A fresh idempotency key against the already-refunded sale must 409.
	code, refundID := postRefund(t, router, token, saleID, "refund-key-2-next", `{}`)
	if code != http.StatusConflict {
		t.Fatalf("second refund: expected 409 sale_not_refundable, got %d id=%q", code, refundID)
	}
}
