-- +goose Up
-- Self-service password reset (email link).
--
-- login is per (tenant, email, password): a `users` row belongs to exactly one
-- tenant, so a reset request names the store the same way POST /v1/auth/login
-- does. Binding the token to tenant_id+user_id (instead of a bare email) keeps
-- every later statement inside one tenant context — RLS then applies normally,
-- and there is no need for a cross-tenant email scan, which would be both a
-- cost amplification and an enumeration channel on an unauthenticated route.
--
-- The table itself is PLATFORM-level (no RLS), like `accounts` and
-- `email_verification_tokens`: the mint/consume lookup is keyed by a random
-- token hash and has to find the row *before* any tenant context exists. It
-- stores only the sha256 digest — the plain token is handed to the mailer once
-- and can never be read back.
CREATE TABLE password_reset_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX password_reset_tokens_user_idx ON password_reset_tokens (user_id, expires_at DESC);
CREATE INDEX password_reset_tokens_expiry_idx ON password_reset_tokens (expires_at);

-- +goose Down
DROP TABLE IF EXISTS password_reset_tokens;
