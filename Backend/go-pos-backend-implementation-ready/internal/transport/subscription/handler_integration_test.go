package subscription

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func trialCfg() config.Config {
	return config.Config{Trial: config.TrialConfig{
		DurationDays: 14, Scope: identity.ScopeOrganization,
		RequireEmailVerification: false, RequirePhoneVerification: false,
		MaxOrganizationsPerAccount: 3, MaxActiveInstallations: 5,
		SuspiciousRegistrationPolicy: "review", RegisterRatePerIPPerHour: 5,
		OfflinePolicy: "grace24h", EmailTokenTTL: 15 * time.Minute, OTPTTL: 5 * time.Minute,
		Mailer: "noop", SMSSender: "noop",
	}}
}

func linkOwnerAccount(t *testing.T, pool *pgxpool.Pool, tenantID, accountID string) {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `UPDATE tenants SET owner_account_id = $2::uuid WHERE id = $1::uuid`, tenantID, accountID); err != nil {
		t.Fatalf("link account: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func newTestRouter(pool *pgxpool.Pool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, testutil.TokenManager(), trialCfg()).Register(router.Group("/v1"))
	return router
}

func doJSON(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestSubscription_TrialGranted(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	router := newTestRouter(pool)

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	acct, err := identity.NewAccountStore(identity.NewAuditRecorder()).Create(ctx, tx,
		fmt.Sprintf("sub-%s@example.com", uuid.NewString()[:8]), "", false, false)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	_ = tx.Commit(ctx)
	linkOwnerAccount(t, pool, seed.TenantID, acct.ID)

	policy, err := identity.LoadPolicy(ctx, pool, trialCfg().Trial)
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	granted, err := identity.GrantTrial(ctx, pool, policy, identity.GrantTrialInput{
		AccountID: acct.ID, TenantID: seed.TenantID, TrialType: "standard",
		EligibilityKey: identity.EligibilityKeyFor(policy.Scope, seed.TenantID),
		Status:         identity.TrialStatusActive, Source: "manual", Reason: "test",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("grant: %v", err)
	}

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "", "manager")
	rec := doJSON(router, http.MethodGet, "/v1/subscription", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data["plan"] != "trial" || resp.Data["trial_status"] != "active" ||
		resp.Data["subscription_status"] != "trialing" || resp.Data["access_allowed"] != true {
		t.Fatalf("unexpected response: %+v", resp.Data)
	}
	if resp.Data["trial_entitlement_id"] != granted.ID {
		t.Fatalf("trial_entitlement_id = %v, want %s", resp.Data["trial_entitlement_id"], granted.ID)
	}
	days, ok := resp.Data["days_remaining"].(float64)
	if !ok || days < 13 || days > 14.5 {
		t.Fatalf("days_remaining = %v, want ~14", resp.Data["days_remaining"])
	}
}

func TestSubscription_PendingTrial(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	router := newTestRouter(pool)

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	acct, err := identity.NewAccountStore(identity.NewAuditRecorder()).Create(ctx, tx,
		fmt.Sprintf("pending-%s@example.com", uuid.NewString()[:8]), "", true, false)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	_ = tx.Commit(ctx)
	linkOwnerAccount(t, pool, seed.TenantID, acct.ID)

	policy, err := identity.LoadPolicy(ctx, pool, trialCfg().Trial)
	if err != nil {
		t.Fatalf("load policy: %v", err)
	}
	if _, err := identity.GrantTrial(ctx, pool, policy, identity.GrantTrialInput{
		AccountID: acct.ID, TenantID: seed.TenantID, TrialType: "standard",
		EligibilityKey: identity.EligibilityKeyFor(policy.Scope, seed.TenantID),
		Status:         identity.TrialStatusPending, Source: "manual", Reason: "test",
	}, time.Now().UTC()); err != nil {
		t.Fatalf("grant: %v", err)
	}

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "", "manager")
	rec := doJSON(router, http.MethodGet, "/v1/subscription", "", token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data["trial_status"] != "pending" || resp.Data["access_allowed"] != true ||
		resp.Data["subscription_status"] != "trialing" || resp.Data["days_remaining"] != float64(0) {
		t.Fatalf("unexpected pending response: %+v", resp.Data)
	}
	if resp.Data["trial_expires_at"] != "" {
		t.Fatalf("pending trial must not expose an expiry, got %v", resp.Data["trial_expires_at"])
	}
}

func TestSubscription_RequiresAuth(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	router := newTestRouter(pool)
	rec := doJSON(router, http.MethodGet, "/v1/subscription", "", "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func insertSub(t *testing.T, pool *pgxpool.Pool, tenantID string) string {
	t.Helper()
	ctx := context.Background()
	var subID string
	err := pool.QueryRow(ctx, `
		INSERT INTO subscriptions (tenant_id, plan_id, status, current_period_end)
		SELECT $1::uuid, id, 'trial', now() + interval '30 days' FROM plans WHERE code = 'trial' AND is_active
		RETURNING id::text`, tenantID).Scan(&subID)
	if err != nil {
		t.Fatalf("insert subscription: %v", err)
	}
	return subID
}

func TestChangePlan_OwnerUpgrades(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	insertSub(t, pool, seed.TenantID)
	router := newTestRouter(pool)

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "", "owner")
	rec := doJSON(router, http.MethodPost, "/v1/subscription/change-plan",
		`{"plan_code":"starter"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Data["plan_code"] != "starter" {
		t.Fatalf("plan_code = %v, want starter", resp.Data["plan_code"])
	}

	var tenantPlan string
	var maxUsers, maxProducts int
	if err := pool.QueryRow(context.Background(),
		`SELECT plan, max_users, max_products FROM tenants WHERE id = $1::uuid`, seed.TenantID).
		Scan(&tenantPlan, &maxUsers, &maxProducts); err != nil {
		t.Fatalf("load tenant: %v", err)
	}
	if tenantPlan != "starter" || maxUsers == 0 || maxProducts == 0 {
		t.Fatalf("tenant plan not synced: plan=%s users=%d products=%d", tenantPlan, maxUsers, maxProducts)
	}
}

func TestChangePlan_CashierForbidden(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	insertSub(t, pool, seed.TenantID)
	router := newTestRouter(pool)

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "", "cashier")
	rec := doJSON(router, http.MethodPost, "/v1/subscription/change-plan",
		`{"plan_code":"starter"}`, token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	var resp struct {
		Error map[string]any `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Error["code"] != "permission_denied" {
		t.Fatalf("error = %v, want permission_denied", resp.Error["code"])
	}
}

func TestChangePlan_UnknownPlan(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	insertSub(t, pool, seed.TenantID)
	router := newTestRouter(pool)

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "", "owner")
	rec := doJSON(router, http.MethodPost, "/v1/subscription/change-plan",
		`{"plan_code":"nonexistent"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestChangePlan_RequiresActiveSubscription(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	router := newTestRouter(pool)

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "", "owner")
	rec := doJSON(router, http.MethodPost, "/v1/subscription/change-plan",
		`{"plan_code":"starter"}`, token)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
