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
- [ ] A5 Inventory adjustments: `POST /v1/inventory/adjustments` with reason codes (damaged, restock, count). Already in AGENTS next phase. *Skipped this session; backlog.*

## Phase B — Operations & security

- [ ] B1 Cashier security: manager PIN for refunds/discounts, discount limits per role enforced **server-side** (mirrors Odoo's "backend validation prevents frontend bypass" lesson).
- [x] B2 End-of-day close: register **cash sessions** (one open per terminal+device): `GET/POST /v1/registers/current|open`, `POST /v1/registers/:id/close` (counted-cash reconciliation), `GET /v1/registers`, `GET /v1/registers/:id` (Z-report). Sales attach via optional `session_id`. Migration `011_registers` (`register_sessions`). Note: `Backend/saas/01_saas_kit_constitution.md` names this capability `cash_sessions` with `pos.open`/`pos.close` permissions under CASHIER — our role check already lets cashiers open/close, but granular `resource.action` permissions are not yet enforced (see decision log).
- [ ] B3 Multi-device sync: full offline→online replay with conflict resolution (idempotency already in place). A3 is the first slice; scale to full change-sequence pull.

## Phase C — Vertical depth (typed tenants come alive)

- [ ] C1 Restaurant: floor plans + table management + kitchen display (KDS) + split bills + tips. *Skipped — large scope, no restaurant client yet.*
- [ ] C2 Pharmacy: expiry-date + lot/serial tracking. *Skipped — needs schema + migrations.*
- [ ] C3 Fashion/textile: size/color variants; textile cutting orders (Odoo best-seller). *Skipped — niche, revisit with a pilot client.*
- [ ] C4 Loyalty & promotions: points, buy-X-get-Y, coupons. Needs `customers` table (schema stubbed in DB-005). *Skipped this session; backlog.*

## Phase D — Analytics & SaaS

- [ ] D1 Dashboard/analytics: `GET /v1/dashboard/summary` — today's revenue, top products, recent sales; extend with per-cashier + payment-method breakdown. Already in AGENTS next phase.
- [ ] D2 SaaS admin polish: per-tenant analytics drill-down, plan limits, billing hooks.

## Phase E — Engineering hardening (tech debt)

- [ ] E1 Rate limiting: move in-memory → Redis-backed (multi-instance). Already in AGENTS.
- [ ] E2 Structured logging: tenant-scoped JSON logs.
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
| 2026-09-11 | B2 naming vs SaaS-kit constitution | `Backend/saas/01_saas_kit_constitution.md` expects `cash_sessions` + granular `pos.open`/`pos.close` permission gating. We ship `register_sessions` with role-check gating (any tenant user). Renaming + `resource.action` RBAC middleware are backlog (D2/plan-engine phase); Flutter must keep reading the shape we return. |