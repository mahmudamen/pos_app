-- +goose Up
-- Stop (from the SaaS control panel) disables a tenant permanently: login is
-- rejected with organization_stopped and the live subscription is cancelled.
-- The tenants.status check constraint must accept 'disabled' (accounts/users
-- already do; tenants predates the identity-trial work).
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_status_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_status_check
    CHECK (status IN ('active', 'suspended', 'closed', 'disabled'));

-- +goose Down
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_status_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_status_check
    CHECK (status IN ('active', 'suspended', 'closed'));