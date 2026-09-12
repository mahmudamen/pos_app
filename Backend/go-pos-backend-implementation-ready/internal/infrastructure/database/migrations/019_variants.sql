-- +goose Up
-- C3 Fashion/textile: size/color variants. A product with has_variants holds
-- the template; stock and price live on product_variants (parent
-- stock_quantity stays as a template value). Sales reference a variant,
-- sale_items records which one, and variant stock is decremented at checkout.

ALTER TABLE products ADD COLUMN has_variants BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE product_variants (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    product_id UUID NOT NULL REFERENCES products(id),
    name TEXT NOT NULL,
    size TEXT NOT NULL DEFAULT '',
    color TEXT NOT NULL DEFAULT '',
    sku TEXT NOT NULL,
    price_minor BIGINT NOT NULL CHECK (price_minor >= 0),
    stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, sku)
);
CREATE INDEX product_variants_tenant_idx ON product_variants (tenant_id);
CREATE INDEX product_variants_product_idx ON product_variants (product_id);

ALTER TABLE sale_items ADD COLUMN variant_id UUID;
ALTER TABLE sale_items ADD COLUMN variant_name TEXT;

ALTER TABLE product_variants ADD COLUMN change_seq BIGINT NOT NULL DEFAULT nextval('change_seq');
CREATE INDEX product_variants_change_seq_idx ON product_variants (tenant_id, change_seq);
CREATE TRIGGER product_variants_updated_at BEFORE UPDATE ON product_variants FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER product_variants_change_seq BEFORE UPDATE ON product_variants FOR EACH ROW EXECUTE FUNCTION bump_change_seq();

ALTER TABLE product_variants ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_variants FORCE ROW LEVEL SECURITY;
CREATE POLICY product_variants_tenant_policy ON product_variants
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP TABLE IF EXISTS product_variants;
ALTER TABLE sale_items DROP COLUMN IF EXISTS variant_id;
ALTER TABLE sale_items DROP COLUMN IF EXISTS variant_name;
ALTER TABLE products DROP COLUMN IF EXISTS has_variants;