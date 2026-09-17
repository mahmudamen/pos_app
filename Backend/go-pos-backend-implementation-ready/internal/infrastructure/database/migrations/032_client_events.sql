-- +goose Up
-- Client telemetry: crash/error and usage events reported by the Flutter app
-- via POST /v1/client/events. Written by the runtime role, read by a future
-- owner/saas_admin inbox. Deliberately EXCLUDED from the sync change feed —
-- this is observability data, not business state that registers must replay.

CREATE TABLE client_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    device_id TEXT NOT NULL DEFAULT '',
    event TEXT NOT NULL,
    app_version TEXT,
    screen TEXT,
    payload JSONB NOT NULL DEFAULT '{}',
    stack_trace TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX client_events_tenant_created_idx ON client_events (tenant_id, created_at DESC);

ALTER TABLE client_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE client_events FORCE ROW LEVEL SECURITY;
CREATE POLICY client_events_tenant_policy ON client_events
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP POLICY IF EXISTS client_events_tenant_policy ON client_events;
ALTER TABLE client_events DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS client_events;