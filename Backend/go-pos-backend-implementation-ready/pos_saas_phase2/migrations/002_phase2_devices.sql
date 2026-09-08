-- +goose Up

CREATE TABLE devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    device_identifier TEXT NOT NULL,
    device_name TEXT NOT NULL,
    device_type TEXT NOT NULL DEFAULT 'pos',
    platform TEXT,
    app_version TEXT,
    credential_hash TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    UNIQUE (tenant_id, device_identifier),
    CONSTRAINT devices_status_ck CHECK (status IN ('active', 'revoked', 'disabled'))
);

CREATE INDEX devices_tenant_idx ON devices(tenant_id);
CREATE INDEX devices_status_idx ON devices(tenant_id, status);

ALTER TABLE user_sessions
    ADD COLUMN IF NOT EXISTS device_uuid UUID REFERENCES devices(id);

CREATE INDEX user_sessions_device_idx ON user_sessions(device_uuid);

ALTER TABLE user_sessions
    ADD COLUMN IF NOT EXISTS token_version BIGINT NOT NULL DEFAULT 1;

-- +goose Down

ALTER TABLE user_sessions DROP COLUMN IF EXISTS token_version;
ALTER TABLE user_sessions DROP COLUMN IF EXISTS device_uuid;
DROP TABLE IF EXISTS devices;
