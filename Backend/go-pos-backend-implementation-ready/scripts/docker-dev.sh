#!/usr/bin/env bash
# ==============================================================================
# POS SaaS — bring up the fully-containerized DEVELOPMENT stack.
# Requires only Docker + Compose v2 (see scripts/setup_ubuntu.sh).
# Everything (postgres, redis, goose migrations, api) runs in containers.
# Source is bind-mounted, so 'docker compose restart api' reloads code changes.
#
# Usage:  ./scripts/docker-dev.sh
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
COMPOSE_FILE=deployments/docker/docker-compose.dev.yml

command -v docker >/dev/null 2>&1 || {
    echo "ERROR: Docker is not installed. Run ./scripts/setup_ubuntu.sh first." >&2
    exit 1
}
docker compose version >/dev/null 2>&1 || {
    echo "ERROR: Docker Compose v2 is required." >&2
    exit 1
}

if [ ! -f .env ]; then
    cp .env.example .env
    echo "Created .env from .env.example"
fi

docker compose -f "$COMPOSE_FILE" up -d --build

echo
echo "Development stack is up:"
echo "  API        http://localhost:8080/v1  (health: /health/ready)"
echo "  PostgreSQL 127.0.0.1:5432"
echo "  Redis      127.0.0.1:6379"
echo
echo "Reload code:  docker compose -f $COMPOSE_FILE restart api"
echo "Stop stack:   docker compose -f $COMPOSE_FILE down"
echo "Logs:         docker compose -f $COMPOSE_FILE logs -f api"