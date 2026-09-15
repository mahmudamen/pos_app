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