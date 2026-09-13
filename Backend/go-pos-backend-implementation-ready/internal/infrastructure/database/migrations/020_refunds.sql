-- +goose Up
-- Refunds (B1 / T037 "US4"): reverse a completed sale end-to-end. A refund
-- restores stock to the exact lots/variants/products the sale consumed,
-- releases the restaurant table, claws back the loyalty points the sale earned,
-- and marks the sale status 'refunded'. The sale_refunds ledger keeps every
-- refund auditable and its (tenant_id, idempotency_key) uniqueness makes
-- retried POSTs replay-safe, exactly like sales.

ALTER TABLE sales DROP CONSTRAINT sales_status_check;
ALTER TABLE sales ADD CONSTRAINT sales_status_check
    CHECK (status IN ('draft', 'completed', 'voided', 'refunded'));

CREATE TABLE sale_refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sale_id UUID NOT NULL,
    created_by UUID,
    idempotency_key TEXT NOT NULL,
    refund_minor BIGINT NOT NULL CHECK (refund_minor > 0),
    reason TEXT CHECK (reason IS NULL OR char_length(reason) <= 255),
    status TEXT NOT NULL DEFAULT 'completed' CHECK (status IN ('pending', 'completed', 'failed')),
    change_seq BIGINT NOT NULL DEFAULT nextval('change_seq'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (tenant_id, sale_id) REFERENCES sales(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, created_by) REFERENCES users(tenant_id, id) ON DELETE SET NULL,
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE INDEX sale_refunds_sale_idx ON sale_refunds (tenant_id, sale_id);
CREATE INDEX sale_refunds_change_seq_idx ON sale_refunds (tenant_id, change_seq);
CREATE TRIGGER sale_refunds_change_seq
    BEFORE UPDATE ON sale_refunds FOR EACH ROW EXECUTE FUNCTION bump_change_seq();

ALTER TABLE sale_refunds ENABLE ROW LEVEL SECURITY;
ALTER TABLE sale_refunds FORCE ROW LEVEL SECURITY;
CREATE POLICY sale_refunds_tenant_policy ON sale_refunds
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- Claw-back must never push a running balance negative; the app floor at 0 and
-- this constraint is the database backstop.
ALTER TABLE customers ADD CONSTRAINT customers_loyalty_points_nonneg
    CHECK (loyalty_points >= 0);

-- +goose Down
ALTER TABLE customers DROP CONSTRAINT IF EXISTS customers_loyalty_points_nonneg;
DROP POLICY IF EXISTS sale_refunds_tenant_policy ON sale_refunds;
ALTER TABLE sale_refunds DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS sale_refunds;
ALTER TABLE sales DROP CONSTRAINT IF EXISTS sales_status_check;
ALTER TABLE sales ADD CONSTRAINT sales_status_check
    CHECK (status IN ('draft', 'completed', 'voided'));