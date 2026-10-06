package registers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

type summaryData struct {
	From                string `json:"from"`
	To                  string `json:"to"`
	SessionsOpened      int64  `json:"sessions_opened"`
	SessionsClosed      int64  `json:"sessions_closed"`
	SalesCount          int64  `json:"sales_count"`
	SubtotalMinor       int64  `json:"subtotal_minor"`
	DiscountMinor       int64  `json:"discount_minor"`
	TaxMinor            int64  `json:"tax_minor"`
	TotalMinor          int64  `json:"total_minor"`
	CashMinor           int64  `json:"cash_minor"`
	CardMinor           int64  `json:"card_minor"`
	MobileMinor         int64  `json:"mobile_minor"`
	CashDifferenceMinor int64  `json:"cash_difference_minor"`
	AverageSaleMinor    int64  `json:"average_sale_minor"`
}

func summaryRange(t *testing.T, router http.Handler, token, query string) (int, summaryData, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/registers/summary"+query, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	var out struct {
		Data summaryData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse summary %q: %v: %s", query, err, rec.Body.String())
	}
	return rec.Code, out.Data, rec.Body.String()
}

// seedRangeSale inserts one completed sale with a split tender plus one voided
// sale (which must never count), inside the tenant's RLS context.
func seedRangeSale(t *testing.T, pool *pgxpool.Pool, tenantID string) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx,
		`SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		t.Fatal(err)
	}
	var saleID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO sales (tenant_id, idempotency_key, status, subtotal_minor, discount_minor, tax_minor, total_minor, currency)
		VALUES ($1::uuid, 'range-summary-sale', 'completed', 500, 0, 0, 500, 'EGP')
		RETURNING id`, tenantID).Scan(&saleID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO sale_payments (tenant_id, sale_id, method, amount_minor)
		VALUES ($1::uuid, $2::uuid, 'cash', 350), ($1::uuid, $2::uuid, 'card', 150)`,
		tenantID, saleID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO sales (tenant_id, idempotency_key, status, subtotal_minor, total_minor, currency)
		VALUES ($1::uuid, 'range-summary-voided', 'voided', 999, 999, 'EGP')`,
		tenantID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestSummaryIntegration_AggregatesSessionsAndSales(t *testing.T) {
	router, seed, pool := setupRegistersIntegration(t)
	cashierToken := regToken(t, seed, seed.CashierID, "cashier")

	// One shift: opening float 100, counted 500 at close, no cash sales tied
	// to it, so the drawer difference is +400.
	sessionID := openSession(t, router, cashierToken)
	if code, body := closeSession(t, router, cashierToken, sessionID, ""); code != http.StatusOK {
		t.Fatalf("close: expected 200, got %d: %s", code, body)
	}
	seedRangeSale(t, pool, seed.TenantID)

	code, sum, body := summaryRange(t, router, cashierToken, "?from=2026-10-01&to=2099-01-01")
	if code != http.StatusOK {
		t.Fatalf("wide window: expected 200, got %d: %s", code, body)
	}
	if sum.SessionsOpened != 1 || sum.SessionsClosed != 1 {
		t.Fatalf("sessions: expected 1 opened and 1 closed, got %d/%d",
			sum.SessionsOpened, sum.SessionsClosed)
	}
	if sum.CashDifferenceMinor != 400 {
		t.Fatalf("cash difference: expected 400, got %d", sum.CashDifferenceMinor)
	}
	if sum.SalesCount != 1 || sum.TotalMinor != 500 || sum.SubtotalMinor != 500 {
		t.Fatalf("sales: expected the one completed sale (1/500/500), got %d/%d/%d",
			sum.SalesCount, sum.TotalMinor, sum.SubtotalMinor)
	}
	if sum.CashMinor != 350 || sum.CardMinor != 150 || sum.MobileMinor != 0 {
		t.Fatalf("tender mix: expected 350/150/0, got %d/%d/%d",
			sum.CashMinor, sum.CardMinor, sum.MobileMinor)
	}
	if sum.AverageSaleMinor != 500 {
		t.Fatalf("average: expected 500, got %d", sum.AverageSaleMinor)
	}

	// A window that misses everything reports clean zeroes rather than failing.
	code, sum, body = summaryRange(t, router, cashierToken, "?from=2099-01-01")
	if code != http.StatusOK {
		t.Fatalf("future window: expected 200, got %d: %s", code, body)
	}
	if sum.SessionsOpened != 0 || sum.SessionsClosed != 0 || sum.SalesCount != 0 ||
		sum.TotalMinor != 0 || sum.CashDifferenceMinor != 0 || sum.AverageSaleMinor != 0 {
		t.Fatalf("future window: expected zeroes, got %+v", sum)
	}

	code, sum, body = summaryRange(t, router, cashierToken, "?to=2020-01-01")
	if code != http.StatusOK {
		t.Fatalf("past window: expected 200, got %d: %s", code, body)
	}
	if sum.SessionsOpened != 0 || sum.SalesCount != 0 {
		t.Fatalf("past window: expected zeroes, got %+v", sum)
	}
}
