-- +goose Up
-- OCR usage metering + credits. Scans of supplier-invoice images are metered
-- per tenant over three rolling windows (day / week / month) — the "limit per
-- plan, per period" the merchant contract asks for — and a scan-borrowing
-- credit balance (`tenants.ocr_credits_remaining`) lets a shop top up points
-- when a window is nearly exhausted instead of waiting for the window to roll.
--
-- A scan is ALWAYS admitted unless it would push the tenant past ANY active
-- window (window limit reached AND credits exhausted); the enforcement decision
-- is computed in Go (internal/transport/purchases/metering.go) from this table
-- plus the credit column, purely and unit-tested. The table itself is only a
-- counter ledger, so it has no money columns.

CREATE TABLE ocr_usage (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    window_kind  TEXT NOT NULL CHECK (window_kind IN ('day', 'week', 'month')),
    window_start DATE NOT NULL,
    scans_used   BIGINT NOT NULL DEFAULT 0 CHECK (scans_used >= 0),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, id)
);

-- exactly one ledger row per tenant+window: upserts bump scans_used.
CREATE UNIQUE INDEX ocr_usage_tenant_window_uq ON ocr_usage (tenant_id, window_kind);
CREATE INDEX ocr_usage_tenant_idx ON ocr_usage (tenant_id, window_kind, window_start);

ALTER TABLE tenants ADD COLUMN ocr_credits_remaining BIGINT NOT NULL DEFAULT 0
    CHECK (ocr_credits_remaining >= 0);

-- Credits are additive bookkeeping; every OCR scan that borrows one is
-- decremented by the same handler that bumped the window, in one transaction.

ALTER TABLE ocr_usage ENABLE ROW LEVEL SECURITY;
ALTER TABLE ocr_usage FORCE ROW LEVEL SECURITY;
CREATE POLICY ocr_usage_tenant_policy ON ocr_usage
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP POLICY IF EXISTS ocr_usage_tenant_policy ON ocr_usage;
ALTER TABLE ocr_usage DISABLE ROW LEVEL SECURITY;
ALTER TABLE tenants DROP COLUMN IF EXISTS ocr_credits_remaining;
DROP TABLE IF EXISTS ocr_usage;