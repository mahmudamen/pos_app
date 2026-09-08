-- Run as a PostgreSQL administrator once on a developer machine.
-- Example: psql -U postgres -d postgres -f scripts/bootstrap_local.sql
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'pos_app') THEN
        CREATE ROLE pos_app LOGIN PASSWORD 'pos_app_dev_password';
    ELSE
        ALTER ROLE pos_app WITH LOGIN PASSWORD 'pos_app_dev_password';
    END IF;
END
$$;

SELECT 'CREATE DATABASE pos OWNER pos_app'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'pos')\gexec

GRANT ALL PRIVILEGES ON DATABASE pos TO pos_app;