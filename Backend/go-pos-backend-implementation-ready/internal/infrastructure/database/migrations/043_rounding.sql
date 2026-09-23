-- +goose Up
-- Cash rounding (Egypt-market POS): when the tenant enables pos.rounding_mode
-- (25/50/100 minor units per EGP), each sale stores the delta added between the
-- nominal total and the payable cash amount. The payable payments must sum to
-- is total_minor + rounding_minor and is never below the nominal total, so the
-- column is non-negative (rounding is a ceiling/nearest-half-up adjustment).
ALTER TABLE sales ADD COLUMN rounding_minor BIGINT NOT NULL DEFAULT 0;
ALTER TABLE sales ADD CONSTRAINT sales_rounding_non_negative CHECK (rounding_minor >= 0);

-- +goose Down
ALTER TABLE sales DROP CONSTRAINT IF EXISTS sales_rounding_non_negative;
ALTER TABLE sales DROP COLUMN IF EXISTS rounding_minor;