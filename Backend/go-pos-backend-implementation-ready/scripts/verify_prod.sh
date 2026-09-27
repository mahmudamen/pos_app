#!/bin/bash
# Smoke-test the POS production hosts. Run on the VPS:
#   bash /usr/local/bin/pos-verify.sh
# Exits non-zero on the first failed check.
set -euo pipefail

base="https://posgo.xamltech.com"
api="https://api.xamltech.com"

say() { echo "[pos-verify] $*"; }

ok_url() {
  local url="$1"
  local code
  code=$(curl -sS -o /dev/null -m 15 -w '%{http_code}' "$url" || echo 000)
  if [[ "$code" == 200 ]]; then
    say "OK  200  $url"
  else
    say "ERR $code  $url"
    exit 1
  fi
}

post_login() {
  local h="$1"
  local body='{"tenant_id":"demo-book-store","email":"admin@demo-book-store.com","password":"admin","device_id":"verify-health","device_name":"verify-health"}'
  local code
  code=$(curl -sS -o /dev/null -m 20 -X POST "https://${h}.xamltech.com/v1/auth/login" \
    -H 'content-type: application/json' -d "$body" -w '%{http_code}' || echo 000)
  if [[ "$code" == 200 ]]; then
    say "OK  200  ${h}.xamltech.com/v1/auth/login"
  else
    say "ERR $code  ${h}.xamltech.com/v1/auth/login"
    exit 1
  fi
}

ok_url "$base/health/live"
ok_url "$base/"
ok_url "$base/private"
ok_url "$base/pricing"
ok_url "$base/delete-account"
ok_url "$api/health/live"
ok_url "$api/admin/"
ok_url "$api/v1/meta/countries"
post_login posgo
post_login api

say "All checks passed."
