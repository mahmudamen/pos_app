-- +goose Up
-- Self-ordering ("order from the table / online menu"). A cashier publishes a
-- QR that only carries a URL (https://api.xamltech.com/selforder?tenant=<slug>).
-- Customers open it in any browser, see ONLY the store's published products
-- (products.selforder_enabled) that are currently in stock (never shown when
-- stock_quantity = 0), and place a request. The request lands here as a
-- pending self_order the cashier approves (turns it into a real sale via the
-- existing sales.CreateSale path, this is where stock actually moves) or
-- cancels.

-- products gain an opt-in "publish for self-order / online menu" flag. When it
-- is off, or stock is 0, the product never appears on the public self-order
-- menu and cannot be purchased through it.
ALTER TABLE products ADD COLUMN selforder_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE self_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    reference TEXT NOT NULL DEFAULT '',
    customer_name TEXT NOT NULL DEFAULT '',
    table_name TEXT NOT NULL DEFAULT '',
    items JSONB NOT NULL DEFAULT '[]'::jsonb,
    subtotal_minor BIGINT NOT NULL DEFAULT 0 CHECK (subtotal_minor >= 0),
    total_minor BIGINT NOT NULL DEFAULT 0 CHECK (total_minor >= 0),
    currency CHAR(3) NOT NULL DEFAULT 'EGP',
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'cancelled')),
    approved_sale_id UUID,
    processed_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (tenant_id, approved_sale_id) REFERENCES sales(tenant_id, id) ON DELETE SET NULL,
    FOREIGN KEY (tenant_id, processed_by) REFERENCES users(tenant_id, id) ON DELETE SET NULL,
    UNIQUE (tenant_id, id)
);
CREATE INDEX self_orders_tenant_status_idx ON self_orders (tenant_id, status, created_at DESC);

-- Customer feedback / "I want this product" requests from the self-order web
-- page. A visitor may attach a name note and opt into browser notifications
-- (web push subscription JSON: endpoint + keys{ p256dh, auth }) so the merchant
-- can notify them the moment a product they asked about is back in stock and
-- published online again. status: open -> fulfilled | closed.
CREATE TABLE product_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    product_id UUID,
    product_name TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    contact TEXT NOT NULL DEFAULT '',
    webpush JSONB,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'fulfilled', 'closed')),
    notified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (tenant_id, product_id) REFERENCES products(tenant_id, id) ON DELETE CASCADE,
    UNIQUE (tenant_id, id)
);
CREATE INDEX product_requests_tenant_status_idx ON product_requests (tenant_id, status, created_at DESC);

ALTER TABLE self_orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE self_orders FORCE ROW LEVEL SECURITY;
CREATE POLICY self_orders_tenant_policy ON self_orders
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE product_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE product_requests FORCE ROW LEVEL SECURITY;
CREATE POLICY product_requests_tenant_policy ON product_requests
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP POLICY IF EXISTS product_requests_tenant_policy ON product_requests;
ALTER TABLE product_requests DISABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS self_orders_tenant_policy ON self_orders;
ALTER TABLE self_orders DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS product_requests;
DROP TABLE IF EXISTS self_orders;
ALTER TABLE products DROP COLUMN IF EXISTS selforder_enabled;