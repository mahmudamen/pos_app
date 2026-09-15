-- +goose Up
-- ma_pos_base parity (Odoo POS Security Framework): per-user discount limits +
-- 5-tier access levels + granular operation permissions, and PIN lockout.

-- PIN brute-force protection on the manager PIN: track failures and lock the
-- credential for a cooldown window once the threshold is reached.
ALTER TABLE users ADD COLUMN pos_pin_failed_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN pos_pin_locked_until TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN pos_pin_last_validated_at TIMESTAMPTZ;

-- Per-user POS security profile (Odoo res.users) with RLS scoped to the tenant.
-- Access levels: none < cashier < advanced < manager < admin. When a user has a
-- row, the handler resolves effective flags from the level OR the per-row
-- custom overrides (use_custom_permissions). Users without a row fall back to
-- the role-derived level (owner->admin, manager->manager, cashier->cashier).
CREATE TABLE users_pos_security (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    access_level TEXT NOT NULL DEFAULT 'cashier'
        CHECK (access_level IN ('none', 'cashier', 'advanced', 'manager', 'admin')),
    max_discount_pct INTEGER
        CHECK (max_discount_pct IS NULL OR (max_discount_pct >= 0 AND max_discount_pct <= 100)),
    use_custom_permissions BOOLEAN NOT NULL DEFAULT FALSE,
    can_delete_order BOOLEAN NOT NULL DEFAULT FALSE,
    can_delete_line BOOLEAN NOT NULL DEFAULT FALSE,
    can_change_qty BOOLEAN NOT NULL DEFAULT FALSE,
    can_negative_qty BOOLEAN NOT NULL DEFAULT FALSE,
    can_price_change BOOLEAN NOT NULL DEFAULT FALSE,
    can_discount BOOLEAN NOT NULL DEFAULT FALSE,
    can_open_session BOOLEAN NOT NULL DEFAULT FALSE,
    can_close_session BOOLEAN NOT NULL DEFAULT FALSE,
    can_payment_modification BOOLEAN NOT NULL DEFAULT FALSE,
    can_refund BOOLEAN NOT NULL DEFAULT FALSE,
    can_negative_stock BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX users_pos_security_tenant_idx ON users_pos_security (tenant_id);
CREATE INDEX users_pos_security_user_idx ON users_pos_security (user_id);

ALTER TABLE users_pos_security ENABLE ROW LEVEL SECURITY;
ALTER TABLE users_pos_security FORCE ROW LEVEL SECURITY;
CREATE POLICY users_pos_security_tenant_policy ON users_pos_security
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP TABLE IF EXISTS users_pos_security;
ALTER TABLE users DROP COLUMN IF EXISTS pos_pin_locked_until;
ALTER TABLE users DROP COLUMN IF EXISTS pos_pin_failed_attempts;
ALTER TABLE users DROP COLUMN IF EXISTS pos_pin_last_validated_at;