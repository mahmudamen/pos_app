-- +goose Up
-- Loyalty (C4): customers earn points on sales; a tenant-scoped ledger records
-- every points movement so balances stay auditable and reversals are possible.

ALTER TABLE customers
    ADD COLUMN loyalty_points BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN loyalty_points_total BIGINT NOT NULL DEFAULT 0;

CREATE TABLE customer_loyalty_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    sale_id UUID REFERENCES sales(id),
    points_delta BIGINT NOT NULL,
    reason TEXT NOT NULL,
    change_seq BIGINT NOT NULL DEFAULT nextval('change_seq'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX customer_loyalty_log_customer_idx
    ON customer_loyalty_log (customer_id, created_at DESC);
CREATE INDEX customer_loyalty_log_sale_idx
    ON customer_loyalty_log (sale_id);

ALTER TABLE customer_loyalty_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_loyalty_log FORCE ROW LEVEL SECURITY;
CREATE POLICY customer_loyalty_log_tenant_policy ON customer_loyalty_log
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose StatementBegin
DO $$
DECLARE
    tid UUID;
BEGIN
    FOR tid IN SELECT id FROM tenants LOOP
        PERFORM set_config('app.current_tenant', tid::text, true);
        INSERT INTO tenant_settings (tenant_id, key, value)
        VALUES (tid, 'loyalty.points_per_100', '1')
        ON CONFLICT (tenant_id, key) DO NOTHING;
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DELETE FROM tenant_settings WHERE key = 'loyalty.points_per_100';
DROP POLICY IF EXISTS customer_loyalty_log_tenant_policy ON customer_loyalty_log;
ALTER TABLE customer_loyalty_log DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS customer_loyalty_log;
ALTER TABLE customers
    DROP COLUMN IF EXISTS loyalty_points,
    DROP COLUMN IF EXISTS loyalty_points_total;