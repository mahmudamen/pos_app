-- +goose Up
-- The public site lists "Beauty" and "Supermarkets" among the advertised
-- verticals, but validBusinessTypes and this CHECK did not accept them, so a
-- merchant following the landing promise (or the web signup page) could not
-- actually register that store type. Extend the CHECK to the two missing
-- types — signup now supports 19 verticals + general.
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_business_type_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_business_type_check
    CHECK (business_type IN (
        'general', 'coffee_shop', 'restaurant', 'retail', 'book_store',
        'mobile_shop', 'computer_shop', 'grocery', 'bakery', 'shawerma',
        'falafel', 'pharmacy', 'butcher', 'fruits_veg', 'clothing', 'sweets',
        'jewelry', 'hardware', 'beauty', 'supermarket'));

-- +goose Down
-- Restore the pre-058 CHECK (the 17 types).
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_business_type_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_business_type_check
    CHECK (business_type IN (
        'general', 'coffee_shop', 'restaurant', 'retail', 'book_store',
        'mobile_shop', 'computer_shop', 'grocery', 'bakery', 'shawerma',
        'falafel', 'pharmacy', 'butcher', 'fruits_veg', 'clothing', 'sweets',
        'jewelry', 'hardware'));
