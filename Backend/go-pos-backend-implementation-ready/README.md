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
