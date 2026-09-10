#!/usr/bin/env bash
# ==============================================================================
# POS SaaS — Ubuntu/Ubuntu-derivative provisioner for the Docker-based stack.
# Installs Docker Engine + Compose v2 (all the container requirements to run the
# portable dev or production stack), then offers to bring up a stack for you.
#
# Usage:  sudo ./scripts/setup_ubuntu.sh
# ==============================================================================
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
err()  { echo -e "${RED}[ERROR]${NC} $1"; }

# --- Pre-flight checks --------------------------------------------------------
if [ "$(id -u)" -ne 0 ] && ! command -v sudo >/dev/null 2>&1; then
    err "Run with root or sudo privileges."
    exit 1
fi
SUDO=""
[ "$(id -u)" -ne 0 ] && SUDO="sudo"

source /etc/os-release
if [ "$ID" != "ubuntu" ] && [ "$ID" != "debian" ]; then
    err "This script targets Ubuntu (or Debian). Detected: $ID."
    exit 1
fi
ARCH=$(dpkg --print-architecture)
case "$ARCH" in amd64|arm64) ;; *)
    err "Unsupported architecture: $ARCH (expected amd64 or arm64)."
    exit 1
;; esac
log "Detected ${PRETTY_NAME:-$ID} ($ARCH)."

# --- Docker already present? ---------------------------------------------------
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    log "Docker $("docker --version") with Compose v2 already installed."
else
    # --- Install Docker Engine + Compose plugin from the official repo ----------
    log "Installing prerequisites..."
    $SUDO apt-get update
    $SUDO apt-get install -y ca-certificates curl gnupg

    $SUDO install -m 0755 -d /etc/apt/keyrings
    if [ ! -f /etc/apt/keyrings/docker.asc ]; then
        curl -fsSL "https://download.docker.com/linux/$ID/gpg" | \
            $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.asc
        $SUDO chmod a+r /etc/apt/keyrings/docker.asc
    fi

    CODENAME="${VERSION_CODENAME:-$(lsb_release -cs)}"
    if [ ! -f /etc/apt/sources.list.d/docker.list ]; then
        echo "deb [arch=$ARCH signed-by=/etc/apt/keyrings/docker.asc] \
https://download.docker.com/linux/$ID $CODENAME stable" | \
            $SUDO tee /etc/apt/sources.list.d/docker.list > /dev/null
    fi

    log "Installing Docker Engine + Compose v2..."
    $SUDO apt-get update
    $SUDO apt-get install -y docker-ce docker-ce-cli containerd.io \
        docker-buildx-plugin docker-compose-plugin

    $SUDO systemctl enable --now docker
    log "Docker installed: $($SUDO docker --version)"
fi

# --- Let the invoking user talk to the daemon without sudo ---------------------
CALLER_USER="${SUDO_USER:-$USER}"
if [ -n "$CALLER_USER" ] && ! id -nG "$CALLER_USER" | grep -qw docker; then
    log "Adding '$CALLER_USER' to the 'docker' group (re-login required for it to apply)."
    $SUDO usermod -aG docker "$CALLER_USER"
else
    log "User '$CALLER_USER' is already in the 'docker' group."
fi

# --- Optional: bring up a stack -------------------------------------------------
echo
echo -e "${CYAN}Choose a stack to start now:${NC}"
echo "  1) Development (pos-dev: postgres + redis + migrate + api)"
echo "  2) Production  (pos-prod: postgres + redis + migrate + api + caddy)"
echo "  3) Skip — I'll start it later"
read -rp "Enter choice [1-3]: " choice

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

case "$choice" in
    1)
        warn "If your user was just added to 'docker', re-login then run:"
        warn "  ./scripts/docker-dev.sh"
        $SUDO docker compose -f deployments/docker/docker-compose.dev.yml up -d --build || true
        ;;
    2)
        if [ ! -f .env.prod ]; then
            cp .env.prod.example .env.prod
            JWT_ACCESS_SECRET=$($SUDO openssl rand -hex 32)
            JWT_REFRESH_SECRET=$($SUDO openssl rand -hex 32)
            POSTGRES_PASSWORD=$($SUDO openssl rand -hex 24)
            $SUDO sed -i \
                -e "s|^JWT_ACCESS_SECRET=.*|JWT_ACCESS_SECRET=$JWT_ACCESS_SECRET|" \
                -e "s|^JWT_REFRESH_SECRET=.*|JWT_REFRESH_SECRET=$JWT_REFRESH_SECRET|" \
                -e "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=$POSTGRES_PASSWORD|" \
                .env.prod
            log "Generated random secrets for .env.prod. Review it before going public."
        fi
        warn "Deploying; set SITE_DOMAIN in .env.prod to get automatic TLS via Caddy."
        $SUDO docker compose --env-file .env.prod \
            -f deployments/docker/docker-compose.prod.yml up -d --build || true
        ;;
    *) log "Skipped. Start later with ./scripts/docker-dev.sh or ./scripts/docker-prod.sh." ;;
esac

echo
log "Done. Docker requirements are in place on $PRETTY_NAME."