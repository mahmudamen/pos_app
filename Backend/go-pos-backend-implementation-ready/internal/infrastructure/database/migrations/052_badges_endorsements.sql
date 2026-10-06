-- +goose Up
-- Community (slice 3): badges + endorsements (POS COPILOT platform workstream).
-- See docs/23_COMMUNITY_JOBS_PROFILES.md. badge_definitions are per-tenant
-- recognitions; user_badges are awards handed to staff; skill_endorsements are
-- peer endorsements of a single skill (one per giver/receiver/skill, no self).
-- All FORCE RLS + tenant policy like every other tenant table. NOTE: excluded
-- from the sync change feed (awareness data, not register state) and from the
-- platform backup catalog (matching slices 1-2 community tables).

CREATE TABLE badge_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    code TEXT NOT NULL,                -- unique per tenant, kebab-case
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT 'military_tech',   -- material icon name
    criteria JSONB NOT NULL DEFAULT '{}',         -- free-form award rule description
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, code),
    CONSTRAINT badge_definitions_name_required CHECK (length(btrim(name)) > 0)
);

CREATE TABLE user_badges (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_id UUID NOT NULL REFERENCES badge_definitions(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    awarded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    note TEXT NOT NULL DEFAULT '',
    awarded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, badge_id)
);

CREATE TABLE skill_endorsements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    giver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    receiver_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill TEXT NOT NULL,               -- lowercase skill label
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, giver_id, receiver_id, skill),   -- one endorsement per (giver, receiver, skill)
    CONSTRAINT endorse_no_self CHECK (giver_id <> receiver_id)
);

ALTER TABLE badge_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE badge_definitions FORCE ROW LEVEL SECURITY;
CREATE POLICY badge_definitions_tenant_policy ON badge_definitions
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE user_badges ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_badges FORCE ROW LEVEL SECURITY;
CREATE POLICY user_badges_tenant_policy ON user_badges
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE skill_endorsements ENABLE ROW LEVEL SECURITY;
ALTER TABLE skill_endorsements FORCE ROW LEVEL SECURITY;
CREATE POLICY skill_endorsements_tenant_policy ON skill_endorsements
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

CREATE INDEX badge_definitions_tenant_idx ON badge_definitions (tenant_id, is_active, name);
CREATE INDEX user_badges_user_idx ON user_badges (tenant_id, user_id);
CREATE INDEX user_badges_badge_idx ON user_badges (tenant_id, badge_id);
CREATE INDEX skill_endorsements_receiver_idx ON skill_endorsements (tenant_id, receiver_id, skill);
CREATE INDEX skill_endorsements_giver_idx ON skill_endorsements (tenant_id, giver_id);

-- +goose Down
DROP POLICY IF EXISTS skill_endorsements_tenant_policy ON skill_endorsements;
DROP TABLE IF EXISTS skill_endorsements;
DROP POLICY IF EXISTS user_badges_tenant_policy ON user_badges;
DROP TABLE IF EXISTS user_badges;
DROP POLICY IF EXISTS badge_definitions_tenant_policy ON badge_definitions;
DROP TABLE IF EXISTS badge_definitions;
