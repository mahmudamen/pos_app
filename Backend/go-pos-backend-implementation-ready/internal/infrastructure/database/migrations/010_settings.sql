-- +goose Up
-- Tenant settings (Odoo-style): key/value configuration scoped to a tenant.
-- Keys mirror Odoo POS settings: default payment method, stock badges, receipt footer.
-- Values are stored as TEXT; validation against a whitelist happens in the API layer.

CREATE TABLE tenant_settings (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by UUID,
    PRIMARY KEY (tenant_id, key)
);

ALTER TABLE tenant_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE tenant_settings FORCE ROW LEVEL SECURITY;
CREATE POLICY tenant_settings_tenant_policy ON tenant_settings
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP POLICY IF EXISTS tenant_settings_tenant_policy ON tenant_settings;
ALTER TABLE tenant_settings DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS tenant_settings;