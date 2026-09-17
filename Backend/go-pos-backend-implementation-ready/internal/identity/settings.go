package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/example/pos-api/internal/config"
	"github.com/jackc/pgx/v5"
)

const platformSettingsKey = "trial_settings"

// PolicyFromConfig derives the runtime trial policy from environment-backed
// configuration. No client-supplied value is involved. Zero values fall back to
// the same backend-owned defaults the env loader uses, so a partially populated
// config (a test harness, an older deployment) can never produce an unusable
// trial policy.
func PolicyFromConfig(cfg config.TrialConfig) TrialPolicy {
	p := TrialPolicy{
		DurationDays:                 cfg.DurationDays,
		Scope:                        cfg.Scope,
		RequireEmailVerification:     cfg.RequireEmailVerification,
		RequirePhoneVerification:     cfg.RequirePhoneVerification,
		RequireDeviceIntegrity:       cfg.RequireDeviceIntegrity,
		MaxOrganizationsPerAccount:   cfg.MaxOrganizationsPerAccount,
		MaxActiveInstallations:       cfg.MaxActiveInstallations,
		SuspiciousRegistrationPolicy: cfg.SuspiciousRegistrationPolicy,
		RegisterRatePerIPPerHour:     cfg.RegisterRatePerIPPerHour,
		PromoTrialsEnabled:           cfg.PromoTrialsEnabled,
		OfflinePolicy:                cfg.OfflinePolicy,
	}
	if p.DurationDays <= 0 {
		p.DurationDays = 14
	}
	if p.Scope == "" {
		p.Scope = ScopeAccount
	}
	if p.MaxOrganizationsPerAccount < 1 {
		p.MaxOrganizationsPerAccount = 3
	}
	if p.MaxActiveInstallations < 1 {
		p.MaxActiveInstallations = 10
	}
	if p.SuspiciousRegistrationPolicy == "" {
		p.SuspiciousRegistrationPolicy = "review"
	}
	if p.RegisterRatePerIPPerHour < 1 {
		p.RegisterRatePerIPPerHour = 5
	}
	if p.OfflinePolicy == "" {
		p.OfflinePolicy = "grace24h"
	}
	return p
}

// LoadPolicy resolves the effective policy: env defaults overlaid with any
// admin overrides persisted in platform_settings. Handlers that serve the
// policy must re-read it inside the relevant transaction.
func LoadPolicy(ctx context.Context, q Querier, cfg config.TrialConfig) (TrialPolicy, error) {
	policy := PolicyFromConfig(cfg)
	var raw []byte
	err := q.QueryRow(ctx,
		`SELECT value FROM platform_settings WHERE key = $1`, platformSettingsKey).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return policy, nil
	}
	if err != nil {
		return TrialPolicy{}, err
	}
	if err := OverlayPolicyJSON(&policy, raw); err != nil {
		return TrialPolicy{}, fmt.Errorf("invalid persisted trial_settings: %w", err)
	}
	return policy, nil
}

// SavePolicy persists admin overrides under platform_settings[trial_settings].
// Only validated values are stored; invalid input is rejected before write.
func SavePolicy(ctx context.Context, q Querier, p TrialPolicy) error {
	if err := ValidatePolicy(p); err != nil {
		return err
	}
	raw, err := json.Marshal(PolicyJSON(p))
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `
		INSERT INTO platform_settings (key, value, updated_at)
		VALUES ($1, $2, now())
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = now()`,
		platformSettingsKey, raw)
	return err
}

// ValidatePolicy checks the invariant set the config loader also enforces,
// so admin-written overrides can never take the service out of bounds.
func ValidatePolicy(p TrialPolicy) error {
	if !oneOf(p.Scope, config.ValidTrialScopes) {
		return fmt.Errorf("scope must be one of %v", config.ValidTrialScopes)
	}
	if !oneOf(p.SuspiciousRegistrationPolicy, config.ValidSuspiciousPolicies) {
		return fmt.Errorf("suspicious_registration_policy must be one of %v", config.ValidSuspiciousPolicies)
	}
	if p.DurationDays < 0 || p.DurationDays > 3650 {
		return fmt.Errorf("duration_days must be between 0 and 3650")
	}
	if p.MaxOrganizationsPerAccount < 1 {
		return fmt.Errorf("max_organizations_per_account must be at least 1")
	}
	if p.MaxActiveInstallations < 1 {
		return fmt.Errorf("max_active_installations must be at least 1")
	}
	if p.RegisterRatePerIPPerHour < 1 {
		return fmt.Errorf("register_rate_per_ip_per_hour must be at least 1")
	}
	if p.OfflinePolicy != "" && !oneOf(p.OfflinePolicy, []string{"grace24h", "grace72h", "block"}) {
		return fmt.Errorf("offline_trial_policy must be one of grace24h, grace72h, block")
	}
	return nil
}

// PolicyJSON is the wire form of the admin-adjustable policy.
func PolicyJSON(p TrialPolicy) map[string]any {
	return map[string]any{
		"trial_duration_days":            p.DurationDays,
		"trial_scope":                    p.Scope,
		"require_email_verification":     p.RequireEmailVerification,
		"require_phone_verification":     p.RequirePhoneVerification,
		"require_device_integrity":       p.RequireDeviceIntegrity,
		"max_organizations_per_account":  p.MaxOrganizationsPerAccount,
		"max_active_installations":       p.MaxActiveInstallations,
		"suspicious_registration_policy": p.SuspiciousRegistrationPolicy,
		"register_rate_per_ip_per_hour":  p.RegisterRatePerIPPerHour,
		"promo_trials_enabled":           p.PromoTrialsEnabled,
		"offline_trial_policy":           p.OfflinePolicy,
	}
}

// OverlayPolicyJSON applies a partial JSONB overlay over env-derived defaults.
func OverlayPolicyJSON(p *TrialPolicy, raw []byte) error {
	var fields map[string]any
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	key := func(name string) (any, bool) {
		if _, ok := fields[name]; !ok {
			return nil, false
		}
		return fields[name], true
	}
	if v, ok := boolField(key("require_email_verification")); ok {
		p.RequireEmailVerification = v
	}
	if v, ok := boolField(key("require_phone_verification")); ok {
		p.RequirePhoneVerification = v
	}
	if v, ok := boolField(key("require_device_integrity")); ok {
		p.RequireDeviceIntegrity = v
	}
	if v, ok := boolField(key("promo_trials_enabled")); ok {
		p.PromoTrialsEnabled = v
	}
	if v, ok := intField(key("trial_duration_days")); ok {
		p.DurationDays = v
	}
	if v, ok := intField(key("max_organizations_per_account")); ok {
		p.MaxOrganizationsPerAccount = v
	}
	if v, ok := intField(key("max_active_installations")); ok {
		p.MaxActiveInstallations = v
	}
	if v, ok := intField(key("register_rate_per_ip_per_hour")); ok {
		p.RegisterRatePerIPPerHour = v
	}
	if v, ok := strField(key("trial_scope")); ok {
		p.Scope = v
	}
	if v, ok := strField(key("suspicious_registration_policy")); ok {
		p.SuspiciousRegistrationPolicy = v
	}
	if v, ok := strField(key("offline_trial_policy")); ok {
		p.OfflinePolicy = v
	}
	return nil
}

func boolField(v any, ok bool) (bool, bool) {
	if !ok {
		return false, false
	}
	if b, isBool := v.(bool); isBool {
		return b, true
	}
	return false, false
}

func intField(v any, ok bool) (int, bool) {
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	}
	return 0, false
}

func strField(v any, ok bool) (string, bool) {
	if !ok {
		return "", false
	}
	if s, isStr := v.(string); isStr {
		return s, true
	}
	return "", false
}

func oneOf(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

// OfflineGraceHours maps the offline_trial_policy string to a clock-independent
// grace window used by the client. "block" means offline POS after first fetch
// is immediately restricted (grace 0).
func OfflineGraceHours(policy string) time.Duration {
	switch policy {
	case "grace72h":
		return 72 * time.Hour
	case "block":
		return 0
	default:
		return 24 * time.Hour
	}
}
