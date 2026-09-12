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
- [ ] AUTH-011 Auth API integration tests.

## CATALOG

- [ ] CAT-001 Category repository.
- [ ] CAT-002 Category service.
- [ ] CAT-003 Category HTTP API.
- [ ] CAT-004 Product repository.
- [ ] CAT-005 Product service and validation.
- [ ] CAT-006 Product/barcode API and tests.

## SALES

- [ ] SALE-001 Define sale domain model.
- [ ] SALE-002 Define money/tax/discount calculation rules.
- [ ] SALE-003 Implement tenant-scoped idempotency keys.
- [ ] SALE-004 Implement atomic sale transaction.
- [ ] SALE-005 Implement stock policy.
- [ ] SALE-006 Implement sale repository.
- [ ] SALE-007 Implement sale HTTP API.
- [ ] SALE-008 Add concurrency and duplicate-request tests.

## SYNC

- [x] SYNC-001 Implement change sequence.
- [x] SYNC-002 Implement pull cursor.
- [x] SYNC-003 Implement cursor expiry.
- [x] SYNC-004 Implement push command IDs.
- [x] SYNC-005 Implement replay-safe command processing.
- [x] SYNC-006 Define per-entity conflict policy.
- [ ] SYNC-007 Add offline replay integration tests. **Partial**: curl E2E covers apply → replay → conflict → dedupe → reject; the Flutter offline queue pushes through `/v1/sync/push`, but an automated device-level offline→online replay test is not written (needs `flutter drive` or an emulator walkthrough).

## OPS

- [x] OPS-001 Add Prometheus metrics.
- [x] OPS-002 Add request latency/error metrics.
- [ ] OPS-003 Add production Dockerfile.
- [ ] OPS-004 Add systemd service.
- [ ] OPS-005 Add Caddy reverse proxy configuration.
- [ ] OPS-006 Add backup/restore runbook.
- [ ] OPS-007 Add production security checklist.

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
