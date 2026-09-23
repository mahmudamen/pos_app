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
	"github.com/example/pos-api/internal/transport/settings"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestSalesIntegration_RoundingRoundTrip exercises the Egypt cash-rounding
// contract end to end against a real PostgreSQL:
//
//  1. The tenant has pos.rounding_mode='25' (client and server agree the
//     policy -- the server stays authoritative and never trusts the client).
//  2. A sale whose nominal total is off a 25-piastre boundary gets the
//     server-computed rounding_minor echoed in the create response.
//  3. A client that sends a rounding_minor that does not match the tenant
//     policy is rejected (400 rounding_error).
//  4. GET /v1/sales/:id surfaces rounding_minor, so a client can show
//     "payable = total + rounding".
//
// DB-gated exactly like the idempotency suite: skipped when no
// TEST_DATABASE_URL / DATABASE_URL is configured.

func TestSalesIntegration_RoundingRoundTrip(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	token := salesToken(t, seed)
	testutil.SetTenantSetting(t, pool, seed.TenantID, settings.KeyRoundingMode, "25")

	// Price in minor units: 2513 piastres, quantity 1 -> nominal total 2513.
	// 2513 % 25 = 13 >= 12.5 → rounds up to 2525 -> expected +12.
	prodID := insertPricedProduct(t, seed, pool, "Mochaccino", "MOCCA-SALE", 2513)

	// Seller omits rounding_minor: server computes 12 and persists it.
	code, body := postSaleBody(t, router, token, "rounding-omitted",
		fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":1}]}`, prodID))
	if code != http.StatusCreated {
		t.Fatalf("omitted rounding: expected 201, got %d (%s)", code, body)
	}
	created := decodeSaleCreate(t, body)
	if created.Data.RoundingMinor != 12 {
		t.Fatalf("omitted rounding: rounding_minor = %d, want 12", created.Data.RoundingMinor)
	}
	if created.Data.TotalMinor != 2513 {
		t.Fatalf("omitted rounding: total_minor = %d, want 2513", created.Data.TotalMinor)
	}

	// Client sends the correct delta: accepted, same rounding.
	code2, body2 := postSaleBody(t, router, token, "rounding-echoed",
		fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":1}],"rounding_minor":12}`, prodID))
	if code2 != http.StatusCreated {
		t.Fatalf("echoed rounding: expected 201, got %d (%s)", code2, body2)
	}
	if got := decodeSaleCreate(t, body2).Data.RoundingMinor; got != 12 {
		t.Fatalf("echoed rounding: rounding_minor = %d, want 12", got)
	}

	// Cashier explicitly tenders the rounded payable (total 2513 + rounding
	// 12 = 2525): accepted -- payments are matched to the payable, not the
	// nominal total.
	code4, body4 := postSaleBody(t, router, token, "rounding-payable",
		fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":1}],"rounding_minor":12,"payments":[{"method":"cash","amount_minor":2525}]}`, prodID))
	if code4 != http.StatusCreated {
		t.Fatalf("payable cash: expected 201, got %d (%s)", code4, body4)
	}

	// A cashier paying the nominal total instead of the payable is rejected:
	// payments must match total + rounding exactly.
	code5, body5 := postSaleBody(t, router, token, "rounding-nominal-pay",
		fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":1}],"rounding_minor":12,"payments":[{"method":"cash","amount_minor":2513}]}`, prodID))
	if code5 != http.StatusBadRequest {
		t.Fatalf("nominal payments under rounding: expected 400, got %d (%s)", code5, body5)
	}

	// Client sends a delta that drifts from the tenant policy: rejected.
	code3, body3 := postSaleBody(t, router, token, "rounding-drift",
		fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":1}],"rounding_minor":11}`, prodID))
	if code3 != http.StatusBadRequest {
		t.Fatalf("drift: expected 400, got %d (%s)", code3, body3)
	}

	// Detail endpoint echoes rounding_minor.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/sales/"+created.Data.ID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var detail struct {
		Data struct {
			RoundingMinor int64 `json:"rounding_minor"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &detail)
	if detail.Data.RoundingMinor != 12 {
		t.Fatalf("detail: rounding_minor = %d, want 12", detail.Data.RoundingMinor)
	}
}

func TestSalesIntegration_RoundingRejectsNegative(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	token := salesToken(t, seed)
	prodID := insertPricedProduct(t, seed, pool, "Mozzarella", "MOZZA-SALE", 1111)

	code, body := postSaleBody(t, router, token, "rounding-negative",
		fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":1}],"rounding_minor":-25}`, prodID))
	if code != http.StatusBadRequest {
		t.Fatalf("negative rounding: expected 400, got %d (%s)", code, body)
	}
}

// insertPricedProduct inserts a catalog product with explicit price + stock
// (10) for the seeded tenant, returning its id.
func insertPricedProduct(t *testing.T, seed testutil.Seed, pool *pgxpool.Pool, name, sku string, priceMinor int64) string {
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
		 VALUES ($1::uuid, $2, $3, $4, 'EGP', 10) RETURNING id::text`,
		seed.TenantID, name, sku, priceMinor).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	return id
}

type saleCreateResponse struct {
	Data struct {
		ID            string `json:"id"`
		TotalMinor    int64  `json:"total_minor"`
		RoundingMinor int64  `json:"rounding_minor"`
	} `json:"data"`
}

// postSaleBody posts a create-sale request and returns the raw response body
// (unlike postSale, which collapses it to just the id).
func postSaleBody(t *testing.T, router http.Handler, token, key, body string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sales", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func decodeSaleCreate(t *testing.T, body string) saleCreateResponse {
	t.Helper()
	var out saleCreateResponse
	if err := json.Unmarshal([]byte(body), &out); err != nil {
		t.Fatalf("decode create response %q: %v", body, err)
	}
	return out
}
