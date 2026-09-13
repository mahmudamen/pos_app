-- +goose Up
-- +goose StatementBegin
ALTER TABLE tenants ADD COLUMN address text NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenants DROP COLUMN address;
-- +goose StatementEnd