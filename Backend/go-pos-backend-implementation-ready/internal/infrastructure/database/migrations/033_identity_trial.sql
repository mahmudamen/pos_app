-- +goose Up
-- Identity + free-trial entitlement foundation (Phase 0 of docs/21_TRIAL_IDENTITY.md).
--
-- Layered identity (deliberately NOT a global-users refactor):
--   accounts (global, one per normalized email) 1─* users (per-tenant login)
--   tenants 1─* users, tenants.owner_user_id → users
--   installations (privacy-conscious app install, spans tenants, no RLS)
--   trial_entitlements (authoritative, eligibility_key encodes the scope)
--   audit_log (append-only security trail)
--
-- accounts/installations/trial_entitlements/audit_log and the verification
-- helpers are PLATFORM-level (no RLS): the eligibility service must read an
-- account's cross-tenant trial history before a tenant context exists. They are
-- never exposed to clients directly; only server services touch them, under
-- least-privilege grants.

-- ── accounts ────────────────────────────────────────────────────────────────
CREATE TABLE accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    primary_email TEXT NOT NULL,
    email_verified_at TIMESTAMPTZ,
    phone_e164 TEXT,
    phone_verified_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('pending', 'active', 'suspended', 'disabled')),
    risk_level TEXT NOT NULL DEFAULT 'low' CHECK (risk_level IN ('low', 'medium', 'high', 'blocked')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX accounts_email_uq ON accounts (lower(primary_email));
CREATE UNIQUE INDEX accounts_phone_uq ON accounts (phone_e164) WHERE phone_e164 IS NOT NULL;
CREATE INDEX accounts_status_idx ON accounts (status);

-- ── installations ───────────────────────────────────────────────────────────
-- installation_public_id is a client-generated random UUID; it is a risk
-- signal, never an identity. It survives until uninstall / explicit rotation.
CREATE TABLE installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    installation_public_id UUID NOT NULL UNIQUE,
    account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    platform TEXT NOT NULL DEFAULT '',
    app_version TEXT NOT NULL DEFAULT '',
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_authenticated_at TIMESTAMPTZ,
    device_integrity_status TEXT CHECK (device_integrity_status IS NULL OR device_integrity_status IN ('verified', 'unavailable', 'failed')),
    risk_level TEXT NOT NULL DEFAULT 'low' CHECK (risk_level IN ('low', 'medium', 'high', 'blocked')),
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX installations_account_idx ON installations (account_id);
CREATE INDEX installations_tenant_idx ON installations (tenant_id);

-- ── trial_entitlements ──────────────────────────────────────────────────────
-- The authoritative trial record. (eligibility_key, trial_type) is unique, so
-- two concurrent grants for the same identity can never both win; GrantTrial
-- additionally takes a transaction advisory lock on the key.
CREATE TABLE trial_entitlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    owner_user_id UUID,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    trial_type TEXT NOT NULL DEFAULT 'standard',
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('pending', 'active', 'expired', 'converted', 'revoked', 'canceled')),
    started_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    trial_days INTEGER NOT NULL DEFAULT 0 CHECK (trial_days BETWEEN 0 AND 3650),
    consumed_at TIMESTAMPTZ,
    source TEXT NOT NULL DEFAULT 'signup',
    eligibility_key TEXT NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (expires_at IS NULL OR started_at IS NULL OR expires_at >= started_at)
);
CREATE UNIQUE INDEX trial_entitlements_scope_uq ON trial_entitlements (eligibility_key, trial_type);
CREATE UNIQUE INDEX trial_entitlements_org_live_uq
    ON trial_entitlements (organization_id)
    WHERE status IN ('pending', 'active', 'expired', 'converted');
CREATE INDEX trial_entitlements_account_idx ON trial_entitlements (account_id);
CREATE INDEX trial_entitlements_status_idx ON trial_entitlements (status, expires_at);

-- ── audit_log (append-only) ─────────────────────────────────────────────────
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor_user_id UUID,
    account_id UUID,
    tenant_id UUID,
    action TEXT NOT NULL,
    entity_type TEXT NOT NULL DEFAULT '',
    entity_id UUID,
    before JSONB,
    after JSONB,
    reason TEXT NOT NULL DEFAULT '',
    ip INET,
    user_agent TEXT NOT NULL DEFAULT ''
);
CREATE INDEX audit_log_created_idx ON audit_log (created_at DESC);
CREATE INDEX audit_log_actor_idx ON audit_log (actor_user_id, created_at DESC);
CREATE INDEX audit_log_tenant_idx ON audit_log (tenant_id, created_at DESC);
CREATE INDEX audit_log_action_idx ON audit_log (action, created_at DESC);

-- ── verification helpers ────────────────────────────────────────────────────
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    purpose TEXT NOT NULL CHECK (purpose IN ('verify', 'email_change')),
    new_email TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX email_verification_account_idx ON email_verification_tokens (account_id, expires_at DESC);

CREATE TABLE phone_otp_challenges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    phone_e164 TEXT NOT NULL,
    otp_hash TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX phone_otp_account_idx ON phone_otp_challenges (account_id, created_at DESC);

CREATE TABLE registration_risk_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    installation_id UUID REFERENCES installations(id) ON DELETE SET NULL,
    ip INET,
    signal TEXT NOT NULL,
    weight INTEGER NOT NULL DEFAULT 0,
    detail JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX registration_risk_events_created_idx ON registration_risk_events (created_at DESC);

-- ── link existing identity tables ───────────────────────────────────────────
ALTER TABLE tenants ADD COLUMN owner_user_id UUID,
    ADD COLUMN status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'closed'));

ALTER TABLE users ADD COLUMN account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    ADD COLUMN email_verified_at TIMESTAMPTZ,
    ADD COLUMN phone TEXT,
    ADD COLUMN phone_verified_at TIMESTAMPTZ,
    ADD COLUMN status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('pending', 'active', 'suspended', 'disabled'));

ALTER TABLE devices ADD COLUMN installation_id UUID REFERENCES installations(id) ON DELETE SET NULL;

-- ── backfill (legacy data is preserved, never rewritten as new trials) ──────
-- One account per already-registered email.
INSERT INTO accounts (primary_email, status)
SELECT lower(email), 'active' FROM store_emails
ON CONFLICT DO NOTHING;

-- users has FORCE RLS, so a backfill UPDATE would match zero rows without a
-- tenant context. Temporarily lift FORCE for the duration of the backfill; it
-- is restored immediately below (fail-open only inside this migration tx).
ALTER TABLE users NO FORCE ROW LEVEL SECURITY;
UPDATE users u SET account_id = a.id
FROM accounts a
WHERE u.account_id IS NULL AND a.primary_email = lower(u.email);

UPDATE tenants t SET owner_user_id = (
    SELECT u.id FROM users u WHERE u.tenant_id = t.id AND u.role = 'owner' ORDER BY u.created_at LIMIT 1
);

-- Preserve every historical trial as an entitlement (source='backfill').
INSERT INTO trial_entitlements (
    organization_id, owner_user_id, account_id, trial_type, status,
    started_at, expires_at, consumed_at, source, eligibility_key, trial_days)
SELECT t.id, t.owner_user_id, u.account_id, 'standard',
       CASE WHEN t.trial_ends_at > now() THEN 'active' ELSE 'expired' END,
       t.trial_ends_at - interval '15 days',
       t.trial_ends_at,
       CASE WHEN t.trial_ends_at <= now() THEN t.trial_ends_at ELSE NULL END,
       'backfill', 'org:' || t.id::text, 15
FROM tenants t
JOIN users u ON u.id = t.owner_user_id AND u.account_id IS NOT NULL
WHERE t.trial_ends_at IS NOT NULL
ON CONFLICT (eligibility_key, trial_type) DO NOTHING;
ALTER TABLE users FORCE ROW LEVEL SECURITY;

-- +goose Down
ALTER TABLE users FORCE ROW LEVEL SECURITY;
ALTER TABLE devices DROP COLUMN IF EXISTS installation_id;
ALTER TABLE users
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS phone_verified_at,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS email_verified_at,
    DROP COLUMN IF EXISTS account_id;
ALTER TABLE tenants DROP COLUMN IF EXISTS status, DROP COLUMN IF EXISTS owner_user_id;
DROP TABLE IF EXISTS registration_risk_events;
DROP TABLE IF EXISTS phone_otp_challenges;
DROP TABLE IF EXISTS email_verification_tokens;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS trial_entitlements;
DROP TABLE IF EXISTS installations;
DROP TABLE IF EXISTS accounts;