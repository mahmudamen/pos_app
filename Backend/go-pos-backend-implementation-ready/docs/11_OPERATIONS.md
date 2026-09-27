# Operations

## Development

PostgreSQL and Redis run through Docker Compose.

## Production topology

```text
Internet
   |
 nginx (host TLS termination, certbot, two vhosts)
   |
   +-- api.xamltech.com   :80  --> /.well-known/acme-challenge/ -> /var/www/certbot
   |                       :443 --> /admin/  -> /var/www/pos-admin/  (React admin SPA)
   |                                everything else -> 127.0.0.1:8080
   |
   +-- posgo.xamltech.com :80  --> /.well-known/acme-challenge/ -> /var/www/certbot
                           :443 --> /apk/   -> /var/www/apk/          (release APK)
                                    everything else -> 127.0.0.1:8080
   |
 Go POS API (127.0.0.1:8080)
   |
  +---------+
  |         |
PostgreSQL Redis
```

PostgreSQL is authoritative.

Redis failure should not corrupt business data.

**The edge is host nginx, not the `caddy` service.** `docker-compose.prod.yml`
still defines `caddy` (it is the edge for a containerised install), but on the
production VPS only `postgres`, `redis`, `migrate` and `api` run — the vhosts in
`/etc/nginx/sites-enabled/` terminate TLS. The live per-host config for the POS
surface is committed at `deployments/nginx/posgo.xamltech.conf` (the console
vhost lives only on the box, in `api.xamltech.conf`). Certificates come from
`certbot certonly --nginx`, and `certbot.timer` renews them; check a renewal with
`certbot renew --cert-name <host> --dry-run`. A new vhost needs its port-80
block (ACME path + 301) first, then the cert, then the 443 block — `nginx -t`
fails while the certificate files are missing.

### Public HTML pages

`GET /`, `GET /pricing`, `GET /private` and `GET /delete-account` are generated
by the API itself (`internal/transport/server/pages.go`, `registerSitePages`) —
no auth, no DB. `GET /` is the branded POS.Go landing (icon, "Admin sign in" →
`/admin/`, privacy link); `GET /private` is the privacy policy the Android app
and the Play listing point at; `GET /delete-account` is the public
data-deletion request page Google Play requires for apps that can create an
account. Because nginx proxies everything except `/admin/`, `/apk/` and
`/.well-known/`, adding a page requires only a Go deploy — no nginx change.

## Post-deploy verification

`scripts/verify_prod.sh` smoke-tests both production hosts from the outside
edge: `/health/live`, the landing, `/pricing`, `/private`, `/delete-account`,
the console SPA, `/v1/meta/countries`, and a real `POST /v1/auth/login` against
`posgo.xamltech.com` and `api.xamltech.com`. It exits non-zero on the first
failure, so it works as a deploy gate:

```bash
# on the VPS
curl -fsSL -o /usr/local/bin/pos-verify.sh \
  https://raw.githubusercontent.com/<org>/<repo>/main/Backend/go-pos-backend-implementation-ready/scripts/verify_prod.sh
bash /usr/local/bin/pos-verify.sh
```

It needs no database, no SSH and no secrets — the login it performs uses the
public `demo-book-store` tenant. Run it after every `compose up -d api` and
after any nginx reload or certificate change.

`scripts/deploy_prod.sh` does it for you: it is **step 6/6** and a failure
aborts the deploy with `FATAL: public verification failed`, so a deploy can
never report success while DNS, TLS or a vhost is broken. It is a plain `ssh`,
not the retrying `sshx` helper, because a non-zero exit means "unhealthy app",
not "unreachable VPS" — retrying only delays the signal.

```bash
# full deploy from the dev box — step 6 runs the gate automatically
SSH_TARGET=root@197.44.6.42 scripts/deploy_prod.sh

# nginx or cert change, no Go rebuild
ssh root@197.44.6.42 'nginx -t && systemctl reload nginx && pos-verify.sh'
```

If you deployed by hand (rsync + `compose up -d api`), run the gate yourself
before calling it done — locally on the dev box with `make verify-prod`, or on
the box with `pos-verify.sh`.

## Deployment

1. Build immutable application artifact.
2. Apply backward-compatible migrations.
3. Deploy application.
4. Check readiness.
5. Monitor error rate/latency.
6. Roll back application if required.
7. Never roll back destructive schema changes blindly.

### Offline image rebuild when the VPS can't reach Go proxy / Docker Hub

The VPS route to `proxy.golang.org` (and Docker Hub) intermittently
TLS-timeouts/connection-resets on fresh fetches, and the Dockerfile's
`go install github.com/pressly/goose/v3/cmd/goose@v3.21.1` (goose is **not** in
go.mod) re-downloads on every build. For **api-only** deploys (no migration
files changed), build just the binary offline by reusing the cached layers:

```bash
# on the VPS, in the repo checkout:
cat > /tmp/Dockerfile.swap <<'EOF'
# syntax=docker/dockerfile:1
FROM golang:1.27-alpine@sha256:8a5910f31396cd4d89662f56c68b3ae31d374308270a1c3bd96672ee5ed43414 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download          # cache hit → no network
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/pos-api ./cmd/api

FROM pos-api:latest AS runtime   # old image already has goose + migrations
COPY --from=build /out/pos-api /app/pos-api
USER pos-api
EOF
docker build --progress=plain -t pos-api:swap -f /tmp/Dockerfile.swap .
docker tag pos-api:swap pos-api:latest && docker image rm pos-api:swap
docker compose --env-file .env.prod -f deployments/docker/docker-compose.prod.yml up -d api
```

Notes:

- Keep `WORKDIR /src`, `COPY go.mod go.sum ./`, `RUN go mod download` byte-for-byte
  identical to the real Dockerfile — otherwise buildkit misses the cached layer.
- Do **not** pass `--network=host`: RUN-step cache keys differ under host
  networking, forcing the flaky download again.
- The migrate service (`goose up`) can be re-verified as a no-op afterwards:
  `docker compose run --rm migrate` should print
  `goose: no migrations to run. current version: <N>`.

### Fresh Ubuntu VPS (first boot)

One-shot provisioner: installs Docker+Compose v2, Go 1.27, python3, `psql`, ufw,
clones `POS_BRANCH`, generates `.env.prod` (random JWT + DB secrets), starts the
whole production stack and health-checks it. Idempotent.

```bash
# On the VPS, as root (recommended):
POS_REPO_URL=https://github.com/<you>/pos.git POS_BRANCH=backend-deploy \
    SITE_DOMAIN=pos.example.com \
    sudo curl -fsSLo /tmp/provision_vps.sh \
    https://raw.githubusercontent.com/<you>/pos/backend-deploy/Backend/go-pos-backend-implementation-ready/scripts/provision_vps.sh \
  && sudo -E bash /tmp/provision_vps.sh
```

After that: set a real `SITE_DOMAIN` in `.env.prod` if not provided, confirm
`/health/ready` (needs Redis — the stack runs it), schedule
`scripts/backup.sh` on cron (`0 3 * * * ...`), and verify a restore once (below).

### Three public domains (surface routing)

Production publishes three hostnames from the same API container, and the API
enforces the split itself (`SURFACE_ROUTING_ENABLED=true`):

| `Host` | Served |
|---|---|
| `xamltech.com` | company site (`/`, `/pricing`, `/private`). Every `/v1/*` is 404. |
| `api.xamltech.com` | `/v1/auth/*`, `/v1/meta/*`, `/v1/saas/*`, `/v1/platform/*`, `/admin/` static app, `/` → `/admin/`. Sign-in requires `saas_admin`. |
| `posgo.xamltech.com` | `/v1/*` store API (except the control plane), landing, `/selforder`, `/sw.js`, `/apk`. Sign-in rejects `saas_admin`. |

Operational consequences:

- Point all three DNS A/AAAA records at the same host and keep
  `SURFACE_ROUTING_ENABLED=true` in `.env.prod`; the Caddyfile serves each
  hostname separately and the app re-checks the `Host` header, so an upstream
  that forwards the wrong vhost cannot leak a surface.
- `unknown_host` (404) means a request arrived on a hostname that is none of the
  three — usually a stale vhost or a typo in DNS. Loopback and the Compose
  service names bypass the check, so `/health/*` and the container-to-container
  hop keep working.
- `not_available_on_host` (404) means the client called a path belonging to
  another surface: a POS build pointed at `api.`, or the console pointed at
  `posgo.`. Fix the client's `API_BASE_URL`, not the server.
- `wrong_surface` (403) is a credentials problem, not a routing one: the account
  exists but belongs to the other audience. Store staff sign in on
  `posgo.`, platform operators on `api.`.
- Keep the routing flag `false` for a single-domain deployment (a staging box with
  one hostname), otherwise every host must be listed.

Hosts that terminate TLS with nginx use
`deployments/nginx/xamltech_surfaces.conf.example` (the containerized stack uses
`deployments/caddy/Caddyfile`). Both are examples: nothing here touches a live
server, and the vhosts still need real certificates and a real ACME location.

Publishing `posgo.xamltech.com` through Cloudflare (DNS record, proxy status,
SSL mode, the edge rules that break a native POS client, origin rollout,
verification, rollback): `docs/25_CLOUDFLARE_POSGO.md`.

See `scripts/provision_vps.sh` for the full env-var switches (`SKIP_*` etc.).

## Backups & Restore runbook (OPS-006)

Backups must be automated and periodically restored into an isolated environment.

A backup that has never been restore-tested is not considered verified.

### Backup (logical, pg_dump)

`scripts/backup.sh` streams a gzipped logical dump (`--no-owner --no-privileges`)
to `$BACKUP_DIR` (default `./backups`) and prunes files older than
`$BACKUP_KEEP_DAYS` (default 14).

```bash
set -a; source .env; set +a        # provides DATABASE_URL
./scripts/backup.sh                # -> backups/pos-<UTC stamp>.sql.gz
```

Schedule daily with cron (`crontab -e`):

```cron
30 2 * * * cd /srv/pos && . .env >/dev/null 2>&1 && ./scripts/backup.sh >>/var/log/pos-backup.log 2>&1
```

Notes:

- `--no-owner/--no-privileges` keep roles out of the dump; tenant RLS and the
  `saas` platform tenant are restored as-is.
- On a multi-instance deployment, run exactly one backup writer to avoid
  concurrent dumps.
- Copy the tarball off-host (object storage / another disk). Retention only
  applies inside `$BACKUP_DIR`.

### Restore

```bash
RESTORE_TARGET_URL=postgres://pos_app:...@db-host:5432/pos ./scripts/restore.sh backups/pos-<stamp>.sql.gz
```

The target database must already exist (e.g. `createdb pos_restore_test`).

### Restore verification (mandatory)

A backup is only "verified" when it successfully restores **and** passes a smoke
test in an isolated database:

1. `createdb pos_restore_test`.
2. Restore into it (URL above, changing the database name).
3. `SELECT count(*) FROM tenants;` — expect the same number as the source.
4. Boot the API against the restored DB and run one login + one sale.
5. Confirm a POS device re-pulls from `change_seq 0` (`GET /v1/sync/pull`) and
   converges with the restored snapshot.

### Rollback

- Application-only rollback is safe any time: keep the previous immutable build
  and redeploy it; PostgreSQL authored the data, migrations are forward-only.
- Never roll back a destructive schema change blindly — that requires a restore
  from the last verified backup taken before the migration.
- Back up *before* running `make migrate` on production: `./scripts/backup.sh`.

## Monitoring

Monitor:

- request rate;
- 4xx/5xx rate;
- latency p50/p95/p99;
- database pool exhaustion;
- database errors;
- Redis errors;
- authentication failures;
- sync failures;
- process restarts.

## Security

Production requires:

- TLS;
- restricted database network access;
- least-privilege DB credentials;
- secret management;
- explicit CORS;
- firewall policy;
- OS patching;
- log retention policy.

## Secret rotation runbook

Rotating secrets invalidates issued tokens: set the new secret, restart the
API, and the old access/refresh tokens stop verifying at a blocking
`security.ErrInvalidToken`. Plan a low-traffic window.

1. Generate replacements offline, >32 bytes, unique per environment:
   `openssl rand -base64 48`. Do not reuse secrets across environments.
2. Update the environment's secret store (`.env.prod` / deployment secrets).
3. Restart the API; verify `/health/live` and `/health/ready` pass.
4. Rotate **refresh tokens with care**: an active refresh token signed with the
   old secret can no longer be validated — clients on `JWT_REFRESH_TTL` (7d
   default) must re-login or handle the 401. Roll a rotation out with a
   client-visible announcement for managed devices; `remembered` logins on the
   Flutter app prompt for credentials again.
5. Back up the old secret value (encrypted, offline) for incident forensics,
   then destroy the working copy.

### Compromised-token revocation

- A leaked access token dies with its `JWT_ACCESS_TTL` (15m default); raising
  the rate limit and rotating both secrets is the emergency stop.
- Revoke a *refresh* token (and its access family) by logging the user out —
  the session row in `auth_sessions` is deleted server-side and blocked at the
  `auth.CheckSession` guard, *before* any JWT validation passes.
- After an account compromise: rotate both JWT secrets, drop the user's
  sessions (`DELETE FROM auth_sessions WHERE user_id = <id>;`), issue a
  password reset, and audit the sync `change_seq` feed for that tenant for
  anomalous writes.
- Never put passwords, JWT secrets, or refresh tokens in logs (structured
  JSON logger writes request IDs and error codes only).
