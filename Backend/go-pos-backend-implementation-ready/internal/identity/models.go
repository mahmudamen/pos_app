// Package identity implements the free-trial and identity foundation for the
// POS SaaS: the platform-level `accounts` / `installations` /
// `trial_entitlements` / `audit_log` tables, the trial eligibility engine, and
// the verification helpers. See docs/21_TRIAL_IDENTITY.md.
//
// Design invariants:
//   - The backend is the sole authority on trial duration, scope and dates.
//     Nothing in this package (or any handler that uses it) accepts a trial
//     duration, expiry, or "has used trial" value from a client.
//   - Account identity (accounts), organization identity (tenants),
//     membership (users), installation identity (installations) and trial
//     entitlement (trial_entitlements) are distinct layers and are never
//     collapsed into a single device/IP/email assertion.
//   - A device/IP/network signal never decides identity by itself; it only
//     contributes to a risk bucket, and every hard decision is governed by
//     configuration.
package identity

import (
	"errors"
	"time"
)

// Account statuses (mirrors the accounts.status CHECK constraint).
const (
	AccountStatusPending   = "pending"
	AccountStatusActive    = "active"
	AccountStatusSuspended = "suspended"
	AccountStatusDisabled  = "disabled"
)

// ValidAccountStatuses lists every allowed account status.
var ValidAccountStatuses = []string{AccountStatusPending, AccountStatusActive, AccountStatusSuspended, AccountStatusDisabled}

// Risk levels shared by accounts and installations.
const (
	RiskLow     = "low"
	RiskMedium  = "medium"
	RiskHigh    = "high"
	RiskBlocked = "blocked"
)

// ValidRiskLevels lists every allowed risk level.
var ValidRiskLevels = []string{RiskLow, RiskMedium, RiskHigh, RiskBlocked}

// Trial entitlement statuses (mirrors trial_entitlements.status).
const (
	TrialStatusPending   = "pending"
	TrialStatusActive    = "active"
	TrialStatusExpired   = "expired"
	TrialStatusConverted = "converted"
	TrialStatusRevoked   = "revoked"
	TrialStatusCanceled  = "canceled"
)

// ValidTrialStatuses lists every allowed trial status.
var ValidTrialStatuses = []string{TrialStatusPending, TrialStatusActive, TrialStatusExpired, TrialStatusConverted, TrialStatusRevoked, TrialStatusCanceled}

func validTrialStatus(s string) bool {
	for _, candidate := range ValidTrialStatuses {
		if s == candidate {
			return true
		}
	}
	return false
}

// TrialSources records how an entitlement came to exist.
const (
	TrialSourceSignup   = "signup"
	TrialSourceManual   = "manual"
	TrialSourceBackfill = "backfill"
	TrialSourcePromo    = "promo"
)

// Account is one row of `accounts` (platform-level, no RLS).
type Account struct {
	ID              string
	PrimaryEmail    string
	EmailVerifiedAt *time.Time
	PhoneE164       string
	PhoneVerifiedAt *time.Time
	Status          string
	RiskLevel       string
}

// IsVerified reports whether email verification is satisfied.
func (a Account) IsEmailVerified() bool { return a.EmailVerifiedAt != nil }

// IsPhoneVerified reports whether phone verification is satisfied.
func (a Account) IsPhoneVerified() bool { return a.PhoneVerifiedAt != nil }

// Installation is one row of `installations`: a privacy-conscious record of an
// app install. It is a risk signal, never a permanent human identity.
type Installation struct {
	ID                    string
	InstallationPublicID  string
	AccountID             string
	TenantID              string
	Platform              string
	AppVersion            string
	FirstSeenAt           time.Time
	LastSeenAt            time.Time
	LastAuthenticatedAt   *time.Time
	DeviceIntegrityStatus string // "" | verified | unavailable | failed
	RiskLevel             string
	RevokedAt             *time.Time
}

// Revoked reports whether the installation was explicitly rotated away.
func (i Installation) Revoked() bool { return i.RevokedAt != nil }

// TrialEntitlement is one row of `trial_entitlements` — the authoritative trial record.
type TrialEntitlement struct {
	ID             string
	OrganizationID string
	OwnerUserID    string
	AccountID      string
	TrialType      string
	Status         string
	StartedAt      *time.Time
	ExpiresAt      *time.Time
	TrialDays      int
	ConsumedAt     *time.Time
	Source         string
	EligibilityKey string
	Reason         string
}

// EligibilityRequest is the fully server-derived input to CheckEligibility.
// It is constructed only inside server code from authenticated state and the
// database; none of these fields come from request bodies.
type EligibilityRequest struct {
	AccountID      string
	TenantID       string
	OwnerUserID    string
	TrialType      string
	EmailVerified  bool
	PhoneVerified  bool
	InstallationID string
	RiskLevel      string
	IP             string
}

// EligibilityResult is the outcome of CheckEligibility. A NeedsReview result
// means the identity looks suspicious but legitimate: the trial is held
// `pending` for manual review rather than denied outright.
type EligibilityResult struct {
	Eligible             bool
	NeedsReview          bool
	VerificationRequired bool
	VerificationHints    []string
	// Reason is a stable machine-readable code (see checks in trial.go).
	Reason string
}

var (
	// ErrTrialAlreadyExists marks a lost grant race or prior consumption.
	ErrTrialAlreadyExists = errors.New("trial already exists")
	// ErrNotFound marks a missing row for a lookup that expects one.
	ErrNotFound = errors.New("not found")
	// ErrStateConflict marks an avoidable duplicate (email/phone owned by
	// another account) or a row that cannot transition.
	ErrStateConflict = errors.New("state conflict")
	// ErrVerificationRequired marks an activation that was requested before
	// the account's configured verification requirements were met.
	ErrVerificationRequired = errors.New("verification required")
)

// AuditEntry is a security-relevant event; it maps to one audit_log row.
type AuditEntry struct {
	ActorUserID string
	AccountID   string
	TenantID    string
	Action      string
	EntityType  string
	EntityID    string
	Before      map[string]any
	After       map[string]any
	Reason      string
	IP          string
	UserAgent   string
}

// Audit action codes (append-only; see docs/21 §13).
const (
	ActionAccountRegistered       = "account.registered"
	ActionAccountSuspended        = "account.suspended"
	ActionEmailVerified           = "email.verified"
	ActionEmailChangeRequested    = "email.change_requested"
	ActionEmailChangeCompleted    = "email.change_completed"
	ActionPhoneVerified           = "phone.verified"
	ActionPhoneChangeRequested    = "phone.change_requested"
	ActionPhoneChangeCompleted    = "phone.change_completed"
	ActionTrialEligibilityChecked = "trial.eligibility_checked"
	ActionTrialGranted            = "trial.granted"
	ActionTrialDenied             = "trial.denied"
	ActionTrialExtended           = "trial.extended"
	ActionTrialRevoked            = "trial.revoked"
	ActionTrialConverted          = "trial.converted"
	ActionOrganizationCreated     = "organization.created"
	ActionOrganizationBlocked     = "organization.blocked"
	ActionOrganizationSuspended   = "organization.suspended"
	ActionInstallationRegistered  = "installation.registered"
	ActionInstallationChanged     = "installation.changed"
	ActionInstallationRevoked     = "installation.revoked"
	ActionSessionCreated          = "session.created"
	ActionSessionRevoked          = "session.revoked"
	ActionSessionLogoutAll        = "session.logout_all"
	ActionSuspiciousRegistration  = "registration.suspicious"
	ActionManualTrialOverride     = "trial.manual_override"
	ActionSubscriptionChanged     = "subscription.changed"
)

// Entity types written into audit_log.entity_type.
const (
	EntityAccount      = "account"
	EntityOrganization = "organization"
	EntityInstallation = "installation"
	EntityTrial        = "trial"
	EntityUser         = "user"
	EntitySession      = "session"
	EntitySubscription = "subscription"
	EntityRiskEvent    = "risk_event"
	EntityDuplicate    = "duplicate"
)

// AuditActionValid reports whether the action code is one we write.
func AuditActionValid(action string) bool {
	switch action {
	case ActionAccountRegistered, ActionAccountSuspended,
		ActionEmailVerified, ActionEmailChangeRequested, ActionEmailChangeCompleted,
		ActionPhoneVerified, ActionPhoneChangeRequested, ActionPhoneChangeCompleted,
		ActionTrialEligibilityChecked, ActionTrialGranted, ActionTrialDenied,
		ActionTrialExtended, ActionTrialRevoked, ActionTrialConverted,
		ActionOrganizationCreated, ActionOrganizationBlocked, ActionOrganizationSuspended,
		ActionInstallationRegistered, ActionInstallationChanged, ActionInstallationRevoked,
		ActionSessionCreated, ActionSessionRevoked, ActionSessionLogoutAll,
		ActionSuspiciousRegistration, ActionManualTrialOverride, ActionSubscriptionChanged:
		return true
	}
	return false
}
