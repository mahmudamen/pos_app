-- +goose Up
-- Merchant self-signup: on-board trial stores without approval.
--  * tenants.business_type gains the coffee_shop vertical.
--  * tenants.trial_ends_at marks a 15-day trial store (NULL = not on trial).
--  * products.description holds the merchant-facing product info text.
--  * store_emails captures signup emails across tenants (RLS-free) so the
--    register handler can enforce global uniqueness in one transaction.

ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_business_type_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_business_type_check
    CHECK (business_type IN ('general', 'coffee_shop', 'restaurant', 'book_store', 'mobile_shop', 'computer_shop', 'grocery', 'retail', 'bakery', 'clothing'));
ALTER TABLE tenants ADD COLUMN trial_ends_at TIMESTAMPTZ;
ALTER TABLE products ADD COLUMN description TEXT NOT NULL DEFAULT '';
CREATE TABLE IF NOT EXISTS store_emails (
    email TEXT PRIMARY KEY
);

-- +goose Down
ALTER TABLE tenants DROP COLUMN IF EXISTS trial_ends_at;
ALTER TABLE products DROP COLUMN IF EXISTS description;
DROP TABLE IF EXISTS store_emails;
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_business_type_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_business_type_check
    CHECK (business_type IN ('general', 'restaurant', 'book_store', 'mobile_shop', 'computer_shop', 'grocery', 'bakery', 'clothing'));