-- +goose Up
-- B1 manager PIN: an optional bcrypt-hashed PIN on managers/owners. When a
-- user has one set, refund requests must echo it back, so a refund is always
-- an explicit credential-confirmed manager action — not just an HTTP call the
-- terminal can make on its own.

ALTER TABLE users ADD COLUMN manager_pin_hash TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS manager_pin_hash;