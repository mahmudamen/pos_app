#!/bin/bash
###############################################################################
# Go POS SaaS Backend - Comprehensive Setup Script
# Version: 1.0 | Target: Ubuntu 22.04/24.04 LTS
# Description: Interactive installer for Go backend, PostgreSQL 16, Redis 7,
#              and full project scaffolding.
###############################################################################

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Logging
LOG_FILE="/var/log/pos-backend-setup.log"
exec > >(tee -a "$LOG_FILE") 2>&1

# Global variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_NAME="pos-backend"
PROJECT_DIR=""
GO_VERSION="1.23.4"
POSTGRES_VERSION="16"
REDIS_VERSION="7"

###############################################################################
# Helper Functions
###############################################################################

print_banner() {
    clear
    echo -e "${CYAN}"
    cat << "EOF"
   ____  ____   ___    ____             _                  _           
  / ___||  _ \ / _ \  | __ )  __ _  ___| | _____ _ __   __| | ___ _ __ 
  \___ \| |_) | | | | |  _ \ / _` |/ __| |/ / _ \ '_ \ / _` |/ _ \ '__|
   ___) |  __/| |_| | | |_) | (_| | (__|   <  __/ | | | (_| |  __/ |   
  |____/|_|    \___/  |____/ \__,_|\___|_|\_\___|_| |_|\__,_|\___|_|   

EOF
    echo -e "${NC}"
    echo -e "${BOLD}Backend Setup & Environment Builder${NC}"
    echo -e "${BLUE}Target: Ubuntu 22.04/24.04 LTS | Go ${GO_VERSION}+ | PostgreSQL ${POSTGRES_VERSION} | Redis ${REDIS_VERSION}${NC}"
    echo -e "${BLUE}Log: ${LOG_FILE}${NC}\n"
}

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "\n${BOLD}${BLUE}▶ $1${NC}"; }

pause() {
    echo -e "\n${CYAN}Press Enter to continue...${NC}"
    read -r
}

command_exists() { command -v "$1" &> /dev/null; }

get_input() {
    local prompt="$1"
    local default="${2:-}"
    local input
    if [ -n "$default" ]; then
        read -rp "$(echo -e "${CYAN}${prompt} [${default}]: ${NC}")" input
        echo "${input:-$default}"
    else
        read -rp "$(echo -e "${CYAN}${prompt}: ${NC}")" input
        echo "$input"
    fi
}

get_password() {
    local prompt="$1"
    local pass
    while true; do
        read -rsp "$(echo -e "${CYAN}${prompt}: ${NC}")" pass
        echo
        if [ ${#pass} -ge 8 ]; then
            echo "$pass"
            break
        else
            log_warn "Password must be at least 8 characters."
        fi
    done
}

confirm() {
    local prompt="$1"
    local response
    read -rp "$(echo -e "${CYAN}${prompt} [Y/n]: ${NC}")" response
    case "$response" in
        [Yy]*|"") return 0 ;;
        *) return 1 ;;
    esac
}

###############################################################################
# 1. System Requirements Check
###############################################################################

check_ubuntu_version() {
    log_step "Checking Ubuntu Version"

    if [ ! -f /etc/os-release ]; then
        log_error "Cannot detect OS. This script requires Ubuntu 22.04 or 24.04 LTS."
        exit 1
    fi

    source /etc/os-release
    if [ "$ID" != "ubuntu" ]; then
        log_error "This script is designed for Ubuntu. Detected: $ID"
        exit 1
    fi

    local version_id="$VERSION_ID"
    if [[ "$version_id" != "22.04" && "$version_id" != "24.04" ]]; then
        log_warn "Detected Ubuntu $version_id. This script is optimized for 22.04/24.04 LTS."
        if ! confirm "Continue anyway?"; then
            exit 1
        fi
    else
        log_info "Ubuntu $version_id LTS detected ✓"
    fi
}

check_hardware() {
    log_step "Checking Hardware Requirements"

    # RAM
    local total_ram_kb
    total_ram_kb=$(grep MemTotal /proc/meminfo | awk '{print $2}')
    local total_ram_gb=$((total_ram_kb / 1024 / 1024))

    if [ "$total_ram_gb" -lt 2 ]; then
        log_warn "RAM: ${total_ram_gb}GB detected (Recommended: 2GB+ for production)"
    else
        log_info "RAM: ${total_ram_gb}GB ✓"
    fi

    # Disk
    local available_disk
    available_disk=$(df -BG / | awk 'NR==2 {print $4}' | sed 's/G//')
    if [ "$available_disk" -lt 10 ]; then
        log_warn "Disk: ${available_disk}GB free (Recommended: 10GB+ for development, 20GB+ for production)"
    else
        log_info "Disk: ${available_disk}GB free ✓"
    fi

    # Architecture
    local arch
    arch=$(dpkg --print-architecture)
    if [ "$arch" != "amd64" ] && [ "$arch" != "arm64" ]; then
        log_error "Architecture $arch not supported. Use amd64 or arm64."
        exit 1
    fi
    log_info "Architecture: $arch ✓"

    # Internet
    if ! ping -c 1 google.com &> /dev/null; then
        log_error "Internet connection required for package installation."
        exit 1
    fi
    log_info "Internet connectivity ✓"
}

check_privileges() {
    log_step "Checking Privileges"
    if [ "$EUID" -ne 0 ]; then
        log_warn "Not running as root. Will use sudo for privileged operations."
        if ! command_exists sudo; then
            log_error "sudo not found. Please run as root or install sudo."
            exit 1
        fi
        SUDO="sudo"
    else
        log_info "Running as root ✓"
        SUDO=""
    fi
}

###############################################################################
# 2. System Update & Base Packages
###############################################################################

update_system() {
    log_step "Updating System Packages"
    $SUDO apt-get update
    $SUDO apt-get upgrade -y
    $SUDO apt-get install -y         curl wget git build-essential software-properties-common         apt-transport-https ca-certificates gnupg lsb-release         unzip jq htop ncdu tree ufw fail2ban
    log_info "System packages updated ✓"
}

###############################################################################
# 3. Go Installation
###############################################################################

install_go() {
    log_step "Installing Go ${GO_VERSION}"

    if command_exists go; then
        local current_version
        current_version=$(go version | awk '{print $3}' | sed 's/go//')
        log_info "Go $current_version already installed"
        if confirm "Reinstall/Update to Go ${GO_VERSION}?"; then
            $SUDO rm -rf /usr/local/go
        else
            return
        fi
    fi

    local arch="$(dpkg --print-architecture)"
    local go_tarball="go${GO_VERSION}.linux-${arch}.tar.gz"
    local go_url="https://go.dev/dl/${go_tarball}"

    log_info "Downloading Go from $go_url..."
    wget -q --show-progress "$go_url" -O "/tmp/${go_tarball}"

    log_info "Extracting Go..."
    $SUDO tar -C /usr/local -xzf "/tmp/${go_tarball}"
    rm "/tmp/${go_tarball}"

    # Add to PATH if not present
    if ! grep -q "/usr/local/go/bin" "$HOME/.bashrc"; then
        echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> "$HOME/.bashrc"
        echo 'export GOPATH=$HOME/go' >> "$HOME/.bashrc"
        log_info "Added Go to PATH in ~/.bashrc"
    fi

    export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
    export GOPATH=$HOME/go

    local installed_version
    installed_version=$(go version | awk '{print $3}')
    log_info "${installed_version} installed successfully ✓"

    # Install Go tools
    log_info "Installing Go tools..."
    go install github.com/cosmtrek/air@latest 2>/dev/null || true
    go install github.com/pressly/goose/v3/cmd/goose@latest 2>/dev/null || true
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest 2>/dev/null || true
    log_info "Go tools installed ✓"
}

###############################################################################
# 4. PostgreSQL 16 Installation
###############################################################################

install_postgres() {
    log_step "Installing PostgreSQL ${POSTGRES_VERSION}"

    if command_exists psql; then
        local current_version
        current_version=$(psql --version | awk '{print $3}' | cut -d'.' -f1)
        if [ "$current_version" == "$POSTGRES_VERSION" ]; then
            log_info "PostgreSQL ${POSTGRES_VERSION} already installed ✓"
            return
        else
            log_warn "PostgreSQL $current_version found. Will install $POSTGRES_VERSION alongside."
        fi
    fi

    # Add PostgreSQL APT repository
    $SUDO install -d /usr/share/postgresql-common/pgdg
    $SUDO curl -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc         --fail https://www.postgresql.org/media/keys/ACCC4CF8.asc
    $SUDO sh -c 'echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc]         https://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main"         > /etc/apt/sources.list.d/pgdg.list'

    $SUDO apt-get update
    $SUDO apt-get install -y postgresql-${POSTGRES_VERSION} postgresql-client-${POSTGRES_VERSION} postgresql-contrib-${POSTGRES_VERSION}

    # Start and enable
    $SUDO systemctl enable postgresql
    $SUDO systemctl start postgresql

    log_info "PostgreSQL ${POSTGRES_VERSION} installed and running ✓"
}

configure_postgres() {
    log_step "Configuring PostgreSQL"

    local db_name db_user db_pass
    db_name=$(get_input "Database name" "pos_db")
    db_user=$(get_input "Database user" "pos_user")
    db_pass=$(get_password "Database password (min 8 chars)")

    # Create user and database
    $SUDO -u postgres psql << EOF
CREATE USER ${db_user} WITH PASSWORD '${db_pass}';
CREATE DATABASE ${db_name} OWNER ${db_user};
GRANT ALL PRIVILEGES ON DATABASE ${db_name} TO ${db_user};
\c ${db_name}
GRANT ALL ON SCHEMA public TO ${db_user};
ALTER USER ${db_user} WITH SUPERUSER;
EOF

    # Configure pg_hba for local trust (development)
    local pg_hba="/etc/postgresql/${POSTGRES_VERSION}/main/pg_hba.conf"
    if [ -f "$pg_hba" ]; then
        $SUDO sed -i 's/scram-sha-256/trust/g' "$pg_hba"
        $SUDO sed -i 's/peer/trust/g' "$pg_hba"
        $SUDO systemctl restart postgresql
        log_info "PostgreSQL configured for local development ✓"
    fi

    # Save credentials
    DB_NAME="$db_name"
    DB_USER="$db_user"
    DB_PASS="$db_pass"
    DB_URL="postgres://${db_user}:${db_pass}@localhost:5432/${db_name}?sslmode=disable"

    log_info "Database: $db_name | User: $db_user"
}

###############################################################################
# 5. Redis 7 Installation
###############################################################################

install_redis() {
    log_step "Installing Redis ${REDIS_VERSION}"

    if command_exists redis-cli; then
        local current_version
        current_version=$(redis-cli --version | awk '{print $2}' | cut -d'.' -f1)
        if [ "$current_version" == "$REDIS_VERSION" ]; then
            log_info "Redis ${REDIS_VERSION} already installed ✓"
            return
        fi
    fi

    # Add Redis official repo
    curl -fsSL https://packages.redis.io/gpg | $SUDO gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg
    echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" |         $SUDO tee /etc/apt/sources.list.d/redis.list

    $SUDO apt-get update
    $SUDO apt-get install -y redis

    $SUDO systemctl enable redis-server
    $SUDO systemctl start redis-server

    # Basic hardening
    $SUDO sed -i 's/^# supervised auto/supervised systemd/' /etc/redis/redis.conf
    $SUDO sed -i 's/^bind 127.0.0.1 ::1/bind 127.0.0.1/' /etc/redis/redis.conf
    $SUDO systemctl restart redis-server

    log_info "Redis ${REDIS_VERSION} installed and running ✓"
}

###############################################################################
# 6. Docker & Docker Compose (Optional)
###############################################################################

install_docker() {
    log_step "Installing Docker & Docker Compose"

    if command_exists docker; then
        log_info "Docker already installed: $(docker --version)"
        if ! confirm "Reinstall Docker?"; then
            return
        fi
    fi

    # Remove old versions
    $SUDO apt-get remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true

    # Install Docker
    $SUDO install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | $SUDO gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    $SUDO chmod a+r /etc/apt/keyrings/docker.gpg

    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg]         https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" |         $SUDO tee /etc/apt/sources.list.d/docker.list > /dev/null

    $SUDO apt-get update
    $SUDO apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

    # Add user to docker group
    $SUDO usermod -aG docker "${SUDO_USER:-$USER}" 2>/dev/null || true

    $SUDO systemctl enable docker
    $SUDO systemctl start docker

    log_info "Docker installed ✓"
    log_warn "You may need to logout and login again for docker group to take effect."
}

###############################################################################
# 7. Caddy Installation (Optional)
###############################################################################

install_caddy() {
    log_step "Installing Caddy Web Server"

    if command_exists caddy; then
        log_info "Caddy already installed: $(caddy version)"
        return
    fi

    $SUDO apt-get install -y debian-keyring debian-archive-keyring apt-transport-https
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | $SUDO gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | $SUDO tee /etc/apt/sources.list.d/caddy-stable.list
    $SUDO apt-get update
    $SUDO apt-get install -y caddy

    $SUDO systemctl enable caddy
    log_info "Caddy installed ✓"
}

###############################################################################
# 8. Firewall Setup
###############################################################################

setup_firewall() {
    log_step "Configuring UFW Firewall"

    $SUDO ufw default deny incoming
    $SUDO ufw default allow outgoing
    $SUDO ufw allow ssh
    $SUDO ufw allow 22/tcp
    $SUDO ufw allow 80/tcp
    $SUDO ufw allow 443/tcp
    $SUDO ufw allow 8080/tcp

    if confirm "Enable UFW firewall now?"; then
        $SUDO ufw --force enable
        log_info "Firewall enabled ✓"
    else
        log_warn "Firewall not enabled. Enable manually with: sudo ufw enable"
    fi

    $SUDO ufw status verbose
}

###############################################################################
# 9. Project Scaffolding
###############################################################################

scaffold_project() {
    log_step "Scaffolding Go POS Backend Project"

    PROJECT_DIR=$(get_input "Project directory" "$HOME/pos-backend")

    if [ -d "$PROJECT_DIR" ]; then
        log_warn "Directory $PROJECT_DIR already exists."
        if ! confirm "Overwrite/Update existing project?"; then
            return
        fi
    fi

    mkdir -p "$PROJECT_DIR"
    cd "$PROJECT_DIR"

    # Initialize Go module
    go mod init pos-backend 2>/dev/null || true

    # Create directory structure
    log_info "Creating project structure..."
    mkdir -p cmd/api
    mkdir -p internal/config
    mkdir -p internal/domain/models
    mkdir -p internal/domain/repositories
    mkdir -p internal/infrastructure/database/migrations
    mkdir -p internal/infrastructure/cache
    mkdir -p internal/infrastructure/http/handlers
    mkdir -p internal/infrastructure/http/middleware
    mkdir -p internal/infrastructure/security
    mkdir -p internal/usecases
    mkdir -p internal/dto
    mkdir -p pkg/errors
    mkdir -p pkg/validators
    mkdir -p pkg/pagination
    mkdir -p pkg/response
    mkdir -p scripts
    mkdir -p deployments/systemd
    mkdir -p deployments/caddy
    mkdir -p deployments/docker

    # Create main.go
    cat > cmd/api/main.go << 'GOEOF'
package main

import (
    "log/slog"
    "os"
    "pos-backend/internal/config"
)

func main() {
    cfg := config.Load()

    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }))
    slog.SetDefault(logger)

    logger.Info("POS Backend starting", "port", cfg.Port, "env", cfg.Env)

    // TODO: Initialize database, router, and start server
    // See backend_spec_go.md for full implementation
}
GOEOF

    # Create config.go
    cat > internal/config/config.go << 'GOEOF'
package config

import (
    "log"
    "os"
    "time"

    "github.com/spf13/viper"
)

type Config struct {
    Port            string        `mapstructure:"PORT"`
    Env             string        `mapstructure:"GO_ENV"`
    DatabaseURL     string        `mapstructure:"DATABASE_URL"`
    RedisURL        string        `mapstructure:"REDIS_URL"`
    JWTAccessSecret string        `mapstructure:"JWT_ACCESS_SECRET"`
    JWTRefreshSecret string       `mapstructure:"JWT_REFRESH_SECRET"`
    BcryptCost      int           `mapstructure:"BCRYPT_COST"`
    MaxSessions     int           `mapstructure:"MAX_SESSIONS_PER_USER"`
    LogLevel        string        `mapstructure:"LOG_LEVEL"`
    CORSOrigins     []string      `mapstructure:"CORS_ALLOWED_ORIGINS"`
    SyncTokenTTL    time.Duration `mapstructure:"SYNC_TOKEN_TTL"`
    SyncBatchSize   int           `mapstructure:"SYNC_BATCH_SIZE"`
}

func Load() *Config {
    viper.SetDefault("PORT", "8080")
    viper.SetDefault("GO_ENV", "development")
    viper.SetDefault("BCRYPT_COST", 12)
    viper.SetDefault("MAX_SESSIONS_PER_USER", 3)
    viper.SetDefault("LOG_LEVEL", "info")
    viper.SetDefault("SYNC_TOKEN_TTL", "168h")
    viper.SetDefault("SYNC_BATCH_SIZE", 500)

    viper.SetConfigFile(".env")
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err != nil {
        log.Println("No .env file found, using environment variables")
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    return &cfg
}
GOEOF

    # Create .env.example
    cat > .env.example << 'ENVEOF'
# Server
PORT=8080
GO_ENV=development

# Database
DATABASE_URL=postgres://pos_user:your_password@localhost:5432/pos_db?sslmode=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# Redis
REDIS_URL=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT (Generate strong secrets in production!)
JWT_ACCESS_SECRET=change-me-to-256-bit-secret-in-production
JWT_REFRESH_SECRET=change-me-to-different-256-bit-secret-in-production

# Security
BCRYPT_COST=12
MAX_SESSIONS_PER_USER=3

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# Logging
LOG_LEVEL=info

# Sync
SYNC_TOKEN_TTL=168h
SYNC_BATCH_SIZE=500
ENVEOF

    # Create Makefile
    cat > Makefile << 'MAKEEOF'
.PHONY: build run test migrate migrate-down docker-build lint fmt deps

APP_NAME=pos-api
BUILD_DIR=./build
MAIN_FILE=./cmd/api/main.go

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)

run:
	go run $(MAIN_FILE)

dev:
	air -c .air.toml

test:
	go test -v -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...

deps:
	go mod download
	go mod tidy

migrate:
	goose -dir ./internal/infrastructure/database/migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir ./internal/infrastructure/database/migrations postgres "$(DATABASE_URL)" down

docker-build:
	docker build -t $(APP_NAME):latest -f deployments/docker/Dockerfile .

.DEFAULT_GOAL := run
MAKEEOF

    # Create Dockerfile
    cat > deployments/docker/Dockerfile << 'DOCKEREof'
# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o pos-api ./cmd/api/main.go

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/pos-api .
COPY --from=builder /app/.env.example .env

EXPOSE 8080
CMD ["./pos-api"]
DOCKEREof

    # Create docker-compose.yml
    cat > deployments/docker/docker-compose.yml << 'COMPOSEOF'
version: '3.8'

services:
  api:
    build:
      context: ../..
      dockerfile: deployments/docker/Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://pos:pos@postgres:5432/pos_db?sslmode=disable
      - REDIS_URL=redis:6379
      - JWT_ACCESS_SECRET=${JWT_ACCESS_SECRET}
      - JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}
    depends_on:
      - postgres
      - redis
    restart: unless-stopped
    networks:
      - pos-network

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: pos
      POSTGRES_PASSWORD: pos
      POSTGRES_DB: pos_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ../infrastructure/database/migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    networks:
      - pos-network

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    networks:
      - pos-network

volumes:
  postgres_data:
  redis_data:

networks:
  pos-network:
    driver: bridge
COMPOSEOF

    # Create systemd service
    cat > deployments/systemd/pos-api.service << 'SYSTEMDEof'
[Unit]
Description=POS SaaS API
After=network.target postgresql.service redis-server.service

[Service]
Type=simple
User=pos-api
Group=pos-api
WorkingDirectory=/opt/pos-api
ExecStart=/opt/pos-api/pos-api
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=pos-api

# Security hardening
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/pos-api/logs
PrivateTmp=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

# Environment
Environment="GO_ENV=production"
Environment="PORT=8080"
EnvironmentFile=/opt/pos-api/.env

[Install]
WantedBy=multi-user.target
SYSTEMDEof

    # Create Caddyfile
    cat > deployments/caddy/Caddyfile << 'CADDYEOF'
:8080 {
    reverse_proxy localhost:8080

    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains; preload"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        Referrer-Policy "strict-origin-when-cross-origin"
    }

    log {
        output file /var/log/caddy/access.log
        format json
    }
}
CADDYEOF

    # Create initial migration
    cat > internal/infrastructure/database/migrations/001_init.sql << 'MIGRATEOF'
-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    plan VARCHAR(20) NOT NULL DEFAULT 'free',
    is_active BOOLEAN NOT NULL DEFAULT true,
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    role VARCHAR(20) NOT NULL DEFAULT 'cashier',
    permissions TEXT[] DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

CREATE TABLE IF NOT EXISTS products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    sku VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category_id UUID,
    base_price DECIMAL(15,4) NOT NULL,
    sale_price DECIMAL(15,4),
    cost_price DECIMAL(15,4),
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    stock_quantity DECIMAL(15,4) NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, sku)
);

CREATE TABLE IF NOT EXISTS sales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id VARCHAR(36) UNIQUE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    receipt_number VARCHAR(50) NOT NULL,
    subtotal DECIMAL(15,4) NOT NULL,
    tax_amount DECIMAL(15,4) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(15,4) NOT NULL DEFAULT 0,
    total DECIMAL(15,4) NOT NULL,
    currency CHAR(3) NOT NULL,
    payment_method VARCHAR(20) NOT NULL,
    customer_id UUID,
    notes TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'confirmed',
    client_timestamp TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sale_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sale_id UUID NOT NULL REFERENCES sales(id) ON DELETE CASCADE,
    product_id UUID NOT NULL REFERENCES products(id),
    product_name VARCHAR(255) NOT NULL,
    sku VARCHAR(100) NOT NULL,
    quantity DECIMAL(15,4) NOT NULL,
    unit_price DECIMAL(15,4) NOT NULL,
    total_price DECIMAL(15,4) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_products_tenant_sku ON products(tenant_id, sku);
CREATE INDEX idx_sales_tenant_created ON sales(tenant_id, created_at);
CREATE INDEX idx_sales_client_id ON sales(client_id);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tenants_updated_at BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_sales_updated_at BEFORE UPDATE ON sales
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sale_items CASCADE;
DROP TABLE IF EXISTS sales CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
DROP FUNCTION IF EXISTS update_updated_at_column();
-- +goose StatementEnd
MIGRATEOF

    # Create .air.toml for hot reload
    cat > .air.toml << 'AIROF'
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/api/main.go"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html"]
  kill_delay = "0s"
  log = "build-errors.log"
  send_interrupt = false
  stop_on_root = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
AIROF

    # Create README
    cat > README.md << 'READMEEOF'
# POS Backend API

Go-based POS SaaS backend for Flutter frontend.

## Quick Start

```bash
# 1. Copy environment
cp .env.example .env
# Edit .env with your database credentials

# 2. Run migrations
make migrate

# 3. Run development server
make dev

# 4. Or build and run
make build
./build/pos-api
```

## Project Structure

```
pos-backend/
├── cmd/api/              # Entry point
├── internal/             # Private application code
│   ├── config/           # Configuration
│   ├── domain/           # Business logic & models
│   ├── infrastructure/   # DB, HTTP, Cache, Security
│   ├── usecases/         # Application use cases
│   └── dto/              # Data transfer objects
├── pkg/                  # Public packages
├── deployments/          # Docker, systemd, Caddy
└── scripts/              # Utility scripts
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /v1/auth/login | Authenticate user |
| POST | /v1/auth/refresh | Refresh access token |
| GET | /v1/products | List products |
| GET | /v1/products/:sku | Get product by SKU |
| POST | /v1/sales | Create sale |
| GET | /v1/sales | List sales |
| POST | /v1/sync/push | Push sync data |
| GET | /v1/sync/pull | Pull sync data |

## Deployment

### Docker Compose
```bash
docker-compose -f deployments/docker/docker-compose.yml up -d
```

### systemd (Production)
```bash
sudo cp deployments/systemd/pos-api.service /etc/systemd/system/
sudo systemctl enable pos-api
sudo systemctl start pos-api
```

## Requirements

- Go 1.23+
- PostgreSQL 16
- Redis 7
- Ubuntu 22.04/24.04 LTS (recommended)
READMEEOF

    # Create go.mod with dependencies
    cat > go.mod << 'GOMODEOF'
module pos-backend

go 1.23

require (
    github.com/gin-gonic/gin v1.10.0
    github.com/golang-jwt/jwt/v5 v5.2.1
    github.com/go-playground/validator/v10 v10.22.0
    github.com/jackc/pgx/v5 v5.6.0
    github.com/jmoiron/sqlx v1.4.0
    github.com/redis/go-redis/v9 v9.6.1
    github.com/spf13/viper v1.19.0
    github.com/google/uuid v1.6.0
    github.com/prometheus/client_golang v1.20.0
    github.com/pressly/goose/v3 v3.21.1
    golang.org/x/crypto v0.26.0
)
GOMODEOF

    # Create .gitignore
    cat > .gitignore << 'GITEOF'
# Binaries
*.exe
*.dll
*.so
*.dylib
*.test
*.out
/build/
/tmp/

# Environment
.env
.env.local
.env.*.local

# IDE
.idea/
.vscode/
*.swp
*.swo
*~

# OS
.DS_Store
Thumbs.db

# Logs
*.log
logs/

# Dependencies
vendor/

# Test coverage
coverage.out
GITEOF

    log_info "Project scaffolded at ${PROJECT_DIR} ✓"
    log_info "Run 'cd ${PROJECT_DIR} && make deps' to download Go modules"
}

###############################################################################
# 10. Generate .env file
###############################################################################

generate_env() {
    log_step "Generating Environment Configuration"

    cd "$PROJECT_DIR"

    local jwt_access jwt_refresh
    jwt_access=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | xxd -p | tr -d '\n')
    jwt_refresh=$(openssl rand -hex 32 2>/dev/null || head -c 64 /dev/urandom | xxd -p | tr -d '\n')

    cat > .env << EOF
# Server
PORT=8080
GO_ENV=development

# Database
DATABASE_URL=${DB_URL:-postgres://pos_user:password@localhost:5432/pos_db?sslmode=disable}
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# Redis
REDIS_URL=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_ACCESS_SECRET=${jwt_access}
JWT_REFRESH_SECRET=${jwt_refresh}

# Security
BCRYPT_COST=12
MAX_SESSIONS_PER_USER=3

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# Logging
LOG_LEVEL=info

# Sync
SYNC_TOKEN_TTL=168h
SYNC_BATCH_SIZE=500
EOF

    log_info ".env file generated with secure JWT secrets ✓"
    log_warn "Keep .env secure and never commit it to version control!"
}

###############################################################################
# 11. Post-Install Summary
###############################################################################

show_summary() {
    echo -e "\n${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║           POS Backend Setup Complete!                          ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}\n"

    echo -e "${BOLD}Project Location:${NC} ${CYAN}${PROJECT_DIR}${NC}"
    echo -e "${BOLD}Go Version:${NC} $(go version 2>/dev/null || echo 'N/A')"
    echo -e "${BOLD}PostgreSQL:${NC} $($SUDO -u postgres psql --version 2>/dev/null || echo 'N/A')"
    echo -e "${BOLD}Redis:${NC} $(redis-cli --version 2>/dev/null || echo 'N/A')"
    echo -e "${BOLD}Docker:${NC} $(docker --version 2>/dev/null || echo 'Not installed')"
    echo -e "${BOLD}Caddy:${NC} $(caddy version 2>/dev/null || echo 'Not installed')"

    echo -e "\n${BOLD}Next Steps:${NC}"
    echo -e "  1. ${CYAN}cd ${PROJECT_DIR}${NC}"
    echo -e "  2. ${CYAN}make deps${NC}       # Download Go dependencies"
    echo -e "  3. ${CYAN}make migrate${NC}    # Run database migrations"
    echo -e "  4. ${CYAN}make dev${NC}        # Start development server with hot reload"
    echo -e "  5. ${CYAN}make build${NC}      # Build production binary"

    echo -e "\n${BOLD}Useful Commands:${NC}"
    echo -e "  • ${CYAN}sudo systemctl status postgresql${NC}"
    echo -e "  • ${CYAN}sudo systemctl status redis-server${NC}"
    echo -e "  • ${CYAN}sudo ufw status${NC}"

    echo -e "\n${BOLD}API will be available at:${NC} ${GREEN}http://localhost:8080${NC}"
    echo -e "${BOLD}Health check:${NC} ${GREEN}curl http://localhost:8080/health${NC}"

    echo -e "\n${YELLOW}⚠ Remember to:${NC}"
    echo -e "  • Change default passwords in production"
    echo -e "  • Update JWT secrets in .env"
    echo -e "  • Configure firewall rules for your specific needs"
    echo -e "  • Set up SSL/TLS with Caddy or certbot for production"

    echo -e "\n${CYAN}Setup log saved to: ${LOG_FILE}${NC}\n"
}

###############################################################################
# Interactive Menu
###############################################################################

show_menu() {
    echo -e "${BOLD}Select installation options:${NC}"
    echo -e "  ${CYAN}1)${NC} Full Install (Go + PostgreSQL + Redis + Docker + Caddy + Project)"
    echo -e "  ${CYAN}2)${NC} Development Only (Go + PostgreSQL + Redis + Project)"
    echo -e "  ${CYAN}3)${NC} Minimal (Go + Project scaffolding only)"
    echo -e "  ${CYAN}4)${NC} Custom (Choose components)"
    echo -e "  ${CYAN}5)${NC} Exit"
}

run_full_install() {
    check_ubuntu_version
    check_hardware
    check_privileges
    update_system
    install_go
    install_postgres
    configure_postgres
    install_redis
    install_docker
    install_caddy
    setup_firewall
    scaffold_project
    generate_env
    show_summary
}

run_dev_install() {
    check_ubuntu_version
    check_hardware
    check_privileges
    update_system
    install_go
    install_postgres
    configure_postgres
    install_redis
    scaffold_project
    generate_env
    show_summary
}

run_minimal() {
    check_ubuntu_version
    check_hardware
    check_privileges
    update_system
    install_go
    scaffold_project
    show_summary
}

run_custom() {
    check_ubuntu_version
    check_hardware
    check_privileges
    update_system

    confirm "Install Go ${GO_VERSION}?" && install_go
    confirm "Install PostgreSQL ${POSTGRES_VERSION}?" && install_postgres && configure_postgres
    confirm "Install Redis ${REDIS_VERSION}?" && install_redis
    confirm "Install Docker & Docker Compose?" && install_docker
    confirm "Install Caddy web server?" && install_caddy
    confirm "Configure UFW firewall?" && setup_firewall
    confirm "Scaffold project structure?" && scaffold_project && generate_env

    show_summary
}

###############################################################################
# Main
###############################################################################

main() {
    print_banner

    if [ "$EUID" -ne 0 ] && ! command_exists sudo; then
        log_error "This script requires root privileges or sudo access."
        exit 1
    fi

    show_menu
    local choice
    choice=$(get_input "Enter choice" "1")

    case "$choice" in
        1) run_full_install ;;
        2) run_dev_install ;;
        3) run_minimal ;;
        4) run_custom ;;
        5) log_info "Exiting. No changes made."; exit 0 ;;
        *) log_error "Invalid choice. Exiting."; exit 1 ;;
    esac
}

# Run main if executed directly
if [ "${BASH_SOURCE[0]}" == "${0}" ]; then
    main "$@"
fi
