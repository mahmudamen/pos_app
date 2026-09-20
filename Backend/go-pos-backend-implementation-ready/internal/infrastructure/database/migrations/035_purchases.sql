-- +goose Up
-- Purchase ledger for supplier invoices. A purchase batches the OCR/manual
-- interpretation of one invoice: the header (supplier, invoice number,
-- currency, money totals) plus one row per bought product. Applying a purchase
-- is the deterministic inventory/price write the OCR layer only *proposes*:
--   * products.stock_quantity  += quantity (converted to the product unit)
--   * products.cost_minor       = invoice unit price (last-purchase cost)
--   * products.unit             = invoice unit when it differs and is known
-- Every products UPDATE is watched by the sync bump trigger, so other devices
-- pick up the new stock/cost through GET /v1/sync/pull automatically.
--
-- duplicate invoices (same tenant + non-empty invoice_no) are blocked by the
-- partial unique index, which makes double-submitted OCR applies safe.

CREATE TABLE purchases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    supplier TEXT NOT NULL DEFAULT '',
    invoice_no TEXT NOT NULL DEFAULT '',
    currency CHAR(3) NOT NULL DEFAULT 'EGP',
    subtotal_minor BIGINT NOT NULL DEFAULT 0 CHECK (subtotal_minor >= 0),
    tax_minor BIGINT NOT NULL DEFAULT 0 CHECK (tax_minor >= 0),
    total_minor BIGINT NOT NULL DEFAULT 0 CHECK (total_minor >= 0),
    ocr_text TEXT NOT NULL DEFAULT '',
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (tenant_id, created_by) REFERENCES users(tenant_id, id) ON DELETE SET NULL,
    UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX purchases_invoice_no_uq ON purchases (tenant_id, invoice_no)
    WHERE invoice_no <> '';
CREATE INDEX purchases_created_idx ON purchases (tenant_id, created_at DESC);

CREATE TABLE purchase_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    purchase_id UUID NOT NULL,
    product_id UUID,
    product_name TEXT NOT NULL,
    quantity NUMERIC(18,3) NOT NULL CHECK (quantity > 0),
    unit TEXT NOT NULL DEFAULT 'piece',
    unit_price_minor BIGINT NOT NULL CHECK (unit_price_minor >= 0),
    total_minor BIGINT NOT NULL CHECK (total_minor >= 0),
    match_score INTEGER NOT NULL DEFAULT 0,
    row_text TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (tenant_id, purchase_id) REFERENCES purchases(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, product_id) REFERENCES products(tenant_id, id) ON DELETE SET NULL,
    UNIQUE (tenant_id, id)
);

CREATE INDEX purchase_items_purchase_idx ON purchase_items (tenant_id, purchase_id);

ALTER TABLE purchases ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchases FORCE ROW LEVEL SECURITY;
CREATE POLICY purchases_tenant_policy ON purchases
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE purchase_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE purchase_items FORCE ROW LEVEL SECURITY;
CREATE POLICY purchase_items_tenant_policy ON purchase_items
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP POLICY IF EXISTS purchase_items_tenant_policy ON purchase_items;
ALTER TABLE purchase_items DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS purchases_tenant_policy ON purchases;
ALTER TABLE purchases DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS purchase_items;
DROP TABLE IF EXISTS purchases;