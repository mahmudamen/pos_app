package identity

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// eligibilityKey builds the durable scope-encoded key that seeds the unique
// trial_entitlements constraint. The key is derived server-side only.
func eligibilityKey(scope, accountID, tenantID, phoneE164 string) string {
	switch scope {
	case "organization":
		return "org:" + tenantID
	case "verified_phone":
		return "phone:" + phoneE164
	case "business_identity":
		// Reserved: requires a business-verification data source. Key is
		// still deterministic so a future implementation is additive.
		return "biz:" + phoneE164 // placeholder (never granted)
	default:
		return "acct:" + accountID
	}
}

// GrantTrialInput carries everything GrantTrial projects onto the tenant and
// subscription. Status is one of TrialStatusActive / TrialStatusPending.
type GrantTrialInput struct {
	AccountID      string
	TenantID       string
	OwnerUserID    string
	TrialType      string
	EligibilityKey string
	Status         string
	DurationDays   int
	Source         string
	Reason         string
}

// GrantTrial creates the authoritative entitlement inside the caller's
// transaction and projects tenants.plan / trial_ends_at plus the live
// subscriptions row, so the login gate and the /v1/subscription endpoint stay
// consistent with the entitlement in one atomic step.
//
// Concurrency: the advisory transaction lock serializes concurrent grants for
// the same eligibility key, and the (eligibility_key, trial_type) unique index
// makes a second winner impossible. Grant lost via race or prior consumption →
// ErrTrialAlreadyExists (the caller denies without a trial; never a 500).
func GrantTrial(ctx context.Context, q Querier, cfg TrialPolicy, in GrantTrialInput, startedAt time.Time) (*TrialEntitlement, error) {
	if in.Status == "" {
		in.Status = TrialStatusActive
	}
	if in.DurationDays <= 0 {
		in.DurationDays = cfg.DurationDays
	}
	if in.DurationDays <= 0 {
		return nil, errors.New("trial duration must be positive")
	}
	if !validTrialStatus(in.Status) {
		return nil, fmt.Errorf("invalid trial status %q", in.Status)
	}
	if _, err := q.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`,
		in.EligibilityKey+":"+in.TrialType); err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	var started *time.Time
	if in.Status == TrialStatusActive && in.DurationDays > 0 {
		started = &startedAt
		e := startedAt.AddDate(0, 0, in.DurationDays).UTC()
		expiresAt = &e
	}

	var ent TrialEntitlement
	err := q.QueryRow(ctx, `
		INSERT INTO trial_entitlements (
			organization_id, owner_user_id, account_id, trial_type, status,
			started_at, expires_at, trial_days, consumed_at, source, eligibility_key, reason)
		VALUES ($1::uuid, NULLIF($2, '')::uuid, $3::uuid, $4, $5,
		        $6, $7, $8, NULL, $9, $10, $11)
		ON CONFLICT (eligibility_key, trial_type) DO NOTHING
		RETURNING id::text, organization_id::text, COALESCE(owner_user_id::text, ''), account_id::text,
		          trial_type, status, started_at, expires_at, trial_days, consumed_at, source, eligibility_key, reason`,
		in.TenantID, in.OwnerUserID, in.AccountID, in.TrialType, in.Status,
		started, expiresAt, in.DurationDays, in.Source, in.EligibilityKey, in.Reason).
		Scan(&ent.ID, &ent.OrganizationID, &ent.OwnerUserID, &ent.AccountID,
			&ent.TrialType, &ent.Status, &ent.StartedAt, &ent.ExpiresAt, &ent.TrialDays,
			&ent.ConsumedAt, &ent.Source, &ent.EligibilityKey, &ent.Reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTrialAlreadyExists
	}
	if err != nil {
		return nil, err
	}

	// Project the tenant + subscription so every downstream consumer (login
	// gate, /v1/subscription, SaaS billing) sees the same dates atomically.
	if err := projectTenantTrial(ctx, q, in.TenantID, expiresAt); err != nil {
		return nil, err
	}
	subStatus := "trial"
	periodEnd := expiresAt
	if in.Status == TrialStatusPending {
		periodEnd = nil
	}
	if err := upsertTrialSubscription(ctx, q, in.TenantID, subStatus, periodEnd); err != nil {
		return nil, err
	}
	return &ent, nil
}

// projectTenantTrial marks the tenant as a (possibly pending) trial holder so
// login's expiry gate and the subscription endpoint agree with the entitlement.
func projectTenantTrial(ctx context.Context, q Querier, tenantID string, expiresAt *time.Time) error {
	_, err := q.Exec(ctx, `
		UPDATE tenants SET plan = 'trial', trial_ends_at = $2, updated_at = now()
		WHERE id = $1::uuid`, tenantID, expiresAt)
	return err
}

// DenyTrialProjection is used when eligibility denied a trial yet still created
// an organization (legitimate user, prior consumption): the tenant is created
// with an instantly-expired trial so the existing login gate blocks access and
// /v1/subscription reports it. Nothing here is decided on the client.
func DenyTrialProjection(ctx context.Context, q Querier, tenantID string) error {
	now := time.Now().UTC()
	expired := now.Add(-time.Second)
	if err := projectTenantTrial(ctx, q, tenantID, &expired); err != nil {
		return err
	}
	return upsertTrialSubscription(ctx, q, tenantID, "cancelled", &now)
}

// upsertTrialSubscription keeps exactly one live (`<> 'cancelled'`)
// subscriptions row per tenant pointing at the zero-priced trial plan.
func upsertTrialSubscription(ctx context.Context, q Querier, tenantID, status string, periodEnd *time.Time) error {
	_, err := q.Exec(ctx, `
		INSERT INTO subscriptions (tenant_id, plan_id, status, trial_ends_at, current_period_start, current_period_end)
		SELECT $1::uuid, id, $2, $3, now(), $3
		FROM plans WHERE code = 'trial'
		ON CONFLICT (tenant_id) WHERE status <> 'cancelled'
		DO UPDATE SET status = EXCLUDED.status,
			trial_ends_at = EXCLUDED.trial_ends_at,
			current_period_end = EXCLUDED.current_period_end,
			updated_at = now()`,
		tenantID, status, periodEnd)
	return err
}

// TrialPolicy is the resolved, backend-owned trial configuration used by the
// eligibility engine. It is assembled by the runtime from env defaults overlaid
// with admin platform_settings (see settings.go); no client ever supplies it.
type TrialPolicy struct {
	DurationDays                 int
	Scope                        string
	RequireEmailVerification     bool
	RequirePhoneVerification     bool
	RequireDeviceIntegrity       bool
	MaxOrganizationsPerAccount   int
	MaxActiveInstallations       int
	SuspiciousRegistrationPolicy string // allow | review | deny
	RegisterRatePerIPPerHour     int
	PromoTrialsEnabled           bool
	OfflinePolicy                string // grace24h | grace72h | block
}

// Trial scopes (mirror config.ValidTrialScopes and the eligibility key prefix).
const (
	ScopeAccount          = "account"
	ScopeOrganization     = "organization"
	ScopeVerifiedPhone    = "verified_phone"
	ScopeBusinessIdentity = "business_identity"
)

// EligibilityKeyFor builds the durable key for a scope+identity pair. It must
// stay in lock-step with eligibilityKey (it is the exported form used by
// register and the admin manual-grant surface), so a key stored by GrantTrial
// always matches the key the eligibility checks search for.
func EligibilityKeyFor(scope, id string) string {
	switch scope {
	case ScopeOrganization:
		return "org:" + id
	case ScopeVerifiedPhone:
		return "phone:" + id
	case ScopeBusinessIdentity:
		return "biz:" + id
	default:
		return "acct:" + id
	}
}

// PolicyReason code constants (stable enum, surfaced to audit + admin).
const (
	ReasonAccountInvalid           = "account_invalid"
	ReasonEmailUnverified          = "email_unverified"
	ReasonPhoneUnverified          = "phone_unverified"
	ReasonOrganizationInvalid      = "organization_invalid"
	ReasonOrgTrialConsumed         = "org_trial_consumed"
	ReasonAccountTrialConsumed     = "account_trial_consumed"
	ReasonPhoneTrialConsumed       = "phone_trial_consumed"
	ReasonInstallationRisk         = "installation_risk"
	ReasonRiskRejected             = "risk_rejected"
	ReasonVelocityExceeded         = "velocity_exceeded"
	ReasonInvalidTrialType         = "invalid_trial_type"
	ReasonMaxOrganizationsReached  = "max_organizations_reached"
	ReasonInvalidEligibility       = "invalid_eligibility"
	ReasonBusinessScopeUnavailable = "business_scope_unavailable"
)

// liveStatusClause is shared by the consumed checks: pending/active/expired/
// converted all block a further trial of the same type for that identity.
const liveStatusClause = `status IN ('pending', 'active', 'expired', 'converted')`

// CheckEligibility evaluates whether a new trial may be granted for the
// request. It is purely evaluative: no rows are written. The caller is
// expected to run it inside the same transaction that will GrantTrial so the
// check cannot race with a concurrent grant.
type EligibilityService struct {
	policy  TrialPolicy
	counter Counter
	audit   AuditRecorder
}

func NewEligibilityService(policy TrialPolicy, counter Counter, audit AuditRecorder) *EligibilityService {
	return &EligibilityService{policy: policy, counter: counter, audit: audit}
}

// Policy exposes the resolved policy (used by the admin settings endpoint).
func (s *EligibilityService) Policy() TrialPolicy { return s.policy }

func (s *EligibilityService) CheckEligibility(ctx context.Context, q Querier, req EligibilityRequest) (EligibilityResult, error) {
	// 1. account exists & is usable.
	var acctStatus, risk string
	err := q.QueryRow(ctx, `SELECT status, risk_level FROM accounts WHERE id = $1::uuid`, req.AccountID).
		Scan(&acctStatus, &risk)
	if errors.Is(err, pgx.ErrNoRows) {
		return EligibilityResult{Eligible: false, Reason: ReasonAccountInvalid}, nil
	}
	if err != nil {
		return EligibilityResult{}, err
	}
	if acctStatus != AccountStatusActive {
		return EligibilityResult{Eligible: false, Reason: ReasonAccountInvalid}, nil
	}

	// 2/3. verification requirements, when configured. Missing verification is
	// not a hard denial: it produces a pending entitlement that self-activates
	// once the account verifies (register → verify → activate).
	if s.policy.RequireEmailVerification && !req.EmailVerified {
		return EligibilityResult{Eligible: false, VerificationRequired: true,
			VerificationHints: []string{"email"}, Reason: ReasonEmailUnverified}, nil
	}
	if s.policy.RequirePhoneVerification && !req.PhoneVerified {
		return EligibilityResult{Eligible: false, VerificationRequired: true,
			VerificationHints: []string{"phone"}, Reason: ReasonPhoneUnverified}, nil
	}
	if s.policy.RequireDeviceIntegrity && req.InstallationID != "" {
		var status string
		if err := q.QueryRow(ctx, `SELECT device_integrity_status FROM installations WHERE id = $1::uuid`, req.InstallationID).
			Scan(&status); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return EligibilityResult{}, err
			}
		} else if status != "verified" {
			return EligibilityResult{Eligible: false, Reason: ReasonInstallationRisk}, nil
		}
	}

	// 4. organization exists and is active.
	var orgStatus string
	err = q.QueryRow(ctx, `SELECT status FROM tenants WHERE id = $1::uuid`, req.TenantID).Scan(&orgStatus)
	if errors.Is(err, pgx.ErrNoRows) || orgStatus != "active" {
		return EligibilityResult{Eligible: false, Reason: ReasonOrganizationInvalid}, nil
	}
	if err != nil {
		return EligibilityResult{}, err
	}

	// 5/6. consumed checks per scope.
	switch s.policy.Scope {
	case "organization":
		var consumed bool
		if err := q.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM trial_entitlements WHERE organization_id = $1::uuid AND `+liveStatusClause+` AND trial_type = $2)`,
			req.TenantID, req.TrialType).Scan(&consumed); err != nil {
			return EligibilityResult{}, err
		}
		if consumed {
			return EligibilityResult{Eligible: false, Reason: ReasonOrgTrialConsumed}, nil
		}
	case "verified_phone":
		var e164 string
		var verified *time.Time
		if err := q.QueryRow(ctx, `SELECT phone_e164, phone_verified_at FROM accounts WHERE id = $1::uuid`, req.AccountID).
			Scan(&e164, &verified); err != nil {
			return EligibilityResult{}, err
		}
		if e164 == "" || verified == nil {
			return EligibilityResult{Eligible: false, Reason: ReasonPhoneUnverified}, nil
		}
		key := eligibilityKey("verified_phone", "", "", e164)
		var consumed bool
		if err := q.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM trial_entitlements WHERE eligibility_key = $1 AND trial_type = $2 AND `+liveStatusClause+`)`,
			key, req.TrialType).Scan(&consumed); err != nil {
			return EligibilityResult{}, err
		}
		if consumed {
			return EligibilityResult{Eligible: false, Reason: ReasonPhoneTrialConsumed}, nil
		}
	case "business_identity":
		// Requires an external business-verification data source that does not
		// exist; never silently grants. An admin can still manually grant.
		return EligibilityResult{Eligible: false, Reason: ReasonBusinessScopeUnavailable}, nil
	default: // account
		if req.AccountID == "" {
			return EligibilityResult{Eligible: false, Reason: ReasonInvalidEligibility}, nil
		}
		var consumed bool
		if err := q.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM trial_entitlements WHERE account_id = $1::uuid AND `+liveStatusClause+` AND trial_type = $2)`,
			req.AccountID, req.TrialType).Scan(&consumed); err != nil {
			return EligibilityResult{}, err
		}
		if consumed {
			return EligibilityResult{Eligible: false, Reason: ReasonAccountTrialConsumed}, nil
		}
	}

	// 7. installation risk signal (never identity, but a blocked/revoked
	// installation contributes to a denial).
	if req.InstallationID != "" {
		var riskLevel string
		var revoked *time.Time
		if err := q.QueryRow(ctx,
			`SELECT risk_level, revoked_at FROM installations WHERE id = $1::uuid`, req.InstallationID).
			Scan(&riskLevel, &revoked); err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return EligibilityResult{}, err
			}
		} else if riskLevel == RiskBlocked || revoked != nil {
			return EligibilityResult{Eligible: false, Reason: ReasonInstallationRisk}, nil
		}
	}

	// 8. trial type must be a configured, active type.
	switch req.TrialType {
	case "", "standard":
	case "promo":
		if !s.policy.PromoTrialsEnabled {
			return EligibilityResult{Eligible: false, Reason: ReasonInvalidTrialType}, nil
		}
	default:
		return EligibilityResult{Eligible: false, Reason: ReasonInvalidTrialType}, nil
	}

	// 9. account organization count stays within policy.
	if s.policy.MaxOrganizationsPerAccount > 0 {
		var orgs int
		if err := q.QueryRow(ctx,
			`SELECT COUNT(*) FROM tenants WHERE owner_account_id = $1::uuid AND status <> 'closed'`, req.AccountID).
			Scan(&orgs); err != nil {
			return EligibilityResult{}, err
		}
		if orgs >= s.policy.MaxOrganizationsPerAccount {
			return s.flagNeedsReviewOrDeny(ReasonMaxOrganizationsReached, req.TrialType), nil
		}
	}

	// 10. registration velocity (IP) — Redis-backed, fail-open.
	velOK := true
	if s.policy.RegisterRatePerIPPerHour > 0 && s.counter != nil && req.IP != "" {
		velOK = s.counter.Take(ctx, "register:ip:"+req.IP, s.policy.RegisterRatePerIPPerHour, time.Hour)
	}
	// 11. risk bucket × suspicious_registration_policy.
	switch risk {
	case RiskBlocked:
		return EligibilityResult{Eligible: false, Reason: ReasonRiskRejected}, nil
	case RiskHigh:
		flag := s.flagNeedsReviewOrDeny(ReasonRiskRejected, req.TrialType)
		if flag.NeedsReview {
			_ = s.audit.Record(ctx, q, AuditEntry{
				AccountID: req.AccountID, TenantID: req.TenantID,
				Action: ActionSuspiciousRegistration, EntityType: "trial_entitlements",
				Reason: ReasonRiskRejected,
			})
		}
		return flag, nil
	}
	if !velOK {
		return s.flagNeedsReviewOrDeny(ReasonVelocityExceeded, req.TrialType), nil
	}
	return EligibilityResult{Eligible: true}, nil
}

// flagNeedsReviewOrDeny maps a suspicion signal through the configured policy:
// allow → still eligible (risk only), review → held for manual approval,
// deny → hard rejection. Users are never permanently banned by a weak signal.
func (s *EligibilityService) flagNeedsReviewOrDeny(reason, trialType string) EligibilityResult {
	switch s.policy.SuspiciousRegistrationPolicy {
	case "allow":
		return EligibilityResult{Eligible: true}
	case "deny":
		return EligibilityResult{Eligible: false, Reason: reason}
	default: // review — the default
		return EligibilityResult{Eligible: false, NeedsReview: true, Reason: reason}
	}
}
