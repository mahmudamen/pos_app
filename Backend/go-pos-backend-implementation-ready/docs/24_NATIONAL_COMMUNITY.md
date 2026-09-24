# 24 — Egypt National Community (POS COPILOT network)

Design spec for the **national, Egypt-only** layer of the POS.Go community — a
counterpart to the store-scoped staff community (`docs/23_COMMUNITY_JOBS_PROFILES.md`).

The store-scoped hub is per-tenant staff tooling (profiles, companies, jobs,
badges, forum). This spec layers a **one-Egypt network** on top: staff from every
store join a single national community via **invitations**, where they read/write a
**blog**, **share jobs** on a national board, build an **expertise score**, earn
**platform badges**, and are kept civil by a **strikes** moderation system.

Scope mirrors the `Pos_Go_copilot_Smart` feasibility study's platform goals.
Contrast with docs/23's core decision: docs/23 deliberately stayed tenant-scoped;
this spec deliberately extends the platform to a national surface, gated by
invitation and enforced by Egyptian market constraints (`EG` country, `EGP`, Arabic).

| Slice | Backend | Flutter | Status |
|-------|---------|---------|--------|
| A. Membership + invitations | migration `048`, module `internal/transport/community/national` | Join/invite + member directory | planned |
| B. Blog / knowledge hub | migration `049` | Blogs list/detail/write | planned |
| C. National job board (share jobs) | migration `050` | National jobs + apply + promote | planned |
| D. Expertise score + leaderboard | migration `051` | My score + leaderboard | planned |
| E. Strikes / moderation | migration `052` | Moderation screen + strike card | planned |
| F. National badges | migration `053` | Badge wall on profiles | planned |

Each slice ships backend + Flutter + tests + grants together, one commit per slice
(house convention, docs/23 §commit boundaries).

## Architecture decisions (decision log)

| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-09-24 | National community tables are **not** `FORCE RLS` tenant tables — they are handler-gated, like the platform/saas tables | Community content is genuinely cross-tenant (Egypt-wide) — a tenant policy cannot express it. Precedent: platform tables (`plans`, `subscriptions`, …) are server-managed. Every national route enforces membership + RBAC in the handler; the DB role gets explicit DML grants only. **Hard rule:** national tables never carry tenant-confidential data; `origin_tenant_id` is attribution only. |
| 2026-09-24 | National membership is **invitation-first** | Product wants a curated, trusted staff network before opening up. Codes are single-use by default, expire, and join is rate-limited per IP (brute-force guard). An `open_join` setting on the community allows self-serve later. |
| 2026-09-24 | Invite codes stored **hashed** (`code_hash`, SHA-256), plaintext shown once to the inviter | Codes are bearer tokens — DB leak should not be a usable code dump. 12-char `EG-` base32 codes are human-typable. |
| 2026-09-24 | Expertise score is a **ledger** (`expertise_events`) + cached total on `community_members`, deterministic pure-helper rules | Same audit pattern as `customer_loyalty_log` (C4): every point change is a row with a reason; totals derive from the ledger; leaderboards compute per-period sums from the ledger. |
| 2026-09-24 | Score levels: bronze 0–199 / silver 200–499 / gold 500–999 / platinum 1000+ | Simple, deterministic, no decaying math for v1. Decay/period-earnings are a v2 knob. |
| 2026-09-24 | Strikes expire (`expires_at = +90 days`); threshold weights 1–3; 3+ → 7-day mute, 5+ → suspended | Punishment should fit the incident and decay with good behavior (typical community norms). Handlers compute effective status from active strikes, not a stored boolean. |
| 2026-09-24 | National badges are a **separate catalog** from tenant badges (`national_*` tables, auto-awarded by rules) | Tenant badges (docs/23 slice 3) are store-brand awards, manually given. National badges are platform achievements — different semantics, no single `scope` column fight. |
| 2026-09-24 | The docs/23 **forum (migration `047`) serves as the national discussion space** | Avoid two overlapping content surfaces (forum + blog). Forum = conversations; blog = authored knowledge articles. Both are national in this plan. |
| 2026-09-24 | Egypt-locality enforced as **`tenants.country_code = 'EG'` gate** + Arabic-first UI | Platform already seeds only `EG`/`EGP`/ar (demo tenants, `countries`/`currencies` are single-entry Egypt). The gate is a belt-and-braces check in the national handler helper — non-EG tenants get `403 egypt_only`. |
| 2026-09-24 | National tables excluded from the sync pull feed (like all community tables) | Awareness/interaction data, never register-critical; devices pull sales/catalog state, not the hub. |

## Preconditions

- Actor is authenticated (bearer JWT) and their tenant is `EG`.
- National routes (except `POST /v1/community/join`, `GET /v1/community/invitations`)
  require a `community_members` row — i.e. **members-only**.
- Effective member state derived from active strikes (`strikeState`):
  `active` · `warning` (1–2 weight) · `muted` (3+ → 7 days, no posting) · `suspended` (5+ → hidden from directory, no writes).

## Slice A — Membership + invitations (migration `048_national_members.sql`)

```sql
CREATE TABLE community_members (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    origin_tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,  -- owning store (attribution only)
    display_name TEXT NOT NULL DEFAULT '',           -- public name; mirror of users.display_name
    joined_via TEXT NOT NULL DEFAULT 'invitation',   -- invitation|open
    invited_by UUID REFERENCES users(id) ON DELETE SET NULL,
    role TEXT NOT NULL DEFAULT 'member',             -- member|moderator|admin
    status TEXT NOT NULL DEFAULT 'active',           -- active|muted|suspended (effective, cached)
    expertise_score INT NOT NULL DEFAULT 0 CHECK (expertise_score >= 0),
    level TEXT NOT NULL DEFAULT 'bronze',            -- bronze|silver|gold|platinum (cached from score)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,         -- soft-delete / self-leave
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT community_members_role_check   CHECK (role   IN ('member','moderator','admin')),
    CONSTRAINT community_members_status_check CHECK (status IN ('active','muted','suspended')),
    CONSTRAINT community_members_level_check  CHECK (level  IN ('bronze','silver','gold','platinum'))
);

CREATE TABLE community_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code_hash TEXT NOT NULL UNIQUE,                  -- SHA-256 of the EG-XXXX… code
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
```

### API (`/v1/community`)

| Method | Route | RBAC | Notes |
|--------|-------|------|-------|
| GET | `/v1/community/me` | member | membership + score + level + recent `expertise_events`; 404 `not_a_member` |
| POST | `/v1/community/join` | auth | `{code?}` → validate (`403 invalid_code`/`410 expired`/`409 already_member`); `open_join` setting allows `{code: ""}`; rate-limited per IP |
| POST | `/v1/community/invitations` | `community.invite` | create code(s) `{email?, note?, max_uses?, expires_at?}`; response includes the **plaintext code once** |
| GET | `/v1/community/invitations` | `community.invite` | my store's codes + used_count/status |
| DELETE | `/v1/community/invitations/:id` | `community.invite` (owner) | revoke |
| GET | `/v1/community/members` | member | directory `?q=&level=&role=&governorate=` |
| GET | `/v1/community/members/:id` | member | profile + score + badges + strike summary |
| PATCH | `/v1/community/members/:id` | `community.moderate` | status/role changes |

RBAC additions (`internal/transport/http/permissions.go`):

```go
grant(all, "community", "invite")       // all members may invite peers
grant(all, "community", "moderate")     // exact role matches on community_members.role:
                                        //   moderator/admin + saas_admin (checked in handler);
                                        //   NOT org roles — national surface
```

## Slice B — Blog / knowledge hub (migration `049_blog.sql`)

```sql
CREATE TABLE blog_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE blog_posts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category_id UUID REFERENCES blog_categories(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    slug TEXT NOT NULL,
    body TEXT NOT NULL DEFAULT '',
    excerpt TEXT NOT NULL DEFAULT '',
    cover_url TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'draft',     -- draft|published|archived
    is_featured BOOLEAN NOT NULL DEFAULT FALSE,
    view_count INT NOT NULL DEFAULT 0,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (slug),
    CONSTRAINT blog_posts_title_required CHECK (length(btrim(title)) > 0),
    CONSTRAINT blog_posts_status_check CHECK (status IN ('draft','published','archived'))
);
```

### API

| Method | Route | RBAC | Notes |
|--------|-------|------|-------|
| GET | `/v1/community/blog/categories` | member | active categories + post counts |
| POST/PATCH | `/v1/community/blog/categories` `/…/:id` | `community.moderate` | |
| GET | `/v1/community/blog/posts` | member | `?q=&category_id=&author_id=&sort=latest|popular|featured&status=` (non-authors see published only) |
| POST | `/v1/community/blog/posts` | member | draft or publish (`publish: true` → +15xp via `expertise_events`) |
| GET | `/v1/community/blog/posts/:id` | member | increments `view_count`; author sees own drafts |
| PATCH/DELETE | `/v1/community/blog/posts/:id` | author or `community.moderate` | edit / archive |

## Slice C — National job board (share jobs) (migration `050_hub_jobs.sql`)

```sql
CREATE TABLE hub_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    origin_job_id UUID REFERENCES job_offers(id) ON DELETE SET NULL,  -- promoted tenant job (attribution)
    author_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    origin_tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    company_name TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    employment_type TEXT NOT NULL DEFAULT 'full_time',  -- full_time|part_time|contract|freelance
    governorate TEXT NOT NULL DEFAULT '',               -- Egypt governorate (القاهرة، الجيزة، …)
    remote BOOLEAN NOT NULL DEFAULT FALSE,
    salary_minor BIGINT NOT NULL DEFAULT 0 CHECK (salary_minor >= 0),
    salary_currency CHAR(3) NOT NULL DEFAULT 'EGP',
    skill_tags TEXT[] NOT NULL DEFAULT '{}',
    contact_email TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'open',                -- open|filled|closed
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    closes_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT hub_jobs_title_required CHECK (length(btrim(title)) > 0),
    CONSTRAINT hub_jobs_type_check   CHECK (employment_type IN ('full_time','part_time','contract','freelance')),
    CONSTRAINT hub_jobs_status_check CHECK (status IN ('open','filled','closed'))
);

CREATE TABLE hub_job_applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hub_job_id UUID NOT NULL REFERENCES hub_jobs(id) ON DELETE CASCADE,
    applicant_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cover_note TEXT NOT NULL DEFAULT '',
    resume JSONB NOT NULL DEFAULT '{}',                 -- snapshot from user_profiles.resume at apply time
    status TEXT NOT NULL DEFAULT 'applied',             -- applied|under_review|accepted|rejected
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (hub_job_id, applicant_id),
    CONSTRAINT hub_job_app_status_check CHECK (status IN ('applied','under_review','accepted','rejected'))
);
```

### API

| Method | Route | RBAC | Notes |
|--------|-------|------|-------|
| GET | `/v1/community/jobs` (national) | member | `?q=&governorate=&type=&mine=&status=` |
| POST | `/v1/community/jobs` | store manager/owner or `community.moderate` | direct posting |
| POST | `/v1/community/jobs/promote` | store manager/owner | `{job_id}` → copy a tenant `job_offers` row here (`origin_job_id` set) |
| GET | `/v1/community/jobs/:id` | member | + `applied` flag + `application_count` (author) |
| PATCH/DELETE | `/v1/community/jobs/:id` | author or `community.moderate` | update / close |
| POST | `/v1/community/jobs/:id/apply` | member | 409 `already_applied`; snapshots resume; accepted later → +50xp |
| GET | `/v1/community/jobs/:id/applications` | author | applicant list |
| PATCH | `/v1/community/applications/:id` | author | status transition |

## Slice D — Expertise score + leaderboard (migration `051_expertise_events.sql`)

```sql
CREATE TABLE expertise_events (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    delta INT NOT NULL CHECK (delta <> 0),
    reason TEXT NOT NULL,       -- first_join +10 | posted_blog +15 | got_endorsement +2 |
                                --   job_accepted +50 | invited_peer_joined +20 |
                                --   strike_issued -25 | suspended -50
    ref_table TEXT NOT NULL DEFAULT '',
    ref_id UUID,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX expertise_events_user_idx ON expertise_events (user_id, created_at DESC);
CREATE INDEX expertise_events_period_idx ON expertise_events (created_at DESC);
```

- Pure helper mirrors `pointsForTotal` (C4): `scoreForReason(reason) → int`, deterministic.
- Every awarding/costing tx writes the ledger row then updates `community_members.expertise_score`
  and re-derives `level` (`levelForScore`). Level upsert is idempotent.
- Leaderboard: `GET /v1/community/leaderboard?period=week|month|all` sums `delta` over active
  members in the period (top 100), `LIMIT 100`.

| Method | Route | RBAC | Notes |
|--------|-------|------|-------|
| GET | `/v1/community/me` | member | includes score + level + last 20 `expertise_events` |
| GET | `/v1/community/leaderboard` | member | `?period=` sum ledger, exclude suspended |

## Slice E — Strikes / moderation (migration `052_strikes.sql`)

```sql
CREATE TABLE community_strikes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    member_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    issued_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    weight SMALLINT NOT NULL DEFAULT 1 CHECK (weight BETWEEN 1 AND 3),
    reason TEXT NOT NULL,   -- spam_posting|abusive_content|plagiarized_blog|fake_job|invite_abuse|review_gaming|other
    note TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '90 days',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT community_strikes_reason_check CHECK (reason IN ('spam_posting','abusive_content','plagiarized_blog','fake_job','invite_abuse','review_gaming','other'))
);
CREATE INDEX community_strikes_member_idx ON community_strikes (member_id, expires_at DESC);
```

- Pure helper `strikeState(sumActiveWeights)` → `warning` (<3) · `muted` (3–4) · `suspended` (5+);
  muted = 7-day post/comment/apply block, suspended = directory-hidden + all writes blocked.
- Handler re-derives `community_members.status` from the ledger on write.

| Method | Route | RBAC | Notes |
|--------|-------|------|-------|
| POST | `/v1/community/members/:id/strikes` | `community.moderate` | `{weight, reason, note}` → re-derives status |
| GET | `/v1/community/members/:id/strikes` | self or `community.moderate` | active + history |
| POST | `/v1/community/members/:id/lift` | `community.moderate` (admin) | expire active strikes, restore |
| DELETE | `/v1/community/strikes/:id` | `community.moderate` (admin) | revoke a strike |

## Slice F — National badges (migration `053_national_badges.sql`)

```sql
CREATE TABLE national_badge_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL UNIQUE,            -- kebab-case
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT 'workspace_premium',
    criteria JSONB NOT NULL DEFAULT '{}', -- human-readable award rule
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE national_user_badges (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    badge_id UUID NOT NULL REFERENCES national_badge_definitions(id) ON DELETE CASCADE,
    awarded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    award_reason TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (user_id, badge_id)
);
```

- Auto-award rules in a pure `badgeCandidates(events…)` helper, evaluated inside the tx that
  triggers them (join, publish, view threshold, invite, hire, endorsement count, clean record);
  idempotent (skip when `national_user_badges` row exists). Seed catalog in the migration
  (plain `INSERT`, no `DO $$`): `first_steps` (joins), `first_post`, `popular_author`
  (500 cumulative views), `people_person` (invite 3), `hired` (job accepted), `respected`
  (10 endorsements), `clean_record` (120 days strike-free).

| Method | Route | RBAC | Notes |
|--------|-------|------|-------|
| GET | `/v1/community/badges` | member | national catalog |
| GET | `/v1/community/members/:id/badges` | member | earned badges for a member |

## Egypt locality

- National handler helper `requireEgypt` checks `tenants.country_code = 'EG'` for the
  actor's tenant → 403 `egypt_only` otherwise. Same helper asserts `community_members`
  presence for member routes.
- UI is Arabic-first; salaries are `salary_minor` + `EGP`; `governorate` free-text with
  an Egyptian governorate seed list in the app.
- Join is rate-limited per IP (reuse `LoginRateLimit` middleware) — invite codes are the
  token of entry and must not be brute-forceable.

## Flutter plan (`lib/features/community/national/`)

- `lib/core/community.dart` extends with: `CommunityMembership`, `Invitation`,
  `BlogPost`, `BlogCategory`, `HubJob`, `HubJobApplication`, `ExpertiseEvent`,
  `Strike`, `NationalBadge` (`@immutable`, tolerant `fromJson`).
- `NationalHubScreen` — new "شبكة مصر" tab in the community hub: My Score card
  (level + recent events), Invite button, Blog list, National jobs, Leaderboard,
  Moderation (role-gated).
- `JoinScreen` — invites currently un-joined users to enter a `EG-` code (or open join).
- `InvitationsScreen` — create/revoke codes; creators see the plaintext once.
- `BlogScreen` / `BlogPostScreen` / `BlogEditorScreen` — cards, read view with view
  count, draft/publish.
- `NationalJobsScreen` / `JobDetailScreen` — governorate/type filters, apply sheet
  (cover note + resume snapshot), "promote job" action from store jobs.
- `LeaderboardScreen` — period chips, ranking list with level chips.
- `BadgesScreen` — national badge wall on every member profile.
- `ModerationScreen` — strike issuance + member lift (moderators/saas_admin).
- `ApiClient` methods mirror the route table; strings under `l10n/strings.dart` (ar).
- Widget tests per screen (in-memory fake `ApiClient`), wire-parsing in
  `test/community_test.dart`.

## Commit boundaries

1. **Spec this doc + roadmap Phase G update** (current commit).
2. Slice A (migration `048` + membership/invitations backend + tests + Flutter join/invite/directory).
3. Slice B (migration `049` + blog backend + UI + tests).
4. Slice C (migration `050` + hub jobs + promote + UI + tests).
5. Slice D (migration `051` + expertise ledger + leaderboard + UI + tests).
6. Slice E (migration `052` + strikes + moderation UI + tests).
7. Slice F (migration `053` + national badges + badge wall + tests).
8. `make openapi` regen + release notes/roadmap update at the end.

## Security & grants

- National tables get **no tenant RLS** (documented exception — see decision log).
  The app roles hold DML via grants only; dev = owner `pos_app`; prod =
  `pos_app_rls` gains DML in `scripts/grants_prod.sql` for
  `community_members`, `community_invitations`, `blog_categories`, `blog_posts`,
  `hub_jobs`, `hub_job_applications`, `expertise_events`, `community_strikes`,
  `national_badge_definitions`, `national_user_badges` (prod-only `permission
  denied` failure otherwise — same gotcha as every new table).
- Invite codes: stored as SHA-256 `code_hash`; plaintext surfaced once at creation.
- No `DO $$` blocks in migrations (goose limitation); badge seed uses plain
  `INSERT` with `ON CONFLICT DO NOTHING`.
- Member moderator/admin roles are national-scope (platform staff), checked in the
  handler — org roles (`owner`/`manager`/`saas_admin`) govern store actions only.
- Everything bearer-auth'd under `/v1/community/*`; no public routes in this workstream.