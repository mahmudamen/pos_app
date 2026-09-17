package identity

import (
	"encoding/json"
	"testing"

	"github.com/example/pos-api/internal/config"
)

func TestPolicyFromConfigDefaults(t *testing.T) {
	cfg := config.TrialConfig{
		DurationDays:                 14,
		Scope:                        "organization",
		RequireEmailVerification:     true,
		MaxOrganizationsPerAccount:   1,
		MaxActiveInstallations:       5,
		SuspiciousRegistrationPolicy: "review",
		RegisterRatePerIPPerHour:     10,
		PromoTrialsEnabled:           false,
		OfflinePolicy:                "grace24h",
	}
	p := PolicyFromConfig(cfg)
	if p.DurationDays != 14 || p.Scope != "organization" || !p.RequireEmailVerification {
		t.Fatalf("unexpected policy defaults: %+v", p)
	}
}

func TestValidatePolicy(t *testing.T) {
	var p TrialPolicy
	if err := ValidatePolicy(p); err == nil {
		t.Error("empty policy should fail validation")
	}
	p = TrialPolicy{
		DurationDays:                 14,
		Scope:                        "organization",
		MaxOrganizationsPerAccount:   1,
		MaxActiveInstallations:       5,
		SuspiciousRegistrationPolicy: "review",
		RegisterRatePerIPPerHour:     10,
		OfflinePolicy:                "grace24h",
	}
	if err := ValidatePolicy(p); err != nil {
		t.Errorf("valid policy rejected: %v", err)
	}
	p.Scope = "nonsense"
	if err := ValidatePolicy(p); err == nil {
		t.Error("invalid scope should fail")
	}
	p.Scope = "account"
	p.SuspiciousRegistrationPolicy = "deny"
	if err := ValidatePolicy(p); err != nil {
		t.Errorf("valid suspicious policy rejected: %v", err)
	}
	p.OfflinePolicy = "forever"
	if err := ValidatePolicy(p); err == nil {
		t.Error("invalid offline policy should fail")
	}
}

func TestOverlayPolicyJSON(t *testing.T) {
	cfg := config.TrialConfig{
		DurationDays:                 14,
		Scope:                        "account",
		RequireEmailVerification:     false,
		RequirePhoneVerification:     false,
		MaxOrganizationsPerAccount:   1,
		MaxActiveInstallations:       5,
		SuspiciousRegistrationPolicy: "review",
		RegisterRatePerIPPerHour:     10,
		PromoTrialsEnabled:           false,
		OfflinePolicy:                "grace24h",
	}
	p := PolicyFromConfig(cfg)
	overlay, _ := json.Marshal(map[string]any{
		"trial_duration_days":            30,
		"require_email_verification":     true,
		"suspicious_registration_policy": "allow",
	})
	if err := OverlayPolicyJSON(&p, overlay); err != nil {
		t.Fatalf("overlay: %v", err)
	}
	if p.DurationDays != 30 || !p.RequireEmailVerification || p.SuspiciousRegistrationPolicy != "allow" {
		t.Fatalf("overlay not applied: %+v", p)
	}
	// untouched values keep their env default
	if p.RequirePhoneVerification || p.OfflinePolicy != "grace24h" {
		t.Fatalf("untouched fields changed: %+v", p)
	}
	// invalid JSON is rejected
	if err := OverlayPolicyJSON(&p, []byte("{not json")); err == nil {
		t.Error("malformed overlay should be rejected")
	}
	// unknown keys are ignored, not fatal
	if err := OverlayPolicyJSON(&p, json.RawMessage(`{"bogus_key": 1}`)); err != nil {
		t.Errorf("unknown keys should be ignored, got %v", err)
	}
}

func TestOfflineGraceHours(t *testing.T) {
	if OfflineGraceHours("grace72h") != 72*3600e9 {
		t.Errorf("grace72h = %v, want 72h", OfflineGraceHours("grace72h"))
	}
	if OfflineGraceHours("block") != 0 {
		t.Errorf("block should be zero grace")
	}
	if OfflineGraceHours("") != 24*3600e9 {
		t.Errorf("empty should default to 24h grace")
	}
}

func TestPolicyJSONRoundTrip(t *testing.T) {
	p := TrialPolicy{
		DurationDays: 21, Scope: "verified_phone", RequireEmailVerification: true,
		MaxOrganizationsPerAccount: 2, MaxActiveInstallations: 3,
		SuspiciousRegistrationPolicy: "deny", RegisterRatePerIPPerHour: 5,
		PromoTrialsEnabled: true, OfflinePolicy: "block",
	}
	raw, err := json.Marshal(PolicyJSON(p))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back TrialPolicy
	if err := OverlayPolicyJSON(&back, raw); err != nil {
		t.Fatalf("overlay: %v", err)
	}
	if back.DurationDays != 21 || back.Scope != "verified_phone" || back.OfflinePolicy != "block" {
		t.Fatalf("round trip changed values: %+v", back)
	}
}
