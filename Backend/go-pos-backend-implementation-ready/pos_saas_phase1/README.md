# Go POS SaaS Backend — Phase 1

Phase 1 establishes the production foundation for the multi-tenant POS SaaS backend.

## Scope

- Go 1.23+
- Gin
- PostgreSQL 16
- Redis 7
- pgx/v5
- Goose migrations
- Structured JSON logging
- Request IDs
- Centralized configuration
- Health/readiness endpoints
- JWT access-token foundation
- Argon2id password hashing
- Users, tenants, memberships, roles, permissions
- Sessions
- Rotating refresh-token foundation
- Tenant isolation foundation
- Audit/security event foundation
- Redis-backed rate-limit foundation
- Docker Compose
- Unit/integration-test foundation

Phase 1 deliberately does not implement the complete POS domain.

## Run

```bash
cp .env.example .env
docker compose up -d postgres redis
go run ./cmd/api
```

Health:

```bash
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

Run tests:

```bash
go test ./...
go test -race ./...
go vet ./...
```
