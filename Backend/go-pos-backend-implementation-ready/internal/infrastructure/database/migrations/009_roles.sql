-- +goose Up
-- Roles: three-tier tenant hierarchy — owner > manager > cashier.
-- saas_admin remains the platform-level role (never assignable per-tenant).
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('owner', 'manager', 'cashier', 'saas_admin'));

-- +goose Down
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('manager', 'cashier', 'saas_admin'));