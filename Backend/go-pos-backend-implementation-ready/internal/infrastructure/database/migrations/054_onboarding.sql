-- +goose Up
-- Onboarding & guided tours (client spec: Flutter/pos_go_app/docs/onboarding).
-- Progress is per user + tour and the SERVER is authoritative: a tour only
-- completes when the business goal it represents is really satisfied
-- (checked against products / sessions / sales, never trusted from the
-- client). onboarding_events records reported business-event ids so a replay
-- never double-counts progress or rewards (ONBOARDING_SYSTEM.md golden rule).
-- Both tables are awareness/progress data, not register state: they are
-- deliberately NOT part of the sync change feed and not in the platform
-- backup catalog.

CREATE TABLE onboarding_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tour_id TEXT NOT NULL,
    tour_version INT NOT NULL DEFAULT 1 CHECK (tour_version > 0),
    status TEXT NOT NULL DEFAULT 'not_started'
        CHECK (status IN ('not_started', 'running', 'paused', 'completed', 'skipped')),
    current_step TEXT NOT NULL DEFAULT '',
    started_at TIMESTAMPTZ,
    paused_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    skipped_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id, tour_id),
    CONSTRAINT onboarding_progress_tour_id_format
        CHECK (tour_id ~ '^[a-z][a-z0-9_]{0,31}$'),
    CONSTRAINT onboarding_progress_step_format
        CHECK (length(current_step) <= 64)
);

CREATE TABLE onboarding_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_id UUID NOT NULL,
    event TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, event_id),
    CONSTRAINT onboarding_events_event_format
        CHECK (event ~ '^[a-z][a-z0-9_.]{0,63}$'),
    CONSTRAINT onboarding_events_payload_object CHECK (jsonb_typeof(payload) = 'object')
);

ALTER TABLE onboarding_progress ENABLE ROW LEVEL SECURITY;
ALTER TABLE onboarding_progress FORCE ROW LEVEL SECURITY;
CREATE POLICY onboarding_progress_tenant_policy ON onboarding_progress
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE onboarding_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE onboarding_events FORCE ROW LEVEL SECURITY;
CREATE POLICY onboarding_events_tenant_policy ON onboarding_events
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

CREATE INDEX onboarding_progress_tenant_user_idx
    ON onboarding_progress (tenant_id, user_id, updated_at DESC);
CREATE INDEX onboarding_events_tenant_created_idx
    ON onboarding_events (tenant_id, created_at DESC);

-- +goose Down
DROP POLICY IF EXISTS onboarding_events_tenant_policy ON onboarding_events;
DROP TABLE IF EXISTS onboarding_events;
DROP POLICY IF EXISTS onboarding_progress_tenant_policy ON onboarding_progress;
DROP TABLE IF EXISTS onboarding_progress;
