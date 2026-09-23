-- +goose Up
-- Community (slice 2): job board (POS COPILOT platform workstream). See
-- docs/23_COMMUNITY_JOBS_PROFILES.md. job_offers are the store's postings
-- (optionally tied to a company); job_applications are staff applications that
-- snapshot the applicant's resume at application time. All FORCE RLS + tenant
-- policy like every other tenant table. NOTE: excluded from the sync change
-- feed (awareness/interaction data, not register state).

CREATE TABLE job_offers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    company_id UUID REFERENCES companies(id) ON DELETE SET NULL,  -- optional: any/unknown employer
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    employment_type TEXT NOT NULL DEFAULT 'full_time', -- full_time|part_time|contract|freelance
    location TEXT NOT NULL DEFAULT '',
    salary_minor BIGINT NOT NULL DEFAULT 0 CHECK (salary_minor >= 0),
    salary_currency CHAR(3) NOT NULL DEFAULT 'EGP',
    skill_tags TEXT[] NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    closes_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, id),
    CONSTRAINT job_offers_title_required CHECK (length(btrim(title)) > 0),
    CONSTRAINT job_offers_type_check CHECK (employment_type IN ('full_time','part_time','contract','freelance'))
);

CREATE TABLE job_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    job_offer_id UUID NOT NULL REFERENCES job_offers(id) ON DELETE CASCADE,
    applicant_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cover_note TEXT NOT NULL DEFAULT '',
    resume JSONB NOT NULL DEFAULT '{}',            -- snapshot at application time
    status TEXT NOT NULL DEFAULT 'applied',        -- applied|under_review|accepted|rejected
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, job_offer_id, applicant_id),  -- one application per person per job
    CONSTRAINT app_status_check CHECK (status IN ('applied','under_review','accepted','rejected'))
);

ALTER TABLE job_offers ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_offers FORCE ROW LEVEL SECURITY;
CREATE POLICY job_offers_tenant_policy ON job_offers
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

ALTER TABLE job_applications ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_applications FORCE ROW LEVEL SECURITY;
CREATE POLICY job_applications_tenant_policy ON job_applications
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID);

CREATE INDEX job_offers_tenant_active_idx ON job_offers (tenant_id, is_active, created_at DESC);
CREATE INDEX job_applications_job_idx ON job_applications (tenant_id, job_offer_id);
CREATE INDEX job_applications_applicant_idx ON job_applications (tenant_id, applicant_id);

-- +goose Down
DROP POLICY IF EXISTS job_applications_tenant_policy ON job_applications;
DROP TABLE IF EXISTS job_applications;
DROP POLICY IF EXISTS job_offers_tenant_policy ON job_offers;
DROP TABLE IF EXISTS job_offers;