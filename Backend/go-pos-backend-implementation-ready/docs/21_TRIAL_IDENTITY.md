# 21 — Free-Trial & Identity System (Design)

Status: **design / awaiting product decisions** (see §15).
Scope: anti-abuse free trials, account/organization identity, verified
contact, installation registry, audit, and the client-side subscription gate.

This document is the source of truth for the trial system. It is grounded in
the current code (migration 032, module `github.com/example/pos-api`). It
supersedes the 15-day hard-coded trial in `transport/auth/register.go`.

---

## 0. Current state (gap analysis)

Already present:

- `tenants` (= the organization), `users` (tenant-scoped login), `devices`
  (per-tenant `client_device_id`), `sessions` (SHA-256 refresh hash, rotation
  via conditional UPDATE, AUTH-007 family revocation), `refresh_tokens` DDL
  (unused).
- `store_emails` — cross-tenant email uniqueness registry used by signup
  (`INSERT` → 23505 → `409 email_taken`).
- Trial = `tenants.plan='trial'` + `tenants.trial_ends_at`, hard-coded
  `trialDays = 15` in `transport/auth/register.go`. Enforced only at login
  (`403 trial_expired`). `subscriptions` (`028_billing.sql`) is backfilled for
  old tenants but **never written at signup**.
- Platform billing control plane under `/v1/saas/*` (`saas_admin` only):
  plans, subscriptions, invoices, status state machine, mock/manual gateways.
- Per-IP login rate limit (memory or Redis), RLS FORCE tenant isolation,
  `client_events` telemetry inbox, structured request logs (`slog`).

Missing (this design adds it):

| Gap | Impact |
|---|---|
| No `accounts` (global identity) | email is only unique per tenant; re-register freely |
| No `trial_entitlements` | no durable, scope-aware "has this identity already trialled" record |
| No `installations` | reinstall = new random device id, no cross-tenant install history |
| No `email_verified` / `phone` / OTP | verification requirements impossible today |
| No mailer / SMS provider | verification cannot be delivered (external dependency) |
| No audit log | security events are not persisted |
| No trial config | duration/scope/limits hard-coded or absent |
| No `GET /v1/subscription` | client shows trial only from the login envelope |
| No email-change workflow | email is effectively immutable client-side; no secure change |

Design principle that resolves the biggest tension: **do not refactor
`users` into a global identity.** Introduce a platform-level `accounts` table
and link `users.account_id` → `accounts.id`. A `users` row remains the
per-organization membership (what the spec calls `organization_members`), so
every existing RLS policy, handler, sale authorship and sync path keeps
working. Entitlements key off `account_id` and/or `tenant_id` per the
configured scope.

---

## 1. Threat model

Assets: free-trial entitlement, account, organization, install history,
audit trail, tenant data.

Adversary: an economically-motivated user who wants unbounded free trials,
without sophisticated tooling; plus scripted bulk registration.

| # | Threat | Layers that counter it | Residual risk |
|---|---|---|---|
| T1 | Re-register with a new email | `accounts` keyed on normalized email; trial `eligibility_key=acct:<id>`; `store_emails`; per-IP/velocity limits | Truly distinct emails are distinct accounts → covered by org/phone/IP risk, not identity |
| T2 | New org under same account | entitlement scope `account` (partial unique on `account_id`) | — |
| T3 | Reinstall / clear storage / new device id | install id is *not* identity; account/organization scope still applies | none to trial (install only raises risk) |
| T4 | Change device clock | all dates from Postgres `now()` (UTC); client gets `server_time` | offline grace window is clock-independent (monotonic elapsed) |
| T5 | Modify Flutter storage (`has_used_trial=false`) | authoritative state is server; client never decides | client can lie but gains nothing |
| T6 | Call `register`/trial API directly | entitlement granted only by the eligibility service inside the signup tx; no client-supplied dates accepted | — |
| T7 | Two concurrent registrations race | per-tx `pg_advisory_xact_lock(hashtext(eligibility_key))` + partial unique index + `ON CONFLICT DO NOTHING` | — |
| T8 | Multiple accounts, same phone | scope `verified_phone` → partial unique on verified `phone_e164` | requires phone verification to be enabled |
| T9 | Same phone verified on another account | phone verification enforces one `accounts.phone_e164` unique; change workflow audited | recycled numbers: support review path |
| T10 | Cross-tenant / other org's trial | RLS FORCE + JWT-derived tenant; platform tables behind `saas_admin`; entitlement lookup scoped by authenticated account/tenant | — |
| T11 | Register a cashier to grab a trial | organization creation / entitlement only for the signup owner path; RBAC `pos.*`; `saas.admin` | — |
| T12 | Replay refresh token / session theft | existing rotation + family revocation; add Redis reuse counters + audit | — |
| T13 | Client sends `trial_days` / `expires_at` / `user_id` | request schema ignores those fields; identity from claims | — |
| T14 | Emulator / rooted device farming | optional `DeviceIntegrityVerifier` (Play Integrity / App Attest) behind config; **never sole control**, never blocks POS by default | determined attacker can pass on a clean device |
| T15 | Bulk signup from one IP/network | Redis velocity limits per IP/prefix/email-domain/phone-prefix; risk score; manual review | shared NAT / CGNAT → risk-based, not hard block |
| T16 | Legitimate shared device / multiple businesses on one network | rules are scoped to account/org/phone, **not** device or IP; device/IP only modulate `risk_level` and review | accepted |
| T17 | Delete trial history | `trial_entitlements` unique constraints + no DELETE grants; only `status` transitions; `audit_log` append-only | — |

Non-goals / explicitly rejected: IMEI/MAC/contacts collection, invasive
fingerprinting, `client_device_id` as identity, "same IP ⇒ same person",
permanent bans from one weak signal.

---

## 2. Identity model

Five distinct layers (never collapsed):

1. **Account identity** — `accounts`: one per normalized primary email.
   Owns verified email/phone. Survives email changes (id is stable).
2. **Organization identity** — `tenants` (a POS business). Gains
   `owner_user_id`, `status`.
3. **Membership** — `users` (existing tenant-scoped login) gains
   `account_id`, `email_verified_at`, `phone`, `phone_verified_at`, `status`.
   This *is* `organization_members`; roles already exist (owner/manager/
   cashier + `saas_admin` platform-only).
4. **Installation identity** — `installations`: a privacy-conscious
   per-app-install record (server-validated random UUIDv4), independent of
   account and tenant, linked to the per-tenant `devices` row.
5. **Trial entitlement** — `trial_entitlements`: the authoritative record,
   with an `eligibility_key` that encodes the configured scope.

Relationship summary:

```
accounts 1─* users *─1 tenants
    │                     │
    └──── trial_entitlements ────┘        installations ── devices (per tenant)
```

---

## 3. Trial eligibility rules

All values configurable (§12). Defaults proposed in §15.

`CheckEligibility(ctx, EligibilityRequest) (EligibilityResult, error)`

Request (server-derived only): `account_id`, `tenant_id`, `owner_user_id`,
`trial_type`, `email_verified`, `phone_verified`, `installation_id`,
`risk_level`, `ip`.

Checks, in order (short-circuit on failure, recorded with a reason code):

| # | Check | Reason code |
|---|---|---|
| 1 | account exists & active | `account_invalid` |
| 2 | email verified (if `require_email_verification`) | `email_unverified` |
| 3 | phone verified (if `require_phone_verification`) | `phone_unverified` |
| 4 | organization exists & active | `organization_invalid` |
| 5 | organization has no non-revoked entitlement | `org_trial_consumed` |
| 6 | account has no non-revoked entitlement (scope `account`) | `account_trial_consumed` |
| 7 | verified phone has no entitlement (scope `verified_phone`) | `phone_trial_consumed` |
| 8 | no blocked/revoked entitlement on this installation/account | `installation_risk` |
| 9 | registration velocity within limits (Redis) | `velocity_exceeded` |
| 10 | `risk_level` ≤ `suspicious_registration_policy` threshold | `risk_rejected` / `needs_review` |
| 11 | `trial_type` is a configured, active type | `invalid_trial_type` |
| 12 | org count for the account ≤ `max_organizations_per_account` | `max_organizations_reached` |

`GrantTrial` runs in **one** transaction:

```sql
SELECT pg_advisory_xact_lock(hashtext($eligibility_key || ':' || $trial_type));
INSERT INTO trial_entitlements (...)
VALUES (...)
ON CONFLICT (eligibility_key, trial_type) DO NOTHING
RETURNING id;
```

If no row is returned the trial was already consumed (or lost the race) →
`CheckEligibility` result is `denied`, and the caller proceeds **without** a
trial. Start/expiry are always `now()`/`now() + make_interval(days => n)`
server-side. On grant, the service projects `tenants.plan='trial'`,
`tenants.trial_ends_at=expires_at`, and upserts the `subscriptions` row
(`status='trial'`), so the existing login gate and billing plane stay valid.

`eligibility_key` by scope:

- `account`: `acct:<account_id>`
- `organization`: `org:<tenant_id>`
- `verified_phone`: `phone:<e164>`
- `business_identity`: `biz:<normalized_tax_id>` (reserved; requires a
  business-verification data source)

Suspicious but legitimate cases return `needs_review` (trial held `pending`,
manual admin approval) rather than a hard deny.

---

## 4. Database design (migration `033_identity_trial.sql`)

New platform-level tables (no RLS, least-privilege grants, never queried by
the client directly):

```sql
accounts (
  id uuid PK default gen_random_uuid(),
  primary_email text NOT NULL,                 -- stored lowercased/trimmed
  email_verified_at timestamptz,
  phone_e164 text,                             -- E.164, nullable
  phone_verified_at timestamptz,
  status text NOT NULL default 'active' check (status in ('pending','active','suspended','disabled')),
  risk_level text NOT NULL default 'low' check (risk_level in ('low','medium','high','blocked')),
  created_at, updated_at
);
create unique index accounts_email_uq on accounts (lower(primary_email));
create unique index accounts_phone_uq on accounts (phone_e164) where phone_e164 is not null;

installations (
  id uuid PK, installation_public_id uuid NOT NULL UNIQUE,
  account_id uuid null references accounts(id) on delete set null,
  tenant_id uuid null references tenants(id) on delete set null,
  platform text, app_version text,
  first_seen_at, last_seen_at, last_authenticated_at,
  device_integrity_status text,                -- 'verified'|'unavailable'|'failed'|null
  risk_level text not null default 'low',
  revoked_at timestamptz null,
  created_at, updated_at
);

trial_entitlements (
  id uuid PK, organization_id uuid NOT NULL references tenants(id) on delete cascade,
  owner_user_id uuid, account_id uuid NOT NULL references accounts(id),
  trial_type text NOT NULL default 'standard',
  status text NOT NULL default 'active'
    check (status in ('pending','active','expired','converted','revoked','canceled')),
  started_at timestamptz, expires_at timestamptz,
  trial_days int NOT NULL check (trial_days between 0 and 3650),
  consumed_at timestamptz, source text NOT NULL default 'signup',
  eligibility_key text NOT NULL, reason text,
  created_at, updated_at
);
create unique index trial_entitlements_scope_uq
  on trial_entitlements (eligibility_key, trial_type);
create unique index trial_entitlements_org_uq
  on trial_entitlements (organization_id) where status in ('pending','active','expired','converted');
create index trial_entitlements_account_idx on trial_entitlements (account_id);

audit_log (
  id uuid PK, created_at timestamptz not null default now(),
  actor_user_id uuid, account_id uuid, tenant_id uuid,
  action text NOT NULL, entity_type text, entity_id uuid,
  before jsonb, after jsonb, reason text,
  ip inet, user_agent text
);  -- append-only: app role gets INSERT+SELECT only

email_verification_tokens (
  id uuid PK, account_id uuid NOT NULL references accounts(id) on delete cascade,
  token_hash text NOT NULL UNIQUE, purpose text NOT NULL,   -- verify|email_change
  new_email text, expires_at timestamptz NOT NULL,
  consumed_at timestamptz, created_at
);
create index email_verification_account_idx on email_verification_tokens (account_id, expires_at desc);

phone_otp_challenges (
  id uuid PK, account_id uuid NOT NULL references accounts(id) on delete cascade,
  phone_e164 text NOT NULL, otp_hash text NOT NULL,
  attempts int NOT NULL default 0, max_attempts int NOT NULL default 5,
  expires_at timestamptz NOT NULL, consumed_at timestamptz, created_at
);

registration_risk_events (
  id uuid PK, account_id uuid, tenant_id uuid, installation_id uuid,
  ip inet, signal text NOT NULL, weight int NOT NULL default 0,
  detail jsonb, created_at timestamptz not null default now()
);
```

Alters to existing tables:

```sql
alter table tenants add column owner_user_id uuid,            -- backfilled from owner user
                    add column status text not null default 'active'
                      check (status in ('active','suspended','closed'));
alter table users   add column account_id uuid references accounts(id),
                    add column email_verified_at timestamptz,
                    add column phone text, add column phone_verified_at timestamptz,
                    add column status text not null default 'active'
                      check (status in ('pending','active','suspended','disabled'));
alter table devices add column installation_id uuid references installations(id);
```

Backfill (single `DO`-free `INSERT ... SELECT ... ON CONFLICT` blocks; goose
cannot parse `DO $$`): create one `accounts` row per distinct
`lower(users.email)` from `store_emails`, link owners, set
`tenants.owner_user_id`. Existing trials are preserved as-is by copying
`tenants.trial_ends_at` into a `trial_entitlements` row with
`eligibility_key = 'org:<tenant_id>'`, `source='backfill'`, `status`
derived from expiry. Rollback drops the new tables/columns; `trial_entitlements`
backfill is intentionally irreversible (historical record).

Constraints/guarantees:

- no duplicate entitlement for the same `(eligibility_key, trial_type)`;
- no second live entitlement for an organization;
- one live owner membership per org (existing role model + partial index on
  `users(tenant_id) where role='owner' and status='active'`);
- no cross-tenant access (RLS unchanged on tenant tables; platform tables
  only reachable through server services);
- app role `pos_app_rls` gets `SELECT,INSERT` (never `DELETE`) on
  `trial_entitlements`, `audit_log`; `SELECT,INSERT,UPDATE` on `accounts`,
  `installations`, verification tables; grants added to `grants_prod.sql`.

---

## 5. API contract

### Public

- `POST /v1/auth/register` — body gains `installation_public_id`,
  `account_id?`(no), `accepted_terms_version`,
  `accepted_privacy_policy_version`; server responds with `account`,
  `organization`, `subscription`, `trial`, `server_time`. Trial is granted
  **only** if §3 passes; otherwise `trial.status='denied'` with a reason and
  the tenant is created without a trial.
- `POST /v1/auth/verify-email` / `POST /v1/auth/resend-verification`
- `POST /v1/auth/verify-phone` / `POST /v1/auth/resend-otp`
- `POST /v1/auth/refresh-email-change` (see §6)

### Authenticated (tenant/account from JWT)

- `GET /v1/subscription` — authoritative state; `days_remaining` computed by
  the server; includes `server_time`, `grace_period`, `renewal_required`,
  `features`, `limits`, `suspension_reason`.
- `POST /v1/email-change/request` → sends token to the new address
- `POST /v1/email-change/confirm` → verifies token, swaps email, audits,
  notifies the old address
- `POST /v1/phone/request-change`, `POST /v1/phone/confirm-change`
- `GET /v1/installations`, `POST /v1/installations`, `POST /v1/installations/:id/revoke`
- `POST /v1/sessions/logout-all`

### Admin (`saas_admin` only, all audited with `reason`)

- `POST /v1/saas/trials/:tenantId/grant` (promotional/manual)
- `POST /v1/saas/trials/:id/extend`, `.../revoke`
- `POST /v1/saas/accounts/:id/suspend`, `POST /v1/saas/tenants/:id/suspend`
- `GET /v1/saas/trials`, `GET /v1/saas/audit`
- `GET/PUT /v1/saas/trial-settings` (scope, duration, verification,
  max orgs/installations, risk policy) — persisted in `tenant_settings`-style
  platform settings table.

Request/response JSON for each is added to `docs/openapi.json` via
`make openapi`; write-body schemas are hand-augmented as the contract grows.

---

## 6. Email-change protection

1. `POST /v1/email-change/request {new_email, current_password}` — auth +
   password re-check; rate-limited per account/IP.
2. Server normalizes `new_email`, rejects if already used by another account
   (`store_emails` / `accounts_email_uq`) **without revealing** whether it
   belongs to an account (uniform response).
3. Insert `email_verification_tokens(purpose='email_change', new_email,
   token_hash=sha256(random32), expires_at=now()+15m)`; email the **new**
   address a signed link.
4. `POST /v1/email-change/confirm {token}` — single-use
   (`consumed_at` set in the same tx that swaps `accounts.primary_email`);
   update `store_emails`; update `users.email` for the account's memberships;
   notify the **old** address; audit `email_change` with before/after.
5. Trial entitlement is keyed on `account_id`/`tenant_id` and is untouched.
   Phone verification is untouched; if `require_phone_verification`, the
   account still needs a verified phone.

Signup email (when `require_email_verification`) uses the same token table
with `purpose='verify'`; until verified, `accounts.status='pending'` and
eligibility check #2 denies the trial (tenant may still be created as
`pending`, or registration can be two-phase — see §15).

---

## 7. Phone verification

- Normalize to E.164 (libphonenumber semantics; `+20…`).
- `phone_otp_challenges`: `otp_hash = sha256(server_random_6_digit)`; never
  stored or logged in plaintext; TTL 5 min.
- Limits (Redis + DB): ≤5 verify attempts per challenge, ≤3 resends per
  30 min, ≤5 challenges per phone/hour, ≤10 per account/day, per-IP limits.
- Verify success sets `accounts.phone_e164` (unique) + `phone_verified_at`.
- Unknown-account responses are uniform (no enumeration).
- Change workflow reuses `purpose='phone_change'` and audits before/after.
- Delivery behind `SMSSender` interface; default `NoopSender` in dev writes
  nothing sensitive (dev-only `/v1/dev/outbox` when `APP_ENV=dev`).

---

## 8. Installation identity & attestation

- Flutter generates `installation_public_id = Uuid().v4()` once, stored in
  `flutter_secure_storage`. Uninstall clears secure storage → new id after a
  genuine reinstall. An in-app "reset" rotates it but is audited.
- `POST /v1/installations` registers/refreshes `platform`, `app_version`,
  `last_seen_at`; linked to `account_id` on auth.
- Rotation/revocation via admin + user device management; multiple
  installations per account are legitimate and never blocked.
- `DeviceIntegrityVerifier` interface: `PlayIntegrityVerifier` /
  `AppleAppAttestVerifier` / `NoopVerifier`. Default `NoopVerifier`
  (`require_device_integrity=false`). Result stored as
  `device_integrity_status` and contributes to `risk_level` only. A client
  boolean is never trusted; the token is verified server-side. POS stays
  usable when attestation is unavailable unless the business opts in.

---

## 9. Registration flow (server)

```
POST /v1/auth/register
 1 validate + normalize (email, phone)
 2 rate limit (IP + email domain + installation)
 3 upsert accounts (pending; email verified or not per config)
 4 create tenant (status pending/active) + owner users row + membership
 5 register/link installation
 6 verify email/phone if required (else mark verified)
 7 CheckEligibility → GrantTrial (one tx, advisory lock)
 8 project tenants.plan/trial_ends_at + upsert subscriptions row
 9 seed catalog, create session (existing code)
10 audit account_registered / organization_created / trial_granted|denied
11 return account + organization + subscription + trial + server_time
```

Idempotency: an `Idempotency-Key` header on register is recorded (Redis +
`sync_commands`-style table) so replayed HTTP requests cannot create two
accounts/tenants. Client never supplies dates/user ids.

---

## 10. Offline policy

Config `offline_trial_policy`:

- While status ∈ {trialing, active, grace_period} and the last successful
  `GET /v1/subscription` is within `offline_grace_hours` (default 24h):
  full offline POS, including `create_sale` queueing.
- After grace elapses without a successful server check: offline **new sales
  blocked**; existing data preserved; sync/catalog still attempted.
- Clock-independent: compute elapsed from `last_verified_at` (server time at
  last fetch) and `Stopwatch`/monotonic elapsed since process start, not from
  `DateTime.now()` deltas against a possibly-wrong clock.
- Trial expiring while offline: sales already queued are synced on reconnect
  and reconciled; server rejects only if subscription is genuinely suspended.
- Local sales are **never deleted** on expiry.
- Server reconciliation on reconnect updates state and audits transitions.

---

## 11. Flutter architecture

Adds, in the existing plain-http/secure-storage style (no new state package):

```
lib/core/subscription/        SubscriptionRepository, trial/subscription models
lib/core/installation/        InstallationIdentityService (uuid + secure storage)
lib/core/auth/                EmailChangeController, PhoneVerificationController
lib/features/subscription/    SubscriptionScreen (plan, countdown, upgrade CTA)
lib/features/auth/            verification screens
```

- `SubscriptionState` + `SubscriptionController` (ChangeNotifier) owned by the
  app shell; `TrialGuard` widget gates POS routes by **server** status.
- Countdown derives from `GET /v1/subscription.days_remaining` and is
  re-synced with `server_time` (skew offset) — never computed from local clock.
- Refresh on foreground (`WidgetsBindingObserver`), on login, and on a
  debounced timer; revoked session → forced sign-out.
- `InstallationIdentityService` sends `installation_public_id` on
  register/login; persisted in secure storage, survives logout (as today),
  rotates only on reinstall or explicit reset.
- `ApiClient` gains `subscription()`, email/phone change, installations,
  logout-all.

---

## 12. Configuration

New env/config keys (defaults in §15):

```
TRIAL_DURATION_DAYS=14
TRIAL_SCOPE=account                 # account|organization|verified_phone|business_identity
REQUIRE_EMAIL_VERIFICATION=false
REQUIRE_PHONE_VERIFICATION=false
REQUIRE_DEVICE_INTEGRITY=false
MAX_ORGANIZATIONS_PER_ACCOUNT=3
MAX_ACTIVE_INSTALLATIONS=10
SUSPICIOUS_REGISTRATION_POLICY=review   # allow|review|deny
OFFLINE_TRIAL_POLICY=grace24h
REGISTER_RATE_PER_IP_PER_HOUR=5
OTP_TTL_SECONDS=300
EMAIL_TOKEN_TTL_MINUTES=15
MAILER=noop                         # noop|smtp
SMS_SENDER=noop                     # noop|provider
PROMO_TRIALS_ENABLED=false
```

Shop-level overrides live in a platform `trial_settings` row (admin API),
falling back to env.

---

## 13. Audit & monitoring

- `audit_log` append-only (INSERT/SELECT only for the app role): registration,
  email/phone verification + change, eligibility checked, trial
  granted/denied/extended/revoked/converted, org created/blocked, install
  registered/changed, session created/revoked, suspicious registration,
  manual override (always with actor + reason + before/after).
- Never log passwords, OTPs, refresh tokens, or token values.
- Prometheus counters: `trial_grants_total{result}`, `trial_denials_total{reason}`,
  `registrations_total{result}`, `otp_failures_total`, `email_changes_total`,
  `installations_total`. Alerts on registration spikes, OTP abuse, denial
  spikes, audit-write failures.
- Data retention: `registration_risk_events`, `phone_otp_challenges`,
  `email_verification_tokens` purged after 90/1/1 days; `audit_log` retained
  24 months; documented in `docs/11_OPERATIONS.md`.

---

## 14. Testing plan (maps to the 33 required scenarios)

Unit: eligibility matrix (every reason code, every scope), config parsing,
E.164 normalization, OTP/token hashing + expiry, risk scoring, state
transitions. Integration (real Postgres, `testutil`): simultaneous grants
(advisory lock + unique index), reinstall/new device, email/phone change
non-reset, cross-tenant access, offline-expiry reconciliation, admin
extend/revoke, refresh reuse, refund/rollback of a failed grant tx. Security:
direct `register` with forged fields, client-supplied dates ignored, RLS
isolation, reuse/revocation, rate-limit boundaries, uniform enumeration
responses. Each test asserts the backend remains authoritative.

---

## 15. Open product decisions (blocking)

1. **Identity model** — recommended: layered `accounts` + existing
   tenant-scoped `users` (low risk). Alternative: full global
   `users`/`organization_members` refactor (high risk, touches every
   handler/RLS path).
2. **Default `trial_scope`** — `account` / `organization` / `verified_phone`.
3. **Verification defaults** — since **no mailer/SMS exists**, do we ship
   Phase 2/3 with `noop` providers + dev outbox, or defer verification until a
   provider is chosen?
4. **Registration with unverified email** — allow tenant creation as
   `pending` (trial withheld until verified) vs two-phase (account only,
   org created after verification).
5. **Implementation order** — Phase 0→5 below, or a subset first.

---

## 16. Implementation phases

- **Phase 0 — foundation (no behavior change):** config keys; migration
  `033_identity_trial.sql`; `accounts`/`installations`/`trial_entitlements`/
  `audit_log`/verification tables; repositories; `TrialEligibilityService`
  (+ unit/integration tests); grants. Existing signup untouched.
- **Phase 1 — wire signup + subscription API:** registration creates
  account/membership/installation, grants via the service, projects
  `tenants`/`subscriptions`; `GET /v1/subscription`; admin trial
  grant/extend/revoke + audit; backfill verified.
- **Phase 2 — email verification + email change** (mailer interface, noop dev).
- **Phase 3 — phone verification/OTP** (SMS interface, Redis limits).
- **Phase 4 — installations lifecycle + device management + attestation
  interface (disabled).**
- **Phase 5 — Flutter:** installation service, subscription repository +
  controller, `TrialGuard`, subscription screen/countdown, email/phone
  screens, offline policy, foreground refresh.
- **Phase 6 — admin control plane, monitoring/alerts, retention + privacy
  docs, OpenAPI.**

Each phase ships with tests + `flutter analyze`/`go vet`/`go test` green and
updates `AGENTS.md`. Migrations are forward-only; no phase enables a stricter
verification default than the current behavior without an explicit decision
in §15.
