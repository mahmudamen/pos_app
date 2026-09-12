-- +goose Up
-- Sync (B3, push side) — lock sync_commands down to its tenant. Migration
-- 005 created the table without RLS; every other tenant-scoped table enforces
-- FORCE ROW LEVEL SECURITY. sync_commands is written/read by the push handler
-- (server-internal, never exposed in /v1/sync/pull), but still needs RLS so ad
-- hoc queries cannot leak across tenants.

ALTER TABLE sync_commands ENABLE ROW LEVEL SECURITY;
ALTER TABLE sync_commands FORCE ROW LEVEL SECURITY;
CREATE POLICY sync_commands_tenant_policy ON sync_commands
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP POLICY IF EXISTS sync_commands_tenant_policy ON sync_commands;
ALTER TABLE sync_commands DISABLE ROW LEVEL SECURITY;