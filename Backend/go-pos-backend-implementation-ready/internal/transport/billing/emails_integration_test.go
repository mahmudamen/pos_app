package billing

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type billingCapture struct {
	sent []identity.EmailMessage
}

func (c *billingCapture) Send(_ context.Context, msg identity.EmailMessage) error {
	c.sent = append(c.sent, msg)
	return nil
}

func setupBillingMailIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool, *billingCapture) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	if _, err := pool.Exec(context.Background(),
		`UPDATE tenants SET contact_email = $2 WHERE id = $1::uuid`,
		seed.TenantID, seed.ManagerEmail); err != nil {
		t.Fatalf("set contact email: %v", err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cap := &billingCapture{}
	h := NewHandler(pool, testutil.TokenManager())
	h.SetMailer(cap)
	h.Register(router.Group("/v1/saas"))
	return router, seed, pool, cap
}

func TestBillingIntegration_InvoiceLifecycleEmails(t *testing.T) {
	router, seed, _, cap := setupBillingMailIntegration(t)
	token := adminToken(t, seed)
	tenantURL := "/v1/saas/tenants/" + seed.TenantID

	// Creating an invoice emails the merchant's contact address.
	rec := mustCall(t, router, http.MethodPost, tenantURL+"/invoices", token,
		`{"amount_minor":29900,"currency":"EGP","description":"Business monthly"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create invoice: got %d: %s", rec.Code, rec.Body.String())
	}
	invoiceID := decode(t, rec)["data"].(map[string]any)["id"].(string)
	if len(cap.sent) != 1 {
		t.Fatalf("after create, sends = %d, want 1", len(cap.sent))
	}
	if cap.sent[0].To != seed.ManagerEmail {
		t.Fatalf("invoice recipient = %q, want %q", cap.sent[0].To, seed.ManagerEmail)
	}
	if !strings.Contains(strings.ToLower(cap.sent[0].Subject), "invoice") &&
		!strings.Contains(cap.sent[0].Subject, "فاتورة") {
		t.Fatalf("invoice subject = %q", cap.sent[0].Subject)
	}

	// Paying it sends a payment-received notice.
	if rec := mustCall(t, router, http.MethodPost, "/v1/saas/invoices/"+invoiceID+"/pay", token,
		`{"provider":"mock"}`); rec.Code != http.StatusOK {
		t.Fatalf("pay: got %d: %s", rec.Code, rec.Body.String())
	}
	if len(cap.sent) != 2 {
		t.Fatalf("after pay, sends = %d, want 2", len(cap.sent))
	}

	// Voiding a fresh open invoice sends a third notice.
	rec = mustCall(t, router, http.MethodPost, tenantURL+"/invoices", token,
		`{"amount_minor":5000,"currency":"EGP","description":"One-off"}`)
	secondID := decode(t, rec)["data"].(map[string]any)["id"].(string)
	if rec := mustCall(t, router, http.MethodPost, "/v1/saas/invoices/"+secondID+"/void", token, ""); rec.Code != http.StatusOK {
		t.Fatalf("void: got %d: %s", rec.Code, rec.Body.String())
	}
	if len(cap.sent) != 4 {
		t.Fatalf("after void, sends = %d, want 4", len(cap.sent))
	}
}

func TestBillingIntegration_NoContactEmailSkipsSend(t *testing.T) {
	router, seed, pool, cap := setupBillingMailIntegration(t)
	token := adminToken(t, seed)
	if _, err := pool.Exec(context.Background(),
		`UPDATE tenants SET contact_email = '' WHERE id = $1::uuid`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	// No owner user exists in the seed (manager/cashier only), so the resolver
	// finds nothing and the create still succeeds without sending.
	rec := mustCall(t, router, http.MethodPost, "/v1/saas/tenants/"+seed.TenantID+"/invoices", token,
		`{"amount_minor":1000,"currency":"EGP","description":"x"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create without contact: got %d: %s", rec.Code, rec.Body.String())
	}
	if len(cap.sent) != 0 {
		t.Fatalf("expected no send, got %d", len(cap.sent))
	}
}

func TestBillingIntegration_SubscriptionCancelledEmail(t *testing.T) {
	router, seed, _, cap := setupBillingMailIntegration(t)
	token := adminToken(t, seed)

	// Assign a subscription so the cancel transition is genuinely observed.
	rec := mustCall(t, router, http.MethodPost, "/v1/saas/tenants/"+seed.TenantID+"/subscription", token,
		`{"plan_code":"starter","status":"active"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("assign: got %d: %s", rec.Code, rec.Body.String())
	}
	subID := decode(t, rec)["data"].(map[string]any)["id"].(string)
	if len(cap.sent) != 0 {
		t.Fatalf("assign should not email, got %d sends", len(cap.sent))
	}

	rec = mustCall(t, router, http.MethodPost, "/v1/saas/subscriptions/"+subID+"/status", token,
		`{"status":"cancelled"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel: got %d: %s", rec.Code, rec.Body.String())
	}
	if len(cap.sent) != 1 {
		t.Fatalf("after cancel, sends = %d, want 1", len(cap.sent))
	}
	if cap.sent[0].To != seed.ManagerEmail {
		t.Fatalf("recipient = %q, want %q", cap.sent[0].To, seed.ManagerEmail)
	}
	if !strings.Contains(cap.sent[0].Subject, "انتهاء الاشتراك") &&
		!strings.Contains(strings.ToLower(cap.sent[0].Subject), "subscription") {
		t.Fatalf("subject = %q, want the subscription-ended subject", cap.sent[0].Subject)
	}
	if cap.sent[0].HTML == "" {
		t.Fatalf("subscription-ended mail should carry the HTML template")
	}

	// Cancelling an already-cancelled id is idempotent: no second email.
	if rec := mustCall(t, router, http.MethodPost, "/v1/saas/subscriptions/"+subID+"/status", token,
		`{"status":"cancelled"}`); rec.Code != http.StatusOK {
		t.Fatalf("re-cancel: got %d: %s", rec.Code, rec.Body.String())
	}
	if len(cap.sent) != 1 {
		t.Fatalf("after re-cancel, sends = %d, want still 1", len(cap.sent))
	}
}
