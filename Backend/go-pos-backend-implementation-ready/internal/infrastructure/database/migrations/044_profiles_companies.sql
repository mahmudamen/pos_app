-- +goose Up
-- Community (slice 1): staff professional profiles + companies (POS COPILOT
-- platform workstream). See docs/23_COMMUNITY_JOBS_PROFILES.md.
-- Profiles are a 1:1 side table on users (keeps auth/sync/session tables
-- untouched); companies are the store's employers; company_members is the
-- many-to-many staff holding (role + display title). All FORCE RLS + tenant
-- policy like every other tenant table. NOTE: excluded from the sync change
-- feed (awareness/interaction data, not register state).

-- 1:1 optional staff profile.
CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    headline TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    years_experience SMALLINT NOT NULL DEFAULT 0 CHECK (years_experience >= 0),
    is_chief BOOLEAN NOT NULL DEFAULT FALSE,
    skills TEXT[] NOT NULL DEFAULT '{}',
    resume JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id)
);

CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    industry TEXT NOT NULL DEFAULT '',
    website TEXT NOT NULL DEFAULT '',
    logo_url TEXT NOT NULL DEFAULT '',
    city TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, id),
    UNIQUE (tenant_id, name),
    CONSTRAINT companies_name_required CHECK (length(btrim(name)) > 0)
);

CREATE TABLE company_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'staff',
    title TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, company_id, user_id),
    CONSTRAINT company_members_role_check CHECK (role IN ('owner', 'manager', 'staff'))
);

ALTER TABLE user_profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_profiles FORCE ROW LEVEL SECURITY;
CREATE POLICY user_profiles_tenant_policy ON user_profiles
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE companies ENABLE ROW LEVEL SECURITY;
ALTER TABLE companies FORCE ROW LEVEL SECURITY;
CREATE POLICY companies_tenant_policy ON companies
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE company_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE company_members FORCE ROW LEVEL SECURITY;
CREATE POLICY company_members_tenant_policy ON company_members
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

CREATE INDEX companies_tenant_active_idx ON companies (tenant_id, is_active, name);
CREATE INDEX company_members_company_idx ON company_members (tenant_id, company_id);
CREATE INDEX company_members_user_idx ON company_members (tenant_id, user_id);

-- +goose Down
DROP POLICY IF EXISTS company_members_tenant_policy ON company_members;
DROP TABLE IF EXISTS company_members;
DROP POLICY IF EXISTS companies_tenant_policy ON companies;
DROP TABLE IF EXISTS companies;
DROP POLICY IF EXISTS user_profiles_tenant_policy ON user_profiles;
DROP TABLE IF EXISTS user_profiles;