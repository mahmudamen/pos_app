-- +goose Up
-- Backorders / negative on-hand: when a tenant opts into
-- inventory.allow_negative_stock, checkout and inventory adjustments may
-- drive the on-hand counters below zero. Drop the >= 0 CHECK constraints on
-- products and product_variants (lot quantities stay physically >= 0 and keep
-- their own constraint, since FEFO consumption needs real lots).
ALTER TABLE products DROP CONSTRAINT IF EXISTS products_stock_quantity_check;
ALTER TABLE product_variants DROP CONSTRAINT IF EXISTS product_variants_stock_quantity_check;

-- +goose Down
ALTER TABLE products ADD CONSTRAINT products_stock_quantity_check CHECK (stock_quantity >= 0);
ALTER TABLE product_variants ADD CONSTRAINT product_variants_stock_quantity_check CHECK (stock_quantity >= 0);