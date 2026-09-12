# Roadmap

Product north star: a **multi-tenant POS SaaS** (Go/Gin backend + Flutter client) for the
Egyptian market — Arabic-first, EGP-native, offline-resilient, vertically typed per industry.

This roadmap was informed by a study of Odoo POS deployments (XamlTech, odoo-pos-egypt) —
what merchants actually buy: offline continuity, real-time stock, receipts, cashier
security, payment flexibility, restaurant mode, and per-industry depth.

Status legend:
- `[x]` done
- `*` in progress
- `-` skipped / explicitly out of scope for now
- `[ ]` planned (next)

## Phase A — Close the checkout gap (merchants feel this immediately)

- [ ] A1 Receipt generation: `POST /v1/receipts/:id` returns ESC/POS thermal layout (Arabic, logo, QR). Flutter sends to Epson/Star via Bluetooth/USB. *Skip for now — hardware-dependent.*
- [x] A2 Stock-on-hand badges on POS product grid (live qty on each card; backend already returns `stock_quantity`).
- [x] A3 Offline sale queue: checkout is queued in SQLite when the API is unreachable and replayed (idempotent) when back online. "Zero lost sales."
- [x] A4 Payments: `payment_method` + split payments (cash/card/mobile) on `POST /v1/sales` (migration `008_payments`; empty `payments` defaults to legacy cash). Verified on device (split card+cash sale; history shows method chip).
- [x] A5 Inventory adjustments: `POST/GET /v1/inventory/adjustments` with reason codes (damaged, restock, count) — migration `012_inventory_adjustments`, RLS + manager-only RBAC (`inventory.adjust`).

## Phase B — Operations & security

- [x] B1 Cashier security (server side): discount limits per role enforced **server-side** (mirrors Odoo's "backend validation prevents frontend bypass" lesson). `POST /v1/sales` accepts optional `discount_minor`; non-zero discounts are RBAC-gated (`pos.discount`) and cashier discounts are capped at `CASHIER_DISCOUNT_PCT` (default 5), managers/owners uncapped. Plus granular `resource.action` RBAC enforced for registers (`pos.open`/`pos.close`/`pos.read`) and inventory (`inventory.adjust`). Manager PIN: pending — **refunds don't exist yet**; PIN gating lands with the refunds feature.
- [x] B2 End-of-day close: register **cash sessions** (one open per terminal+device): `GET/POST /v1/registers/current|open`, `POST /v1/registers/:id/close` (counted-cash reconciliation), `GET /v1/registers`, `GET /v1/registers/:id` (Z-report). Sales attach via optional `session_id`. Migration `011_registers` (`register_sessions`). Naming stays `register_sessions` — granular `pos.open`/`pos.close`/`pos.read` RBAC is now enforced on top (see B1 and decision log).
- [x] B3 Multi-device sync: `GET /v1/sync/pull?cursor=&limit=` streams tenant changes ordered by the monotonic `change_seq` (migration `014_sync_pull` adds `change_seq` to `register_sessions`, `inventory_adjustments`, `tenant_settings`). Entities: categories, products, customers, sales, customer_loyalty_log, register_sessions, inventory_adjustments, tenant_settings — soft-deleted catalog rows decode as `change_type: delete`. Bounded pages (`limit` ≤ 500), `has_more`/next-`cursor` for incremental replay, and `cursor_expired` (409) when a cursor predates the retained window. Users/devices/sync_commands excluded (sensitive). `POST /v1/sync/push` (migration `015_sync_push_rls` FORCE-RLSs `sync_commands`) applies replay-safe commands in one tx: each `command_id`+payload pair is recorded then applied, with retries replaying the stored outcome (`replayed`), reused `command_id` + different payload → `conflict`/`command_conflict`, duplicate (device,operation,payload) deduped, and per-command `applied|replayed|conflict|rejected` + `error_code`. `sale.create` is the supported operation (shared `CreateSale` from `internal/transport/sales/sale.go`); the offline `create_sale` queue pushes through `/v1/sync/push`. Remaining: more write operations + UI conflict surfacing.

## Phase C — Vertical depth (typed tenants come alive)

- [ ] C1 Restaurant: floor plans + table management + kitchen display (KDS) + split bills + tips. *Skipped — large scope, no restaurant client yet.*
- [ ] C2 Pharmacy: expiry-date + lot/serial tracking. *Skipped — needs schema + migrations.*
- [ ] C3 Fashion/textile: size/color variants; textile cutting orders (Odoo best-seller). *Skipped — niche, revisit with a pilot client.*
- [x] C4 Loyalty & promotions: customers CRUD (`GET/POST/PATCH /v1/customers`, search) + **points on sales** — `POST /v1/sales` accepts optional `customer_id`, accrues `(total_minor/100) × rate` points from tenant setting `loyalty.points_per_100` (default 1), updates `customers.loyalty_points` (+total), writes a `customer_loyalty_log` ledger row, and echoes `customer_id` + `loyalty_points_earned`. Migration `013_loyalty`; `customers.read` (all tenant roles) / `customers.write` (manager+). Buy-X-get-Y + coupons remain backlog.

## Phase D — Analytics & SaaS

- [x] D1 Dashboard/analytics: `GET /v1/dashboard/summary` — today's revenue, top products, recent sales; extended with per-cashier + payment-method breakdown. Flutter `DashboardScreen` wired into the POS app bar.
- [ ] D2 SaaS admin polish: per-tenant analytics drill-down, plan limits, billing hooks.

## Phase E — Engineering hardening (tech debt)

- [x] E1 Rate limiting: move in-memory → Redis-backed (multi-instance). Already in AGENTS.
- [x] E2 Structured logging: tenant-scoped JSON logs.
- [ ] E3 Integration tests: Docker Compose env w/ real Postgres + Redis.
- [ ] E4 Flutter `flutter drive` / integration tests: login → sale → history.
- [ ] E5 OpenAPI spec generation from Gin routes for frontend codegen.

## Definition of done

Same as docs/04_TASKS.md: gofmt-clean, vet-clean, tested, tenant-isolated, typed errors,
context-respecting, no secret logging, docs updated.

## Decision log

| Date | Item | Decision |
|------|------|----------|
| 2026-09-10 | A2 stock badges | Done & verified on Android device (qty badges on every POS card via existing `stock_quantity`). |
| 2026-09-10 | A3 offline sale queue | Done & verified on Android device: checkout offline queues `create_sale`, replays via client idempotency key on reconnect (`POST /v1/sales` 201). |
| 2026-09-10 | Login form persistence | Added "Remember logins" (secure storage) — prefills the form and auto-restores session on relaunch; verified on device. |
| 2026-09-10 | A4 split payments | Done & verified: migration `008_payments` (`sale_payments` + `sales.payment_method`), `POST /v1/sales` accepts `payments[]` summing to the total (empty → legacy cash). Device demo: card 5.00 + cash 1.00 on a 6.00 sale, history shows method chip; unit tests (9 Go, 17 Dart). |
| 2026-09-10 | A5/A6 inventory register | Next candidates after A4 — inventory adjustments table + register cash-out. |
| 2026-09-10 | A1 receipts | Skipped now — hardware (thermal printer) dependent; revisit after A2/A3 proven on-device. |
| 2026-09-10 | B1 PIN security | Backlog — must land before refunds/discounts role-outs. |
| 2026-09-10 | C1–C3 verticals | Deferred — aligned with "typed tenants" but no committed industry client yet. |
| 2026-09-11 | Login field order | Fixed "Next skips Email" in login (Root cause: IME "Next" falls back to reading-order `nextFocus()` which misbehaves in Arabic RTL). All four fields now carry explicit `focusNode` + `textInputAction` (next/done) + `onFieldSubmitted` chain; covered by widget tests (94 Dart tests total). |
| 2026-09-11 | B2 cash sessions | Done: one open session per tenant+device (partial unique index), opening cash → sales accumulate live summary → close with counted cash computes expected (opening + cash tender) and difference (over/short). Migrations `011_registers`; Go unit tests incl. procurer; curl E2E verified (open→sale w/ `session_id`→close→Z-report→409s); Flutter `register_sessions` API + POS session bar + Z-report screen (94 Dart tests). Scope confirmed with product owner: **per terminal**, expected-cash **auto** + counted-cash **optional/manual**. |
| 2026-09-11 | B2 naming vs SaaS-kit constitution | `Backend/saas/01_saas_kit_constitution.md` expects `cash_sessions` + granular `pos.open`/`pos.close` permission gating. We ship `register_sessions`; `resource.action` RBAC is now enforced (B1) with the `pos.*` names the constitution uses. |
| 2026-09-11 | A5 inventory adjustments | Done: migration `012_inventory_adjustments` (reasons damaged/restock/count, `quantity_delta <> 0`, optional note), `POST/GET /v1/inventory/adjustments`, RLS + manager-only RBAC. E2E curl verified (restock +5, invalid reason 400, list meta). |
| 2026-09-11 | D1 dashboard | Done: `GET /v1/dashboard/summary` (today revenue/sales/avg/items, top products, recent sales, per-cashier, payment mix — one tx, RLS-safe across tenant tables) + Flutter `DashboardScreen` (stat cards, payment mix, top products, per-cashier, recent sales, pull-to-refresh) from the POS app bar. |
| 2026-09-11 | B1 discount limits + RBAC | Done (server-side only): `sales.discount_minor` honored on `POST /v1/sales` (total = subtotal − discount; payments must sum to the discounted total). Discounts require `pos.discount`; cashier capped at `CASHIER_DISCOUNT_PCT` (default 5% of subtotal), managers/owners uncapped. New pure unit-tested helpers (`validateDiscount`, `HasPermission`). Registered routes for registers + inventory are now permission-gated. Manager **PIN** deferred until refunds exist. |
| 2026-09-11 | Password hashing vs SaaS-kit constitution | `Backend/saas/01_saas_kit_constitution.md` mandates Argon2id; the repo ships bcrypt (`BCRYPT_COST 12`, `internal/config`). **Deviation not previously recorded.** Decision: keep bcrypt for backend compatibility + cost-tuning config; Argon2id swap is a security-hardening backlog item (affects stored hashes) — note in E2/E3. |
| 2026-09-11 | Local-AI doc (`Backend/local_ai_pos_ocr_rag_agent_architecture.md`) | Reference-only design doc, not a committed workstream. Foundation we already match: RLS tenant isolation, `/v1/products/barcode/:barcode`, sale idempotency, integer `*_minor`. Not implemented: `unit`/UoM on products, inventory ledger, purchases, OCR/S3/pgvector/Ollama, agents, audit service. Doc's `/api/v1/...` prefix is illustrative — our convention is `/v1/...`. |
| 2026-09-12 | B3 sync push | Done (push side): migration `015_sync_push_rls` FORCE-RLSs `sync_commands` (the last tenant table without RLS). `POST /v1/sync/push` applies replay-safe commands in one tx — the command row is recorded first (pending) then applied; a unique violation is caught inside a SAVEPOINT so the tx survives Postgres's abort-on-violation. Same `command_id`+payload → stored outcome replayed (`replayed: true`, nothing re-applied); same `command_id` + different payload → `status: conflict`/`command_conflict`; duplicate (device,operation,payload) under a new id → dedupe replay. `sale.create` (shared `businesssales.CreateSale` extracted into `internal/transport/sales/sale.go`) is the only operation; failures map by HTTP status (409 business conflicts → `conflict`, other 4xx/5xx → `rejected`); idempotency key defaults to `sync:<command_id>` when the payload omits one. Curl E2E verified: applied → replay → conflict-on-reused-id → dedupe → unknown op rejected → 400 over-limit → insufficient_stock conflict, stock unchanged on replay. Flutter: `ApiClient.syncPush` + `SyncPushCommand`/`SyncPushResult`/`SyncPushPage`; the offline `create_sale` queue now replays through push (payload carries client `idempotency_key`). Also unblocked pre-existing `TestRLSIsolatesTenants` (DB-010) by seeding users with `set_config('app.current_tenant', …)` in `testutil.SeedTenant`, and fixed stale catalog soft-delete expectation; full suite runs 238 tests (249 pass incl. live DB tests, 6 validation-order skips in sales). |
| 2026-09-11 | B3 sync pull | Done (pull side): migration `014_sync_pull` adds `change_seq` + `bump_change_seq` triggers to `register_sessions`, `inventory_adjustments`, `tenant_settings` (append-only column for adjustments). `GET /v1/sync/pull?cursor=&limit=` returns a single `UNION ALL` stream ordered by `change_seq` with bounded pages, `has_more`, next-`cursor`, and `cursor_expired` (409) when the cursor predates the retained window; soft-deleted catalog rows decode as `change_type: delete`. Curl E2E verified (full dump → incremental pull → delete decode → cursor_expired). |
| 2026-09-11 | C4 loyalty | Done (first slice): customers CRUD + points-on-sales. Migration `013_loyalty` seeds `loyalty.points_per_100 = '1'` per tenant via an RLS-safe `DO` loop (`set_config('app.current_tenant', ...)` per tenant — a plain `INSERT ... SELECT` violates the FORCE RLS policy on `tenant_settings`; also needs `-- +goose StatementBegin/End` around the `DO $$` block). New permission actions `customers.read` (owner/manager/cashier) + `customers.write` (manager/owner). `POST /v1/sales` validates `customer_id` in-tx (404 `customer_not_found`), accrues points, updates counters + ledger atomically with the sale, echoes `customer_id`/`loyalty_points_earned` on create + detail. Idempotent replay fetch now includes `customer_id` + `loyalty_points_earned` too (no double-credit on replay). E2E curl verified: create→list→search→PATCH, cashier create 403, sale 10 cappuccino w/ customer → 35 pts (+ ledger entry), detail shows entry, replay no double-credit, unknown customer 404. Flutter: `Customer`/`CustomersPage` models, `ApiClient.listCustomers/createCustomer`, `CustomersScreen` (search + add sheet + points) from the POS app bar. |
| 2026-09-11 | Loyalty rate storage | Points rate is an Odoo-ish `tenant_settings` key (`loyalty.points_per_100`, `BIGINT` parse, missing/invalid → default 1) rather than a per-plan config — keeps multi-tenant flexibility and matches the existing settings mechanism. Buy-X-get-Y and coupons deferred (need promotion rules schema). |
| 2026-09-12 | E1 Redis rate limiting | Done: `ratelimit.Limiter` (interface) with `RedisLimiter` (Lua-free INCR+EXPIRE, bounded attempts/window, fail-open on Redis error) + `MemoryLimiter` fallback; `LoginRateLimitWith` middleware enforces per-IP on `/auth/login`; config `login_rate_max`/`login_rate_window`. Committed `ef2a24f`. |
| 2026-09-12 | E2 structured logging | Done: tenant-scoped JSON `slog` handler (`RequestLogger` middleware adds `tenant_id`/`user_id`/`request_id`/route/method/status/duration; `Recovery(logger)` logs panics w/ stack structured). Committed `ef2a24f`. |
| 2026-09-12 | OPS-001/002 Prometheus metrics | Done: `internal/infrastructure/metrics` — `pos_api_http_requests_total` (CounterVec by method/route/status_class), `pos_api_http_request_duration_seconds` (HistogramVec by method/route), `pos_api_http_requests_inflight` (Gauge). `Registry` keeps vectors unregistered until `Register(reg, gather)` — main wires default registerer/gatherer, tests use a private `prometheus.NewRegistry()` to avoid global duplicate-registration panics; `Handler()` serves `promhttp` exposition from that same gatherer so `/metrics` always reflects the middleware's vectors. `statusClass` buckets codes by hundreds digit (`2xx`/`4xx`/`5xx`) for stable cardinality. 3 unit tests; wired in `cmd/api/main.go` (`/metrics` route + middleware). Remaining OPS-003..007 (Dockerfile/systemd/Caddy/backup/SecChecklist). |