-- +goose Up
-- Five more sellable verticals the consultant round asked for: auto-parts
-- shops, opticians, pet shops, flower shops and laundries. They run on the
-- same general POS core (no dedicated server logic), so this only widens the
-- tenant type vocabulary and unlocks their demo catalogs at signup.
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_business_type_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_business_type_check
    CHECK (business_type IN (
        'general', 'coffee_shop', 'restaurant', 'retail', 'book_store',
        'mobile_shop', 'computer_shop', 'grocery', 'bakery', 'shawerma',
        'falafel', 'pharmacy', 'butcher', 'fruits_veg', 'clothing', 'sweets',
        'jewelry', 'hardware', 'beauty', 'supermarket', 'auto_parts',
        'optics', 'pets', 'flowers', 'laundry'));

-- +goose Down
-- Restore the pre-059 CHECK (the 19 types).
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_business_type_check;
ALTER TABLE tenants ADD CONSTRAINT tenants_business_type_check
    CHECK (business_type IN (
        'general', 'coffee_shop', 'restaurant', 'retail', 'book_store',
        'mobile_shop', 'computer_shop', 'grocery', 'bakery', 'shawerma',
        'falafel', 'pharmacy', 'butcher', 'fruits_veg', 'clothing', 'sweets',
        'jewelry', 'hardware', 'beauty', 'supermarket'));
