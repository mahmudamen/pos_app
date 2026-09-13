#!/usr/bin/env bash
# ==============================================================================
# POS SaaS — Ubuntu VPS first-boot provisioner for the production Docker stack.
# Idempotent: safe to re-run after a failed or interrupted run; everything that
# is already in place is left untouched.
#
# What it does, in order:
#   1. Preflight    : root/sudo, Ubuntu/Debian, amd64/arm64, >=2 GB RAM
#   2. Baseline     : apt packages (curl, git, ca-certificates, gnupg, ufw,
#                     unattended-upgrades, python3+pip+venv, postgresql-client)
#   3. Go toolchain : go1.23.x from the official tarball -> /usr/local/go
#   4. Docker       : Docker Engine + Compose v2 + Buildx from the docker repo
#   5. Firewall     : ufw allow OpenSSH/80/443; enable
#   6. Requirements : verify docker, compose, go, python3, psql, git, openssl
#   7. Clone        : git clone -b $POS_BRANCH $POS_REPO_URL into $POS_HOME
#   8. .env.prod    : create from .env.prod.example, auto-generate secrets,
#                     set SITE_DOMAIN / CORS / SCRAPE_IP
#   9. Deploy       : docker compose up -d --build (migrate -> api -> caddy)
#  10. Verify       : wait for /health/live + /health/ready; print summary
#
# Usage (recommended — lets you clone straight from the branch):
#   POS_REPO_URL=https://github.com/you/pos.git POS_BRANCH=backend-deploy \
#       SITE_DOMAIN=pos.example.com \
#       sudo -E ./scripts/provision_vps.sh
#
# Or after the repo is already checked out (no clone needed — run from inside it):
#   sudo ./scripts/provision_vps.sh
#
# Overridable env (all optional):
#   POS_REPO_URL   git URL to clone from (skips clone if unset + already in a checkout)
#   POS_BRANCH     branch to clone            [default: backend-deploy]
#   POS_HOME       where to clone/expect repo [default: /opt/pos]
#   SITE_DOMAIN    public hostname (Caddy TLS) [default: localhost]
#   SCRAPE_IP      metrics collector source IP [default: 127.0.0.1]
#   CORS_ALLOWED   Origin allowlist for browsers [default: https://$SITE_DOMAIN]
#   GO_VERSION     Go toolchain version        [default: 1.23.12]
#   SKIP_PKGS=1    skip apt baseline package install
#   SKIP_GO=1      skip Go toolchain install
#   SKIP_DOCKER=1  skip Docker install
#   SKIP_UFW=1     skip firewall enable
#   SKIP_CLONE=1   skip clone/checkout step
#   SKIP_ENV=1     skip .env.prod generation (keeps existing)
#   SKIP_DEPLOY=1  skip compose up (just provision + configure)
# ==============================================================================
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

log()  { echo -e "${GREEN}[INFO]${NC}  $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC}  $1"; }
err()  { echo -e "${RED}[ERROR]${NC} $1"; }
step() { echo -e "\n${CYAN}==> ${NC}$1"; }

# --- Pre-flight ---------------------------------------------------------------
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

RAM_GB=$(awk '/MemTotal/ { printf "%.0f", $2/1024/1024 }' /proc/meminfo)
if [ "${RAM_GB:-0}" -lt 2 ]; then
    warn "Only ~${RAM_GB} GB RAM — the prod stack is happy on 2 GB, but resize if slow."
fi

# --- Config -------------------------------------------------------------------
POS_REPO_URL="${POS_REPO_URL:-}"
POS_BRANCH="${POS_BRANCH:-backend-deploy}"
POS_HOME="${POS_HOME:-/opt/pos}"
BACKEND_DIR="${POS_HOME}/Backend/go-pos-backend-implementation-ready"
SITE_DOMAIN="${SITE_DOMAIN:-localhost}"
SCRAPE_IP="${SCRAPE_IP:-127.0.0.1}"
CORS_ALLOWED="${CORS_ALLOWED:-https://${SITE_DOMAIN}}"
GO_VERSION="${GO_VERSION:-1.23.12}"
SKIP_PKGS="${SKIP_PKGS:-0}"
SKIP_GO="${SKIP_GO:-0}"
SKIP_DOCKER="${SKIP_DOCKER:-0}"
SKIP_UFW="${SKIP_UFW:-0}"
SKIP_CLONE="${SKIP_CLONE:-0}"
SKIP_ENV="${SKIP_ENV:-0}"
SKIP_DEPLOY="${SKIP_DEPLOY:-0}"

# --- 2. Baseline packages -----------------------------------------------------
if [ "$SKIP_PKGS" != "1" ]; then
    step "Installing baseline packages"
    $SUDO apt-get update
    DEBIAN_FRONTEND=noninteractive $SUDO apt-get install -y \
        ca-certificates curl gnupg git lsb-release \
        ufw unattended-upgrades \
        python3 python3-pip python3-venv \
        postgresql-client

    # Turn on automatic security updates (non-interactive, no reboot schedule).
    if [ -d /etc/apt/apt.conf.d ] && [ ! -f /etc/apt/apt.conf.d/20auto-upgrades ]; then
        echo 'APT::Periodic::Update-Package-Lists "1";' | $SUDO tee /etc/apt/apt.conf.d/20auto-upgrades >/dev/null
        echo 'APT::Periodic::Unattended-Upgrade "1";'   | $SUDO tee -a /etc/apt/apt.conf.d/20auto-upgrades >/dev/null
    fi
else
    log "Skipping baseline packages (SKIP_PKGS=1)."
fi

# --- 3. Go toolchain ----------------------------------------------------------
if [ "$SKIP_GO" != "1" ]; then
    step "Installing Go ${GO_VERSION}"
    if [ -x /usr/local/go/bin/go ] && [ "$(/usr/local/go/bin/go version | awk '{print $3}')" = "go${GO_VERSION}" ]; then
        log "Go ${GO_VERSION} already installed at /usr/local/go."
    else
        GO_TARBALL="go${GO_VERSION}.linux-${ARCH}.tar.gz"
        GO_URL="https://go.dev/dl/${GO_TARBALL}"
        log "Downloading ${GO_URL}"
        curl -fsSL -o /tmp/${GO_TARBALL} "${GO_URL}"
        $SUDO rm -rf /usr/local/go
        $SUDO tar -C /usr/local -xzf /tmp/${GO_TARBALL}
        rm -f /tmp/${GO_TARBALL}
    fi
    if [ ! -f /etc/profile.d/gopath.sh ]; then
        printf 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin\n' | $SUDO tee /etc/profile.d/gopath.sh >/dev/null
    fi
    export PATH="$PATH:/usr/local/go/bin:$HOME/go/bin"
    log "Go: $(command -v go >/dev/null && go version || echo "avail after re-login")"
else
    log "Skipping Go toolchain (SKIP_GO=1)."
fi

# --- 4. Docker Engine + Compose v2 --------------------------------------------
if [ "$SKIP_DOCKER" != "1" ]; then
    step "Installing Docker Engine + Compose v2"
    CODENAME="${VERSION_CODENAME:-$(lsb_release -cs)}"
    DOCKER_KEY=/etc/apt/keyrings/docker.asc
    if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
        log "Docker $("docker --version") with Compose v2 already installed."
    else
        $SUDO install -m 0755 -d /etc/apt/keyrings
        # Docker rotated its repo signing subkey; the /gpg endpoint can lag behind
        # the signer used by apt InRelease. Fetch the armored key, validate, and if
        # apt reports NO_PUBKEY, append the requested key from a keyserver.
        curl -fsSL "https://download.docker.com/linux/${ID}/gpg" -o "${DOCKER_KEY}"
        $SUDO chmod a+r "${DOCKER_KEY}"
        DOCKER_ERR=/tmp/docker-apt-update.log
        for _try in 1 2 3; do
            if $SUDO apt-get update 2>"${DOCKER_ERR}" | grep -qE "Err:|NO_PUBKEY"; then
                NO_PUBKEY=$(grep -oE 'NO_PUBKEY [0-9A-F]+' "${DOCKER_ERR}" | awk '{print $2}' | sort -u | head -1)
            else
                NO_PUBKEY=""
            fi
            if [ -z "$NO_PUBKEY" ]; then break; fi
            log "apt needs docker key ${NO_PUBKEY} — fetching from keyserver and appending"
            GOPG=/tmp/docker-extra-key
            rm -f "${GOPG}"
            gpg --no-default-keyring --keyring "${GOPG}" --keyserver keyserver.ubuntu.com \
                --recv-keys "${NO_PUBKEY}" >/dev/null 2>&1
            if gpg --no-default-keyring --keyring "${GOPG}" --list-keys "${NO_PUBKEY}" >/dev/null 2>&1; then
                gpg --no-default-keyring --keyring "${GOPG}" --export --armor "${NO_PUBKEY}" >> "${DOCKER_KEY}"
                $SUDO chmod a+r "${DOCKER_KEY}"
            else
                warn "Could not fetch ${NO_PUBKEY} from keyserver — apt may still fail."
                break
            fi
        done
        if [ ! -f /etc/apt/sources.list.d/docker.sources ] && [ ! -f /etc/apt/sources.list.d/docker.list ]; then
            printf 'Types: deb\nURIs: https://download.docker.com/linux/%s\nSuites: %s\nComponents: stable\nArchitectures: %s\nSigned-By: %s\n' \
                "$ID" "$CODENAME" "$ARCH" "$DOCKER_KEY" | $SUDO tee /etc/apt/sources.list.d/docker.sources >/dev/null
        fi
        $SUDO apt-get update
        $SUDO apt-get install -y docker-ce docker-ce-cli containerd.io \
            docker-buildx-plugin docker-compose-plugin
        $SUDO systemctl enable --now docker
    fi

    CALLER_USER="${SUDO_USER:-$USER}"
    if [ -n "$CALLER_USER" ] && ! id -nG "$CALLER_USER" | grep -qw docker; then
        log "Adding '$CALLER_USER' to the 'docker' group (re-login required for it to apply)."
        $SUDO usermod -aG docker "$CALLER_USER"
    else
        log "User '$CALLER_USER' is already in the 'docker' group."
    fi
else
    log "Skipping Docker install (SKIP_DOCKER=1)."
fi

# --- 5. Firewall --------------------------------------------------------------
if [ "$SKIP_UFW" != "1" ]; then
    step "Configuring ufw (OpenSSH, 80, 443)"
    $SUDO ufw allow OpenSSH >/dev/null 2>&1 || true
    $SUDO ufw allow 80/tcp >/dev/null 2>&1 || true
    $SUDO ufw allow 443/tcp >/dev/null 2>&1 || true
    $SUDO ufw --force enable >/dev/null 2>&1 || warn "ufw enable failed (check 'sudo ufw status')."
fi

# --- 6. Requirement checks -----------------------------------------------------
step "Checking required tooling"
declare -A REQS=(
    [docker]="Docker Engine — SKIP_DOCKER=1"
    [psql]="postgresql-client — install via SKIP_PKGS=0"
    [python3]="python3 — install via SKIP_PKGS=0"
    [git]="git — install via SKIP_PKGS=0"
    [openssl]="openssl — install via SKIP_PKGS=0"
    [curl]="curl — install via SKIP_PKGS=0"
)
[ "$SKIP_GO" = "1" ] || REQS[go]="Go toolchain — install via SKIP_GO=0"
missing=0
for tool in "${!REQS[@]}"; do
    if command -v "$tool" >/dev/null 2>&1; then
        log "  ok    ${tool} ($(command -v "$tool"))"
    else
        err "  MISS  ${tool} (${REQS[$tool]})"
        missing=1
    fi
done
if docker compose version >/dev/null 2>&1; then
    log "  ok    compose v2"
else
    err "  MISS  docker compose plugin (install via SKIP_DOCKER=0)"
    missing=1
fi
command -v docker >/dev/null 2>&1 && log "  ok    docker ($(docker --version))"
if [ "$missing" = "1" ]; then
    err "Missing required tools above. Fix and re-run (script is idempotent)."
    exit 1
fi

# --- 7. Clone / checkout the branch -------------------------------------------
step "Resolving the repository"
reuse_checkout=0
if [ -z "$POS_REPO_URL" ]; then
    if git -C "$POS_HOME" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        reuse_checkout=1
    elif git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
        POS_HOME="$(git rev-parse --show-toplevel)"
        BACKEND_DIR="${POS_HOME}/Backend/go-pos-backend-implementation-ready"
        reuse_checkout=1
    fi
fi
if [ "$SKIP_CLONE" = "1" ] && [ -d "$BACKEND_DIR" ]; then
    log "SKIP_CLONE=1 — using existing checkout at ${POS_HOME}"
elif [ -d "$BACKEND_DIR" ] && [ -d "$POS_HOME/.git" ]; then
    log "Checkout already present at ${POS_HOME}; fetching ${POS_BRANCH}"
    git -C "$POS_HOME" fetch --all --prune >/dev/null 2>&1 || true
    git -C "$POS_HOME" checkout "$POS_BRANCH" 2>/dev/null || \
        git -C "$POS_HOME" checkout -b "$POS_BRANCH" "origin/$POS_BRANCH" 2>/dev/null || true
elif [ "$reuse_checkout" = "1" ]; then
    log "Running from inside a checkout at ${POS_HOME}; branch: $(git -C "$POS_HOME" branch --show-current 2>/dev/null || echo 'detached')"
else
    [ -n "$POS_REPO_URL" ] || { err "POS_REPO_URL not set and not inside a repo."; exit 1; }
    log "Cloning ${POS_BRANCH} from ${POS_REPO_URL} into ${POS_HOME}"
    $SUDO mkdir -p "$POS_HOME"
    $SUDO git clone --depth 1 --branch "$POS_BRANCH" "$POS_REPO_URL" "$POS_HOME"
    $SUDO chown -R "${SUDO_USER:-root}":"${SUDO_USER:-root}" "$POS_HOME" 2>/dev/null || true
fi
[ -d "$BACKEND_DIR" ] || { err "No backend checkout found at ${BACKEND_DIR}."; exit 1; }
log "Backend checkout: ${BACKEND_DIR}"

# --- 8. .env.prod --------------------------------------------------------------
step "Preparing .env.prod"
if [ "$SKIP_ENV" = "1" ]; then
    log "SKIP_ENV=1 — leaving any existing .env.prod untouched."
elif [ -f "${BACKEND_DIR}/.env.prod" ]; then
    warn ".env.prod already exists — leaving it untouched. Edit SITE_DOMAIN/secrets manually if needed."
else
    cp "${BACKEND_DIR}/.env.prod.example" "${BACKEND_DIR}/.env.prod"
    JWT_ACCESS_SECRET=$($SUDO openssl rand -hex 32)
    JWT_REFRESH_SECRET=$($SUDO openssl rand -hex 32)
    POSTGRES_PASSWORD=$($SUDO openssl rand -hex 24)
    sed -i \
        -e "s|^SITE_DOMAIN=.*|SITE_DOMAIN=${SITE_DOMAIN}|" \
        -e "s|^CORS_ALLOWED_ORIGINS=.*|CORS_ALLOWED_ORIGINS=${CORS_ALLOWED}|" \
        -e "s|^POSTGRES_PASSWORD=.*|POSTGRES_PASSWORD=${POSTGRES_PASSWORD}|" \
        -e "s|^JWT_ACCESS_SECRET=.*|JWT_ACCESS_SECRET=${JWT_ACCESS_SECRET}|" \
        -e "s|^JWT_REFRESH_SECRET=.*|JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}|" \
        -e "s|^JWT_ISSUER=.*|JWT_ISSUER=pos-api|" \
        -e "s|^LOG_LEVEL=.*|LOG_LEVEL=info|" \
        "${BACKEND_DIR}/.env.prod"
    if ! grep -q "^SCRAPE_IP=" "${BACKEND_DIR}/.env.prod"; then
        echo "SCRAPE_IP=${SCRAPE_IP}" >> "${BACKEND_DIR}/.env.prod"
    fi
    chown "${SUDO_USER:-root}":"${SUDO_USER:-root}" "${BACKEND_DIR}/.env.prod" 2>/dev/null || true
    log "Generated ${BACKEND_DIR}/.env.prod with fresh random secrets."
fi

# --- 9. Deploy -----------------------------------------------------------------
if [ "$SKIP_DEPLOY" = "1" ]; then
    log "SKIP_DEPLOY=1 — provisioning only. Deploy later from ${BACKEND_DIR}:"
    log "  docker compose --env-file .env.prod -f deployments/docker/docker-compose.prod.yml up -d --build"
else
    step "Building and starting the production stack (postgres, redis, migrate, api, caddy)"
    if ! docker info >/dev/null 2>&1; then
        warn "The '${SUDO_USER:-$USER}' user cannot reach the daemon yet. Retrying that one step with sudo..."
        (cd "$BACKEND_DIR" && "$SUDO" docker compose --env-file .env.prod \
            -f deployments/docker/docker-compose.prod.yml up -d --build)
    else
        (cd "$BACKEND_DIR" && docker compose --env-file .env.prod \
            -f deployments/docker/docker-compose.prod.yml up -d --build)
    fi
    log "Stack started (or already running)."
fi

# --- 10. Verify ----------------------------------------------------------------
step "Health checks"
deadline=$((SECONDS + 120))
until [ $SECONDS -ge "$deadline" ]; do
    if curl -fsS "http://127.0.0.1:8080/health/live" >/dev/null 2>&1; then
        log "  /health/live  OK"
        break
    fi
    sleep 3
done
curl -fsS "http://127.0.0.1:8080/health/live" >/dev/null 2>&1 || \
    warn "/health/live not OK yet — check 'docker compose -f deployments/docker/docker-compose.prod.yml ps'."

deadline=$((SECONDS + 60))
until [ $SECONDS -ge "$deadline" ]; do
    if curl -fsS "http://127.0.0.1:8080/health/ready" >/dev/null 2>&1; then
        log "  /health/ready OK"
        break
    fi
    sleep 3
done
curl -fsS "http://127.0.0.1:8080/health/ready" >/dev/null 2>&1 || \
    warn "/health/ready not OK (needs Redis — check the redis container)."

# --- Summary ------------------------------------------------------------------
step "Done"
echo -e "${GREEN}POS backend is provisioned.${NC}"
echo "  Repo            : $POS_HOME (branch: $POS_BRANCH)"
echo "  Backend dir     : $BACKEND_DIR"
echo "  Config          : $BACKEND_DIR/.env.prod"
echo "  API             : http://$SITE_DOMAIN/  (Caddy TLS, reverse_proxy -> api:8080)"
echo "  Health          : http://$SITE_DOMAIN/health/live  /health/ready"
echo "  Metrics         : http://127.0.0.1:9090/metrics (Caddy :9090, source IP $SCRAPE_IP only)"
if [ "$SITE_DOMAIN" = "localhost" ]; then
    warn "SITE_DOMAIN is still 'localhost' — set it in .env.prod to a public domain for automatic TLS."
fi
echo
echo "Back it up (cron, daily 3:00 AM):"
echo "  0 3 * * * $BACKEND_DIR/scripts/backup.sh >> /var/log/pos-backup.log 2>&1"
echo
echo "Flutter production APK (from a dev machine, then sign/adb install):"
echo "  flutter build apk --release --dart-define=API_BASE_URL=https://${SITE_DOMAIN}"
echo
echo "User '$SUDO_USER' may need to re-login for the docker group + Go PATH to apply."