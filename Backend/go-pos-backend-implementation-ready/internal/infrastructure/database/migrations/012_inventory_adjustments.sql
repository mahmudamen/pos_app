-- +goose Up
-- Inventory adjustments — A5 from the roadmap. A manager-visible change to
-- stock that is not a sale: damaged goods, restock, or a physical stock count
-- correction. Inventory always moves through this deterministic service (the
-- AI/label-extraction layer, if added later, may only propose adjustments).

CREATE TABLE inventory_adjustments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID NOT NULL,
    reason TEXT NOT NULL CHECK (reason IN ('damaged', 'restock', 'count')),
    quantity_delta BIGINT NOT NULL CHECK (quantity_delta <> 0),
    note TEXT CHECK (note IS NULL OR char_length(note) <= 255),
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (tenant_id, product_id) REFERENCES products(tenant_id, id) ON DELETE CASCADE,
    FOREIGN KEY (tenant_id, created_by) REFERENCES users(tenant_id, id) ON DELETE SET NULL,
    UNIQUE (tenant_id, id)
);

CREATE INDEX inventory_adjustments_product_idx ON inventory_adjustments (tenant_id, product_id, created_at DESC);
CREATE INDEX inventory_adjustments_created_idx ON inventory_adjustments (tenant_id, created_at DESC);

ALTER TABLE inventory_adjustments ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_adjustments FORCE ROW LEVEL SECURITY;
CREATE POLICY inventory_adjustments_tenant_policy ON inventory_adjustments
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP POLICY IF EXISTS inventory_adjustments_tenant_policy ON inventory_adjustments;
ALTER TABLE inventory_adjustments DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS inventory_adjustments;