# Implementation Plan: POS SaaS Platform

**Branch**: `001-pos-saas-platform` | **Date**: 2026-09-08 | **Spec**: [spec.md](spec.md)

## Summary

Build a Firebase-free, multi-tenant POS platform in staged vertical slices. The Go API and PostgreSQL database own authentication, catalog, stock, sales, tenant isolation, synchronization, and operational reporting. The Flutter client provides an Android-first cashier experience with secure sessions, local cache, durable offline commands, checkout, receipts, and management workflows.

## Technical Context

**Language/Version**: Go 1.23+, Dart/Flutter 3.3+/Dart 3.3+

**Primary Dependencies**: Gin, pgx/v5, PostgreSQL 16, Redis 7, Goose, JWT v5, bcrypt, Flutter `http`, `flutter_secure_storage`, SQLite-compatible local persistence

**Storage**: PostgreSQL server source of truth; Redis sessions/rate limits; Flutter secure storage plus local SQLite cache/queue

**Testing**: Go unit/integration tests, race detector, vet, Goose validation, Flutter analyze, Flutter unit/widget tests, API contract tests

**Target Platform**: Go Linux service; Android POS terminals first; responsive Flutter desktop/web support later

**Project Type**: Multi-project mobile client plus API service

**Performance Goals**: Barcode lookup target under 200 ms at repository/API level; normal checkout under 60 seconds for a trained cashier; bounded sync pages

**Constraints**: Tenant isolation at app and PostgreSQL RLS layers; exact integer money; offline-capable client; no sensitive logs; idempotent writes; backward-compatible `/v1` contract

**Scale/Scope**: Multi-tenant retail/food POS; initial MVP supports authentication, product catalog, checkout, stock, and durable synchronization; management/reporting follows the core checkout path

## Constitution Check

- **Correctness**: Pass. PostgreSQL transactions and constraints own business invariants.
- **Multi-tenancy**: Pass. JWT claims, authorization, transaction-local RLS context, and RLS policies are required.
- **Security**: Pass. Secrets stay outside source, access tokens are short-lived, refresh tokens rotate and are hashed.
- **Data integrity**: Pass. Minor-unit money, idempotency keys, foreign keys, stock locks, and unique constraints are specified.
- **Maintainability**: Pass. Thin handlers, use-case/service boundaries, repository-owned SQL, context-aware I/O.
- **Compatibility**: Pass. Flutter consumes the versioned `/v1` contract; breaking changes require a new version.
- **Observability**: Pass. Request IDs, structured logs, health/readiness, metrics, and sanitized errors are required.
- **Quality**: Pass. Every phase has tests, static checks, documentation, and operational validation.

## Delivery Phases

### Phase 0 - SpecKit and project foundation

Install and initialize SpecKit, preserve the constitution, document the product specification, establish API contracts, and make Go/Flutter validation commands reproducible.

### Phase 1 - Backend platform and tenant security

Finish migrations, local database bootstrap, RLS transaction helpers, repository boundaries, integration tests, JWT middleware, session revocation, replay detection, and role claims.

### Phase 2 - Flutter shell and authenticated catalog

Build the new `Flutter/pos_go_app` app shell, secure session lifecycle, API client, local catalog cache, responsive cashier layout, loading/error/offline states, and product/barcode lookup.

### Phase 3 - Atomic checkout

Implement server-calculated sales, locked stock updates, idempotency, payment metadata, Flutter cart checkout, receipt-ready sale state, and retry-safe client behavior.

### Phase 4 - Offline synchronization

Implement local SQLite tables, pending command queue, server change cursor, pull/push APIs, conflict results, retry policy, and reconnect UI.

### Phase 5 - Operations and management

Add categories/products/customers management, invoices, refunds, cash sessions, roles/permissions, low-stock alerts, reports, receipt printing, and Arabic RTL.

### Phase 6 - Production hardening

Add metrics, rate limiting, security headers, backups/restore drills, production deployment, monitoring, CI gates, performance tests, and release documentation.

## Project Structure

```text
Backend/go-pos-backend-implementation-ready/
├── cmd/api/
├── internal/config/
├── internal/infrastructure/database/
├── internal/infrastructure/security/
├── internal/transport/auth/
├── internal/transport/catalog/
├── internal/transport/sales/
├── internal/usecases/
├── internal/infrastructure/database/migrations/
├── specs/001-pos-saas-platform/
└── deployments/

Flutter/pos_go_app/
├── lib/core/                 # API, secure session, connectivity, local storage
├── lib/features/auth/
├── lib/features/catalog/
├── lib/features/pos/
├── lib/features/sync/
├── lib/features/operations/
└── test/
```

## Dependency Graph

```text
SpecKit/spec artifacts
        |
Backend migrations + RLS + auth --------------------+
        |                                             |
Flutter shell + API/session --------------------------+--> Catalog and checkout
                                                      |
                                      Offline sync ---+--> Operations/management
                                                      |
                                             Hardening/release
```

## Implementation Strategy

Deliver the P1 MVP first: sign-in, tenant-safe catalog, cart, atomic checkout, and receipt-ready result. Keep each phase independently testable. Do not build management screens or payment integrations before the core sale transaction and offline command model are stable.

## Risks and Mitigations

- **Local database setup drift**: provide Docker Compose and administrator bootstrap SQL; validate migrations in CI.
- **Tenant leakage**: require a shared tenant transaction helper and cross-tenant integration tests before catalog/sales completion.
- **Offline duplicate sales**: require stable command IDs and server idempotency before enabling offline checkout.
- **Flutter/backend contract drift**: keep API contracts beside the SpecKit feature and add client contract tests for every write endpoint.
- **Printer/payment failures**: treat printing as post-sale fulfillment and payment providers as replaceable adapters.
