# 23 — Community, Jobs & Profiles (POS COPILOT platform slice)

Design spec for the staff-facing community workstream on the POS.Go platform:
professional profiles (+ chief/chef flag), companies, skill endorsements,
badges, resumes, a job board, and an in-app forum.

Scope mirrors the `Pos_Go_copilot_Smart` feasibility study's platform goals,
but is deliberately **tenant-scoped** (per-store staff), not cross-tenant,
so it reuses the existing FORCE-RLS multi-tenancy, RBAC, and sync machinery.
Each slice ships backend + Flutter + tests together, one commit per slice.

> **National layer:** `docs/24_NATIONAL_COMMUNITY.md` extends this to a
> cross-store, Egypt-only network (invitation-gated membership, blog, shared job
> board, expertise score, strikes, platform badges). The forum (slice 4, migration
> `047`) becomes the national discussion space there.

| Slice | Backend | Flutter | Status |
|-------|---------|---------|--------|
| 1. Profiles + companies | migration `044`, module `internal/transport/community` (profiles.go, companies.go), RBAC `community.*` | `lib/core/community.dart`, `lib/features/community/`, hub entry from POS app bar | planned |
| 2. Job board | migration `045`, jobs.go + apply/approve | Jobs tab, apply sheet, manager approval | planned |
| 3. Badges + endorsements | migration `046`, badges.go + endorsements.go | Badges tab + profile chips | planned |
| 4. Forum | migration `047`, forum.go | Forum tab (categories, posts, comments, likes) | planned |

## Architecture decisions (decision log)

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-09-23 | Tenant-scoped staff feature, not cross-tenant marketplace | Product is one store + its staff. Reuses FORCE RLS + RBAC + hander patterns; a cross-tenant public marketplace would need a separate public-API surface and is out of scope for POS.Go v1. |
| 2026-09-23 | Profiles as separate `user_profiles` 1:1 table, not columns on `users` | `users` is touch-point-heavy (auth, sync, sessions, security). A 1:1 side table keeps blast radius small and makes the profile optional (only staff who opt in). |
| 2026-09-23 | Excluded from the sync pull feed (like `notifications`, `client_events`) | Community content is awareness/interaction data, never register-critical. Adding 6 more `change_seq` bump triggers buys nothing on devices. |
| 2026-09-23 | Jobs salaries as `salary_minor` + `salary_currency` (integer minor units) | Matches money convention everywhere (money is integer minor units). Salary is a single value + currency, not a band (bands are free text in description). |
| 2026-09-23 | Endorsements on **skills strings**, not badge definitions | An endorsement means "peer attests this skill label". Badges are store-issued awards. Two different primitives, two tables. |
| 2026-09-23 | Everything behind bearer auth `/v1/*` (no public routes in this workstream) | Self-order already covers the public surface. Staff forum/jobs/profiles are authenticated. |
| 2026-09-23 | Forum likes per-user (unique row), not counters | Enables "did I like this" state for the client; counts derived with `COUNT(*)` (small data, no materialized counters needed). |
| 2026-09-23 | Badges catalog is **tenant-defined**, not global Odoo-like | A store issues its own badges (their brand/culture). Seeded defaults optional. No plans/global-taxonomy coupling. |
| 2026-09-23 | `users`/`user_profiles` can belong to many companies | A chef at restaurant A also consults at B. `company_members` is many-to-many with a per-membership role/title. |

## Slice 1 — Profiles + companies

### Schema (migration `044_profiles_companies.sql`)

```sql
-- 1:1 optional staff profile. Every column on the profile is a candidate for
-- being shown on the POS.Go community feed / job applications.
CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    headline TEXT NOT NULL DEFAULT '',          -- e.g. "Head Chef · Cairo"
    bio TEXT NOT NULL DEFAULT '',
    avatar_url TEXT NOT NULL DEFAULT '',
    location TEXT NOT NULL DEFAULT '',
    years_experience SMALLINT NOT NULL DEFAULT 0 CHECK (years_experience >= 0),
    is_chief BOOLEAN NOT NULL DEFAULT FALSE,     -- "chief/chef" verified flag
    skills TEXT[] NOT NULL DEFAULT '{}',         -- lowercase skill labels
    resume JSONB NOT NULL DEFAULT '{}'           -- {experience:[{role,org,years}], education:[{school,degree,year}]}
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
    UNIQUE (tenant_id, name)
);
ALTER TABLE companies ADD CONSTRAINT companies_name_required CHECK (length(btrim(name)) > 0);

CREATE TABLE company_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'staff',          -- owner|manager|staff
    title TEXT NOT NULL DEFAULT '',              -- display title at that company
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, company_id, user_id),
    CONSTRAINT company_members_role_check CHECK (role IN ('owner','manager','staff'))
);

-- RLS FORCE on all three + tenant policy (same 3-statement pattern as every
-- other tenant table).
```

### API (`/v1`)

| Method | Route | RBAC | Notes |
|--------|-------|------|-------|
| GET | `/v1/community/profiles` | `community.read` | list, `?q=` name/skill/headline, `?role=` |
| GET | `/v1/community/profiles/:id` | `community.read` | profile + company memberships + top skills + badge count |
| GET | `/v1/community/profiles/me` | `community.read` (self allowed: all roles) | the caller's own profile |
| PUT | `/v1/community/profiles/me` | self or `community.write` | create/update own profile |
| GET | `/v1/community/companies` | `community.read` | list, `?q=` |
| POST | `/v1/community/companies` | `community.write` | create company |
| GET | `/v1/community/companies/:id` | `community.read` | company + members + active job offers |
| PATCH | `/v1/community/companies/:id` | `community.write` | update |
| DELETE | `/v1/community/companies/:id` | `community.write` | soft-delete (is_active=false) |
| POST | `/v1/community/companies/:id/members` | `community.write` | add member `{user_id, role, title}` |
| DELETE | `/v1/community/companies/:id/members/:userId` | `community.write` | remove member |

### RBAC additions (`internal/transport/http/permissions.go`)

```go
grant(all, "community", "read")
grant([]string{"owner","manager","saas_admin"}, "community", "write")
// profile self-edit is guarded in the handler (user_id == actor), not RBAC.
```

## Slice 2 — Job board

### Schema (migration `045_jobs.sql`)

```sql
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

-- RLS FORCE + tenant policy on both.
```

### API

| Method | Route | RBAC | Notes |
|--------|-------|------|-------|
| GET | `/v1/community/jobs` | `community.read` | `?q=&company_id=&type=&mine=` (staff may filter own applications) |
| POST | `/v1/community/jobs` | `jobs.write` | create offer (store or company manager) |
| GET | `/v1/community/jobs/:id` | `community.read` | offer + `applied` flag for the caller |
| PATCH | `/v1/community/jobs/:id` | `jobs.write` | update |
| DELETE | `/v1/community/jobs/:id` | `jobs.write` | soft-delete |
| POST | `/v1/community/jobs/:id/apply` | `community.read` (any staff) | 409 `already_applied` on duplicate; snapshots resume |
| GET | `/v1/community/jobs/:id/applications` | `jobs.write` | manager view of applicants (id, name, headline, resume, cover) |
| PATCH | `/v1/community/applications/:id` | `jobs.write` | `{status}` transition |

RBAC: add `grant(all, "jobs", "read")` (aliasing `community.read` is fine) and
`grant([]string{"owner","manager","saas_admin"}, "jobs", "write")`.

## Slice 3 — Badges + endorsements

### Schema (migration `046_badges_endorsements.sql`)

```sql
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
    UNIQUE (tenant_id, code)
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

-- RLS FORCE + tenant policy on all three.
```

### API

| Method | Route | RBAC | Notes |
|--------|-------|------|-------|
| GET | `/v1/community/badges` | `community.read` | catalog (active defs) |
| POST | `/v1/community/badges` | `community.write` | create definition |
| PATCH | `/v1/community/badges/:id` | `community.write` | update |
| GET | `/v1/community/profiles/:id/badges` | `community.read` | badges + counts earned for that user |
| POST | `/v1/community/profiles/:id/badges` | `community.write` | award `{badge_id, note}` (409 `already_awarded`) |
| DELETE | `/v1/community/profiles/:id/badges/:badgeId` | `community.write` | revoke |
| GET | `/v1/community/profiles/:id/endorsements` | `community.read` | group by skill w/ giver list + count |
| POST | `/v1/community/endorsements` | `community.read` (any staff to a peer) | `{receiver_id, skill, note}`; self blocked 400 `cannot_endorse_self`; dup 409 |
| DELETE | `/v1/community/endorsements/:id` | self or `community.write` | withdraw |

## Slice 4 — Forum

### Schema (migration `047_forum.sql`)

```sql
CREATE TABLE forum_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, name)
);

CREATE TABLE forum_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES forum_categories(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
    is_closed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, id),
    CONSTRAINT forum_posts_title_required CHECK (length(btrim(title)) > 0)
);

CREATE TABLE forum_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    post_id UUID NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT forum_comments_body_required CHECK (length(btrim(body)) > 0)
);

CREATE TABLE forum_post_likes (
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    post_id UUID NOT NULL REFERENCES forum_posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, user_id)
);

-- RLS FORCE + tenant policy on all four.
```

### API

| Method | Route | RBAC | Notes |
|--------|-------|------|-------|
| GET | `/v1/community/forum/categories` | `community.read` | active categories + post counts |
| POST | `/v1/community/forum/categories` | `community.write` | create |
| PATCH | `/v1/community/forum/categories/:id` | `community.write` | update |
| GET | `/v1/community/forum/posts` | `community.read` | `?category_id=&q=&sort=latest|top|pinned` |
| POST | `/v1/community/forum/posts` | `forum.write` (all staff) | create post |
| GET | `/v1/community/forum/posts/:id` | `community.read` | post + comments + like state |
| PATCH | `/v1/community/forum/posts/:id` | author or `community.write` | edit |
| DELETE | `/v1/community/forum/posts/:id` | author or `community.write` | delete |
| POST | `/v1/community/forum/posts/:id/comments` | `forum.write` | comment |
| POST | `/v1/community/forum/posts/:id/like` | `forum.write` | toggle-like (idempotent) |

RBAC: add `grant(all, "forum", "write")` — the whole team posts to the kitchen/server/recipe forum. Moderation (edit/delete others) rides on `community.write`.

## Flutter plan (`lib/features/community/`)

- `lib/core/community.dart` — models + wire parsing: `StaffProfile`, `Company`, `CompanyMember`, `JobOffer`, `JobApplication`, `Badge`, `UserBadge`, `SkillEndorsement`, `ForumCategory`, `ForumPost`, `ForumComment` (all `@immutable`, `fromJson` with tolerant defaults).
- `lib/features/community/community_hub_screen.dart` — bottom-nav hub: Profiles / Jobs / Forum / Companies tabs, reachable via the POS app bar (app bar is crowd — use the existing overflow; see decision from B4 overflow items).
- `ProfilesScreen` — searchable staff grid/cards (avatar, headline, chief chip, skills), tap → `ProfileDetailScreen` (bio, memberships, badges, endorsements, resume, "endorse" action).
- `CompaniesScreen` — company cards + `CompanyDetailScreen` (members, open jobs).
- `JobsScreen` — filterable offer list, `JobDetailScreen` (salary, tags, apply sheet w/ cover note + resume snapshot), `MyApplications` section; manager sees applicants + status actions.
- `BadgesScreen` — catalog grid + award flow (manager) per profile.
- `ForumScreen` — categories + post list (latest/top/pinned), `ForumPostScreen` (body, comments, like toggle, comment box).
- `ApiClient` methods mirror the route table; strings under `l10n/strings.dart` (ar + en).
- Widget tests per screen using the in-memory fake `ApiClient` pattern; wire-parsing tests in `test/community_test.dart`.

## Commit boundaries (house convention: one workstream per commit)

1. **Spec this doc** (current commit).
2. Slice 1 (migration `044` + profiles/companies backend + tests + Flutter hub + profiles/companies UI + tests).
3. Slice 2 (migration `045` + jobs backend + UI + tests).
4. Slice 3 (migration `046` + badges/endorsements backend + UI + tests).
5. Slice 4 (migration `047` + forum backend + UI + tests).
6. `make openapi` regen + release notes/roadmap update at the end.

## Security & grants

- All tables `ENABLE ROW LEVEL SECURITY` + `FORCE ROW LEVEL SECURITY` + tenant
  policy (same 3 lines as every tenant table).
- `scripts/grants_prod.sql` must gain DML for `user_profiles`, `companies`,
  `company_members`, `job_offers`, `job_applications`, `badge_definitions`,
  `user_badges`, `skill_endorsements`, `forum_categories`, `forum_posts`,
  `forum_comments`, `forum_post_likes` or prod-only `permission denied`
  failures appear (same gotcha as every new table).
- No `DO $$` blocks in migrations (goose limitation); seed default badges via a
  `-- +goose StatementBegin/End` `DO $$` if shipped, or via an idempotent SQL
  function run by the API first-time (prefer the latter — see C4 loyalty precedent).