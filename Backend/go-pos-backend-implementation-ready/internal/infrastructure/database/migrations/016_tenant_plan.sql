-- +goose Up
-- Tenant plan & resource limits (D2 SaaS admin polish).
-- plan is informational; max_users / max_products cap creation with
-- 0 = unlimited so existing tenants stay unaffected and limits are opt-in.

ALTER TABLE tenants
    ADD COLUMN plan TEXT NOT NULL DEFAULT 'standard',
    ADD COLUMN max_users INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN max_products INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE tenants
    DROP COLUMN IF EXISTS plan,
    DROP COLUMN IF EXISTS max_users,
    DROP COLUMN IF EXISTS max_products;