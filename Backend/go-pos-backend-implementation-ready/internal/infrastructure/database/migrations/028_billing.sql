-- +goose Up
-- Platform billing core for the SaaS control plane (web admin).
--
-- These are PLATFORM-level tables: they are intentionally NOT tenant-scoped
-- and carry no RLS, because they are read/written only by the saas_admin role
-- through /v1/saas/*. Tenant data stays isolated behind its own RLS policies;
-- subscriptions/invoices merely reference tenants(id).
--
-- Money is integer minor units, matching the rest of the schema. The default
-- currency is EGP (Egypt-first demo configuration).

CREATE TABLE plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price_minor BIGINT NOT NULL DEFAULT 0 CHECK (price_minor >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'EGP',
    billing_period TEXT NOT NULL DEFAULT 'monthly'
        CHECK (billing_period IN ('monthly', 'yearly')),
    -- features is a JSON array of feature keys (e.g. ["inventory.advanced"]).
    features JSONB NOT NULL DEFAULT '[]'::jsonb,
    max_users INTEGER NOT NULL DEFAULT 0 CHECK (max_users >= 0),
    max_products INTEGER NOT NULL DEFAULT 0 CHECK (max_products >= 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id UUID NOT NULL REFERENCES plans(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'trial'
        CHECK (status IN ('trial', 'active', 'grace_period', 'past_due', 'suspended', 'cancelled')),
    -- provider is the payment gateway used for this subscription's invoices.
    provider TEXT NOT NULL DEFAULT 'mock',
    trial_ends_at TIMESTAMPTZ,
    current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    current_period_end TIMESTAMPTZ,
    cancel_at_period_end BOOLEAN NOT NULL DEFAULT FALSE,
    cancelled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- At most one live (non-cancelled) subscription per tenant.
CREATE UNIQUE INDEX subscriptions_tenant_live_uq
    ON subscriptions (tenant_id) WHERE status <> 'cancelled';
CREATE INDEX subscriptions_tenant_idx ON subscriptions (tenant_id);
CREATE INDEX subscriptions_status_idx ON subscriptions (status);

CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    amount_minor BIGINT NOT NULL CHECK (amount_minor >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'EGP',
    status TEXT NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'paid', 'void', 'refunded')),
    provider TEXT NOT NULL DEFAULT 'manual',
    provider_ref TEXT,
    description TEXT NOT NULL DEFAULT '',
    due_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX invoices_tenant_idx ON invoices (tenant_id, created_at DESC);
CREATE INDEX invoices_status_idx ON invoices (status);

CREATE TRIGGER plans_updated_at BEFORE UPDATE ON plans FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER subscriptions_updated_at BEFORE UPDATE ON subscriptions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER invoices_updated_at BEFORE UPDATE ON invoices FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Seed the default plan catalog (idempotent). 0 = unlimited.
INSERT INTO plans (code, name, description, price_minor, currency, billing_period, features, max_users, max_products)
VALUES
    ('starter', 'Starter', 'Solo store getting started', 0, 'EGP', 'monthly',
     '["pos.basic", "inventory.basic", "dashboard.basic"]'::jsonb, 3, 100),
    ('business', 'Business', 'Growing multi-cashier store', 29900, 'EGP', 'monthly',
     '["pos.basic", "inventory.advanced", "dashboard.advanced", "restaurant", "loyalty"]'::jsonb, 10, 1000),
    ('enterprise', 'Enterprise', 'Unlimited stores and verticals', 99900, 'EGP', 'monthly',
     '["pos.basic", "inventory.advanced", "dashboard.advanced", "restaurant", "pharmacy", "textile", "loyalty", "sync.multi_device"]'::jsonb, 0, 0)
ON CONFLICT (code) DO NOTHING;

-- Backfill a live subscription for every tenant that predates billing so the
-- control panel has data on day one. Trial tenants keep their trial window;
-- everyone else starts active on Starter. Plain INSERT..SELECT so it is safe
-- for goose (no DO $$ blocks).
INSERT INTO subscriptions (tenant_id, plan_id, status, trial_ends_at, current_period_end)
SELECT
    t.id,
    p.id,
    CASE WHEN t.plan = 'trial' THEN 'trial' ELSE 'active' END,
    CASE WHEN t.plan = 'trial' THEN t.trial_ends_at ELSE NULL END,
    NOW() + INTERVAL '30 days'
FROM tenants t
JOIN plans p ON p.code = 'starter'
WHERE NOT EXISTS (
    SELECT 1 FROM subscriptions s WHERE s.tenant_id = t.id AND s.status <> 'cancelled'
);

-- +goose Down
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plans;
