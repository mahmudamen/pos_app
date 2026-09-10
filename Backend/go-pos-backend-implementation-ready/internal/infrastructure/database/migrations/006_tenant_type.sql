-- +goose Up
ALTER TABLE tenants ADD COLUMN business_type TEXT NOT NULL DEFAULT 'general';
ALTER TABLE tenants ADD CONSTRAINT tenants_business_type_check
    CHECK (business_type IN ('general', 'restaurant', 'book_store', 'mobile_shop', 'computer_shop', 'grocery', 'bakery', 'clothing'));

-- +goose Down
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_business_type_check;
ALTER TABLE tenants DROP COLUMN IF EXISTS business_type;