-- +goose Up
-- C1 Restaurant: floor plans + table status + split bills + tips.
-- sales gains an optional table_id (NULL for counter sales), a parent link
-- for split-bill children, and an aggregated tips_minor; sale_payments gains
-- tip_minor so a tender line can carry a gratuity on top of the exact sum.

ALTER TABLE sales ADD COLUMN table_id UUID;
ALTER TABLE sales ADD COLUMN parent_sale_id UUID;
ALTER TABLE sales ADD COLUMN tips_minor BIGINT NOT NULL DEFAULT 0;

CREATE TABLE floors (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);
CREATE INDEX floors_tenant_idx ON floors (tenant_id);

CREATE TABLE restaurant_tables (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    floor_id UUID NOT NULL REFERENCES floors(id),
    name TEXT NOT NULL,
    seats INTEGER NOT NULL DEFAULT 2 CHECK (seats > 0),
    status TEXT NOT NULL DEFAULT 'free' CHECK (status IN ('free', 'occupied', 'reserved', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, floor_id, name)
);
CREATE INDEX restaurant_tables_tenant_idx ON restaurant_tables (tenant_id);
CREATE INDEX restaurant_tables_floor_idx ON restaurant_tables (floor_id);

ALTER TABLE sale_payments ADD COLUMN tip_minor BIGINT NOT NULL DEFAULT 0;

ALTER TABLE floors ADD COLUMN change_seq BIGINT NOT NULL DEFAULT nextval('change_seq');
CREATE INDEX floors_change_seq_idx ON floors (tenant_id, change_seq);
CREATE TRIGGER floors_updated_at BEFORE UPDATE ON floors FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER floors_change_seq BEFORE UPDATE ON floors FOR EACH ROW EXECUTE FUNCTION bump_change_seq();

ALTER TABLE restaurant_tables ADD COLUMN change_seq BIGINT NOT NULL DEFAULT nextval('change_seq');
CREATE INDEX restaurant_tables_change_seq_idx ON restaurant_tables (tenant_id, change_seq);
CREATE TRIGGER restaurant_tables_updated_at BEFORE UPDATE ON restaurant_tables FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER restaurant_tables_change_seq BEFORE UPDATE ON restaurant_tables FOR EACH ROW EXECUTE FUNCTION bump_change_seq();

ALTER TABLE floors ENABLE ROW LEVEL SECURITY;
ALTER TABLE floors FORCE ROW LEVEL SECURITY;
CREATE POLICY floors_tenant_policy ON floors
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE restaurant_tables ENABLE ROW LEVEL SECURITY;
ALTER TABLE restaurant_tables FORCE ROW LEVEL SECURITY;
CREATE POLICY restaurant_tables_tenant_policy ON restaurant_tables
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP TABLE IF EXISTS restaurant_tables;
DROP TABLE IF EXISTS floors;
ALTER TABLE sale_payments DROP COLUMN IF EXISTS tip_minor;
ALTER TABLE sales DROP COLUMN IF EXISTS table_id;
ALTER TABLE sales DROP COLUMN IF EXISTS parent_sale_id;
ALTER TABLE sales DROP COLUMN IF EXISTS tips_minor;