# Tasks: POS SaaS Platform

## Phase 1 - Setup

- [x] T001 [P] Add SpecKit-managed project instructions and feature metadata in `.specify/feature.json`
- [x] T002 [P] Document backend and Flutter quickstart validation in `specs/001-pos-saas-platform/quickstart.md`
- [ ] T003 [P] Add CI jobs for Go tests/vet and Flutter analyze/test in `.github/workflows/ci.yml`

## Phase 2 - Foundational Backend

- [x] T004 Add PostgreSQL local role/database bootstrap in `scripts/bootstrap_local.sql`
- [x] T005 Complete repeatable Goose migrations and rollback checks in `internal/infrastructure/database/migrations/`
- [x] T006 Implement tenant transaction helper and context-aware repository base in `internal/infrastructure/database/tenant.go`
- [ ] T007 [P] Add RLS cross-tenant integration fixtures in `internal/infrastructure/database/tenant_integration_test.go`
- [x] T008 [P] Add typed application errors and response mapping in `internal/errors/` and `internal/transport/http/`
- [x] T009 Add bearer authentication middleware and claims context in `internal/transport/http/auth.go`
- [ ] T010 Add password policy, refresh replay detection, revocation, and session limits in `internal/transport/auth/handler.go`
- [ ] T011 Add auth integration tests against PostgreSQL in `internal/transport/auth/handler_integration_test.go`

## Phase 3 - User Story 1: Sign In a Terminal

**Goal**: A cashier can securely sign in and maintain a tenant/device session.

**Independent test**: Seed tenant/user/device data, call login and refresh, then verify valid claims and rejected replay.

- [ ] T012 [P] [US1] Add login/refresh contract fixtures in `specs/001-pos-saas-platform/contracts/auth-fixtures.json`
- [x] T013 [US1] Return role and permission claims from `internal/transport/auth/handler.go`
- [x] T014 [US1] Add logout and session revocation endpoints in `internal/transport/auth/handler.go`
- [x] T015 [US1] Add Flutter secure session restore and logout in `Flutter/pos_go_app/lib/core/session_store.dart`
- [x] T016 [US1] Add Flutter token-refresh retry behavior in `Flutter/pos_go_app/lib/core/api_client.dart`
- [x] T017 [US1] Add login/session widget tests in `Flutter/pos_go_app/test/auth_test.dart`

## Phase 4 - User Story 2: Complete a Sale

**Goal**: A cashier can find products, build a cart, and commit one exact, stock-safe sale.

**Independent test**: Authenticate against seeded PostgreSQL data, submit checkout, verify sale/items/stock, then retry the same key.

- [ ] T018 [P] [US2] Add category/product repository interfaces in `internal/domain/repositories/`
- [ ] T019 [P] [US2] Add product management validation in `internal/usecases/catalog.go`
- [x] T020 [US2] Add authenticated category/product CRUD endpoints in `internal/transport/catalog/handler.go`
- [ ] T021 [US2] Add sale repository and transaction service in `internal/usecases/sales.go`
- [ ] T022 [US2] Add atomic stock/idempotency integration tests in `internal/transport/sales/handler_integration_test.go`
- [x] T023 [US2] Add product ID/cart line model and quantity controls in `Flutter/pos_go_app/lib/features/pos/pos_screen.dart`
- [ ] T024 [US2] Add payment method and checkout confirmation state in `Flutter/pos_go_app/lib/features/pos/payment_sheet.dart`
- [x] T025 [US2] Add sale success/error/retry tests in `Flutter/pos_go_app/test/api_client_test.dart`

## Phase 5 - User Story 3: Offline and Synchronize

**Goal**: The terminal continues checkout with cached data and synchronizes each command once after reconnection.

**Independent test**: Disable network, enqueue a sale, restore network, push it, and verify one server result and one completed local command.

- [x] T026 [P] [US3] Add local SQLite schema for cached products and pending commands in `Flutter/pos_go_app/lib/core/storage/local_database.dart`
- [ ] T027 [P] [US3] Add connectivity state service in `Flutter/pos_go_app/lib/core/connectivity/connectivity_service.dart`
- [x] T028 [US3] Add sync command and cursor migrations in `internal/infrastructure/database/migrations/005_sync.sql`
- [ ] T029 [US3] Add pull/push synchronization contracts in `internal/transport/sync/handler.go`
- [ ] T030 [US3] Add server command replay and conflict service in `internal/usecases/sync.go`
- [ ] T031 [US3] Add Flutter pending-command repository and retry worker in `Flutter/pos_go_app/lib/features/sync/`
- [ ] T032 [US3] Add offline catalog/cart and reconnect UI state in `Flutter/pos_go_app/lib/features/pos/`
- [ ] T033 [US3] Add offline replay integration tests in `internal/transport/sync/handler_integration_test.go` and `Flutter/pos_go_app/test/sync_test.dart`

## Phase 6 - User Story 4: Manage Store Operations

**Goal**: Managers can operate the store while cashier permissions remain limited.

**Independent test**: Exercise manager and cashier sessions against products, customers, invoices, refunds, sessions, and reports.

- [x] T034 [P] [US4] Add role/permission migration and policies in `internal/infrastructure/database/migrations/004_permissions.sql`
- [ ] T035 [P] [US4] Add customer and category APIs in `internal/transport/operations/handler.go`
- [ ] T036 [US4] Add invoice history and sale retrieval in `internal/transport/sales/handler.go`
- [ ] T037 [US4] Add refund transaction with stock restoration in `internal/usecases/refunds.go`
- [ ] T038 [US4] Add manager dashboard/report endpoints in `internal/transport/operations/handler.go`
- [ ] T039 [US4] Add Flutter manager navigation and permission guards in `Flutter/pos_go_app/lib/features/operations/`
- [ ] T040 [US4] Add receipt preview and printer adapter boundary in `Flutter/pos_go_app/lib/features/operations/printing/`
- [ ] T041 [US4] Add product, invoice, refund, and permission widget tests in `Flutter/pos_go_app/test/operations_test.dart`

## Phase 7 - Polish and Production

- [ ] T042 [P] Add Prometheus request/sale/sync metrics in `internal/observability/`
- [ ] T043 [P] Add rate limiting and security headers in `internal/transport/http/`
- [ ] T044 [P] Add Flutter Arabic localization and RTL support in `Flutter/pos_go_app/lib/l10n/`
- [ ] T045 [P] Add backup/restore and release runbook in `docs/11_OPERATIONS.md`
- [ ] T046 Add production image, systemd, and Caddy deployment files in `deployments/`
- [ ] T047 Run race, load, migration, Flutter, and API contract checks and record results in `specs/001-pos-saas-platform/quickstart.md`

## Dependencies

`T004-T011` block all user stories. `US1` must complete before authenticated `US2`; `US2` must complete before `US3`; `US4` depends on stable sales and permissions. `T042-T047` may begin incrementally after the relevant story surfaces exist.

## Parallel Opportunities

- T007, T008, and T012 can proceed after the foundational schema is stable.
- T018/T019 and T023 can proceed in parallel once the product contract is fixed.
- T026/T027 and T028 can proceed in parallel before sync handler integration.
- T034/T035 and T039 can proceed in parallel after roles are defined.
- T042-T046 are independently parallelizable by subsystem.

## MVP Scope

Deliver T004-T025: tenant-safe authentication, remote catalog, cart, atomic checkout, and Flutter tests. Defer offline queue, operations, reporting, printing, and production deployment until the MVP sale path is proven against PostgreSQL.

## Definition of Done

Every completed task must include relevant tests, pass `go test ./...`, `go vet ./...`, Flutter analysis/tests where applicable, preserve tenant isolation, avoid sensitive logging, respect context cancellation, and update the contract or quickstart when behavior changes.
