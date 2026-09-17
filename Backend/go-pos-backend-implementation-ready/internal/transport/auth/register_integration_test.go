package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Merchant self-signup (onboarding) integration coverage: registering creates a
// 15-day trial tenant with an owner + seeded demo catalog, duplicate emails are
// rejected, and an expired trial blocks further logins. Skips without a DB.

func registerBody(businessType, email string) string {
	payload := map[string]string{
		"store_name":    "Onboard Demo",
		"business_type": businessType,
		"email":         email,
		"password":      "secret-pass-123",
		"display_name":  "Onboard Owner",
		"device_id":     "unit-reg-device",
		"device_name":   "Onboarding",
	}
	b, _ := json.Marshal(payload)
	return string(b)
}

func registerPayload(t *testing.T, router http.Handler, businessType, email string) map[string]any {
	t.Helper()
	rec := doPost(t, router, "/v1/auth/register", registerBody(businessType, email), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register(%s,%s): expected 201, got %d body=%s", businessType, email, rec.Code, rec.Body.String())
	}
	var resp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal register response: %v (body=%s)", err, rec.Body.String())
	}
	return resp.Data
}

func TestRegisterIntegration_CreatesTrialStoreWithCatalog(t *testing.T) {
	pool := testutil.Pool(t)
	if pool == nil {
		t.Skip("no TEST_DATABASE_URL/DATABASE_URL")
	}
	testutil.Migrate(t, testutil.DatabaseURL(t))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, authConfig()).Register(router.Group("/v1"))

	email := fmt.Sprintf("owner-%s@example.com", uuid.NewString()[:8])
	data := registerPayload(t, router, "coffee_shop", email)

	if data["access_token"] == "" || data["refresh_token"] == "" {
		t.Fatal("register should return tokens")
	}
	user := data["user"].(map[string]any)
	if user["role"] != "owner" {
		t.Fatalf("expected owner role, got %v", user["role"])
	}
	tenant := data["tenant"].(map[string]any)
	if tenant["business_type"] != "coffee_shop" {
		t.Fatalf("expected coffee_shop business_type, got %v", tenant["business_type"])
	}
	if tenant["plan"] != "trial" {
		t.Fatalf("expected trial plan, got %v", tenant["plan"])
	}
	trialEnds := tenant["trial_ends_at"].(string)
	end, err := time.Parse(time.RFC3339, trialEnds)
	if err != nil {
		t.Fatalf("trial_ends_at not parseable: %v", err)
	}
	if days := time.Until(end); days < 13*24*time.Hour || days > 16*24*time.Hour {
		t.Fatalf("trial window should be ~14-15 days, got %v", days)
	}

	ctx := context.Background()
	tenantID := tenant["id"].(string)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		t.Fatal(err)
	}
	var categories, products int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM categories WHERE tenant_id = $1`, tenantID).Scan(&categories); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM products WHERE tenant_id = $1 AND barcode <> ''`, tenantID).Scan(&products); err != nil {
		t.Fatal(err)
	}
	if categories == 0 {
		t.Fatal("register should seed demo categories")
	}
	if products == 0 {
		t.Fatal("register should seed demo products")
	}
	var withImage, withInfo int
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM products WHERE tenant_id = $1 AND image_url <> '' AND description <> ''`, tenantID).Scan(&withImage); err != nil {
		t.Fatal(err)
	}
	if err = tx.QueryRow(ctx, `SELECT count(*) FROM products WHERE tenant_id = $1`, tenantID).Scan(&withInfo); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if withImage != withInfo {
		t.Fatalf("all seeded products should carry image_url + description (%d/%d)", withImage, withInfo)
	}

	// The freshly registered owner can log in with the chosen email/password.
	login := `{"tenant_id":"` + tenantID + `","email":"` + email + `","password":"secret-pass-123","device_id":"d","device_name":"c"}`
	rec := doPost(t, router, "/v1/auth/login", login, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login after register: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Expiring the trial blocks the next login with trial_expired.
	if _, err = pool.Exec(ctx, `UPDATE tenants SET trial_ends_at = now() - interval '1 day' WHERE id = $1`, tenantID); err != nil {
		t.Fatal(err)
	}
	rec = doPost(t, router, "/v1/auth/login", login, "")
	if rec.Code != http.StatusForbidden || !strings.Contains(rec.Body.String(), "trial_expired") {
		t.Fatalf("expired trial login: expected 403 trial_expired, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRegisterIntegration_DuplicateEmailRejected(t *testing.T) {
	pool := testutil.Pool(t)
	if pool == nil {
		t.Skip("no TEST_DATABASE_URL/DATABASE_URL")
	}
	testutil.Migrate(t, testutil.DatabaseURL(t))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, authConfig()).Register(router.Group("/v1"))

	email := "dup-" + uuid.NewString()[:8] + "@example.com"
	registerPayload(t, router, "grocery", email)
	rec := doPost(t, router, "/v1/auth/register", registerBody("book_store", email), "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "email_taken") {
		t.Fatalf("duplicate register: expected 409 email_taken, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRegisterIntegration_UnsupportedBusinessType(t *testing.T) {
	pool := testutil.Pool(t)
	if pool == nil {
		t.Skip("no TEST_DATABASE_URL/DATABASE_URL")
	}
	testutil.Migrate(t, testutil.DatabaseURL(t))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, authConfig()).Register(router.Group("/v1"))

	body := registerBody("hotel", "owner-"+uuid.NewString()[:8]+"@example.com")
	rec := doPost(t, router, "/v1/auth/register", body, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unsupported business type: expected 400, got %d", rec.Code)
	}
}
