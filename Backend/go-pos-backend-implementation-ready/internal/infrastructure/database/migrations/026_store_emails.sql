-- +goose Up
-- store_emails is the cross-tenant email uniqueness table used by merchant
-- signup (POST /v1/auth/register). Some databases recorded an earlier 025
-- payload that omitted it; IF NOT EXISTS makes this safe on both fresh and
-- migrated databases.
CREATE TABLE IF NOT EXISTS store_emails (
    email TEXT PRIMARY KEY
);

-- +goose Down
DROP TABLE IF EXISTS store_emails;