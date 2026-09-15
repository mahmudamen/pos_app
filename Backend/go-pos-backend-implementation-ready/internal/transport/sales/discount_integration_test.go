package sales

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/example/pos-api/internal/transport/settings"
)

// DISCOUNT-POLICY DB-backed slice: cap/block/warn enforcement and the manager
// PIN override against a real PostgreSQL. Skips when no database is set,
// matching the other integration tests.

type fullSaleResponse struct {
	Data struct {
		ID              string `json:"id"`
		SubtotalMinor   int64  `json:"subtotal_minor"`
		DiscountMinor   int64  `json:"discount_minor"`
		TotalMinor      int64  `json:"total_minor"`
		DiscountCapped  bool   `json:"discount_capped"`
		DiscountWarning string `json:"discount_warning"`
	} `json:"data"`
	Error *struct {
		Code string `json:"code"`
	} `json:"error"`
}

func postSaleFull(t *testing.T, router http.Handler, token, key, body string) (int, fullSaleResponse) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/sales", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	router.ServeHTTP(rec, req)
	var s fullSaleResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &s)
	return rec.Code, s
}

func cashierSalesToken(t *testing.T, seed testutil.Seed) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, seed.CashierID, seed.DeviceID, "sess-disc", "cashier")
}

func managerSalesToken(t *testing.T, seed testutil.Seed) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-disc", "manager")
}

func saleWithDiscount(prodID string, quantity int64, discountMinor int64, extra string) string {
	body := fmt.Sprintf(`{"items":[{"product_id":%q,"quantity":%d}],"discount_minor":%d`, prodID, quantity, discountMinor)
	if extra != "" {
		body += `,` + extra
	}
	return body + `}`
}

func TestDiscountIntegration_CapModeClampsToLimit(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	token := cashierSalesToken(t, seed)
	prodID := insertSaleProduct(t, seed, pool)

	// 20% discount on a 1000-subtotal; cashier ceiling is 5%.
	code, resp := postSaleFull(t, router, token, "disc-key-cap", saleWithDiscount(prodID, 2, 200, ""))
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %+v", code, resp.Error)
	}
	if !resp.Data.DiscountCapped {
		t.Fatalf("expected discount_capped=true: %+v", resp.Data)
	}
	if resp.Data.DiscountMinor != 50 || resp.Data.TotalMinor != 950 {
		t.Fatalf("capped discount: got discount=%d total=%d, want 50/950", resp.Data.DiscountMinor, resp.Data.TotalMinor)
	}
}

func TestDiscountIntegration_GlobalMaxPctNarrowsManager(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	token := managerSalesToken(t, seed)
	prodID := insertSaleProduct(t, seed, pool)
	testutil.SetTenantSetting(t, pool, seed.TenantID, settings.KeyMaxDiscountPct, "10")

	// 30% discount; manager level allows 50% but the tenant caps at 10%.
	code, resp := postSaleFull(t, router, token, "disc-key-global", saleWithDiscount(prodID, 2, 300, ""))
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %+v", code, resp.Error)
	}
	if !resp.Data.DiscountCapped {
		t.Fatalf("expected cap when global limit applies: %+v", resp.Data)
	}
	if resp.Data.DiscountMinor != 100 || resp.Data.TotalMinor != 900 {
		t.Fatalf("global-cap discount: got discount=%d total=%d, want 100/900", resp.Data.DiscountMinor, resp.Data.TotalMinor)
	}
}

func TestDiscountIntegration_BlockModeRequiresManagerPIN(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	testutil.SetTenantSetting(t, pool, seed.TenantID, settings.KeyDiscountMode, "block")
	token := cashierSalesToken(t, seed)
	prodID := insertSaleProduct(t, seed, pool)
	testutil.SetManagerPIN(t, pool, seed.TenantID, seed.CashierID, "4321")

	// No pin -> rejected.
	code, resp := postSaleFull(t, router, token, "disc-key-block-no", saleWithDiscount(prodID, 2, 200, ""))
	if code != http.StatusForbidden || resp.Error == nil || resp.Error.Code != "discount_manager_pin_required" {
		t.Fatalf("expected 403 discount_manager_pin_required, got %d %+v", code, resp.Error)
	}

	// Wrong pin -> rejected.
	code, resp = postSaleFull(t, router, token, "disc-key-block-badpin",
		saleWithDiscount(prodID, 2, 200, `"manager_pin":"9999"`))
	if code != http.StatusForbidden || resp.Error == nil || resp.Error.Code != "invalid_pin" {
		t.Fatalf("expected 403 invalid_pin, got %d %+v", code, resp.Error)
	}

	// Valid pin -> the full discount is approved (no clamp).
	code, resp = postSaleFull(t, router, token, "disc-key-block-ok",
		saleWithDiscount(prodID, 2, 200, `"manager_pin":"4321"`))
	if code != http.StatusCreated {
		t.Fatalf("expected 201 with valid pin, got %d: %+v", code, resp.Error)
	}
	if resp.Data.DiscountCapped || resp.Data.DiscountMinor != 200 || resp.Data.TotalMinor != 800 {
		t.Fatalf("approved block discount: got discount=%d total=%d capped=%v, want 200/800/false",
			resp.Data.DiscountMinor, resp.Data.TotalMinor, resp.Data.DiscountCapped)
	}
}

func TestDiscountIntegration_BlockModeWithoutManagerSettingRejects(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	testutil.SetTenantSetting(t, pool, seed.TenantID, settings.KeyDiscountMode, "block")
	testutil.SetTenantSetting(t, pool, seed.TenantID, settings.KeyManagerDiscount, "false")
	token := cashierSalesToken(t, seed)
	prodID := insertSaleProduct(t, seed, pool)

	code, resp := postSaleFull(t, router, token, "disc-key-block-no-mgr", saleWithDiscount(prodID, 2, 200, ""))
	if code != http.StatusForbidden {
		t.Fatalf("expected 403 without manager override, got %d: %+v", code, resp.Error)
	}
}

func TestDiscountIntegration_WarnModeAllowsWithNotice(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	testutil.SetTenantSetting(t, pool, seed.TenantID, settings.KeyDiscountMode, "warn")
	token := cashierSalesToken(t, seed)
	prodID := insertSaleProduct(t, seed, pool)

	code, resp := postSaleFull(t, router, token, "disc-key-warn", saleWithDiscount(prodID, 2, 200, ""))
	if code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %+v", code, resp.Error)
	}
	if resp.Data.DiscountCapped || resp.Data.DiscountWarning == "" ||
		resp.Data.DiscountMinor != 200 || resp.Data.TotalMinor != 800 {
		t.Fatalf("warn result wrong: %+v", resp.Data)
	}
}

func TestDiscountIntegration_AdminBypassesBlockMode(t *testing.T) {
	router, seed, pool := setupSalesIntegration(t)
	testutil.SetTenantSetting(t, pool, seed.TenantID, settings.KeyDiscountMode, "block")
	prodID := insertSaleProduct(t, seed, pool)

	// owner maps to the admin level and never needs a pin.
	ownerToken := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-disc", "owner")
	code, resp := postSaleFull(t, router, ownerToken, "disc-key-admin", saleWithDiscount(prodID, 2, 200, ""))
	if code != http.StatusCreated {
		t.Fatalf("admin should bypass block mode, got %d: %+v", code, resp.Error)
	}
	if resp.Data.TotalMinor != 800 {
		t.Fatalf("admin full discount: got total %d, want 800", resp.Data.TotalMinor)
	}
}
