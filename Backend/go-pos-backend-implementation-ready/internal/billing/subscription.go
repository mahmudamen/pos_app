package billing

import "fmt"

// Subscription lifecycle states.
const (
	StatusTrial       = "trial"
	StatusActive      = "active"
	StatusGracePeriod = "grace_period"
	StatusPastDue     = "past_due"
	StatusSuspended   = "suspended"
	StatusCancelled   = "cancelled"
)

// SubscriptionStatuses lists every valid state, in lifecycle order.
func SubscriptionStatuses() []string {
	return []string{StatusTrial, StatusActive, StatusGracePeriod, StatusPastDue, StatusSuspended, StatusCancelled}
}

// ValidSubscriptionStatus reports whether s is a known state.
func ValidSubscriptionStatus(s string) bool {
	for _, candidate := range SubscriptionStatuses() {
		if s == candidate {
			return true
		}
	}
	return false
}

// transition table: from -> allowed destinations. A subscription only ever
// moves along these edges; anything else is rejected with ErrInvalidTransition.
//
//	trial        -> active, past_due, suspended, cancelled
//	active       -> past_due, grace_period, suspended, cancelled
//	past_due     -> active, grace_period, suspended, cancelled
//	grace_period -> active, suspended, cancelled
//	suspended    -> active, cancelled
//	cancelled    -> (terminal)
var transitions = map[string]map[string]bool{
	StatusTrial: {
		StatusActive:    true,
		StatusPastDue:   true,
		StatusSuspended: true,
		StatusCancelled: true,
	},
	StatusActive: {
		StatusPastDue:     true,
		StatusGracePeriod: true,
		StatusSuspended:   true,
		StatusCancelled:   true,
	},
	StatusPastDue: {
		StatusActive:      true,
		StatusGracePeriod: true,
		StatusSuspended:   true,
		StatusCancelled:   true,
	},
	StatusGracePeriod: {
		StatusActive:    true,
		StatusSuspended: true,
		StatusCancelled: true,
	},
	StatusSuspended: {
		StatusActive:    true,
		StatusCancelled: true,
	},
	StatusCancelled: {},
}

// CanTransition reports whether a subscription may move from -> to. A no-op
// transition (same state) is allowed so callers can make idempotent requests.
func CanTransition(from, to string) bool {
	if from == to {
		return ValidSubscriptionStatus(from)
	}
	if !ValidSubscriptionStatus(from) || !ValidSubscriptionStatus(to) {
		return false
	}
	return transitions[from][to]
}

// ValidateTransition returns a descriptive error when from -> to is rejected.
func ValidateTransition(from, to string) error {
	if !ValidSubscriptionStatus(to) {
		return fmt.Errorf("unknown subscription status %q", to)
	}
	if !CanTransition(from, to) {
		return fmt.Errorf("cannot move subscription from %q to %q", from, to)
	}
	return nil
}

// StatusGrantsAccess reports whether a tenant in this state may use the POS.
// Suspended and cancelled tenants are blocked; everything else has access.
func StatusGrantsAccess(status string) bool {
	switch status {
	case StatusSuspended, StatusCancelled:
		return false
	default:
		return true
	}
}

// PlanChangeDecision is the outcome of evaluating an upgrade/downgrade against
// the tenant's current usage.
type PlanChangeDecision struct {
	Allowed     bool   `json:"allowed"`
	Code        string `json:"code,omitempty"`
	Required    int    `json:"required,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Remediation string `json:"remediation,omitempty"`
}

// EvaluatePlanChange decides whether a tenant with `currentUsers` users and
// `currentProducts` products can move to `target`. A limit of 0 means
// unlimited. A downgrade that would strand existing rows is blocked with a
// remediation message rather than silently disabling data (per the SaaS kit
// plan, PLAN_DOWNGRADE_BLOCKED).
func EvaluatePlanChange(currentUsers, currentProducts int, target Plan) PlanChangeDecision {
	if target.MaxUsers > 0 && currentUsers > target.MaxUsers {
		return PlanChangeDecision{
			Allowed:     false,
			Code:        "PLAN_DOWNGRADE_BLOCKED",
			Required:    currentUsers,
			Limit:       target.MaxUsers,
			Remediation: fmt.Sprintf("remove %d user(s) before downgrading", currentUsers-target.MaxUsers),
		}
	}
	if target.MaxProducts > 0 && currentProducts > target.MaxProducts {
		return PlanChangeDecision{
			Allowed:     false,
			Code:        "PLAN_DOWNGRADE_BLOCKED",
			Required:    currentProducts,
			Limit:       target.MaxProducts,
			Remediation: fmt.Sprintf("remove %d product(s) before downgrading", currentProducts-target.MaxProducts),
		}
	}
	return PlanChangeDecision{Allowed: true}
}
