package subscription

import (
	"testing"
	"time"

	"github.com/example/pos-api/internal/identity"
)

func TestResolveTrialStatus(t *testing.T) {
	now := time.Date(2026, 9, 17, 12, 0, 0, 0, time.UTC)
	future := now.Add(24 * time.Hour)
	past := now.Add(-time.Hour)

	cases := []struct {
		name string
		ent  *identity.TrialEntitlement
		want string
	}{
		{"none", nil, "none"},
		{"active-future", &identity.TrialEntitlement{ID: "x", Status: identity.TrialStatusActive, ExpiresAt: &future}, identity.TrialStatusActive},
		{"active-expired", &identity.TrialEntitlement{ID: "x", Status: identity.TrialStatusActive, ExpiresAt: &past}, identity.TrialStatusExpired},
		{"pending", &identity.TrialEntitlement{ID: "x", Status: identity.TrialStatusPending}, identity.TrialStatusPending},
		{"revoked", &identity.TrialEntitlement{ID: "x", Status: identity.TrialStatusRevoked}, identity.TrialStatusRevoked},
		{"converted", &identity.TrialEntitlement{ID: "x", Status: identity.TrialStatusConverted}, identity.TrialStatusConverted},
	}
	for _, tc := range cases {
		if got := resolveTrialStatus(tc.ent, now); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestResolveSubscriptionStatus(t *testing.T) {
	cases := []struct {
		name          string
		subStatus     string
		accountStatus string
		hasSub        bool
		suspendedAcct bool
		trialStatus   string
		want          string
	}{
		{"no sub no trial", "", "active", false, false, "none", "none"},
		{"trialing", "trial", "active", true, false, identity.TrialStatusActive, "trialing"},
		{"pending-hold", "trial", "active", true, false, identity.TrialStatusPending, "trialing"},
		{"trial-expired", "trial", "active", true, false, identity.TrialStatusExpired, "expired"},
		{"paid-active", "active", "active", true, false, identity.TrialStatusExpired, "active"},
		{"grace", "grace_period", "active", true, false, "none", "grace_period"},
		{"cancelled-sub", "cancelled", "active", true, false, "none", "cancelled"},
		{"cancelled-wins-over-trial", "cancelled", "active", true, false, identity.TrialStatusActive, "cancelled"},
		{"suspended-account-wins", "active", "suspended", true, true, identity.TrialStatusActive, "suspended"},
	}
	for _, tc := range cases {
		if got := resolveSubscriptionStatus(tc.subStatus, tc.accountStatus, tc.hasSub, tc.suspendedAcct, tc.trialStatus); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestAccessAllowed(t *testing.T) {
	allowed := []struct{ sub, trial string }{
		{"trialing", identity.TrialStatusPending},
		{"trialing", identity.TrialStatusActive},
		{"active", "none"},
		{"grace_period", "none"},
		{"past_due", "none"},
	}
	for _, c := range allowed {
		if !accessAllowed(c.sub, c.trial) {
			t.Errorf("expected access for %v", c)
		}
	}
	for _, c := range []struct{ sub, trial string }{
		{"suspended", "none"}, {"cancelled", "none"}, {"expired", identity.TrialStatusExpired},
		{"suspended", identity.TrialStatusActive},
	} {
		if accessAllowed(c.sub, c.trial) {
			t.Errorf("expected denial for %v", c)
		}
	}
}

func TestRenewalRequired(t *testing.T) {
	if !renewalRequired("expired", identity.TrialStatusExpired) {
		t.Error("expired should require renewal")
	}
	if !renewalRequired("past_due", "none") {
		t.Error("past_due should require renewal")
	}
	if !renewalRequired("active", identity.TrialStatusExpired) {
		t.Error("expired trial with active sub should maybe not require renewal")
	}
	if renewalRequired("active", identity.TrialStatusActive) {
		t.Error("active trial should not require renewal")
	}
}

func TestDaysRemaining(t *testing.T) {
	now := time.Date(2026, 9, 17, 10, 0, 0, 0, time.UTC)
	expires := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	if got := daysRemaining(&expires, now); got != 7 {
		t.Errorf("daysRemaining = %d, want 7", got)
	}
	if got := daysRemaining(nil, now); got != 0 {
		t.Errorf("nil expiry should be 0, got %d", got)
	}
	past := time.Date(2026, 9, 10, 10, 0, 0, 0, time.UTC)
	if got := daysRemaining(&past, now); got != -7 {
		t.Errorf("past expiry should be negative, got %d", got)
	}
}
