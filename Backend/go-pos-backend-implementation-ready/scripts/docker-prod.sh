#!/usr/bin/env bash
# ==============================================================================
# POS SaaS — deploy the PRODUCTION stack with Docker + Compose v2.
# Combines a one-shot goose migration with an immutable API image and Caddy.
#
# Usage:  ./scripts/docker-prod.sh
#
# First deployment: it creates .env.prod from .env.prod.example and generates
# random DB/JWT secrets automatically. Set SITE_DOMAIN to your real domain to
# get automatic TLS from Caddy, then re-run this script.
# ==============================================================================
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
COMPOSE_FILE=deployments/docker/docker-compose.prod.yml

command -v docker >/dev/null 2>&1 || {
    echo "ERROR: Docker is not installed. Run ./scripts/setup_ubuntu.sh first." >&2
    exit 1
}
docker compose version >/dev/null 2>&1 || {
    echo "ERROR: Docker Compose v2 is required." >&2
    exit 1
}

if [ ! -f .env.prod ]; then
    cp .env.prod.example .env.prod
    JWT_ACCESS_SECRET=$(openssl rand -hex 32)
    JWT_REFRESH_SECRET=$(openssl rand -hex 32)
    POSTGRES_PASSWORD=$(openssl rand -hex 24)
    sed -i \
        -e "s|^JWT_ACCESS_SECRET=.*|JWT_ACCESS_SECRET=$JWT_ACCESS_SECRET|" \
        -e "s|^JWT_REFRESH_SECRET=.*|JWT_REFRESH_SECRET=$JWT_REFRESH_SECRET|" \
        -e "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$POSTGRES_PASSWORD|" \
        .env.prod
    echo "Created .env.prod with random secrets."
    echo "IMPORTANT: edit .env.prod and set SITE_DOMAIN (and CORS) before going public."
fi

docker compose --env-file .env.prod -f "$COMPOSE_FILE" up -d --build

echo
echo "Production stack deployed:"
echo "  API   http://127.0.0.1:${API_PORT:-8080}/health/ready"
echo "  TLS   https://${SITE_DOMAIN:-localhost}   (Caddy)"
echo
echo "Logs:   docker compose --env-file .env.prod -f $COMPOSE_FILE logs -f api"
echo "Status: docker compose --env-file .env.prod -f $COMPOSE_FILE ps"
echo "Update: re-run ./scripts/docker-prod.sh"