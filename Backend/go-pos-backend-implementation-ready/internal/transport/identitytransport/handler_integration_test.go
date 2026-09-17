package identitytransport

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

func testCfg() config.Config {
	return config.Config{Trial: config.TrialConfig{
		DurationDays: 14, Scope: identity.ScopeOrganization,
		RequireEmailVerification: true, RequirePhoneVerification: true,
		MaxOrganizationsPerAccount: 3, MaxActiveInstallations: 5,
		SuspiciousRegistrationPolicy: "review", RegisterRatePerIPPerHour: 5,
		OfflinePolicy: "grace24h", EmailTokenTTL: 15 * time.Minute, OTPTTL: 5 * time.Minute,
		Mailer: "noop", SMSSender: "noop",
	}}
}

// uniquePhone returns E.164-ish digits derived from a random UUID so the
// accounts_phone_uq unique index never collides across tests/runs.
func uniquePhone() string {
	var digits []rune
	for _, r := range strings.ReplaceAll(uuid.NewString(), "-", "") {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	return "+2010" + string(digits[:8])
}

// identityFixture creates an account (email + phone, both unverified initially
// with the verification-required policy, so a pending trial can self-activate)
// and links it to a seeded tenant as the owner.
func identityFixture(t *testing.T, pool *pgxpool.Pool, email string) (seed testutil.Seed, account identity.Account) {
	t.Helper()
	seed = testutil.SeedTenant(t, pool)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	acct, err := identity.NewAccountStore(identity.NewAuditRecorder()).Create(ctx, tx, email, uniquePhone(),
		true, true)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE tenants SET owner_account_id = $2::uuid WHERE id = $1::uuid`, seed.TenantID, acct.ID); err != nil {
		t.Fatalf("link account: %v", err)
	}
	// A pending trial (verification forthcoming) so we can observe activation.
	_, err = identity.GrantTrial(ctx, tx, identity.PolicyFromConfig(testCfg().Trial), identity.GrantTrialInput{
		AccountID: acct.ID, TenantID: seed.TenantID, TrialType: "standard",
		EligibilityKey: identity.EligibilityKeyFor(identity.ScopeOrganization, seed.TenantID),
		Status:         identity.TrialStatusPending, Source: "signup", Reason: "test",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("grant pending: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit fixture: %v", err)
	}
	return seed, acct
}

func identityRouter(pool *pgxpool.Pool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, testutil.TokenManager(), testCfg()).Register(router.Group("/v1"))
	return router
}

func doIdent(router http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func trialStatus(t *testing.T, pool *pgxpool.Pool, tenantID string) string {
	t.Helper()
	ctx := context.Background()
	var status string
	if err := pool.QueryRow(ctx, `SELECT status FROM trial_entitlements
		WHERE organization_id = $1::uuid ORDER BY created_at DESC LIMIT 1`, tenantID).Scan(&status); err != nil {
		t.Fatalf("read trial status: %v", err)
	}
	return status
}

func TestEmailVerify_ActivatesPendingTrial(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	email := fmt.Sprintf("verify-%s@example.com", uuid.NewString()[:8])
	seed, account := identityFixture(t, pool, email)
	router := identityRouter(pool)

	if got := trialStatus(t, pool, seed.TenantID); got != "pending" {
		t.Fatalf("pre fixture trial status = %s, want pending", got)
	}

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	plain, err := identity.CreateEmailToken(context.Background(), tx, account.ID, identity.TokenPurposeVerify, "", testCfg().Trial.EmailTokenTTL)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	_ = tx.Commit(context.Background())

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "", "owner")
	body := fmt.Sprintf(`{"verification_token":%q}`, plain)
	rec := doIdent(router, http.MethodPost, "/v1/identity/email/verify", body, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify status = %d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.Data["email_verified"] != true {
		t.Fatalf("email_verified = %v, want true", resp.Data["email_verified"])
	}

	// Email alone does NOT satisfy the phone requirement → trial stays pending.
	if got := trialStatus(t, pool, seed.TenantID); got != "pending" {
		t.Fatalf("trial status after email only = %s, want pending", got)
	}
}

func TestPhoneVerify_SelfActivatesOnceAllRequirementsMet(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	email := fmt.Sprintf("activ-%s@example.com", uuid.NewString()[:8])
	seed, account := identityFixture(t, pool, email)
	router := identityRouter(pool)

	ctx := context.Background()
	// Satisfy email via the store directly, then drive phone via the API.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := identity.NewAccountStore(identity.NewAuditRecorder()).MarkEmailVerified(ctx, tx, account.ID); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("mark email verified: %v", err)
	}
	code, err := identity.CreateOTPChallenge(ctx, tx, account.ID, account.PhoneE164, testCfg().Trial.OTPTTL)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("create otp: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "", "owner")
	rec := doIdent(router, http.MethodPost, "/v1/identity/phone/verify", `{"otp":"`+code+`"}`, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("phone verify status = %d body=%s", rec.Code, rec.Body.String())
	}
	// Both requirements met now → pending trial self-activates.
	if got := trialStatus(t, pool, seed.TenantID); got != "active" {
		t.Fatalf("trial status after phone verify = %s, want active", got)
	}
}

func TestIdentityEndpoints_RejectNonOwner(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	router := identityRouter(pool)

	token := testutil.MintAccess(t, seed.TenantID, seed.CashierID, seed.DeviceID, "", "cashier")
	rec := doIdent(router, http.MethodGet, "/v1/identity", "", token)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestEmailVerify_RejectsWrongAccountToken(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	email := fmt.Sprintf("mismatch-%s@example.com", uuid.NewString()[:8])
	seed, _ := identityFixture(t, pool, email)
	router := identityRouter(pool)

	// Token issued for an account that is NOT the tenant owner → 400 mismatch.
	ctx := context.Background()
	other, err := identity.NewAccountStore(identity.NewAuditRecorder()).Create(ctx, pool,
		fmt.Sprintf("other-%s@example.com", uuid.NewString()[:8]), "", false, false)
	if err != nil {
		t.Fatalf("create other account: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	plain, err := identity.CreateEmailToken(ctx, tx, other.ID, identity.TokenPurposeVerify, "", testCfg().Trial.EmailTokenTTL)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatalf("create token: %v", err)
	}
	_ = tx.Commit(ctx)

	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "", "owner")
	rec := doIdent(router, http.MethodPost, "/v1/identity/email/verify",
		fmt.Sprintf(`{"verification_token":%q}`, plain), token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}