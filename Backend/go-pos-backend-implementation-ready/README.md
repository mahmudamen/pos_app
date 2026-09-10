# Go POS SaaS Backend — Implementation-Ready Blueprint

This repository is the implementation baseline for a multi-tenant Go POS SaaS backend designed to work with a Flutter POS client.

## Stack

- Go 1.23+
- Gin HTTP API
- PostgreSQL 16
- Redis 7
- pgx/v5
- JWT access + rotating refresh tokens
- Goose migrations
- Prometheus metrics
- Structured JSON logging
- Docker Compose for development
- systemd + Caddy for production

## Source of truth

- `docs/00_CONSTITUTION.md` — engineering constitution
- `docs/01_REQUIREMENTS.md` — requirements
- `docs/02_ARCHITECTURE.md` — architecture
- `docs/03_PLAN.md` — implementation plan
- `docs/04_TASKS.md` — coding task backlog
- `docs/05_CODING_STANDARDS.md` — coding rules
- `docs/06_SECURITY.md` — security requirements
- `docs/07_API_CONTRACT.md` — API contract
- `docs/08_DATABASE.md` — database model
- `docs/09_SYNC.md` — offline synchronization
- `docs/10_TESTING.md` — quality strategy
- `docs/11_OPERATIONS.md` — deployment and operations
- `speckit.constitution`, `speckit.plan`, `speckit.tasks` — SpecKit-compatible execution files

## Bootstrap

```bash
cp .env.example .env
./scripts/setup_dev.sh
make migrate
make run
```

For a machine using a locally installed PostgreSQL instead of Docker, run
`scripts/bootstrap_local.sql` once as a PostgreSQL administrator, then use the
same `make migrate` and `make run` commands. Redis is optional for the initial
login flow but is required for readiness when configured.

Health:

```bash
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

## Important

`go.mod` and `go.sum` are the dependency manifests. A Python `requirements.txt` is intentionally not used.

The repository starts with the infrastructure and database foundation. Domain APIs are implemented task-by-task rather than as one large uncontrolled change.

## Portable Docker (Ubuntu)

The whole stack is containerized: **dev** and **deployment** versions both run with only Docker + Compose v2 installed (no host Go/Postgres/Redis/Caddy needed).

### Fresh Ubuntu host

```bash
sudo ./scripts/setup_ubuntu.sh   # installs Docker Engine + Compose v2, then helps start a stack
```

### Development version

```bash
./scripts/docker-dev.sh          # or: make docker-dev
```

Builds `pos-dev`: postgres + redis + goose migration runner + API with the source tree
bind-mounted (`http://localhost:8080`). Reload code with
`docker compose -f deployments/docker/docker-compose.dev.yml restart api`.
Running it requires only Docker — not a local Go toolchain.

### Deployment version

```bash
./scripts/docker-prod.sh         # or: make docker-prod
```

The first run creates `.env.prod` from `.env.prod.example` and fills in random
DB/JWT secrets. Set `SITE_DOMAIN` (and `CORS_ALLOWED_ORIGINS`) in `.env.prod`,
then re-run. Builds `pos-prod`: postgres + redis + one-shot migrate service +
immutable API image + Caddy (automatic TLS for a real domain, ports 80/443).
Migrations run automatically inside the stack (same image, goose binary +
migrations baked in). The API is only bound to `127.0.0.1:8080`; the network
between Caddy → API → Postgres/Redis stays private.

### Notes

- The API container image (`Dockerfile`) is self-contained: API binary, goose
  CLI, and `internal/infrastructure/database/migrations` are baked in, so the
  same image both migrates and serves.
- `deployments/docker/docker-compose.yml` still provides just the Postgres/Redis
  containers for the host-based `make db-up` / `./scripts/setup_dev.sh` workflow.
- Compose reads secrets via environment interpolation (`${VAR:?...}`), so a
  missing production secret fails fast instead of silently. Keep `.env` and
  `.env.prod` out of version control (they are gitignored).
