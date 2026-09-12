-- +goose Up
-- C2 Pharmacy: expiry-date + lot/serial tracking. Products opt in with
-- track_lots (lot ledger is authoritative for stock on this product) and/or
-- expiry_required (a sale line must be satisfied from a lot that has an
-- expiry_date). Lots are consumed first-expiry-first-out at checkout and each
-- consumed lot is recorded on sale_items for recall.

ALTER TABLE products ADD COLUMN track_lots BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE products ADD COLUMN expiry_required BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE product_lots (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    product_id UUID NOT NULL REFERENCES products(id),
    lot_number TEXT NOT NULL,
    expiry_date DATE,
    quantity INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, product_id, lot_number)
);
CREATE INDEX product_lots_tenant_idx ON product_lots (tenant_id);
CREATE INDEX product_lots_product_expiry_idx ON product_lots (product_id, expiry_date);

ALTER TABLE sale_items ADD COLUMN lot_id UUID;
ALTER TABLE sale_items ADD COLUMN lot_number TEXT;

ALTER TABLE product_lots ADD COLUMN change_seq BIGINT NOT NULL DEFAULT nextval('change_seq');
CREATE INDEX product_lots_change_seq_idx ON product_lots (tenant_id, change_seq);
CREATE TRIGGER product_lots_updated_at BEFORE UPDATE ON product_lots FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER product_lots_change_seq BEFORE UPDATE ON product_lots FOR EACH ROW EXECUTE FUNCTION bump_change_seq();

ALTER TABLE product_lots ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_lots FORCE ROW LEVEL SECURITY;
CREATE POLICY product_lots_tenant_policy ON product_lots
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP TABLE IF EXISTS product_lots;
ALTER TABLE sale_items DROP COLUMN IF EXISTS lot_id;
ALTER TABLE sale_items DROP COLUMN IF EXISTS lot_number;
ALTER TABLE products DROP COLUMN IF EXISTS track_lots;
ALTER TABLE products DROP COLUMN IF EXISTS expiry_required;