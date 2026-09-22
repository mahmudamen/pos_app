-- Least-privilege production grants for the pos-api app role.
-- Run ONCE as a PostgreSQL administrator against the production database after
-- migrations have been applied:
--   psql -U postgres -d pos -f scripts/grants_prod.sql
--
-- The app role gets:
--   * DML on the tenant data schemas only (no DDL, no superuser),
--   * USAGE on sequences it actually increments (sale counter, sync cursors),
--   * maintenance reserved to the owner/saas_admin roles which stay separate.
--
-- This script is idempotent and safe to re-run after a rollout.

\set ON_ERROR_STOP on

-- Make sure the app role exists (plain LOGIN, NO superuser/createdb/createrole).
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app') THEN
        CREATE ROLE pos_app LOGIN;
    END IF;
END
$$;

-- Tenant data lives in the public schema unless you adopt a per-tenant schema
-- layout. Grant only what the runtime needs.
GRANT USAGE ON SCHEMA public TO pos_app;
GRANT SELECT, INSERT, UPDATE, DELETE
  ON ALL TABLES IN SCHEMA public TO pos_app;

-- Forward rows are protected by FORCE ROW LEVEL SECURITY; the app role is
-- *not* given BYPASSRLS, so tenant isolation applies to it like everyone else.
-- Confirmed: cross-tenant reads return nothing without app.current_tenant set.

-- Sequences the app actually advances at runtime (sale idempotency, sync
-- cursors, register sessions). Grant USAGE/SELECT on all and let ownership of
-- the tables drive the rest; this covers future migrations' sequences too.
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO pos_app;

-- Merchant signup stores email uniqueness in store_emails, which is a plain
-- (non-RLS) table so concurrent registrations across tenants are safe. Grant
-- the app rows INSERT+SELECT; the least-privilege pos_app_rls role gets the
-- same when it is provisioned (production only).
GRANT SELECT, INSERT ON TABLE store_emails TO pos_app;
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app_rls') THEN
        GRANT SELECT, INSERT ON TABLE store_emails TO pos_app_rls;
    END IF;
END
$$;

-- The sync/analytics handlers call set_config with app.current_tenant; that is
-- allowed for any role (it is a per-session GUC), verified by the RLS tests.
-- No special grant required beyond the above.

-- users_pos_security (migration 027) is written by the users manager and read
-- by auth/login + the sales checkout; the least-privilege role needs its DML.
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app_rls') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE users_pos_security TO pos_app_rls;
    END IF;
END
$$;

-- Billing tables (migration 028) are platform-level (plans, subscriptions,
-- invoices): read/written by the saas_admin control plane endpoints. Grant the
-- least-privilege role its DML like every other table the API touches.
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app_rls') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE plans TO pos_app_rls;
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE subscriptions TO pos_app_rls;
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE invoices TO pos_app_rls;
    END IF;
END
$$;

-- Client telemetry (migration 032) is written by the app role and read by a
-- future owner/saas_admin inbox; INSERT is all the runtime role needs.
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app_rls') THEN
        GRANT SELECT, INSERT ON TABLE client_events TO pos_app_rls;
    END IF;
END
$$;

-- Units + bundled Egyptian price catalog (migrations 029/030) are platform
-- reference data: read-only for the runtime role. The price-refresh job reuses
-- the products DML grant (already above) to update seeded rows.
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app_rls') THEN
        GRANT SELECT ON TABLE units TO pos_app_rls;
        GRANT SELECT ON TABLE unit_conversions TO pos_app_rls;
        GRANT SELECT ON TABLE egypt_price_catalog TO pos_app_rls;
    END IF;
END
$$;

-- Purchase ledger + OCR-scanned line items (migration 035). Tenant-RLS rows
-- written by the purchases module; the runtime role needs the same DML it
-- exercises when an invoice is applied/listed.
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app_rls') THEN
        GRANT SELECT, INSERT ON TABLE purchases TO pos_app_rls;
        GRANT SELECT, INSERT ON TABLE purchase_items TO pos_app_rls;
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE ocr_usage TO pos_app_rls;
        GRANT UPDATE (ocr_credits_remaining) ON TABLE tenants TO pos_app_rls;
    END IF;
END
$$;

-- Self-ordering (migration 038). Public web orders INSERT self_orders through
-- the tenant-scoped tx; the staff module reads/updates both tables.
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app_rls') THEN
        GRANT SELECT, INSERT, UPDATE ON TABLE self_orders TO pos_app_rls;
        GRANT SELECT, INSERT, UPDATE ON TABLE product_requests TO pos_app_rls;
    END IF;
END
$$;

-- ── Identity + trial platform tables (migrations 033/034) ───────────────────
-- These are platform-level (no RLS) and are only ever read/written by server
-- services, never exposed to clients. audit_log is append-only: no UPDATE or
-- DELETE grants so no code path can rewrite history. trial_entitlements is
-- append-only too: status transitions are UPDATEs (allowed), but rows are
-- never deleted. accounts.primary_email uniqueness must stay immutable to
-- whoever could write, so the role gets the same DML it exercises in signup.
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app_rls') THEN
        GRANT INSERT, SELECT, UPDATE ON TABLE accounts TO pos_app_rls;
        GRANT INSERT, SELECT, UPDATE ON TABLE installations TO pos_app_rls;
        GRANT INSERT, SELECT, UPDATE ON TABLE trial_entitlements TO pos_app_rls;
        GRANT INSERT, SELECT ON TABLE audit_log TO pos_app_rls;
        GRANT INSERT, SELECT, UPDATE ON TABLE platform_settings TO pos_app_rls;
        GRANT INSERT, SELECT, UPDATE ON TABLE email_verification_tokens TO pos_app_rls;
        GRANT INSERT, SELECT, UPDATE ON TABLE phone_otp_challenges TO pos_app_rls;
        GRANT INSERT, SELECT ON TABLE registration_risk_events TO pos_app_rls;
        GRANT INSERT, SELECT, UPDATE ON TABLE sessions TO pos_app_rls;
    END IF;
END
$$;

-- Notifications inbox + manager web-push subscriptions (migration 040).
-- Tenant-RLS rows written/read by the notifications module; the runtime role
-- needs SELECT/INSERT on the inbox and full DML on push subs (subscribe upserts,
-- a dead endpoint triggers DELETE during a push sweep).
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app_rls') THEN
        GRANT SELECT, INSERT, UPDATE ON TABLE notifications TO pos_app_rls;
        GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE notification_push_sub TO pos_app_rls;
    END IF;
END
$$;

-- Maintenance permissions stay OUTSIDE this script by design:
--   * schema migrations / create table     -> owner or a dedicated migrator role
--   * pg_dump backups                      -> run as owner (scripts/backup.sh)
--   * saas_admin cross-tenant aggregation  -> same app role, RLS-set loops

REVOKE CREATE ON SCHEMA public FROM pos_app;
REVOKE ALL PRIVILEGES ON DATABASE pos FROM PUBLIC;

-- pos_app_rls is the least-privilege runtime role; it still needs CONNECT,
-- which the REVOKE-from-PUBLIC above removes (the database ACL is non-empty,
-- so PUBLIC's default CONNECT no longer applies). Without this the app works
-- only until its connection pool recycles.
DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app_rls') THEN
        GRANT CONNECT ON DATABASE pos TO pos_app_rls;
    END IF;
END
$$;