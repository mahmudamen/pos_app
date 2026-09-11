-- +goose Up
-- Sync (B3) — make the remaining client-facing tables syncable through the
-- monotonic change sequence. categories/products/customers/sales already carry
-- change_seq + the bump trigger; register_sessions, inventory_adjustments and
-- tenant_settings are the client-facing additions the POS app caches.

ALTER TABLE register_sessions
    ADD COLUMN change_seq BIGINT NOT NULL DEFAULT nextval('change_seq');
CREATE INDEX register_sessions_change_seq_idx
    ON register_sessions (tenant_id, change_seq);
CREATE TRIGGER register_sessions_change_seq
    BEFORE UPDATE ON register_sessions FOR EACH ROW EXECUTE FUNCTION bump_change_seq();

ALTER TABLE inventory_adjustments
    ADD COLUMN change_seq BIGINT NOT NULL DEFAULT nextval('change_seq');
CREATE INDEX inventory_adjustments_change_seq_idx
    ON inventory_adjustments (tenant_id, change_seq);

ALTER TABLE tenant_settings
    ADD COLUMN change_seq BIGINT NOT NULL DEFAULT nextval('change_seq');
CREATE INDEX tenant_settings_change_seq_idx
    ON tenant_settings (tenant_id, change_seq);
CREATE TRIGGER tenant_settings_change_seq
    BEFORE UPDATE ON tenant_settings FOR EACH ROW EXECUTE FUNCTION bump_change_seq();

-- -- +goose Down
DROP TRIGGER IF EXISTS register_sessions_change_seq ON register_sessions;
DROP TRIGGER IF EXISTS tenant_settings_change_seq ON tenant_settings;
DROP INDEX IF EXISTS register_sessions_change_seq_idx;
DROP INDEX IF EXISTS inventory_adjustments_change_seq_idx;
DROP INDEX IF EXISTS tenant_settings_change_seq_idx;

ALTER TABLE register_sessions DROP COLUMN IF EXISTS change_seq;
ALTER TABLE inventory_adjustments DROP COLUMN IF EXISTS change_seq;
ALTER TABLE tenant_settings DROP COLUMN IF EXISTS change_seq;