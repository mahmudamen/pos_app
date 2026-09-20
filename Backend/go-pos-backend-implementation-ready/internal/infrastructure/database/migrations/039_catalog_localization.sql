-- +goose Up
-- Bilingual catalog: Egyptian merchants run Arabic-first UIs while keeping an
-- English master name. Categories carry an optional Arabic display name;
-- products carry an optional Arabic name and description. NULL means fall back
-- to the English-master fields on the client. New columns inherit the existing
-- FORCE RLS policies and table grants, so no ACL changes are required.

ALTER TABLE categories
    ADD COLUMN name_ar TEXT;

ALTER TABLE products
    ADD COLUMN name_ar TEXT,
    ADD COLUMN description_ar TEXT;

-- +goose Down
ALTER TABLE products
    DROP COLUMN IF EXISTS description_ar,
    DROP COLUMN IF EXISTS name_ar;
ALTER TABLE categories
    DROP COLUMN IF EXISTS name_ar;