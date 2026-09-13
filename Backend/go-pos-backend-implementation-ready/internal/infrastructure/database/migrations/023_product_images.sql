-- +goose Up
-- +goose StatementBegin
ALTER TABLE products ADD COLUMN image_url text NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE products DROP COLUMN IF EXISTS image_url;
-- +goose StatementEnd