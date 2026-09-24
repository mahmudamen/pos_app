-- +goose Up
-- Egypt National Community (slice A): national membership + invitation codes
-- (POS COPILOT network). See docs/24_NATIONAL_COMMUNITY.md.
--
-- Deliberately NOT tenant-RLS'd (like the platform/saas tables): the community
-- is cross-tenant (Egypt-wide). Access is gated in the handlers by authentication
-- + membership + RBAC, never by row-level tenant policy. origin_tenant_id is
-- attribution only and must never carry tenant-confidential data. Excluded from
-- the sync change feed (awareness/interaction data).

-- One row per user who joined the Egypt national community.
CREATE TABLE community_members (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    origin_tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,  -- owning store (attribution only)
    display_name TEXT NOT NULL DEFAULT '',           -- public name; copied from users.display_name at join
    joined_via TEXT NOT NULL DEFAULT 'invitation',   -- invitation|open
    invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    role TEXT NOT NULL DEFAULT 'member',             -- member|moderator|admin
    status TEXT NOT NULL DEFAULT 'active',           -- active|muted|suspended (effective, cached)
    expertise_score INT NOT NULL DEFAULT 0 CHECK (expertise_score >= 0),
    level TEXT NOT NULL DEFAULT 'bronze',            -- bronze|silver|gold|platinum (derived from score)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,         -- soft-delete / self-leave
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT community_members_role_check   CHECK (role   IN ('member','moderator','admin')),
    CONSTRAINT community_members_status_check CHECK (status IN ('active','muted','suspended')),
    CONSTRAINT community_members_level_check  CHECK (level  IN ('bronze','silver','gold','platinum'))
);

CREATE INDEX community_members_directory_idx ON community_members (is_active, status, level, display_name);
CREATE INDEX community_members_tenant_idx    ON community_members (origin_tenant_id) WHERE origin_tenant_id IS NOT NULL;
CREATE INDEX community_members_invited_idx   ON community_members (invited_by) WHERE invited_by IS NOT NULL;

-- Single-use-by-default invitation codes. Only the SHA-256 of the code is
-- stored; the plaintext EG-XXXX… code is shown once to the inviter at creation.
CREATE TABLE community_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code_hash TEXT NOT NULL UNIQUE,                  -- SHA-256 of the canonical code
    inviter_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    origin_tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    email TEXT,                                      -- optional: pre-target a staff member
    note TEXT NOT NULL DEFAULT '',
    max_uses INT NOT NULL DEFAULT 1 CHECK (max_uses > 0),
    used_count INT NOT NULL DEFAULT 0 CHECK (used_count <= max_uses),
    status TEXT NOT NULL DEFAULT 'active',           -- active|revoked|expired
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT community_invitations_status_check CHECK (status IN ('active','revoked','expired'))
);

CREATE INDEX community_invitations_inviter_idx ON community_invitations (inviter_id, created_at DESC);
CREATE INDEX community_invitations_tenant_idx  ON community_invitations (origin_tenant_id) WHERE origin_tenant_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS community_invitations;
DROP TABLE IF EXISTS community_members;