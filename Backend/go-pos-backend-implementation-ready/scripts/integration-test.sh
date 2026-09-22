#!/usr/bin/env bash
# E3: run the DB-backed Go test suites against a real PostgreSQL + Redis in
# an ephemeral Docker Compose stack. Requires Docker + Compose v2.
#
# Usage:
#   ./scripts/integration-test.sh           # go test ./...
#   RACE=1 ./scripts/integration-test.sh    # go test -race ./... (slower)
#   ./scripts/integration-test.sh -run TestCatalogIntegration ./internal/transport/...
set -euo pipefail

cd "$(dirname "$0")/.."
COMPOSE=(docker compose -f deployments/docker/docker-compose.integration.yml)
export TEST_DATABASE_URL="${TEST_DATABASE_URL:-postgres://pos_app:pos_app_test_password@127.0.0.1:15432/pos_test}"

"${COMPOSE[@]}" up -d --wait
trap '"${COMPOSE[@]}" down >/dev/null 2>&1' EXIT

args=("$@")
if [ "${#args[@]}" -eq 0 ]; then
  args=(./...)
fi

echo "TEST_DATABASE_URL=$TEST_DATABASE_URL"
if [ "${RACE:-0}" = "1" ]; then
  go test -race "${args[@]}"
else
  go test "${args[@]}"
fi