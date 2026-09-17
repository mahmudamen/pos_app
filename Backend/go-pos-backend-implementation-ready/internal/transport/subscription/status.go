package subscription

import (
	"time"

	"github.com/example/pos-api/internal/identity"
)

// DerivedStatus is the deterministic, clock-safe view of a tenant's
// subscription + trial. serverTime always comes from the database; a shifted
// client clock can never flip these values.
type DerivedStatus struct {
	Plan               string
	SubscriptionStatus string // trialing|active|grace_period|past_due|suspended|cancelled|none
	TrialStatus        string // pending|active|expired|converted|revoked|canceled|none
	TrialStart         *time.Time
	TrialExpires       *time.Time
	DaysRemaining      int
	ServerTime         time.Time
	AccessAllowed      bool
	RenewalRequired    bool
	SuspensionReason   string
}

// resolveTrialStatus derives a trial status from the entitlement row + DB time.
func resolveTrialStatus(e *identity.TrialEntitlement, dbNow time.Time) string {
	if e == nil || e.ID == "" {
		return "none"
	}
	switch e.Status {
	case identity.TrialStatusPending, identity.TrialStatusConverted, identity.TrialStatusRevoked, identity.TrialStatusCanceled:
		return e.Status
	default: // active / expired
		if e.ExpiresAt != nil && !e.ExpiresAt.After(dbNow) {
			return identity.TrialStatusExpired
		}
		return identity.TrialStatusActive
	}
}

// resolveSubscriptionStatus folds the subscription row, the account status and
// the trial into a single "how the API treats this tenant" status.
func resolveSubscriptionStatus(subStatus, accountStatus string, hasSubscription, suspendedAccount bool, trialStatus string) string {
	if suspendedAccount {
		return "suspended"
	}
	if hasSubscription && subStatus != "" && subStatus != "trial" {
		// A paid row wins; the trial is irrelevant once converted.
		return subStatus
	}
	switch trialStatus {
	case identity.TrialStatusActive:
		return "trialing"
	case identity.TrialStatusPending:
		return "trialing" // pending means granted but unaudited; still usable
	case identity.TrialStatusExpired:
		return "expired"
	default:
		if hasSubscription {
			return subStatus
		}
		return "none"
	}
}

// accessAllowed decides whether the POS stays usable. Grace periods are NOT
// applied here — offline grace is the client's offline cache policy, it does
// not extend server access.
func accessAllowed(subStatus, trialStatus string) bool {
	if subStatus == "suspended" || subStatus == "cancelled" {
		return false
	}
	if subStatus == "trialing" || subStatus == "active" || subStatus == "grace_period" || subStatus == "past_due" {
		return true
	}
	if trialStatus == identity.TrialStatusPending || trialStatus == identity.TrialStatusActive {
		return true
	}
	return false
}

func renewalRequired(subStatus, trialStatus string) bool {
	if subStatus == "expired" || trialStatus == identity.TrialStatusExpired {
		return true
	}
	return subStatus == "past_due" || subStatus == "cancelled"
}

// daysRemaining floors to whole days; zero is "today is the last usable day",
// negative is "already past". Clamped at 0 by the caller for display.
func daysRemaining(expires *time.Time, dbNow time.Time) int {
	if expires == nil {
		return 0
	}
	return int(expires.Truncate(24*time.Hour).Sub(dbNow.Truncate(24*time.Hour)) / (24 * time.Hour))
}
