#!/usr/bin/env bash
# Finish the POS.Go production deploy: run migrations, then restart the API.
#
# The images are already built (compose build migrate api completed). This runs
# the two remaining steps in the order they must happen in: 050 repairs the
# dangling tenant plan codes, and the new binary refuses invoice capture for any
# plan that does not resolve. Migrating first is what keeps that from turning
# into a customer-facing 402.
#
# Run as root on the VPS, from the code directory:
#   cd /opt/pos/Backend/go-pos-backend-implementation-ready
#   bash deploy_finish.sh
#
# Add --dry-run to see what would run without touching anything.

set -euo pipefail

CODE_DIR="/opt/pos/Backend/go-pos-backend-implementation-ready"
DRY=0
[[ "${1:-}" == "--dry-run" ]] && DRY=1

cd "$CODE_DIR"

DBU=$(grep -m1 '^POSTGRES_USER=' .env.prod | cut -d= -f2-)
DBN=$(grep -m1 '^POSTGRES_DB=' .env.prod | cut -d= -f2-)
compose() { docker compose --env-file .env.prod -f deployments/docker/docker-compose.prod.yml "$@"; }
q() { docker exec pos-prod-postgres psql -U "$DBU" -d "$DBN" -tAc "$1"; }
say() { printf '\033[1m==>\033[0m %s\n' "$*"; }
info() { printf '  ----  %s\n' "$*"; }
bad() { printf '\033[31mERROR:\033[0m %s\n' "$*" >&2; }

# --- 1. Preflight -----------------------------------------------------------
say "Preflight"

for f in .env.prod deployments/docker/docker-compose.prod.yml; do
  [[ -f "$f" ]] || { bad "missing $f -- is this the right directory?"; exit 1; }
done

before_version=$(q "select coalesce(max(version_id),0) from goose_db_version where is_applied")
info "goose version before: $before_version"

# The whole point of 050. If this returns rows after migrating, the gate will
# 402 real customers, so it is worth stopping for.
dangling_before=$(q "select count(*) from tenants t left join plans p on p.code=t.plan where p.code is null")
info "tenants on an unresolvable plan code: $dangling_before"

# Back up first. Forward-only migrations are meant to be irreversible-safe, but
# "intended" is not the same as "checked".
if [[ $DRY -eq 0 ]]; then
  info "taking a pre-migration dump"
  docker exec pos-prod-postgres pg_dump -U "$DBU" -d "$DBN" \
    > "/root/pos-predeploy-$(date +%Y%m%d-%H%M%S).sql"
  info "dump written under /root/pos-predeploy-*.sql"
fi

# --- 2. Migrations ----------------------------------------------------------
if [[ $DRY -eq 1 ]]; then
  say "DRY RUN: would run: compose run --rm migrate"
else
  say "Running migrations"
  compose run --rm migrate
fi

after_version=$(q "select coalesce(max(version_id),0) from goose_db_version where is_applied")
info "goose version after: $after_version"

if [[ "$after_version" -lt 50 ]]; then
  bad "goose is at $after_version, expected 50. The new binary would serve the"
  bad "OCR gate against un-repaired plan codes. Not restarting the API."
  exit 1
fi
[[ $DRY -eq 1 ]] || info "migrations applied"

# --- 3. Verify the repair actually took ------------------------------------
say "Verifying the plan repair"
dangling_after=$(q "select count(*) from tenants t left join plans p on p.code=t.plan where p.code is null")
info "tenants on an unresolvable plan code: $dangling_after  (was $dangling_before)"

legacy_active=$(q "select count(*) from plans where code in ('standard','test') and is_active")
info "legacy default tiers marked sellable: $legacy_active  (must be 0)"

if [[ "$dangling_after" -ne 0 ]]; then
  bad "$dangling_after tenants still resolve to no plan. The OCR gate would"
  bad "refuse them. Not restarting the API."
  exit 1
fi
if [[ "$legacy_active" -ne 0 ]]; then
  bad "standard/test are is_active=true, which would put them on the public"
  bad "pricing page. Not restarting the API."
  exit 1
fi

ocr_missing=$(q "select count(*) from plans where not (features @> '[\"ocr_capture\"]'::jsonb)")
info "plan rows without ocr_capture: $ocr_missing  (0 is fine only for the inactive legacy rows)"
info "active plans: $(q "select string_agg(code, ',' order by code) from plans where is_active")"

# --- 4. Restart the API -----------------------------------------------------
if [[ $DRY -eq 1 ]]; then
  say "DRY RUN: would run: compose up -d api"
  say "Then: curl -fsS https://api.xamltech.com/health/live && curl -fsS .../health/ready"
  exit 0
fi

say "Restarting the API"
compose up -d api

# --- 5. Post-deploy checks --------------------------------------------------
say "Waiting for health"
for i in $(seq 1 30); do
  if curl -fsS --max-time 5 https://api.xamltech.com/health/live >/dev/null 2>&1; then
    info "live OK after ${i}s"
    break
  fi
  [[ $i -eq 30 ]] && { bad "API did not come up in 30s"; docker logs --tail 40 pos-prod-api; exit 1; }
  sleep 1
done

ready=$(curl -fsS --max-time 5 https://api.xamltech.com/health/ready || true)
info "ready: $ready"

# The OCR meter should now be live and bounded. Verified via env because the
# endpoint itself needs a tenant token we do not hold here.
info "OCR env in the running container:"
docker exec pos-prod-api printenv | grep '^OCR_' | sed 's/^/        /'

# A 402 here would mean the gate is live but the repair missed something.
status=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 -X POST \
  https://api.xamltech.com/v1/purchases/ocr || true)
case "$status" in
  401|403) info "unauthenticated OCR call -> $status (expected; route is registered and gated)" ;;
  402)     bad "OCR returned 402 to an unauthenticated call -- the gate is misbehaving" ;;
  404)     bad "OCR route is missing; the new binary is not serving" ;;
  *)       info "unauthenticated OCR call -> $status" ;;
esac

say "Done. Verify the site: https://api.xamltech.com/ and /pricing"
info "Confirm the pricing page still shows only: $(q "select string_agg(code, ', ' order by code) from plans where is_active")"
