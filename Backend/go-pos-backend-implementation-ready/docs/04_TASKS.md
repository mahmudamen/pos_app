# Coding Tasks

## BOOT

- [x] BOOT-001 Create application configuration package.
- [x] BOOT-002 Implement structured logger.
- [x] BOOT-003 Implement PostgreSQL pool.
- [x] BOOT-004 Implement Redis client.
- [x] BOOT-005 Implement HTTP server and graceful shutdown.
- [x] BOOT-006 Implement request ID/recovery middleware.
- [x] BOOT-007 Add `/health/live` and `/health/ready`.
- [x] BOOT-008 Add Docker Compose development environment.
- [x] BOOT-009 Add CI quality gates.

## DATABASE

- [x] DB-001 Create extensions migration.
- [x] DB-002 Create tenants and users.
- [x] DB-003 Create devices/sessions/refresh tokens.
- [x] DB-004 Create categories/products.
- [x] DB-005 Create customers.
- [x] DB-006 Create sales/sale_items.
- [x] DB-007 Add tenant-scoped indexes and constraints.
- [x] DB-008 Add RLS policies to every tenant-scoped table.
- [x] DB-009 Add monotonic change sequence.
- [x] DB-010 Add integration tests proving cross-tenant access is denied.

## AUTH

- [x] AUTH-001 Password hashing service.
- [x] AUTH-002 Credential validation.
- [x] AUTH-003 Access JWT issuance.
- [x] AUTH-004 Access JWT validation with issuer/algorithm/type checks.
- [x] AUTH-005 Refresh-token hashing.
- [x] AUTH-006 Refresh rotation.
- [x] AUTH-007 Refresh replay/reuse detection.
- [x] AUTH-008 Session registration.
- [x] AUTH-009 Session revocation.
- [x] AUTH-010 Maximum-session eviction using explicit ordering.
- [x] AUTH-011 Auth API integration tests. DB-backed (`handler_integration_test.go`): login happy path, wrong-password 401, refresh rotation + AUTH-007 reuse revocation, logout revocation. Runs against real Postgres via `make integration-test`; skips without `TEST_DATABASE_URL`.

## CATALOG

- [x] CAT-001 Category repository. CRUD backed by the shared handler/SQL concurrency (soft-delete via `is_active`); no separate repo layer — matches the codebase convention (see `05_CODING_STANDARDS.md`).
- [x] CAT-002 Category service. Validation + business rules live in the categories/products handlers; `is_active` soft-delete and slug/parent handling covered by handler tests.
- [x] CAT-003 Category HTTP API. `GET/POST/PATCH/DELETE /v1/categories/:id` with 401/404/400 handling.
- [x] CAT-004 Product repository. Product SQL access in the catalog handler: create, update (PATCH), soft-delete, pagination, `stock_quantity`.
- [x] CAT-005 Product service and validation. Name/sku/price/currency validation, duplicate-SKU and stock handling, unit-tested.
- [x] CAT-006 Product/barcode API and tests. `GET /v1/products/barcode/:barcode`, pagination/search, 5 DB-backed integration tests (create/read/soft-delete).

## SALES

- [x] SALE-001 Define sale domain model. `Sale`/`SaleItem`/`Payment`/`CreateSaleRequest` in `internal/transport/sales/`.
- [x] SALE-002 Define money/tax/discount calculation rules. Integer minor units; subtotal/ discount/tax/total + `validateDiscount` (cashier pct cap) in `discounts.go`.
- [x] SALE-003 Implement tenant-scoped idempotency keys. `UNIQUE(tenant_id, idempotency_key)`; replay returns the stored sale; race → 409 `idempotency_conflict`.
- [x] SALE-004 Implement atomic sale transaction. Sale + items + payments + stock + loyalty in one tx with row locks (`FOR UPDATE`).
- [x] SALE-005 Implement stock policy. `FOR UPDATE` lock, `insufficient_stock` 409, non-negative invariant.
- [x] SALE-006 Implement sale repository. SQL access in `sale.go` (shared with sync push via `CreateSale`).
- [x] SALE-007 Implement sale HTTP API. `GET /v1/sales`, `GET /v1/sales/:id`, `POST /v1/sales` (split tender, session_id, customer_id, discount).
- [x] SALE-008 Add concurrency and duplicate-request tests. Unit suites + DB-backed `handler_integration_test.go`: idempotent duplicate returns same sale (stock unchanged) and 5-way concurrent same-key requests yield exactly 1 sale row.

## SYNC

- [x] SYNC-001 Implement change sequence.
- [x] SYNC-002 Implement pull cursor.
- [x] SYNC-003 Implement cursor expiry.
- [x] SYNC-004 Implement push command IDs.
- [x] SYNC-005 Implement replay-safe command processing.
- [x] SYNC-006 Define per-entity conflict policy.
- [ ] SYNC-007 Add offline replay integration tests. **Partial**: curl E2E covers apply → replay → conflict → dedupe → reject; server-side is now also DB-backed (`internal/transport/sync/handler_integration_test.go` — push apply/replay/conflict/dedupe repro against real Postgres). Remaining: an automated device-level offline→online replay test (needs `flutter drive` or an emulator walkthrough).

## OPS

- [x] OPS-001 Add Prometheus metrics.
- [x] OPS-002 Add request latency/error metrics.
- [x] OPS-003 Add production Dockerfile. Multistage `golang:1.23-alpine` build → `alpine:3.20` runtime, non-root `pos-api` user, goose + migrations, `/health/live` HEALTHCHECK.
- [x] OPS-004 Add systemd service. `deployments/systemd/pos-api.service` (hardened unit: NoNewPrivileges, ProtectSystem).
- [x] OPS-005 Add Caddy reverse proxy configuration. `deployments/caddy/Caddyfile` — TLS, HSTS, security headers; wired into `docker-compose.prod.yml`.
- [x] OPS-006 Add backup/restore runbook. `scripts/backup.sh` / `scripts/restore.sh` + runbook section in `11_OPERATIONS.md` (schedule, restore verification, rollback).
- [x] OPS-007 Add production security checklist. `14_SECURITY_CHECKLIST.md` (secrets, transport, hardening, ops, tenant data, deployment).

## Definition of Done

Every task must:

- compile;
- be gofmt-clean;
- pass vet;
- include appropriate tests;
- preserve tenant isolation;
- use typed/sanitized errors;
- respect context cancellation;
- avoid secret/sensitive logging;
- update relevant documentation;
- pass the applicable CI quality gates.
