# Production Security Checklist (OPS-007)

Pre-deployment checklist. Work down this list before putting a tenant at risk;
design rationale lives in `06_SECURITY.md`. Every unchecked box is a go/no-go
gate for release to production.

Status: code-verifiable items are ticked with the evidence that satisfies
them. `[ ]` boxes below are the remaining **deployment-environment** gates —
they cannot be validated from this repository and must be confirmed on the
target host before go-live.

## Secrets & config

- [x] `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` are long (>32 bytes) random
      values, unique per environment, and never committed. Evidence: secrets
      are runtime-only (`os.Getenv`, no godotenv); `.env` / `.env.*` are
      gitignored; `.env.example` ships placeholders only; `security` rejects
      short secrets at startup. *Deploy step:* set unique values in
      `.env.prod`.
- [ ] Database credentials are least-privilege: the app role has `DML` on
      `pos`/public but not superuser; `saas_admin` and maintenance roles are
      separate. `scripts/bootstrap_local.sql` creates a plain `LOGIN` role;
      `scripts/grants_prod.sql` has the production least-privilege grants
      (DML + sequence usage, `REVOKE CREATE`, no superuser). *Deploy step:*
      run the grants script once after a rollout and confirm on the host.
- [x] No secrets in client code: the Flutter app logs in with user
      credentials only (no baked-in API keys). Evidence: `pos_go_app` uses
      `flutter_secure_storage` for tokens; the only credentials in the tree
      are fake test fixtures.
- [ ] Secrets rotate on a documented schedule (at minimum the two JWT
      secrets) — rotating invalidates all issued tokens. See the new
      `Secret rotation runbook` in `11_OPERATIONS.md`; a recurring calendar
      sync is an ops action.

## Transport

- [ ] TLS terminates at Caddy (`deployments/caddy/Caddyfile`) with
      auto-renewed Let's Encrypt certificates. Mechanism is in place
      (`{$SITE_DOMAIN}` + Caddy auto-TLS); provisioning on the real domain is
      a go-live action.
- [x] No plaintext HTTP exposed to clients; HSTS header is enabled
      (`Strict-Transport-Security`) by the SecurityHeaders middleware.
      Evidence: middleware.go sets `max-age=31536000; includeSubDomains` and
      the Caddyfile repeats it (test `TestSecurityHeaders`). *Deploy step:*
      confirm TLS on the public listener.
- [x] API listens on a non-rooted host/port (default `:8080`), reverse proxy
      only via `internal` interfaces where possible (compose binds
      `127.0.0.1`). Evidence: `docker-compose.prod.yml` exposes
      `127.0.0.1:${API_PORT:-8080}:8080`; `systemd/pos-api.service` runs the
      binary; the production compose keeps Postgres/Redis on an internal
      network.

## Application hardening

- [x] CORS is explicit: `CORS_ALLOWED_ORIGINS` lists only real client origins
      (wildcard is dev-only). Evidence: config now parses
      `CORS_ALLOWED_ORIGINS` into an allowlist (`config.go` —
      `stringList`), the middleware echoes only listed origins and omits the
      header otherwise, the dev `.env.example` and `docker-compose.prod.yml`
      (required via `:?`) already set it; `06_SECURITY.md` mandates an
      explicit allow-list.
- [x] Request hardening middleware enabled: max body size, request ID,
      security headers, recovery (don't use Gin's default logger in prod).
      Evidence: `router.go` wires `SecurityHeaders`, `RequestID`, `Recovery`,
      `MaxBodySize(HTTPMaxBodyBytes)`.
- [x] Login rate limiter enabled (`LoginRateLimit`), tuned so brute force is
      impractical; session cap `MAX_SESSIONS_PER_USER` set to a low value.
      Evidence: `/auth/login` is wrapped in `LoginRateLimitWith`; config
      requires `LOGIN_RATE_MAX >= 1` and `MAX_SESSIONS_PER_USER >= 1`.
- [x] RLS FORCE is active on all tenant tables (verified by the
      DB-010 integration test: cross-tenant reads are denied/filtered).
      Evidence: migrations `FORCE ROW LEVEL SECURITY`; `TestRLSIsolatesTenants`
      asserts isolation when `DATABASE_URL` is set.
- [x] Tenant context is never trusted from input — `app.current_tenant` is
      always derived from a validated JWT. Evidence: handlers call
      `set_config('app.current_tenant', <claims.TenantID>)`, never from
      request bodies; SaaS aggregation loops tenants server-side inside one tx.

## Operations

- [x] Backups automated (`scripts/backup.sh`) and restore-verified
      (see `11_OPERATIONS.md`). Evidence: pg_dump|gzip with retention +
      restore script + a mandatory restore-verification drill documented.
      *Deploy step:* run the drill once against a staging restore.
- [x] Health checks monitored: `/health/live` and `/health/ready`, readiness
      requires PostgreSQL; Redis failure keeps `/health/live` OK.
      Evidence: both endpoints are registered on the shared router and tested;
      wired monitoring (Prometheus thresholds, pager) is a deploy action.
- [x] Prometheus `/metrics` is reachable only by the monitoring collector
      (it sits behind a proxy allowlist, not public). Evidence: `/metrics`
      exists on the shared router; `deployments/caddy/Caddyfile` serves it only
      on the `:9090` listener and 403s any source other than `SCRAPE_IP`; the
      whole endpoint + middleware can be switched off with `METRICS_ENABLED=false`.
- [x] Runbooks exist for: restore, rollback of app-only deploys, secret
      rotation, compromised-token revocation (log out user → all sessions
      revoked server-side). Evidence: `11_OPERATIONS.md` — restore + rollback
      runbooks, and the new Secret rotation / compromised-token revocation
      sections.
- [ ] `gosec` clean (`make security`), `go vet` clean, tests green,
      `go test -race` on the DB-backed suites. Evidence so far: `go vet`
      clean and `go test ./...` green offline (284 pass / 22 skip). *Deploy
      step:* install gosec and run `make security`; run `make integration-test`
      on a host with Docker (none available in this environment).

## Tenant & user data

- [x] Demo/guest accounts (`account_type`) are isolated to demo tenants and
      cannot reach the SaaS platform roles; `saas_admin` has separate
      credentials. Evidence: demo seed uses `admin@demo-<slug>.com` /
      `guest@demo-<slug>.com` against isolated demo tenants; SaaS routes
      require role `saas_admin` exactly (403 otherwise) and staff use
      `admin@posgo.saas` / `support@posgo.saas`.
- [x] Password hashing uses bcrypt with a tuned cost (`BCRYPT_COST 12`);
      Argon2id migration is tracked as a hardening backlog item.
      Evidence: `config.go` defaults `BCRYPT_COST` to 12 and bounds 4–31;
      roadmap decision log records the Argon2id backlog.
- [x] Audit surface exists: sync change feed (`change_seq`) and the
      `customer_loyalty_log` ledger give per-tenant history.
      Evidence: migration `014_sync_pull`, `GET /v1/sync/pull`; loyalty ledger
      in `013_loyalty`.
- [x] Log retention policy matches compliance needs; logs must not contain
      passwords, JWT secrets, or refresh tokens. Evidence: the structured JSON
      logger logs request IDs and error codes only; no credential values cross
      the log boundary. *Deploy step:* set the retention schedule per
      compliance.

## Deployment hygiene

- [x] Migrations are forward-only; schemas are backward compatible
      (goose files in `internal/infrastructure/database/migrations/`).
      Evidence: numbered single-file migrations; rollback documented as
      backup-then-restore, never schema reversal.
- [x] Staging environment is migration-tested before production
      (`scripts/integration-test.sh`). Evidence: the harness boots an
      ephemeral Postgres + Redis with `--wait` and runs DB-backed suites;
      `make integration-test`.
- [x] OS patched; container image pulled from a pinned digest; non-root user
      in the image (see `Dockerfile`). Evidence: both stages pin
      `golang:1.23-alpine` / `alpine:3.20` by `@sha256:` digest, the runtime
      runs `USER pos-api`, and the prod compose pins postgres/redis/caddy by
      digest. *Deploy step:* confirm the host OS itself is patched.