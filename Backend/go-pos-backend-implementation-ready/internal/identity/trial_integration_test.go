package identity_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

func basePolicy() identity.TrialPolicy {
	return identity.TrialPolicy{
		DurationDays:                 14,
		Scope:                        identity.ScopeOrganization,
		RequireEmailVerification:     false,
		RequirePhoneVerification:     false,
		RequireDeviceIntegrity:       false,
		MaxOrganizationsPerAccount:   3,
		MaxActiveInstallations:       5,
		SuspiciousRegistrationPolicy: "review",
		RegisterRatePerIPPerHour:     0,
		PromoTrialsEnabled:           false,
		OfflinePolicy:                "grace24h",
	}
}

// newOrgAccount inserts a platform account + a standalone tenant (no owner
// user yet) and returns both ids. Emails/slugs are suffixed with the clock so
// repeated test runs never collide on the unique indexes.
func newOrgAccount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, email string) (accountID, tenantID string) {
	t.Helper()
	uniq := fmt.Sprint(time.Now().UnixNano())
	email = email[:len(email)-len("@example.com")] + "-" + uniq + "@example.com"
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	store := identity.NewAccountStore(identity.NewAuditRecorder())
	acct, err := store.Create(ctx, tx, email, "", false, false)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if err := tx.QueryRow(ctx,
		`INSERT INTO tenants (name, slug) VALUES ($1, $2) RETURNING id::text`,
		"Org "+email, "org-"+fmt.Sprint(time.Now().UnixNano())).Scan(&tenantID); err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit fixtures: %v", err)
	}
	return acct.ID, tenantID
}

func TestGrantTrial_ActiveProjection(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, tenantID := newOrgAccount(t, ctx, pool, "grant-active@example.com")
	start := time.Now().UTC().Add(-time.Minute)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ent, err := identity.GrantTrial(ctx, tx, basePolicy(), identity.GrantTrialInput{
		AccountID: accountID, TenantID: tenantID, TrialType: "standard",
		EligibilityKey: identity.EligibilityKeyFor(identity.ScopeOrganization, tenantID),
		Status:         identity.TrialStatusActive, Source: "signup", Reason: "new_org",
	}, start)
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if ent.Status != identity.TrialStatusActive {
		t.Fatalf("status = %s, want active", ent.Status)
	}
	if ent.ExpiresAt == nil || ent.TrialDays != 14 {
		t.Fatalf("unexpected entitlement: %+v", ent)
	}
	// grant may have started a second early; loosen the bound
	if ent.ExpiresAt.Sub(start) < 13*24*time.Hour {
		t.Fatalf("expires = %v, want ~14 days after start %v", ent.ExpiresAt, start)
	}

	// tenant projection
	var plan string
	var ends *time.Time
	if err := tx.QueryRow(ctx, `SELECT plan, trial_ends_at FROM tenants WHERE id = $1::uuid`, tenantID).
		Scan(&plan, &ends); err != nil {
		t.Fatalf("read tenant: %v", err)
	}
	if plan != "trial" || ends == nil || !ends.Equal(*ent.ExpiresAt) {
		t.Fatalf("tenant projection = (%s, %v), want (trial, %v)", plan, ends, ent.ExpiresAt)
	}

	// subscriptions row
	var subStatus string
	var subEnd *time.Time
	if err := tx.QueryRow(ctx, `SELECT status, trial_ends_at FROM subscriptions WHERE tenant_id = $1::uuid`, tenantID).
		Scan(&subStatus, &subEnd); err != nil {
		t.Fatalf("read subscription: %v", err)
	}
	if subStatus != "trial" || subEnd == nil {
		t.Fatalf("subscription = (%s, %v), want (trial, non-nil)", subStatus, subEnd)
	}
}

func TestGrantTrial_PendingThenActivate(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, tenantID := newOrgAccount(t, ctx, pool, "pending-before@example.com")

	policy := basePolicy()
	policy.RequireEmailVerification = true
	policy.RequirePhoneVerification = true

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ent, err := identity.GrantTrial(ctx, tx, policy, identity.GrantTrialInput{
		AccountID: accountID, TenantID: tenantID, TrialType: "standard",
		EligibilityKey: identity.EligibilityKeyFor(identity.ScopeOrganization, tenantID),
		Status:         identity.TrialStatusPending, Source: "signup",
		Reason: "waiting for verification",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	if ent.Status != identity.TrialStatusPending || ent.StartedAt != nil || ent.ExpiresAt != nil {
		t.Fatalf("pending entitlement should have nil dates: %+v", ent)
	}

	// Tenant is on trial with NULL end → the login gate reports trial_pending.
	var ends *time.Time
	if err := tx.QueryRow(ctx, `SELECT trial_ends_at FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&ends); err != nil {
		t.Fatalf("read tenant: %v", err)
	}
	if ends != nil {
		t.Fatalf("pending tenant trial_ends_at = %v, want NULL", ends)
	}

	// Activation before verification is refused.
	acct, err := identity.NewAccountStore(identity.NewAuditRecorder()).GetByID(ctx, tx, accountID)
	if err != nil {
		t.Fatalf("get account: %v", err)
	}
	if _, err := identity.NewTrialStore().ActivatePending(ctx, tx, tenantID, accountID, acct, policy); !errors.Is(err, identity.ErrVerificationRequired) {
		t.Fatalf("ActivatePending before verification = %v, want ErrVerificationRequired", err)
	}

	// Satisfy the requirements, then activation must start the timer.
	acct.EmailVerifiedAt = &time.Time{}
	acct.PhoneVerifiedAt = &time.Time{}
	acct.Status = identity.AccountStatusActive
	activated, err := identity.NewTrialStore().ActivatePending(ctx, tx, tenantID, accountID, acct, policy)
	if err != nil {
		t.Fatalf("activate: %v", err)
	}
	if activated.Status != identity.TrialStatusActive || activated.StartedAt == nil || activated.ExpiresAt == nil {
		t.Fatalf("activated entitlement wrong: %+v", activated)
	}

	var tenantEnds *time.Time
	if err := tx.QueryRow(ctx, `SELECT trial_ends_at FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&tenantEnds); err != nil {
		t.Fatalf("read tenant: %v", err)
	}
	if tenantEnds == nil || !tenantEnds.Equal(*activated.ExpiresAt) {
		t.Fatalf("tenant trial_ends_at = %v, want %v", tenantEnds, activated.ExpiresAt)
	}

	// Running activation again is a no-op, not an error.
	if _, err := identity.NewTrialStore().ActivatePending(ctx, tx, tenantID, accountID, acct, policy); err != nil {
		t.Fatalf("re-activate should be a no-op: %v", err)
	}
}

func TestGrantTrial_DoubleGrantFails(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, tenantID := newOrgAccount(t, ctx, pool, "double-grant@example.com")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	key := identity.EligibilityKeyFor(identity.ScopeOrganization, tenantID)
	in := identity.GrantTrialInput{
		AccountID: accountID, TenantID: tenantID, TrialType: "standard",
		EligibilityKey: key, Status: identity.TrialStatusActive, Source: "signup",
	}
	if _, err := identity.GrantTrial(ctx, tx, basePolicy(), in, time.Now().UTC()); err != nil {
		t.Fatalf("first grant: %v", err)
	}
	if _, err := identity.GrantTrial(ctx, tx, basePolicy(), in, time.Now().UTC()); !errors.Is(err, identity.ErrTrialAlreadyExists) {
		t.Fatalf("second grant = %v, want ErrTrialAlreadyExists", err)
	}
}

func TestGrantTrial_ConcurrentSameKey(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, tenantID := newOrgAccount(t, ctx, pool, "concurrent-grant@example.com")
	key := identity.EligibilityKeyFor(identity.ScopeOrganization, tenantID)

	const attempts = 8
	winners := make(chan string, attempts)
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx, err := pool.Begin(ctx)
			if err != nil {
				errs <- err
				return
			}
			defer func() { _ = tx.Rollback(ctx) }()
			in := identity.GrantTrialInput{
				AccountID: accountID, TenantID: tenantID, TrialType: "standard",
				EligibilityKey: key, Status: identity.TrialStatusActive, Source: "signup",
			}
			ent, err := identity.GrantTrial(ctx, tx, basePolicy(), in, time.Now().UTC())
			if err == nil {
				winners <- ent.Status
				_ = tx.Commit(ctx)
				return
			}
			if errors.Is(err, identity.ErrTrialAlreadyExists) {
				errs <- nil
				return
			}
			errs <- err
		}()
	}
	wg.Wait()
	close(winners)
	close(errs)
	var winnerCount int
	for range winners {
		winnerCount++
	}
	for err := range errs {
		if err != nil {
			t.Fatalf("unexpected error in race: %v", err)
		}
	}
	if winnerCount != 1 {
		t.Fatalf("grant winners = %d, want exactly 1", winnerCount)
	}
}

func TestCheckEligibility_OrganizationConsumed(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, tenantID := newOrgAccount(t, ctx, pool, "elig-consumed@example.com")
	policy := basePolicy()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	svc := identity.NewEligibilityService(policy, nil, identity.NewAuditRecorder())
	req := identity.EligibilityRequest{
		AccountID: accountID, TenantID: tenantID, TrialType: "standard",
		EmailVerified: true, PhoneVerified: true,
	}
	res, err := svc.CheckEligibility(ctx, tx, req)
	if err != nil {
		t.Fatalf("check before grant: %v", err)
	}
	if !res.Eligible {
		t.Fatalf("expected eligible, got %+v", res)
	}

	if _, err := identity.GrantTrial(ctx, tx, policy, identity.GrantTrialInput{
		AccountID: accountID, TenantID: tenantID, TrialType: "standard",
		EligibilityKey: identity.EligibilityKeyFor(identity.ScopeOrganization, tenantID),
		Status:         identity.TrialStatusActive, Source: "signup",
	}, time.Now().UTC()); err != nil {
		t.Fatalf("grant: %v", err)
	}

	res, err = svc.CheckEligibility(ctx, tx, req)
	if err != nil {
		t.Fatalf("check after grant: %v", err)
	}
	if res.Eligible || res.Reason != identity.ReasonOrgTrialConsumed {
		t.Fatalf("expected org_trial_consumed, got %+v", res)
	}
}

func TestCheckEligibility_VerificationGate(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, tenantID := newOrgAccount(t, ctx, pool, "elig-verify@example.com")
	policy := basePolicy()
	policy.RequireEmailVerification = true

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	svc := identity.NewEligibilityService(policy, nil, identity.NewAuditRecorder())
	res, err := svc.CheckEligibility(ctx, tx, identity.EligibilityRequest{
		AccountID: accountID, TenantID: tenantID, TrialType: "standard",
	})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if res.Eligible || !res.VerificationRequired {
		t.Fatalf("expected verification_required, got %+v", res)
	}
	found := false
	for _, hint := range res.VerificationHints {
		if hint == "email" {
			found = true
		}
	}
	if !found {
		t.Fatalf("hints = %v, want email", res.VerificationHints)
	}
}

func TestDenyTrialProjection(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	_, tenantID := newOrgAccount(t, ctx, pool, "deny-proj@example.com")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := identity.DenyTrialProjection(ctx, tx, tenantID); err != nil {
		t.Fatalf("deny projection: %v", err)
	}
	var plan string
	var ends time.Time
	if err := tx.QueryRow(ctx, `SELECT plan, trial_ends_at FROM tenants WHERE id = $1::uuid`, tenantID).
		Scan(&plan, &ends); err != nil {
		t.Fatalf("read tenant: %v", err)
	}
	if plan != "trial" || !ends.Before(time.Now()) {
		t.Fatalf("tenant = (%s, %v), want (trial, past)", plan, ends)
	}
	var subStatus string
	if err := tx.QueryRow(ctx, `SELECT status FROM subscriptions WHERE tenant_id = $1::uuid`, tenantID).
		Scan(&subStatus); err != nil {
		t.Fatalf("read subscription: %v", err)
	}
	if subStatus != "cancelled" {
		t.Fatalf("subscription status = %s, want cancelled", subStatus)
	}
}

func TestListAndLoadWithNullReason(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, tenantID := newOrgAccount(t, ctx, pool, "nullreason@example.com")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	store := identity.NewTrialStore()
	ent, err := identity.GrantTrial(ctx, tx, basePolicy(), identity.GrantTrialInput{
		AccountID: accountID, TenantID: tenantID, TrialType: "standard",
		EligibilityKey: identity.EligibilityKeyFor(identity.ScopeOrganization, tenantID),
		Status:         identity.TrialStatusActive, Source: "backfill",
	}, time.Now().UTC().Add(-time.Minute))
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	// reason column stays NULL — backfilled rows carry no reason.
	loaded, err := store.GetByID(ctx, tx, ent.ID)
	if err != nil {
		t.Fatalf("GetByID with NULL reason: %v", err)
	}
	if loaded.Reason != "" {
		t.Fatalf("reason = %q, want '' from NULL", loaded.Reason)
	}
	byOrg, err := store.GetByOrganization(ctx, tx, tenantID)
	if err != nil {
		t.Fatalf("GetByOrganization with NULL reason: %v", err)
	}
	if byOrg == nil || byOrg.Reason != "" {
		t.Fatalf("org load failed: %+v", byOrg)
	}
	items, err := store.List(ctx, tx, 50, 0)
	if err != nil {
		t.Fatalf("List with NULL reason: %v", err)
	}
	found := false
	for _, e := range items {
		if e.ID == ent.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("List did not include the NULL-reason entitlement")
	}
	extended, err := store.Extend(ctx, tx, ent.ID, 3)
	if err != nil {
		t.Fatalf("Extend RETURNING with NULL reason: %v", err)
	}
	if extended.Reason != "" {
		t.Fatalf("extended reason = %q, want ''", extended.Reason)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestExtend_MovesExpiryInLockStep(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, tenantID := newOrgAccount(t, ctx, pool, "extend@example.com")
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	store := identity.NewTrialStore()
	start := time.Now().UTC().Add(-24 * time.Hour)
	ent, err := identity.GrantTrial(ctx, tx, basePolicy(), identity.GrantTrialInput{
		AccountID: accountID, TenantID: tenantID, TrialType: "standard",
		EligibilityKey: identity.EligibilityKeyFor(identity.ScopeOrganization, tenantID),
		Status:         identity.TrialStatusActive, Source: "signup",
	}, start)
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	extended, err := store.Extend(ctx, tx, ent.ID, 7)
	if err != nil {
		t.Fatalf("extend: %v", err)
	}
	if extended.TrialDays != 14+7 {
		t.Fatalf("trial_days = %d, want 21", extended.TrialDays)
	}
	if extended.StartedAt == nil || extended.ExpiresAt == nil {
		t.Fatalf("extended dates nil: %+v", extended)
	}
	if extended.ExpiresAt.Sub(*extended.StartedAt) < 20*24*time.Hour {
		t.Fatalf("expiry not advanced: %v -> %v", extended.StartedAt, extended.ExpiresAt)
	}
	var tenantEnds *time.Time
	if err := tx.QueryRow(ctx, `SELECT trial_ends_at FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&tenantEnds); err != nil {
		t.Fatalf("read tenant: %v", err)
	}
	if tenantEnds == nil || !tenantEnds.Equal(*extended.ExpiresAt) {
		t.Fatalf("tenant trial_ends_at = %v, want %v", tenantEnds, extended.ExpiresAt)
	}
}

func TestRevoke_ThenEligibleAgain(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, tenantID := newOrgAccount(t, ctx, pool, "revoke@example.com")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	store := identity.NewTrialStore()
	policy := basePolicy()

	ent, err := identity.GrantTrial(ctx, tx, policy, identity.GrantTrialInput{
		AccountID: accountID, TenantID: tenantID, TrialType: "standard",
		EligibilityKey: identity.EligibilityKeyFor(identity.ScopeOrganization, tenantID),
		Status:         identity.TrialStatusActive, Source: "signup",
	}, time.Now().UTC())
	if err != nil {
		t.Fatalf("grant: %v", err)
	}
	revoked, err := store.Revoke(ctx, tx, ent.ID)
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if revoked.Status != identity.TrialStatusRevoked || revoked.ConsumedAt == nil {
		t.Fatalf("revoked entitlement wrong: %+v", revoked)
	}

	// The org_live index and the organization consumed-check both exclude
	// revoked rows, so the account stays eligible again.
	svc := identity.NewEligibilityService(policy, nil, identity.NewAuditRecorder())
	res, err := svc.CheckEligibility(ctx, tx, identity.EligibilityRequest{
		AccountID: accountID, TenantID: tenantID, TrialType: "standard",
		EmailVerified: true, PhoneVerified: true,
	})
	if err != nil {
		t.Fatalf("check after revoke: %v", err)
	}
	if !res.Eligible {
		t.Fatalf("expected eligible after revoke, got %+v", res)
	}
}

func TestEmailToken_ConsumeOnce(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, _ := newOrgAccount(t, ctx, pool, "email-token@example.com")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	plain, err := identity.CreateEmailToken(ctx, tx, accountID, "verify", "", time.Hour)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if len(plain) != 64 {
		t.Fatalf("token length = %d, want 64", len(plain))
	}
	payload, err := identity.ConsumeEmailToken(ctx, tx, plain, "verify", time.Hour)
	if err != nil {
		t.Fatalf("consume: %v", err)
	}
	if payload.AccountID != accountID {
		t.Fatalf("payload account = %s, want %s", payload.AccountID, accountID)
	}
	if _, err := identity.ConsumeEmailToken(ctx, tx, plain, "verify", time.Hour); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("second consume = %v, want ErrInvalidToken", err)
	}
}

func TestOTPFlow(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, _ := newOrgAccount(t, ctx, pool, "otp-flow@example.com")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	code, err := identity.CreateOTPChallenge(ctx, tx, accountID, "+201001234567", 10*time.Minute)
	if err != nil {
		t.Fatalf("create otp: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("otp length = %d, want 6", len(code))
	}
	if err := identity.VerifyOTP(ctx, tx, accountID, "+201001234567", code, 10*time.Minute); err != nil {
		t.Fatalf("verify otp: %v", err)
	}
	// Replay must be rejected (single-use).
	if err := identity.VerifyOTP(ctx, tx, accountID, "+201001234567", code, 10*time.Minute); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("otp replay = %v, want ErrInvalidToken", err)
	}
}

func TestAccountChangeEmail(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()
	accountID, _ := newOrgAccount(t, ctx, pool, "change-email@example.com")

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	store := identity.NewAccountStore(identity.NewAuditRecorder())

	newEmail := fmt.Sprintf("new-%d@example.com", time.Now().UnixNano())
	if err := store.ChangeEmail(ctx, tx, accountID, newEmail); err != nil {
		t.Fatalf("change email: %v", err)
	}
	updated, err := store.GetByID(ctx, tx, accountID)
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.PrimaryEmail != newEmail || updated.EmailVerifiedAt != nil {
		t.Fatalf("updated account = %+v", updated)
	}

	// audit trail captured the change
	var actions []string
	rows, err := tx.Query(ctx, `SELECT action FROM audit_log WHERE account_id = $1::uuid`, accountID)
	if err != nil {
		t.Fatalf("audit query: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var a string
		if err := rows.Scan(&a); err != nil {
			t.Fatalf("audit scan: %v", err)
		}
		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("audit rows: %v", err)
	}
	found := false
	for _, a := range actions {
		if a == identity.ActionEmailChangeCompleted {
			found = true
		}
	}
	if !found {
		t.Fatalf("audit actions = %v, want email.change_completed", actions)
	}
}
