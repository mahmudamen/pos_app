# Feature Specification: POS SaaS Platform

**Feature Branch**: `001-pos-saas-platform`
**Created**: 2026-09-08
**Status**: Draft
**Input**: Build a multi-tenant POS SaaS product with a Go/PostgreSQL backend and a new Flutter client, replacing Firebase-backed workflows.

## User Scenarios & Testing

### User Story 1 - Sign in a terminal (Priority: P1)

As a cashier, I can sign in to a store on a named POS terminal so that my work is associated with the correct store, user, device, and session.

**Why this priority**: No protected POS operation is possible without trusted identity and tenant context.

**Independent Test**: Seed one store, user, and terminal; sign in through the Flutter client; verify an access session is returned and protected API calls use the store context.

**Acceptance Scenarios**:

1. **Given** an active store user and terminal, **When** valid credentials are submitted, **Then** the client receives an access token, rotated refresh token, user identity, and store identity.
2. **Given** invalid credentials, **When** sign-in is submitted, **Then** no session is created and a sanitized authentication error is returned.
3. **Given** an expired or reused refresh token, **When** refresh is requested, **Then** the request is rejected and the related session is revoked.

### User Story 2 - Complete a sale (Priority: P1)

As a cashier, I can search or scan products, add them to a cart, review exact totals, accept payment, and receive a sale result.

**Why this priority**: Checkout is the core business value of a POS system.

**Independent Test**: Seed products with prices and stock; authenticate; add products in the Flutter client; submit checkout; verify one atomic sale, sale items, and stock decrement.

**Acceptance Scenarios**:

1. **Given** active products with available stock, **When the cashier completes checkout,** **Then** the server calculates totals from stored prices and commits the sale and stock changes atomically.
2. **Given** insufficient stock, **When checkout is submitted,** **Then** the sale is rejected and no stock or sale rows change.
3. **Given** a retried checkout with the same tenant-scoped idempotency key, **When the request is repeated,** **Then** the original sale result is returned without creating a duplicate sale.
4. **Given** products belonging to another tenant, **When their identifiers are submitted,** **Then** the sale cannot read or modify them.

### User Story 3 - Continue offline and synchronize (Priority: P2)

As a cashier, I can continue viewing cached products and record sales when the network is unavailable, then synchronize safely when connectivity returns.

**Why this priority**: Retail checkout must remain usable during unreliable connectivity.

**Independent Test**: Disconnect the client, create a queued sale, restore connectivity, run synchronization, and verify the command is applied once and marked complete.

**Acceptance Scenarios**:

1. **Given previously synchronized catalog data and no network,** **When a cashier opens checkout,** **Then cached products remain searchable and the client clearly shows offline state.
2. **Given an offline sale,** **When connectivity returns,** **Then the client uploads a stable command ID and the server applies it at most once.
3. **Given a synchronization conflict,** **When the server evaluates the command,** **Then the client receives an explicit accepted, rejected, or conflict result.

### User Story 4 - Manage store operations (Priority: P3)

As a manager, I can manage products, stock, customers, invoices, refunds, sessions, and reports with role-appropriate permissions.

**Why this priority**: Operational workflows make the POS useful beyond the checkout counter, but depend on the P1 flows.

**Independent Test**: Sign in as manager and cashier; verify permitted screens/actions and denied actions; create a product, inspect a sale, and produce a daily report.

**Acceptance Scenarios**:

1. **Given manager permissions,** **When a product is created or stock is adjusted,** **Then the change is tenant-scoped and appears in the catalog.
2. **Given cashier permissions,** **When a restricted management action is attempted,** **Then the action is denied without exposing other tenant data.
3. **Given completed sales,** **When a manager opens reports or invoices,** **Then totals and transaction history agree with committed sales.

## Edge Cases

- A duplicate product identifier or barcode is rejected within a tenant but may exist in another tenant.
- A sale with an empty item list, zero quantity, negative amount, unknown product, or inactive product is rejected.
- Concurrent sales for the last units of stock cannot produce negative stock.
- A client retry after a timeout returns the original idempotent result.
- A refresh token replay revokes the session family and requires sign-in.
- Missing, malformed, or cross-tenant authorization context returns a sanitized error.
- Local storage corruption or an expired offline queue must be surfaced without silently losing pending work.
- A printer or payment device failure must not create a sale unless the configured payment workflow confirms success.

## Requirements

### Functional Requirements

- **FR-001**: System MUST authenticate users within a tenant and bind sessions to a registered device.
- **FR-002**: System MUST issue short-lived access tokens and rotating refresh tokens, storing only refresh-token hashes.
- **FR-003**: System MUST enforce tenant isolation in application authorization and PostgreSQL row-level security.
- **FR-004**: System MUST provide versioned APIs for authentication, catalog, sales, synchronization, and operations.
- **FR-005**: System MUST store money as exact integer minor units with an explicit currency.
- **FR-006**: System MUST calculate sale totals from server-owned product prices and commit sale items and stock changes atomically.
- **FR-007**: System MUST support tenant-scoped idempotency for retryable sale commands.
- **FR-008**: Flutter MUST cache catalog and durable pending commands locally for offline checkout.
- **FR-009**: Synchronization MUST use stable command identifiers and a monotonic server cursor.
- **FR-010**: System MUST provide product search, SKU/barcode lookup, cart editing, checkout, and sale history.
- **FR-011**: System MUST provide manager and cashier authorization boundaries.
- **FR-012**: System MUST provide structured errors, request IDs, health/readiness checks, and sanitized logs.
- **FR-013**: System MUST support receipt generation/printing through a client-side printer integration without making printer availability the business source of truth.
- **FR-014**: System MUST support backups, restore verification, metrics, rate limiting, and production deployment documentation before release.

### Key Entities

- **Tenant**: A store or business boundary containing users, devices, catalog, customers, and sales.
- **User**: A staff identity with a role and active/inactive state.
- **Device**: A registered POS terminal bound to a tenant and session.
- **Session**: A revocable login state with refresh-token rotation metadata.
- **Product**: A tenant-owned sellable item with SKU/barcode, minor-unit price, currency, and stock.
- **Customer**: An optional tenant-owned customer record associated with sales.
- **Sale**: A committed transaction with exact totals and a tenant-scoped idempotency key.
- **SaleItem**: A snapshot of a product and quantity at sale time.
- **SyncCommand**: A durable client command with a stable ID and processing result.
- **ChangeCursor**: A monotonic server position used to pull tenant changes.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A trained cashier completes a normal product-to-sale workflow in under 60 seconds after sign-in.
- **SC-002**: Barcode/product lookup responds within 200 ms at the repository/API target under representative load.
- **SC-003**: Retrying the same sale command 10 times creates exactly one logical sale.
- **SC-004**: Cross-tenant read/write isolation tests demonstrate zero unauthorized records exposed or changed.
- **SC-005**: Offline sales synchronize successfully after reconnection without duplicate application or lost queued commands.
- **SC-006**: 95% of primary checkout test users complete checkout without assistance.
- **SC-007**: Backend tests, race checks, static analysis, Flutter analysis, and Flutter tests pass before each release.

## Assumptions

- The first client target is Android POS terminals; desktop support may follow using the same Flutter domain/data layers.
- Go/PostgreSQL replaces Firebase as the backend source of truth.
- Redis is an optimization and session/rate-limit dependency, not the business data store.
- Payments are initially recorded as cash/card method metadata; external payment gateway integration is a later phase.
- The existing GitHub-derived Flutter repositories are references only; the new client is `Flutter/pos_go_app`.
- English is the initial client locale; Arabic RTL is planned before production rollout.
- Local SQLite/secure storage is used for client durability, while the backend remains authoritative after synchronization.
