-- +goose Up
-- Expand tenants.business_type to all 17 onboarding verticals. Migrations
-- 006/025 capped it at the 10 legacy types, but validBusinessTypes in
-- register.go already accepts 17, so signup for shawerma/falafel/pharmacy/
-- butcher/fruits_veg/sweets/jewelry/hardware failed with a lagging CHECK
-- (insert rejected -> 500 unable to create tenant).
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_business_type_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_business_type_check
    CHECK (business_type IN (
        'general', 'coffee_shop', 'restaurant', 'retail', 'book_store',
        'mobile_shop', 'computer_shop', 'grocery', 'bakery', 'shawerma',
        'falafel', 'pharmacy', 'butcher', 'fruits_veg', 'clothing', 'sweets',
        'jewelry', 'hardware'));

-- +goose Down
-- Restore the pre-031 CHECK (the 10 legacy types) for rollbacks.
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_business_type_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_business_type_check
    CHECK (business_type IN (
        'general', 'coffee_shop', 'restaurant', 'book_store', 'mobile_shop',
        'computer_shop', 'grocery', 'retail', 'bakery', 'clothing'));