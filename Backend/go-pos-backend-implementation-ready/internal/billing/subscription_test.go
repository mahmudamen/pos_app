package billing

import "testing"

func TestSubscriptionStatuses(t *testing.T) {
	want := []string{"trial", "active", "grace_period", "past_due", "suspended", "cancelled"}
	got := SubscriptionStatuses()
	if len(got) != len(want) {
		t.Fatalf("status count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("status[%d] = %q, want %q", i, got[i], want[i])
		}
		if !ValidSubscriptionStatus(want[i]) {
			t.Fatalf("%q should be valid", want[i])
		}
	}
	if ValidSubscriptionStatus("nonsense") || ValidSubscriptionStatus("") {
		t.Fatal("unknown statuses must be invalid")
	}
}

func TestCanTransition(t *testing.T) {
	allowed := [][2]string{
		{"trial", "active"},
		{"trial", "past_due"},
		{"trial", "suspended"},
		{"trial", "cancelled"},
		{"active", "past_due"},
		{"active", "grace_period"},
		{"active", "suspended"},
		{"active", "cancelled"},
		{"past_due", "active"},
		{"past_due", "grace_period"},
		{"grace_period", "active"},
		{"grace_period", "suspended"},
		{"grace_period", "cancelled"},
		{"suspended", "active"},
		{"suspended", "cancelled"},
		{"active", "active"}, // idempotent
	}
	for _, edge := range allowed {
		if !CanTransition(edge[0], edge[1]) {
			t.Fatalf("%s -> %s should be allowed", edge[0], edge[1])
		}
	}
	denied := [][2]string{
		{"trial", "grace_period"},
		{"past_due", "trial"},
		{"grace_period", "past_due"},
		{"suspended", "past_due"},
		{"suspended", "grace_period"},
		{"cancelled", "active"},
		{"cancelled", "trial"},
		{"cancelled", "suspended"},
		{"active", "trial"},
	}
	for _, edge := range denied {
		if CanTransition(edge[0], edge[1]) {
			t.Fatalf("%s -> %s should be denied", edge[0], edge[1])
		}
	}
	if CanTransition("bogus", "active") || CanTransition("active", "bogus") {
		t.Fatal("unknown states must never transition")
	}
}

func TestValidateTransition(t *testing.T) {
	if err := ValidateTransition("trial", "active"); err != nil {
		t.Fatalf("valid transition rejected: %v", err)
	}
	if err := ValidateTransition("active", "trial"); err == nil {
		t.Fatal("active -> trial should error")
	}
	if err := ValidateTransition("active", "nope"); err == nil {
		t.Fatal("unknown target should error")
	}
}

func TestStatusGrantsAccess(t *testing.T) {
	allowed := []string{"trial", "active", "grace_period", "past_due"}
	for _, s := range allowed {
		if !StatusGrantsAccess(s) {
			t.Fatalf("%s should grant access", s)
		}
	}
	for _, s := range []string{"suspended", "cancelled"} {
		if StatusGrantsAccess(s) {
			t.Fatalf("%s should block access", s)
		}
	}
}

func TestEvaluatePlanChange(t *testing.T) {
	target := Plan{Code: "starter", MaxUsers: 3, MaxProducts: 100}
	if d := EvaluatePlanChange(2, 50, target); !d.Allowed {
		t.Fatalf("under limits should be allowed: %+v", d)
	}
	// Exactly at the limit is fine.
	if d := EvaluatePlanChange(3, 100, target); !d.Allowed {
		t.Fatalf("at limits should be allowed: %+v", d)
	}
	d := EvaluatePlanChange(7, 50, target)
	if d.Allowed || d.Code != "PLAN_DOWNGRADE_BLOCKED" || d.Required != 7 || d.Limit != 3 {
		t.Fatalf("user overage decision wrong: %+v", d)
	}
	if d.Remediation == "" {
		t.Fatal("blocked downgrade must include remediation")
	}
	d = EvaluatePlanChange(2, 250, target)
	if d.Allowed || d.Code != "PLAN_DOWNGRADE_BLOCKED" || d.Required != 250 || d.Limit != 100 {
		t.Fatalf("product overage decision wrong: %+v", d)
	}
	// 0 = unlimited, never blocks.
	unlimited := Plan{Code: "enterprise", MaxUsers: 0, MaxProducts: 0}
	if d := EvaluatePlanChange(9999, 9999, unlimited); !d.Allowed {
		t.Fatalf("unlimited plan should allow anything: %+v", d)
	}
}
