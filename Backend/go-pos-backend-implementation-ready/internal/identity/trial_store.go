package identity

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// TrialStore owns read/transition operations on trial_entitlements. The table
// is append-only from the application's viewpoint: transitions are expressed
// as status updates, and the app role has no DELETE grant. Historical rows are
// never removed.
type TrialStore struct{}

func NewTrialStore() *TrialStore { return &TrialStore{} }

// GetByOrganization loads the live entitlement for an organization, if any.
func (s *TrialStore) GetByOrganization(ctx context.Context, q Querier, tenantID string) (*TrialEntitlement, error) {
	var e TrialEntitlement
	err := q.QueryRow(ctx, `
		SELECT id::text, organization_id::text, COALESCE(owner_user_id::text, ''), account_id::text,
		       trial_type, status, started_at, expires_at, trial_days, consumed_at, source, eligibility_key, COALESCE(reason, '')
		FROM trial_entitlements
		WHERE organization_id = $1::uuid
		ORDER BY created_at DESC LIMIT 1`, tenantID).
		Scan(&e.ID, &e.OrganizationID, &e.OwnerUserID, &e.AccountID,
			&e.TrialType, &e.Status, &e.StartedAt, &e.ExpiresAt, &e.TrialDays,
			&e.ConsumedAt, &e.Source, &e.EligibilityKey, &e.Reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &e, err
}

// GetByID loads one entitlement by id.
func (s *TrialStore) GetByID(ctx context.Context, q Querier, id string) (*TrialEntitlement, error) {
	var e TrialEntitlement
	err := q.QueryRow(ctx, `
		SELECT id::text, organization_id::text, COALESCE(owner_user_id::text, ''), account_id::text,
		       trial_type, status, started_at, expires_at, trial_days, consumed_at, source, eligibility_key, COALESCE(reason, '')
		FROM trial_entitlements WHERE id = $1::uuid`, id).
		Scan(&e.ID, &e.OrganizationID, &e.OwnerUserID, &e.AccountID,
			&e.TrialType, &e.Status, &e.StartedAt, &e.ExpiresAt, &e.TrialDays,
			&e.ConsumedAt, &e.Source, &e.EligibilityKey, &e.Reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &e, err
}

// List returns entitlements (newest first), used by the SaaS trial admin view.
func (s *TrialStore) List(ctx context.Context, q Querier, limit, offset int) ([]TrialEntitlement, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := q.Query(ctx, `
		SELECT id::text, organization_id::text, COALESCE(owner_user_id::text, ''), account_id::text,
		       trial_type, status, started_at, expires_at, trial_days, consumed_at, source, eligibility_key, COALESCE(reason, '')
		FROM trial_entitlements ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TrialEntitlement, 0, limit)
	for rows.Next() {
		var e TrialEntitlement
		if err := rows.Scan(&e.ID, &e.OrganizationID, &e.OwnerUserID, &e.AccountID,
			&e.TrialType, &e.Status, &e.StartedAt, &e.ExpiresAt, &e.TrialDays,
			&e.ConsumedAt, &e.Source, &e.EligibilityKey, &e.Reason); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// Extend pushes an active/pending trial's expiry forward (admin action).
// started_at is preserved; expires_at and trial_days move. Returns the updated
// entitlement together with the previous status for auditing.
func (s *TrialStore) Extend(ctx context.Context, q Querier, id string, extraDays int) (*TrialEntitlement, error) {
	if extraDays < 1 {
		return nil, errors.New("extra days must be positive")
	}
	var existing TrialEntitlement
	if err := q.QueryRow(ctx, `
		SELECT status, COALESCE(started_at, now()), expires_at FROM trial_entitlements WHERE id = $1::uuid`, id).
		Scan(&existing.Status, &existing.StartedAt, &existing.ExpiresAt); err != nil {
		return nil, err
	}
	base := existing.StartedAt
	if existing.ExpiresAt != nil && existing.ExpiresAt.After(*base) {
		base = existing.ExpiresAt
	}
	newExpiry := base.AddDate(0, 0, extraDays).UTC()
	var updated TrialEntitlement
	err := q.QueryRow(ctx, `
		UPDATE trial_entitlements
		SET expires_at = $2, trial_days = trial_days + $3, status = 'active', updated_at = now()
		WHERE id = $1::uuid AND status IN ('pending', 'active', 'expired')
		RETURNING id::text, organization_id::text, COALESCE(owner_user_id::text, ''), account_id::text,
		          trial_type, status, started_at, expires_at, trial_days, consumed_at, source, eligibility_key, COALESCE(reason, '')`,
		id, newExpiry, extraDays).
		Scan(&updated.ID, &updated.OrganizationID, &updated.OwnerUserID, &updated.AccountID,
			&updated.TrialType, &updated.Status, &updated.StartedAt, &updated.ExpiresAt, &updated.TrialDays,
			&updated.ConsumedAt, &updated.Source, &updated.EligibilityKey, &updated.Reason)
	if err != nil {
		return nil, err
	}
	// Keep the tenant projection in lock-step so /v1/subscription stays correct.
	if err := projectTenantTrial(ctx, q, updated.OrganizationID, updated.ExpiresAt); err != nil {
		return nil, err
	}
	if err := upsertTrialSubscription(ctx, q, updated.OrganizationID, "trial", updated.ExpiresAt); err != nil {
		return nil, err
	}
	return &updated, nil
}

// Revoke ends an entitlement immediately (admin action); the row is kept for
// audit, and the organization becomes eligible again once the org was merely
// revoked (the org_live partial index excludes 'revoked').
func (s *TrialStore) Revoke(ctx context.Context, q Querier, id string) (*TrialEntitlement, error) {
	_, err := q.Exec(ctx, `
		UPDATE trial_entitlements
		SET status = 'revoked', consumed_at = COALESCE(consumed_at, now()), updated_at = now()
		WHERE id = $1::uuid AND status IN ('pending', 'active', 'expired', 'converted')`, id)
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, q, id)
}

// ActivatePending flips a pending entitlement to active once the account has
// satisfied every configured verification requirement (or none was required
// and the hold was a review hold). The trial timer starts at activation, never
// at signup. projectTenantTrial + the subscriptions row move in lock-step.
func (s *TrialStore) ActivatePending(ctx context.Context, q Querier, tenantID, accountID string, account Account, policy TrialPolicy) (*TrialEntitlement, error) {
	if account.Status != AccountStatusActive && account.Status != "pending" {
		return nil, ErrStateConflict
	}
	if policy.RequireEmailVerification && !account.IsEmailVerified() {
		return nil, ErrVerificationRequired
	}
	if policy.RequirePhoneVerification && !account.IsPhoneVerified() {
		return nil, ErrVerificationRequired
	}
	days := policy.DurationDays
	if days <= 0 {
		days = 14
	}
	now := time.Now().UTC()
	expires := now.AddDate(0, 0, days)
	var ent TrialEntitlement
	err := q.QueryRow(ctx, `
		UPDATE trial_entitlements
		SET status = 'active', started_at = $3, expires_at = $4, trial_days = $5,
		    reason = 'verification_complete', updated_at = now()
		WHERE organization_id = $1::uuid AND account_id = $2::uuid AND status = 'pending' AND trial_type = 'standard'
		RETURNING id::text, organization_id::text, COALESCE(owner_user_id::text, ''), account_id::text,
		          trial_type, status, started_at, expires_at, trial_days, consumed_at, source, eligibility_key, COALESCE(reason, '')`,
		tenantID, accountID, now, expires, days).
		Scan(&ent.ID, &ent.OrganizationID, &ent.OwnerUserID, &ent.AccountID,
			&ent.TrialType, &ent.Status, &ent.StartedAt, &ent.ExpiresAt, &ent.TrialDays,
			&ent.ConsumedAt, &ent.Source, &ent.EligibilityKey, &ent.Reason)
	// A missing pending row simply means there is nothing to activate.
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := projectTenantTrial(ctx, q, tenantID, &expires); err != nil {
		return nil, err
	}
	if err := upsertTrialSubscription(ctx, q, tenantID, "trial", &expires); err != nil {
		return nil, err
	}
	return &ent, nil
}

// Convert marks a consumed/active trial as converted to a paid subscription
// (admin action mirroring the billing transition).
func (s *TrialStore) Convert(ctx context.Context, q Querier, id string) (*TrialEntitlement, error) {
	_, err := q.Exec(ctx, `
		UPDATE trial_entitlements
		SET status = 'converted', consumed_at = COALESCE(consumed_at, now()), updated_at = now()
		WHERE id = $1::uuid AND status IN ('active', 'expired')`, id)
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, q, id)
}
