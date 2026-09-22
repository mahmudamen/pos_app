-- +goose Up
-- Tenant notifications: in-app inbox for manager-level staff + optional RFC 8030
-- web-push delivery to subscribed browser endpoints. Emitted by business events
-- (low stock, OCR window limit reached, refund applied, new manager user) and
-- consumed via GET /v1/notifications. EXCLUDED from the sync change feed — like
-- client_events this is awareness data, not business state for registers to
-- replay. user_id NULL broadcasts to the whole tenant (managers filter by
-- read/severity); the in-app inbox reads rows regardless of role while push
-- delivery targets manager subscriptions only.

CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    type TEXT NOT NULL,
    key TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL DEFAULT 'info',
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}',
    read_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT notifications_severity_check CHECK (severity IN ('info', 'warning', 'critical')),
    CONSTRAINT notifications_type_check CHECK (type IN ('low_stock', 'ocr_limit', 'refund', 'user_created', 'info'))
);

CREATE INDEX notifications_tenant_created_idx ON notifications (tenant_id, created_at DESC);
CREATE INDEX notifications_tenant_read_idx ON notifications (tenant_id, read_at NULLS FIRST);
CREATE UNIQUE INDEX notifications_tenant_key_unique ON notifications (tenant_id, key) WHERE key <> '';

ALTER TABLE notifications ENABLE ROW LEVEL SECURITY;
ALTER TABLE notifications FORCE ROW LEVEL SECURITY;
CREATE POLICY notifications_tenant_policy ON notifications
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- Manager web-push subscriptions (RFC 8030) used to forward in-app notifications
-- to the browser when the manager is away from the console. One row per (user,
-- endpoint); endpoint JSONB mirrors the PushSubscription wire shape.
CREATE TABLE notification_push_sub (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    sub JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT notification_push_sub_unique UNIQUE (user_id, endpoint)
);

ALTER TABLE notification_push_sub ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_push_sub FORCE ROW LEVEL SECURITY;
CREATE POLICY notification_push_sub_tenant_policy ON notification_push_sub
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

-- +goose Down
DROP POLICY IF EXISTS notification_push_sub_tenant_policy ON notification_push_sub;
DROP TABLE IF EXISTS notification_push_sub;
DROP POLICY IF EXISTS notifications_tenant_policy ON notifications;
DROP TABLE IF EXISTS notifications;