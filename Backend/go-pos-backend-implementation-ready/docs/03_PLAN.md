# Implementation Plan

## Phase 0 — Bootstrap

Deliver:

- Go module.
- Configuration loader.
- Docker Compose.
- DB/Redis connectivity.
- Goose migrations.
- HTTP server.
- health endpoints.
- graceful shutdown.
- CI quality gates.

Exit: developer can clone, configure, migrate, run, and verify health.

## Phase 1 — Database foundation

Deliver:

- extensions.
- tenants.
- users.
- devices.
- sessions.
- refresh tokens.
- categories.
- products.
- customers.
- sales.
- sale items.
- indexes.
- RLS.
- audit/change sequence primitives.

Exit: migrations are repeatable and RLS tests prove tenant isolation.

## Phase 2 — Authentication

Deliver:

- password hashing.
- login.
- access JWT.
- refresh rotation.
- session management.
- logout/revocation.
- rate limits.

Exit: authentication and refresh replay tests pass.

## Phase 3 — Catalog

Deliver:

- category APIs.
- product APIs.
- barcode lookup.
- authorization.
- validation.
- repository integration tests.

Exit: CRUD and lookup APIs are tested and tenant-safe.

## Phase 4 — Sales

Deliver:

- sale command model.
- idempotency.
- atomic transaction.
- totals.
- stock rules.
- sale retrieval.

Exit: duplicate retry creates one logical sale.

## Phase 5 — Sync

Deliver:

- monotonic `change_seq`.
- pull cursor.
- push command IDs.
- replay safety.
- conflict rules.
- cursor expiry.

Exit: offline replay tests pass.

## Phase 6 — Hardening

Deliver:

- metrics.
- tracing/log correlation.
- rate limiting.
- security headers.
- body limits.
- benchmarks.
- load tests.
- backup verification.

## Phase 7 — Production

Deliver:

- container/image build.
- systemd or container deployment.
- Caddy/TLS.
- monitoring.
- backups.
- restore drill.
- rollback runbook.
