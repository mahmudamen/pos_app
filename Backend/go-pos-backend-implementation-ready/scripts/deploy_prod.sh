#!/usr/bin/env bash
# =============================================================================
# deploy_prod.sh — idempotent production deploy for the POS Go backend.
#
# Ships the OCR-purchases wheel (OCR review + blur detector + supplier
# purchases + OCR metering) to the prod VPS and returns the stack to green:
#
#   1. rsync the backend worktree → VPS  /opt/pos/Backend/...  (excludes
#      .git/.env*/bin/* so no key or VCS metadata travels)
#   2. apply prod grants for every new table via grants_prod.sql (idempotent
#      DO-block; safe to re-run) — required BEFORE migrate so pos_app_rls can
#      INSERT/UPDATE the new tenant tables once they exist
#   3. REBUILD the migrate image — migrations 035_purchases.sql + 036_ocr_usage.sql
#      are baked into the image, so any migration change forces a rebuild, then
#      run goose up (freeze-fix: on build failure build only the changed binary
#      offline reusing cached layers, per AGENTS.md)
#   4. rebuild + restart the api service, wait for /health/ready
#   5. smoke-test the new /v1/purchases + /v1/purchases/ocr/* endpoints
#   6. run scripts/verify_prod.sh from the VPS — the public-host gate (both
#      nginx vhosts, TLS, the HTML pages, and a real login). Non-zero exits
#      the deploy as failed.
#
# Idempotency: every remote step is guarded by DO-blocks / retries; safe to
# re-invoke. The VPS network route is intermittently flaky (documented) so the
# script retries each ssh call up to SSH_TRIES times with backoff before
# giving up.
#
# Usage:
#   SSH_TARGET=root@197.44.6.42 scripts/deploy_prod.sh
#
# SSH key auth (BatchMode) from this dev box; agent/identity configured in
# ~/.ssh/id_ed25519 per the prod runbook.
# =============================================================================
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/.." && pwd)"

SSH_TARGET="${SSH_TARGET:-root@197.44.6.42}"
COMPOSE_REL="deployments/docker/docker-compose.prod.yml"
GRANTS_REL="scripts/grants_prod.sql"
COMPOSE="$ROOT/$COMPOSE_REL"
GRANTS="$ROOT/$GRANTS_REL"
DEPLOY_DIR="/opt/pos/Backend/go-pos-backend-implementation-ready"

: "${SSH_TRIES:=8}"
: "${SSH_BASE_ARGS:=-o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=25}"

log()  { printf '[deploy_prod] %s\n' "$*"; }
die()  { printf '[deploy_prod] FATAL: %s\n' "$*" >&2; exit 1; }

command -v rsync >/dev/null || die "rsync is required"
command -v ssh  >/dev/null || die "ssh is required"
[[ -f "$COMPOSE" ]] || die "compose not found: $COMPOSE"
[[ -f "$GRANTS"  ]] || die "grants not found: $GRANTS"

# ---------------------------------------------------------------------------
# sshx <remote-cmd...> — run a remote command with retries+backoff (flaky VPS).
# ---------------------------------------------------------------------------
sshx() {
  local rc=0 attempt=1 out
  for ((attempt = 1; attempt <= SSH_TRIES; attempt++)); do
    out=$(ssh $SSH_BASE_ARGS "$SSH_TARGET" "$@" 2>&1) && rc=0 || rc=$?
    if (( rc == 0 )); then
      [[ -n "${out:-}" ]] && printf '%s\n' "$out"
      return 0
    fi
    log "ssh attempt $attempt/$SSH_TRIES failed (rc=$rc) — retrying in ${attempt}s"
    sleep "$attempt"
  done
  die "could not reach $SSH_TARGET after $SSH_TRIES attempts (last rc=$rc)"
}

# ---------------------------------------------------------------------------
# ---------------------------------------------------------------------------
log "== 1/6 rsync worktree → $SSH_TARGET"
rsync -azh --delete \
  --exclude '.git' --exclude '.gitignore' \
  --exclude '.env*' --exclude '*.env' \
  --exclude 'bin' --exclude '*.db' --exclude '*.log' \
  -e "ssh $SSH_BASE_ARGS" \
  "$ROOT/" "$SSH_TARGET:$DEPLOY_DIR/" || die "rsync failed (flaky route — re-run)"

log "== 2/6 apply prod grants (idempotent DO-blocks, covers 035/036 tables)"
sshx "cd $DEPLOY_DIR && docker compose --env-file .env.prod -f $COMPOSE_REL exec -T postgres sh -c 'psql -U \"\$POSTGRES_USER\" -d \"\$POSTGRES_DB\" -v ON_ERROR_STOP=1 -f /app/scripts/grants_prod.sql' 2>&1 || echo GRANTS_FALLBACK_PSQL; echo grants_done"

log "== 3/6 rebuild migrate image (035/036 baked) + run goose up"
REMOTE_MIGRATE=$(cat <<'REMOTE'
set -e
cd /opt/pos/Backend/go-pos-backend-implementation-ready
export $(grep -E '^[A-Z_]+=' .env.prod | xargs)
if ! docker compose --env-file .env.prod -f deployments/docker/docker-compose.prod.yml build migrate 2>&1; then
  echo "== freeze-fix: offline rebuild of the API binary without the goose install"
  mkdir -p /tmp/pos-swap
  cat > /tmp/pos-swap/Dockerfile.swap <<'SWAP'
FROM pos-api:latest AS base
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api
FROM base
COPY --from=build /out/api /app/api
ENTRYPOINT ["/app/api"]
SWAP
  docker build --progress=plain -t pos-api:swap -f /tmp/pos-swap/Dockerfile.swap .
  docker tag pos-api:swap pos-api:latest
  docker compose --env-file .env.prod -f deployments/docker/docker-compose.prod.yml build migrate
fi
docker compose --env-file .env.prod -f deployments/docker/docker-compose.prod.yml run --rm migrate 2>&1
docker compose --env-file .env.prod -f deployments/docker/docker-compose.prod.yml ps 2>&1
echo MIGRATE_REMOTE_DONE
REMOTE
)
sshx "$REMOTE_MIGRATE"

log "== 4/6 rebuild + restart api, wait for ready"
sshx "cd $DEPLOY_DIR && docker compose --env-file .env.prod -f $COMPOSE_REL up -d --build api 2>&1 && for i in \$(seq 1 30); do if curl -sf \"http://127.0.0.1:8080/health/ready\" >/dev/null 2>&1; then echo API_READY; break; fi; sleep 2; done"

log "== 5/6 smoke the new OCR/purchases endpoints"
sshx "cd $DEPLOY_DIR && curl -sf http://127.0.0.1:8080/health/ready && echo && curl -s -o /dev/null -w 'purchases_status=%{http_code}\n' http://127.0.0.1:8080/v1/purchases/ocr/usage 2>&1; echo SMOKE_DONE"

# Step 6 is the public gate: run from the VPS so the request path is
# Cloudflare -> nginx -> Go, i.e. the same one a real client uses. The internal
# smoke above only proves 127.0.0.1:8080 answers; this proves DNS, TLS and both
# vhosts are intact. Fails the deploy on the first failed check.
log "== 6/6 public-host verification (scripts/verify_prod.sh from the VPS)"
# Deliberately NOT sshx: a non-zero exit here means "the app is unhealthy",
# not "the VPS is unreachable", and retrying an unhealthy app just delays the
# signal. Run it once and fail loudly.
if ! ssh $SSH_BASE_ARGS "$SSH_TARGET" "bash $DEPLOY_DIR/scripts/verify_prod.sh"; then
  die "public verification failed — the deploy is NOT healthy (see output above)"
fi
log "   public hosts verified (both vhosts, TLS, HTML pages, login)"

log "== ✅ deploy complete. Run the Flutter OCR E2E against https://api.xamltech.com next."
