# Production Security Checklist (OPS-007)

Pre-deployment checklist. Work down this list before putting a tenant at risk;
design rationale lives in `06_SECURITY.md`. Every unchecked box is a go/no-go
gate for release to production.

## Secrets & config

- [ ] `JWT_ACCESS_SECRET` / `JWT_REFRESH_SECRET` are long (>32 bytes) random
      values, unique per environment, and never committed (`.env` is in
      `.gitignore`).
- [ ] Database credentials are least-privilege: the app role has `DML` on
      `pos`/public but not superuser; `saas_admin` and maintenance roles are
      separate.
- [ ] No secrets in client code: the Flutter app logs in with user
      credentials only (no baked-in API keys).
- [ ] Secrets rotate on a documented schedule (at minimum the two JWT
      secrets) — rotating invalidates all issued tokens.

## Transport

- [ ] TLS terminates at Caddy (`deployments/caddy/Caddyfile`) with
      auto-renewed Let's Encrypt certificates.
- [ ] No plaintext HTTP exposed to clients; HSTS header is enabled
      (`Strict-Transport-Security`) by the SecurityHeaders middleware.
- [ ] API listens on a non-rooted host/port (default `:8080`), reverse proxy
      only via `internal` interfaces where possible (compose binds
      `127.0.0.1`).

## Application hardening

- [ ] CORS is explicit: `CORS_ALLOWED_ORIGINS` lists only real client origins
      (wildcard is dev-only).
- [ ] Request hardening middleware enabled: max body size, request ID,
      security headers, recovery (don't use Gin's default logger in prod).
- [ ] Login rate limiter enabled (`LoginRateLimit`), tuned so brute force is
      impractical; session cap `MAX_SESSIONS_PER_USER` set to a low value.
- [ ] RLS FORCE is active on all tenant tables (verified by the
      DB-010 integration test: cross-tenant reads are denied/filtered).
- [ ] Tenant context is never trusted from input — `app.current_tenant` is
      always derived from a validated JWT.

## Operations

- [ ] Backups automated (`scripts/backup.sh`) and restore-verified
      (see `11_OPERATIONS.md`).
- [ ] Health checks monitored: `/health/live` and `/health/ready`, readiness
      requires PostgreSQL; Redis failure keeps `/health/live` OK.
- [ ] Prometheus `/metrics` is reachable only by the monitoring collector
      (put it behind the proxy with an allowlist, not public).
- [ ] Runbooks exist for: restore, rollback of app-only deploys, secret
      rotation, compromised-token revocation (log out user → all sessions
      revoked server-side).
- [ ] `gosec` clean (`make security`), `go vet` clean, tests green,
      `go test -race` on the DB-backed suites.

## Tenant & user data

- [ ] Demo/guest accounts (`account_type`) are isolated to demo tenants and
      cannot reach the SaaS platform roles; `saas_admin` has separate
      credentials.
- [ ] Password hashing uses bcrypt with a tuned cost (`BCRYPT_COST 12`);
      Argon2id migration is tracked as a hardening backlog item.
- [ ] Audit surface exists: sync change feed (`change_seq`) and the
      `customer_loyalty_log` ledger give per-tenant history.
- [ ] Log retention policy matches compliance needs; logs must not contain
      passwords, JWT secrets, or refresh tokens.

## Deployment hygiene

- [ ] Migrations are forward-only; schemas are backward compatible
      (goose files in `internal/infrastructure/database/migrations/`).
- [ ] Staging environment is migration-tested before production
      (`scripts/integration-test.sh`).
- [ ] OS patched; container image pulled from a pinned digest; non-root user
      in the image (see `Dockerfile`).