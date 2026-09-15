package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB-backed coverage for the SaaS billing control plane: plan lifecycle,
// subscription state machine + plan change with usage checks, invoice charging
// through the mock gateway, and the billing summary. Skips without a test DB.

func setupBillingIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	tokens := testutil.TokenManager()
	NewHandler(pool, tokens).Register(router.Group("/v1/saas"))
	return router, seed, pool
}

func adminToken(t *testing.T, seed testutil.Seed) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-bill", "saas_admin")
}

func randomSuffix() string {
	return strings.ToLower(fmt.Sprintf("%x", time.Now().UnixNano()))[:8]
}

func mustCall(t *testing.T, router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	body := map[string]any{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
	return body
}

func insertTinyPlan(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO plans (code, name, price_minor, billing_period, max_users, max_products)
		VALUES ('tiny', 'Tiny', 5000, 'monthly', 1, 1) ON CONFLICT (code) DO NOTHING`); err != nil {
		t.Fatal(err)
	}
}

func TestBillingIntegration_PlanLifecycle(t *testing.T) {
	router, seed, pool := setupBillingIntegration(t)
	_ = pool
	token := adminToken(t, seed)
	code := "cafe" + randomSuffix()

	create := fmt.Sprintf(`{"code":%q, "name":"Cafe", "price_minor":8000, "currency":"EGP", "billing_period":"monthly", "features":["pos.basic"], "max_users":5, "max_products":500}`, code)
	rec := mustCall(t, router, http.MethodPost, "/v1/saas/plans", token, create)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create plan: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	// Duplicate code → 409.
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/plans", token, create)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate plan: expected 409, got %d", rec.Code)
	}
	// Invalid payload → 400.
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/plans", token,
		`{"code":"BAD!", "name":"x", "billing_period":"weekly"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad plan: expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	// List contains the seeded plans + our new one.
	rec = mustCall(t, router, http.MethodGet, "/v1/saas/plans", token, "")
	body := decode(t, rec)
	list, _ := json.Marshal(body["data"])
	if !bytes.Contains(list, []byte(`"code":"business"`)) || !bytes.Contains(list, []byte(`"code":"`+code+`"`)) {
		t.Fatalf("plan list should include seeded + created plans: %s", list)
	}
	// Update + soft delete.
	id := extractID(t, list, code)
	rec = mustCall(t, router, http.MethodPatch, "/v1/saas/plans/"+id, token, `{"price_minor":9000}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update plan: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = mustCall(t, router, http.MethodDelete, "/v1/saas/plans/"+id, token, "")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete plan: expected 204, got %d", rec.Code)
	}
	rec = mustCall(t, router, http.MethodDelete, "/v1/saas/plans/"+id, token, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("re-delete plan: expected 404, got %d", rec.Code)
	}
}

func TestBillingIntegration_SubscriptionLifecycle(t *testing.T) {
	router, seed, pool := setupBillingIntegration(t)
	token := adminToken(t, seed)

	// Assign a business subscription to the seeded trial tenant.
	rec := mustCall(t, router, http.MethodPost,
		"/v1/saas/tenants/"+seed.TenantID+"/subscription", token, `{"plan_code":"business","status":"trial","trial_days":15}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("assign subscription: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	body := decode(t, rec)
	id := body["data"].(map[string]any)["id"].(string)

	// Assigning again while live → 409.
	rec = mustCall(t, router, http.MethodPost,
		"/v1/saas/tenants/"+seed.TenantID+"/subscription", token, `{"plan_code":"starter"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("double assign: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}

	// Read back the live subscription.
	rec = mustCall(t, router, http.MethodGet,
		"/v1/saas/tenants/"+seed.TenantID+"/subscription", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get subscription: expected 200, got %d", rec.Code)
	}
	body = decode(t, rec)
	sub := body["data"].(map[string]any)
	if sub["status"] != "trial" || sub["plan_code"] != "business" {
		t.Fatalf("live subscription wrong: %v", sub)
	}

	// State machine: trial -> suspended -> active.
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/subscriptions/"+id+"/status", token, `{"status":"suspended"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("trial->suspended: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	// Invalid: suspended -> grace_period.
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/subscriptions/"+id+"/status", token, `{"status":"grace_period"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("suspended->grace_period: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/subscriptions/"+id+"/status", token, `{"status":"active"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("suspended->active: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	// Unknown status → 400.
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/subscriptions/"+id+"/status", token, `{"status":"banana"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("banana status: expected 400, got %d", rec.Code)
	}

	// Change plan within limits: business -> starter (seed has 2 users / 0 products).
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/subscriptions/"+id+"/change-plan", token, `{"plan_code":"starter"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("change plan under limits: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Change plan to one with a 1-user cap → blocked by usage.
	insertTinyPlan(t, pool)
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/subscriptions/"+id+"/change-plan", token, `{"plan_code":"tiny"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("plan downgrade blocked: expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	body = decode(t, rec)
	if body["error"].(map[string]any)["code"] != "plan_downgrade_blocked" {
		t.Fatalf("expected plan_downgrade_blocked, got %s", rec.Body.String())
	}
	data := body["data"].(map[string]any)
	if data["required"].(float64) != 2 || data["limit"].(float64) != 1 {
		t.Fatalf("usage block payload wrong: %v", data)
	}

	// Cancel (terminal) then confirm cancelling again works but active->trial is refused.
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/subscriptions/"+id+"/status", token, `{"status":"cancelled"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("active->cancelled: expected 200, got %d", rec.Code)
	}
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/subscriptions/"+id+"/status", token, `{"status":"cancelled"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("idempotent cancel: expected 200, got %d", rec.Code)
	}
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/subscriptions/"+id+"/status", token, `{"status":"active"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("cancelled->active: expected 409, got %d", rec.Code)
	}
}

func TestBillingIntegration_InvoicePayVoidRefund(t *testing.T) {
	router, seed, pool := setupBillingIntegration(t)
	_ = pool
	token := adminToken(t, seed)
	tenantURL := "/v1/saas/tenants/" + seed.TenantID

	// A live subscription is needed to attach invoices; assign one first.
	if rec := mustCall(t, router, http.MethodPost, tenantURL+"/subscription", token,
		`{"plan_code":"business","status":"active"}`); rec.Code != http.StatusCreated {
		t.Fatalf("assign: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	subBody := decode(t, mustCall(t, router, http.MethodGet, tenantURL+"/subscription", token, ""))
	subID := subBody["data"].(map[string]any)["id"].(string)

	create := `{"amount_minor":29900,"currency":"EGP","description":"Business monthly","subscription_id":"` + subID + `"}`
	rec := mustCall(t, router, http.MethodPost, tenantURL+"/invoices", token, create)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create invoice: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	body := decode(t, rec)
	invoiceID := body["data"].(map[string]any)["id"].(string)

	// Negative amounts are rejected by validation (binding: required, >= 0).
	bad := mustCall(t, router, http.MethodPost, tenantURL+"/invoices", token,
		`{"amount_minor":-5,"currency":"EGP","description":"x"}`)
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("negative invoice: expected 400, got %d", bad.Code)
	}

	// Pay through the mock gateway.
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/invoices/"+invoiceID+"/pay", token, `{"provider":"mock"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("pay invoice: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body = decode(t, rec)
	paid := body["data"].(map[string]any)
	if paid["status"] != "paid" || !strings.HasPrefix(paid["provider_ref"].(string), "mock_") {
		t.Fatalf("paid invoice payload wrong: %v", paid)
	}

	// Re-pay → 409 (already paid). Refund → 200. Refund again → 409.
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/invoices/"+invoiceID+"/pay", token, `{"provider":"mock"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("re-pay: expected 409, got %d", rec.Code)
	}
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/invoices/"+invoiceID+"/refund", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("refund: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/invoices/"+invoiceID+"/refund", token, "")
	if rec.Code != http.StatusConflict {
		t.Fatalf("re-refund: expected 409, got %d", rec.Code)
	}

	// A second open invoice can be voided; a voided one cannot be paid.
	rec = mustCall(t, router, http.MethodPost, tenantURL+"/invoices", token, create)
	body = decode(t, rec)
	secondID := body["data"].(map[string]any)["id"].(string)
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/invoices/"+secondID+"/void", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("void: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = mustCall(t, router, http.MethodPost, "/v1/saas/invoices/"+secondID+"/pay", token, `{"provider":"mock"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("pay voided: expected 409, got %d", rec.Code)
	}
}

func TestBillingIntegration_BillingSummary(t *testing.T) {
	router, seed := func() (*gin.Engine, testutil.Seed) {
		r, s, _ := setupBillingIntegration(t)
		return r, s
	}()
	token := adminToken(t, seed)
	tenantURL := "/v1/saas/tenants/" + seed.TenantID

	mustCall(t, router, http.MethodPost, tenantURL+"/subscription", token, `{"plan_code":"business","status":"active"}`)
	mustCall(t, router, http.MethodPost, tenantURL+"/invoices", token,
		`{"amount_minor":29900,"currency":"EGP","description":"Monthly"}`)

	rec := mustCall(t, router, http.MethodGet, "/v1/saas/billing/summary", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("summary: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	body := decode(t, rec)
	data := body["data"].(map[string]any)
	if data["active_plans"].(float64) < 3 {
		t.Fatalf("active_plans should include seeded catalog: %v", data["active_plans"])
	}
	subs := data["subscriptions"].(map[string]any)
	if subs["active"].(float64) < 1 {
		t.Fatalf("active subscriptions missing: %v", subs)
	}
	if data["mrr_minor"].(float64) < 29900 {
		t.Fatalf("mrr should include the business plan: %v", data["mrr_minor"])
	}
	if data["outstanding_minor"].(float64) < 29900 {
		t.Fatalf("outstanding should include the open invoice: %v", data["outstanding_minor"])
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"providers":["manual","mock"]`)) {
		t.Fatalf("providers missing: %s", rec.Body.String())
	}
}

// extractID finds the id of a plan given a JSON array of plan objects.
func extractID(t *testing.T, data []byte, code string) string {
	t.Helper()
	var plans []map[string]any
	if err := json.Unmarshal(data, &plans); err != nil {
		t.Fatal(err)
	}
	for _, p := range plans {
		if p["code"] == code {
			return p["id"].(string)
		}
	}
	t.Fatalf("plan %q not found", code)
	return ""
}
