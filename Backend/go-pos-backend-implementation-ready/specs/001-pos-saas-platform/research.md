# Research: POS SaaS Platform

## Decision: Keep Go/PostgreSQL as the business source of truth

**Rationale**: PostgreSQL transactions, constraints, and RLS enforce stock, money, idempotency, and tenant invariants centrally. Redis remains optional infrastructure for sessions, rate limits, and acceleration.

**Alternatives considered**: Firebase/Firestore was rejected because the product requires server-authoritative inventory, atomic sales, tenant RLS, and a versioned API shared by multiple clients.

## Decision: Flutter local-first client with a durable sync queue

**Rationale**: POS checkout must work during connectivity loss. SQLite/local durable storage should hold the catalog cache, cart drafts, completed local receipts, and pending commands. Secure storage holds credentials and device identity.

**Alternatives considered**: Remote-only Flutter requests were rejected because a network outage must not stop checkout. A fully local source of truth was rejected because multiple terminals need server reconciliation.

## Decision: Feature-first Flutter structure with thin API/data boundaries

**Rationale**: Reference POS projects show that checkout, inventory, invoices, sessions, and reports evolve independently. Feature folders keep UI and state discoverable while shared API, storage, sync, and design-system code stays in core/data packages.

**Alternatives considered**: Copying an existing Firebase Clean Architecture project was rejected because it would preserve the wrong backend contract and create migration debt.

## Decision: Exact integer minor-unit money

**Rationale**: Product prices, taxes, discounts, sale totals, and payment amounts must be deterministic across Go, PostgreSQL, and Dart. Floating-point money is not acceptable.

## Decision: Idempotent sale commands and monotonic cursor sync

**Rationale**: Offline retries and request timeouts are normal. A client command ID/idempotency key guarantees at-most-once business application, while a server sequence provides stable pull ordering.

## Decision: Android-first, responsive Flutter UI

**Rationale**: The primary device is an Android POS terminal, but the reference projects demonstrate useful desktop-width layouts. The client will support compact mobile/tablet layouts first and keep the domain/data layers platform-neutral.

## Decision: Cash/card metadata before payment gateway integration

**Rationale**: The first release needs a working checkout workflow without coupling business completion to a provider. Payment gateway integrations can be added behind a payment service after the sale invariants are stable.
