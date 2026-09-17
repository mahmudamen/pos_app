-- +goose Up
-- Free-trial platform control plane (docs/21_TRIAL_IDENTITY.md §5 admin API).
--
--   platforms_settings — key/value store for admin-tunable platform policy.
--     The admin GET /v1/saas/trial-settings merges env defaults with the
--     overrides persisted here; the service always falls back to env when a
--     key is absent, so a fresh install has sane behavior before any admin
--     writes.
--   plans rows for 'trial' — signup grants land on a zero-priced trial plan so
--     the subscription row written at registration is self-consistent with the
--     rest of the billing control plane.
--   users partial unique index — at most one *active* owner per organization
--     (the membership invariant §2 of docs/21). Deactivated owners do not
--     block a successor.

CREATE TABLE platform_settings (
    key TEXT PRIMARY KEY,
    value JSONB NOT NULL,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- The trial plan is a real plans row so subscriptions written at signup are
-- consistent with /v1/saas/* billing and /v1/subscription. Price 0; the trial
-- is an entitlement, not a paid plan.
INSERT INTO plans (code, name, description, price_minor, currency, billing_period, features, max_users, max_products)
VALUES ('trial', 'Trial', 'Trial entitlement granted at signup', 0, 'EGP', 'monthly',
        '["pos.basic", "inventory.basic", "dashboard.basic"]'::jsonb, 3, 100)
ON CONFLICT (code) DO NOTHING;

-- One active owner per organization. signup and the SaaS control plane already
-- honor this; the constraint makes the invariant impossible to violate even
-- under a concurrent admin action.
-- Exactly one active owner per tenant. Legacy data created before the
-- invariant may carry duplicates; keep the earliest owner and demote the rest
-- to manager so the unique index can be created (and the invariant holds from
-- now on). The demotion is explicit and discoverable, not silent data loss.
-- users is FORCE-RLS, so this repair must lift FORCE for its duration
-- (fail-open only inside this migration transaction; restored immediately).
ALTER TABLE users NO FORCE ROW LEVEL SECURITY;
WITH ranked AS (
    SELECT u.id, u.tenant_id,
           row_number() OVER (PARTITION BY u.tenant_id ORDER BY u.created_at, u.id) AS rn
    FROM users u
    WHERE u.role = 'owner' AND u.is_active
)
UPDATE users SET role = 'manager', updated_at = now()
FROM ranked r WHERE r.id = users.id AND r.rn > 1;
ALTER TABLE users FORCE ROW LEVEL SECURITY;
CREATE UNIQUE INDEX users_one_active_owner_uq
    ON users (tenant_id)
    WHERE role = 'owner' AND is_active;

-- Owner lookups for accounts (org counts, admin attribution) are otherwise a
-- full scan on a table that grows with every merchant. owner_account_id is a
-- denormalized owner attribution that avoids joining FORCE-RLS users.
ALTER TABLE tenants ADD COLUMN owner_account_id UUID REFERENCES accounts(id);
-- users is FORCE-RLS, so the backfill join must lift FORCE for its duration
-- (fail-open only inside this migration transaction; restored immediately).
ALTER TABLE users NO FORCE ROW LEVEL SECURITY;
UPDATE tenants t SET owner_account_id = (
    SELECT u.account_id FROM users u WHERE u.id = t.owner_user_id LIMIT 1
);
ALTER TABLE users FORCE ROW LEVEL SECURITY;
CREATE INDEX tenants_owner_account_idx ON tenants (owner_account_id);

-- +goose Down
DROP INDEX IF EXISTS tenants_owner_account_idx;
ALTER TABLE tenants DROP COLUMN IF EXISTS owner_account_id;
DROP INDEX IF EXISTS tenants_owner_idx;
DROP INDEX IF EXISTS users_one_active_owner_uq;
DELETE FROM plans WHERE code = 'trial';
DROP TABLE IF EXISTS platform_settings;