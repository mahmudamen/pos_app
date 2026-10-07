-- +goose Up
-- Onboarding trial reward (EPIC 11, docs/onboarding/TRIAL_REWARD.md).
-- Decision 2026-10-07: the trial stays 15 days; the 30-day extension in the
-- spec is a documented, deliberately-not-implemented deviation. The reward is
-- therefore a SERVER-AUTHORITATIVE idempotent milestone record, not a date
-- change: only an activated tenant (first sale completed, verified against the
-- real sales feed) can claim it, the server decides, and a user can never be
-- rewarded twice. The client is only told eligible=true and shows the
-- congratulations UI; it can never grant itself.

CREATE TABLE onboarding_rewards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reward_type TEXT NOT NULL DEFAULT 'ONBOARDING_ACTIVATION',
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id, reward_type),
    CONSTRAINT onboarding_rewards_type_format
        CHECK (reward_type ~ '^[A-Z0-9_]{1,64}$')
);

ALTER TABLE onboarding_rewards ENABLE ROW LEVEL SECURITY;
ALTER TABLE onboarding_rewards FORCE ROW LEVEL SECURITY;
CREATE POLICY onboarding_rewards_tenant_policy ON onboarding_rewards
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

CREATE INDEX onboarding_rewards_tenant_granted_idx
    ON onboarding_rewards (tenant_id, granted_at DESC);

-- +goose Down
DROP TABLE IF EXISTS onboarding_rewards;