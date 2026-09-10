-- +goose Up
-- Payments: split tender per sale (Phase A — A4). A sale may be paid by one
-- or several methods whose amounts sum to total_minor. sales.payment_method
-- is the primary (largest) line, denormalized so list queries stay cheap.

ALTER TABLE sales ADD COLUMN payment_method TEXT;
ALTER TABLE sales ADD CONSTRAINT sales_payment_method_check
    CHECK (payment_method IN ('cash', 'card', 'mobile'));

CREATE TABLE sale_payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sale_id UUID NOT NULL,
    method TEXT NOT NULL CHECK (method IN ('cash', 'card', 'mobile')),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (tenant_id, sale_id) REFERENCES sales(tenant_id, id) ON DELETE CASCADE,
    UNIQUE (tenant_id, id)
);

CREATE INDEX sale_payments_sale_idx ON sale_payments (tenant_id, sale_id);

ALTER TABLE sale_payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE sale_payments FORCE ROW LEVEL SECURITY;
CREATE POLICY sale_payments_tenant_policy ON sale_payments
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP POLICY IF EXISTS sale_payments_tenant_policy ON sale_payments;
ALTER TABLE sale_payments DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS sale_payments;
ALTER TABLE sales DROP CONSTRAINT IF EXISTS sales_payment_method_check;
ALTER TABLE sales DROP COLUMN IF EXISTS payment_method;