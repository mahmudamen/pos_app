#!/usr/bin/env bash
# ==============================================================================
# Go POS SaaS — System Requirements & Environment Provisioning Script
# Target: Ubuntu 22.04/24.04 LTS
# ==============================================================================

set -euo pipefail

# --- Color formatting ---
RED='\030[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# --- Step 1: Pre-flight Verification ---
echo -e "${BLUE}[1/6] Validating System Requirements...${NC}"

# Check Sudo / Root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Error: This script must be run with sudo privileges.${NC}"
    exit 1
fi

# Check OS Version (Ubuntu 22.04 or 24.04)
if [ -f /etc/os-release ]; then
    . /etc/os-release
    if [[ "$ID" != "ubuntu" ]]; then
        echo -e "${RED}Error: Unsupported distribution. Expected Ubuntu.${NC}"
        exit 1
    fi
    echo -e "${GREEN}Detected OS:${NC} $PRETTY_NAME"
else
    echo -e "${RED}Error: Cannot identify OS release version.${NC}"
    exit 1
fi

# Check Architecture
ARCH=$(uname -m)
if [[ "$ARCH" != "x86_64" && "$ARCH" != "aarch64" ]]; then
    echo -e "${RED}Error: Unsupported architecture $ARCH. Expected x86_64 or aarch64.${NC}"
    exit 1
fi

# Check Minimum Memory (2GB recommended)
RAM_MB=$(free -m | awk '/^Mem:/{print $2}')
if [ "$RAM_MB" -lt 1800 ]; then
    echo -e "${YELLOW}Warning: Total RAM is lower than 2GB (${RAM_MB}MB). Build processes or DB services may run low on memory.${NC}"
fi

# --- Step 2: Interactive Prompt for Setup Type ---
echo -e "\n${BLUE}[2/6] Select Environment Setup Type:${NC}"
echo "1) Development Environment (Installs Go 1.23+, PostgreSQL, Redis, Docker Compose, build tools)"
echo "2) Production Backend Environment (Installs Go 1.23+, PostgreSQL 16, Redis 7, Caddy proxy, systemd layout)"
read -p "Enter choice [1-2]: " SETUP_CHOICE

case $SETUP_CHOICE in
    1) ENV_TYPE="dev" ;;
    2) ENV_TYPE="prod" ;;
    *) echo -e "${RED}Invalid selection. Exiting.${NC}"; exit 1 ;;
esac

echo -e "${GREEN}Configuring environment for: ${ENV_TYPE}${NC}\n"

# --- Step 3: Base Package Installation ---
echo -e "${BLUE}[3/6] Updating packages and installing dependencies...${NC}"
apt-get update -y
apt-get install -y curl wget git build-essential ufw software-properties-common ca-certificates gnupg

# --- Step 4: Install Go 1.23+ ---
echo -e "${BLUE}[4/6] Installing Go 1.23+...${NC}"
GO_VERSION="1.23.6"

if command -v go &> /dev/null; then
    CURRENT_GO=$(go version | awk '{print $3}')
    echo -e "${YELLOW}Existing Go installation found (${CURRENT_GO}). Upgrading/Overwriting to Go ${GO_VERSION}...${NC}"
    rm -rf /usr/local/go
fi

if [[ "$ARCH" == "x86_64" ]]; then
    GO_ARCH="amd64"
else
    GO_ARCH="arm64"
fi

wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -O /tmp/go.tar.gz
tar -C /usr/local -xzf /tmp/go.tar.gz
rm /tmp/go.tar.gz

# Export PATH dynamically for the current script session
export PATH=$PATH:/usr/local/go/bin

# Set persistent environment variables
if ! grep -q "/usr/local/go/bin" /etc/profile; then
    echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> /etc/profile
fi

echo -e "${GREEN}Go installed successfully:${NC} $(/usr/local/go/bin/go version)"

# --- Step 5: Install Services & Infrastructure ---
if [[ "$ENV_TYPE" == "dev" ]]; then
    echo -e "${BLUE}[5/6] Installing Development Tooling...${NC}"
    
    # Install PostgreSQL & Redis via APT for quick dev iteration
    apt-get install -y postgresql postgresql-contrib redis-server

    # Install Docker & Docker Compose
    if ! command -v docker &> /dev/null; then
        echo -e "${YELLOW}Installing Docker...${NC}"
        curl -fsSL https://get.docker.com | sh
    fi

    # Install Goose migration tool
    export GOPATH=${HOME}/go
    /usr/local/go/bin/go install github.com/pressly/goose/v3/cmd/goose@latest || true

elif [[ "$ENV_TYPE" == "prod" ]]; then
    echo -e "${BLUE}[5/6] Provisioning Production Services...${NC}"
    
    # Install PostgreSQL 16 official repository
    install -d /etc/apt/keyrings
    curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /etc/apt/keyrings/postgresql.gpg
    echo "deb [signed-by=/etc/apt/keyrings/postgresql.gpg] http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list
    
    apt-get update -y
    apt-get install -y postgresql-16 postgresql-contrib-16 redis-server caddy

    # Create dedicated user for API service execution
    if ! id -u pos-api &>/dev/null; then
        useradd -r -m -d /opt/pos-api -s /bin/false pos-api
        echo -e "${GREEN}Created system user 'pos-api'.${NC}"
    fi

    # Setup Directory layout
    mkdir -p /opt/pos-api/logs
    chown -R pos-api:pos-api /opt/pos-api

    # Configure Firewall (UFW)
    ufw allow 80/tcp
    ufw allow 443/tcp
    ufw allow 22/tcp
    ufw --force enable
fi

# Ensure basic services are enabled
systemctl enable --now postgresql
systemctl enable --now redis-server

# --- Step 6: Summary ---
echo -e "\n${GREEN}======================================================================${NC}"
echo -e "${GREEN} Environment Setup Completed Successfully!${NC}"
echo -e "${GREEN}======================================================================${NC}"
echo -e "Go Binary Path : /usr/local/go/bin/go"
echo -e "Go Version     : $(/usr/local/go/bin/go version)"
echo -e "Environment    : $ENV_TYPE"
echo -e "\n${YELLOW}Note:${NC} Run 'source /etc/profile' or re-log in to update your shell PATH."
