-- +goose Up
-- Register (POS) sessions — B2 "end-of-day close". A register session is one
-- cashier shift on one terminal: it carries the starting float and, once
-- closed, the Z-report figures (expected/actually-counted cash, difference).
-- There is at most one OPEN session per tenant+device at a time.

CREATE TABLE register_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id UUID NOT NULL,
    user_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    opening_cash_minor BIGINT NOT NULL DEFAULT 0 CHECK (opening_cash_minor >= 0),
    closing_cash_minor BIGINT CHECK (closing_cash_minor >= 0),
    expected_cash_minor BIGINT CHECK (expected_cash_minor >= 0),
    cash_difference_minor BIGINT,
    opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    FOREIGN KEY (tenant_id, device_id) REFERENCES devices(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX register_sessions_open_uq
    ON register_sessions (tenant_id, device_id) WHERE status = 'open';
CREATE INDEX register_sessions_device_idx
    ON register_sessions (tenant_id, device_id, opened_at DESC);

ALTER TABLE sales ADD COLUMN register_session_id UUID;
ALTER TABLE sales ADD CONSTRAINT sales_register_session_fk
    FOREIGN KEY (tenant_id, register_session_id)
    REFERENCES register_sessions(tenant_id, id) ON DELETE SET NULL;
CREATE INDEX sales_register_session_idx ON sales (tenant_id, register_session_id);

ALTER TABLE register_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE register_sessions FORCE ROW LEVEL SECURITY;
CREATE POLICY register_sessions_tenant_policy ON register_sessions
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP POLICY IF EXISTS register_sessions_tenant_policy ON register_sessions;
ALTER TABLE register_sessions DISABLE ROW LEVEL SECURITY;
ALTER TABLE sales DROP CONSTRAINT IF EXISTS sales_register_session_fk;
ALTER TABLE sales DROP COLUMN IF EXISTS register_session_id;
DROP TABLE IF EXISTS register_sessions;