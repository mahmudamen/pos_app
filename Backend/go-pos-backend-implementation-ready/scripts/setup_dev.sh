#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "ERROR: missing command: $1" >&2
    exit 1
  }
}

need_cmd go
need_cmd docker

if ! docker compose version >/dev/null 2>&1; then
  echo "ERROR: Docker Compose v2 is required." >&2
  exit 1
fi

if [ ! -f .env ]; then
  cp .env.example .env
  echo "Created .env from .env.example"
fi

go version
go mod download

docker compose -f deployments/docker/docker-compose.yml up -d postgres redis

echo "Waiting for PostgreSQL..."
for i in {1..30}; do
  if docker compose -f deployments/docker/docker-compose.yml exec -T postgres pg_isready -U pos_app -d pos >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "Waiting for Redis..."
for i in {1..30}; do
  if docker compose -f deployments/docker/docker-compose.yml exec -T redis redis-cli ping >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

echo "Development environment is ready."
echo "Next:"
echo "  set -a; source .env; set +a"
echo "  make migrate"
echo "  make run"
