-- +goose Up
-- Platform support inbox + tenant billing contact address.
--
-- support_messages is a PLATFORM-level table (no RLS), like `accounts` and
-- `audit_log`: the SaaS support console reads one cross-tenant queue, and an
-- inbound message from an unknown sender has no tenant context to resolve
-- before it is stored. tenant_id is an attribution column (set when the sender
-- address matches a known store owner), never an access boundary.
--
-- provider_ref is the IMAP UID (as text) so a poller replay cannot double
-- insert a message; the partial unique index scopes it to inbound rows in the
-- same mailbox. Outbound rows carry NULL.
CREATE TABLE support_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    thread_id UUID NOT NULL,
    thread_key TEXT NOT NULL DEFAULT '',
    direction TEXT NOT NULL CHECK (direction IN ('inbound', 'outbound')),
    mailbox TEXT NOT NULL DEFAULT 'support',
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    message_id TEXT,
    in_reply_to TEXT,
    from_email TEXT NOT NULL DEFAULT '',
    from_name TEXT NOT NULL DEFAULT '',
    to_email TEXT NOT NULL DEFAULT '',
    subject TEXT NOT NULL DEFAULT '',
    body_text TEXT NOT NULL DEFAULT '',
    body_html TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'archived')),
    provider_ref TEXT,
    error TEXT,
    received_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX support_messages_provider_uq
    ON support_messages (mailbox, provider_ref)
    WHERE provider_ref IS NOT NULL AND direction = 'inbound';
CREATE INDEX support_messages_thread_idx ON support_messages (thread_id, created_at);
CREATE INDEX support_messages_thread_key_idx ON support_messages (thread_key, created_at DESC);
CREATE INDEX support_messages_status_idx ON support_messages (status, created_at DESC);
CREATE INDEX support_messages_tenant_idx ON support_messages (tenant_id, created_at DESC);

-- IMAP resume cursor per mailbox. uid_validity detects a server-side UID reset
-- (mailbox rebuilt), which invalidates last_uid and forces a re-sync.
CREATE TABLE support_sync_state (
    mailbox TEXT PRIMARY KEY,
    uid_validity BIGINT NOT NULL DEFAULT 0,
    last_uid BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Billing/notification recipient for a tenant. Signup populates it from the
-- owner's email; this backfill covers stores created before the column existed.
ALTER TABLE tenants ADD COLUMN contact_email TEXT NOT NULL DEFAULT '';

-- Backfill the owner email for existing tenants. users is FORCE RLS, but
-- migrations run as the table owner (rolbypassrls), so the correlated read
-- sees every row on the target databases.
UPDATE tenants t SET contact_email = COALESCE((
    SELECT u.email FROM users u
    WHERE u.tenant_id = t.id AND u.role = 'owner'
    ORDER BY u.created_at
    LIMIT 1), '')
WHERE t.contact_email = '';

-- +goose Down
ALTER TABLE tenants DROP COLUMN IF EXISTS contact_email;
DROP TABLE IF EXISTS support_sync_state;
DROP TABLE IF EXISTS support_messages;
