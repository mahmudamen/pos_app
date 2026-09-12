# AGENTS.md

Workspace for a POS (point-of-sale) SaaS: a Go/Gin backend plus a Flutter client. Most of the tree is **cloned reference code**; only two directories are real work.

## Layout: real code vs. reference clones

Do not edit anything outside these three paths:

- `Backend/go-pos-backend-implementation-ready/` — the Go backend (module `github.com/example/pos-api`, Go 1.23). This is the active backend.
- `Flutter/pos_go_app/` — the Flutter client for that backend. Intentionally has **no Firebase**; uses plain `http`, `sqflite`, `flutter_secure_storage`.
- `Backend/backend_spec_go.md`, `Flutter/flutter_pos_spec.md`, `Flutter/skills_tasks_spec.md` — design specs.

Everything else under `Flutter/` (e.g. `flutter-pos-system`, `pos_go_app`'s siblings `bloc`, `fl_chart`, `flutterfire`, `postgresql-dart`, `POS-APP`, …) are unmodified third-party clones used as reference. `Backend/go-pos-backend-implementation-ready (2)/` is a stale older copy — never edit it. `Frontend/` and `desktopapplication/` are empty placeholders. No directory here is a git repo, so git-based workflows (e.g. `make bump` in `flutter-pos-system`) will fail.

The Flutter spec (`flutter_pos_spec.md`) describes a target (Riverpod/Drift/Dio/Freezed) that the current `pos_go_app` deliberately does not follow; trust the code over the spec.

## Backend (`Backend/go-pos-backend-implementation-ready/`)

Bootstrap (requires `go` and Docker Compose v2):

```bash
cp .env.example .env
./scripts/setup_dev.sh          # go mod download + starts postgres & redis in Docker
set -a; source .env; set +a     # REQUIRED
make migrate
make run                        # serves :8080, /v1/* API
```

Gotchas:

- The Go app and Makefile **do not auto-load `.env`** (no godotenv; `config.Load()` reads real env vars). Export `.env` yourself or `make migrate`/`make run` will see no `DATABASE_URL`.
- No Docker? Run `scripts/bootstrap_local.sql` once as a Postgres administrator, then `make migrate`/`make run` as usual. Redis is optional for basic login; if absent, `/health/ready` reports not ready but `/health/live` stays OK.
- Health probes: `curl 127.0.0.1:8080/health/live` and `/health/ready`.
- `make test` (`go test ./...`) needs **no database** — handler tests assert "unavailable" with a nil pool. CI runs gofmt-check → `go vet` → `go test` → `go test -race`; `make check` = fmt + vet + test, `make lint` = golangci-lint, `make security` = gosec.
- Migrations are goose files in `internal/infrastructure/database/migrations/` (single-file `.sql`, numbered `001_extensions`, `003_core`, `004_permissions`, `005_sync`, `006_tenant_type`, `007_localization`, `008_payments`, `010_settings`, `011_registers`, `012_inventory_adjustments`, `013_loyalty`, `014_sync_pull`, `015_sync_push_rls`); apply with `make migrate`, roll back `make migrate-down`.
- API: everything under `/v1`; auth (`/auth/login|refresh|logout`), catalog (`/categories`, `/categories/:id`, `/products`, `/products/:id`, `/products/barcode/:barcode`, PATCH `/products/:id`), sales (`GET /v1/sales` list, `GET /v1/sales/:id` detail with items + payments, `POST /v1/sales` with idempotency, stock locking, RLS and split tender), users (`GET /users`, `POST /users`), sync (`GET /v1/sync/pull` change feed, `POST /v1/sync/push` replay-safe commands), meta (`/meta/countries`, `/meta/currencies` — public), platform (`/saas/summary`, `/saas/tenants` — require `saas_admin` role exactly). Money is integer minor units (`price_minor`, `subtotal_minor`, `amount_minor`).
- RLS is FORCE-enabled on all tenant tables (`FORCE ROW LEVEL SECURITY`), so **even table owners see nothing without** `SET app.current_tenant`. The SaaS handler aggregates cross-tenant by looping tenants inside one tx and calling `SELECT set_config('app.current_tenant', $1, true)` per tenant. Platform-level ad-hoc SQL must set tenant context similarly (`SELECT set_config('app.current_tenant', <uuid>, false)` in psql).
- SaaS demo seed: `scripts/seed_demo.sql` (idempotent) creates 5 typed demo tenants (restaurant, book_store, mobile_shop, computer_shop, grocery) + `demo-store` + `saas` platform tenant. All demo logins password `admin`: `admin@demo-<slug>.com` (manager/demo), `guest@demo-<slug>.com` (guest), and SaaS staff `admin@posgo.saas` / `support@posgo.saas` (`saas_admin`). `GET /v1/saas/summary` currently returns `total_users` = 20 (3 users × 6 demo tenants + 2 SaaS staff).
- `docs/*.md` (esp. `05_CODING_STANDARDS.md`, `09_SYNC.md`) are the source of truth for architecture; `speckit.*` and `.specify/` are SpecKit workflow scaffolding, `.github/skills/speckit-*` are GitHub skills.

## Flutter client (`Flutter/pos_go_app/`)

Run tests/lint from this directory:

```bash
flutter analyze                  # lints: avoid_print, prefer_single_quotes
flutter test                     # DB tests use sqflite_common_ffi
```

Run the app with the backend up:

```bash
flutter run --dart-define=API_BASE_URL=http://127.0.0.1:8080
```

`API_BASE_URL` defaults to `http://127.0.0.1:8080` via `String.fromEnvironment` (`lib/core/api_client.dart`). The backend must be running to log in; login goes to `POST /v1/auth/login` with a `tenant_id`/`device_id`/`device_name` payload and expects the `{"data": ...}` envelope.

Gotcha: `flutter test` can crash with a `RangeError ... 0..97` from `test_core`'s compact reporter (see `flutter_01.log`) — an upstream terminal-width bug, not a test failure; work around it with a wider terminal or `--reporter expanded`.

## Completed work

### Backend — features
- **Sales endpoints**: `GET /v1/sales` (paginated), `GET /v1/sales/:id` (detail with items + payments), `POST /v1/sales` (with idempotency, stock locking, RLS)
- **Split payments**: migration `008_payments` adds `sale_payments` (method cash/card/mobile + `amount_minor`, RLS) and `sales.payment_method` (primary = largest line). `POST /v1/sales` accepts `payments: [{method, amount_minor}]`; lines must sum exactly to the total, an empty array defaults to cash covering the full sale (legacy clients keep working), and responses expose `payment_method`. Validation lives in pure, unit-tested helpers (`normalizePayments`, `primaryPaymentMethod` in `internal/transport/sales/payments.go`).
- **Product pagination**: `?page=&limit=` on `GET /v1/products` with total count in meta
- **Categories**: full CRUD — `GET`, `POST`, `PATCH`, `DELETE /v1/categories/:id` (soft-delete via `is_active`)
- **Product soft-delete**: `DELETE /v1/products/:id` (sets `is_active = false`)
- **User management**: `GET /v1/users` (paginated), `POST /v1/users` (role cashier/manager + `account_type` standard/demo/guest, bcrypt password)
- **Register cash sessions**: migration `011_registers` — `register_sessions` (status open/closed, `opening_cash_minor`, `closing_cash_minor`, `expected_cash_minor`, `cash_difference_minor`, opened/closed_at, device, user; partial unique index = one open session per tenant+device; RLS) + `sales.register_session_id` FK/index. Routes: `GET /v1/registers/current` (404 `no_open_session`), `POST /v1/registers/open`, `POST /v1/registers/:id/close` (optional `closing_cash_minor`; expected = opening + cash tender, difference = counted − expected), `GET /v1/registers` (paginated lean list), `GET /v1/registers/:id` (full + live `summary`). `POST /v1/sales` accepts optional `session_id` (must be open, same tenant+device). Module `internal/transport/registers/`.
- **Multi-tenant SaaS demo**: migration `006_tenant_type` (`tenants.business_type`), migration `007_localization` (tenants `country_code`/`currency_code`/`default_language`, reference `countries`/`currencies` tables — Egypt `EG`/EGP/ar only, demo users `account_type`, role check now allows `saas_admin`); login returns tenant `business_type`/`country_code`/`currency_code`/`default_language` + user `account_type`
- **Customers + loyalty (C4)**: migration `013_loyalty` adds `customers.loyalty_points`/`loyalty_points_total` + `customer_loyalty_log` ledger (RLS FORCE, rows carry `reason`/`change_seq`); seeds `loyalty.points_per_100 = '1'` per tenant (RLS-safe `DO` loop). Routes `GET /v1/customers` (page/limit + `q` search on name/email/phone), `GET /v1/customers/:id` (customer + last 20 ledger entries), `POST /v1/customers`, `PATCH /v1/customers/:id` — `customers.read` (all tenant roles), `customers.write` (manager/owner). `POST /v1/sales` accepts optional `customer_id` (404 `customer_not_found`), accrues `(total_minor/100) × rate` points atomically with the sale (ledger row + counter update), echoes `customer_id` + `loyalty_points_earned`. Module `internal/transport/customers/`.
- **Meta + SaaS endpoints**: public `GET /v1/meta/countries`, `GET /v1/meta/currencies`; `GET /v1/saas/summary` and `GET /v1/saas/tenants` behind `saas_admin` role (cross-tenant aggregates loop tenants inside one tx setting `app.current_tenant` per tenant — RLS-safe)
- **SaaS admin polish (D2)**: `GET /v1/saas/tenants/:id/analytics` — per-tenant drill-down (plan/resource facts, user+product counts, today's stats, 7-day revenue trend, top products, recent sales) via one tx with per-tenant RLS context, 404 `tenant_not_found`/400 `invalid_tenant_id`. **Plan limits**: migration `016_tenant_plan` adds `tenants.plan` (default `standard`) + `max_users`/`max_products` (0 = unlimited); enforced 409 `plan_limit_exceeded` on `POST /v1/users` and `POST /v1/products` (checked inside the tenant tx so the count is RLS-scoped); tenant list exposes plan fields. Billing/webhook hooks remain backlog.
- **CORS middleware**: config-driven allowlist (`CORS_ALLOWED_ORIGINS`, dev default `*`; `docker-compose.prod.yml` requires it), OPTIONS preflight, wired in `internal/transport/server/router.go`
- **SecurityHeaders middleware**: HSTS, X-Frame-Options DENY, nosniff, no-store, XSS protection
- **LoginRateLimit middleware**: in-memory, configurable attempts/window per IP, wired to `/auth/login`
- **JWT**: `TokenManager` with typed access/refresh tokens, `IssueWithRole`, short-secret rejection
- **Inventory adjustments (A5)**: migration `012_inventory_adjustments` (reasons `damaged|restock|count`, `quantity_delta <> 0`, optional note ≤255, RLS FORCE); `POST/GET /v1/inventory/adjustments`; `created_by` echoes actor display name; stock cannot go negative
- **Dashboard/analytics (D1)**: `GET /v1/dashboard/summary` — `today` (revenue/sales/avg/items), `top_products`, `recent_sales`, `per_cashier`, `payment_mix`; one tx with per-tenant RLS context
- **Granular RBAC (B1)**: static `resource.action` matrix in `internal/transport/http/permissions.go` (`HasPermission`) — `pos.read|open|close|sale|discount`, `inventory.adjust` (manager/owner/saas_admin), `dashboard.read`, `saas.admin`; enforced on registers (read/open/close → 403 `permission_denied` otherwise) and inventory create
- **Server-enforced discount limits (B1)**: `POST /v1/sales` accepts optional `discount_minor`; total = subtotal − discount and payments must sum to the discounted total; non-zero discounts require `pos.discount`; cashiers capped at `CASHIER_DISCOUNT_PCT` (config, default 5% of subtotal), managers/owners uncapped; pure helpers `validateDiscount` (`internal/transport/sales/discounts.go`)
- **Prometheus HTTP metrics (OPS-001/002)**: `internal/infrastructure/metrics/` — `Registry` owns `pos_api_http_requests_total` (CounterVec by method/route/status_class), `pos_api_http_request_duration_seconds` (HistogramVec by method/route), `pos_api_http_requests_inflight` (Gauge). `NewScoped()` returns fresh unregistered vectors; `Register(reg, gather)` wires them onto a caller-chosen registerer+gatherer (main: DefaultRegisterer; tests: private `prometheus.NewRegistry()`). `Middleware()` records count/latency/inflight; `Handler()` serves `promhttp` exposition from the captured gatherer. `statusClass` buckets codes into hundreds-digit labels (`2xx`/`4xx`/`5xx`) for stable cardinality. Registered in `cmd/api/main.go` alongside the `/metrics` route.
- **Shared route table (E5)**: `internal/transport/server/router.go` (`server.Register(engine, Deps{...})`) is the single place all middleware, `/health/*`, `/metrics`, and `/v1/*` routes are wired. `cmd/api/main.go` and the OpenAPI generator both use it, so the spec cannot drift from the served routes.
- **OpenAPI generation (E5)**: `cmd/openapi` builds the engine via `server.Register` with a nil pool + bare config (runs offline, no DB), walks the live Gin route table, and emits `docs/openapi.json` (29 paths; `{id}` params; bearerAuth security except `/v1/auth/login` + `/v1/meta/*` + `/health/*`; shared data/error `Envelope`). `make openapi` regenerates it.
- **Integration test env (E3)**: `deployments/docker/docker-compose.integration.yml` (ephemeral Postgres on 127.0.0.1:15432 + Redis on :16379) + `scripts/integration-test.sh` (boots with `--wait`, exports `TEST_DATABASE_URL`, runs `go test ./...` or `RACE=1 go test -race`, tears down) + `make integration-test`. DB-backed suites (database, catalog, auth, sales, sync) skip without `TEST_DATABASE_URL`/`DATABASE_URL`, so `make test` stays green offline.
- **DB-backed integration suites (AUTH-011/SYNC-007/SALE-008)**: `internal/testutil.Seed` now also exposes `ManagerEmail`/`CashierEmail` (emails are generated deterministically). New suites: `auth/handler_integration_test.go` (login 200 + user/tenant echo, wrong-password 401, refresh rotation + AUTH-007 reuse-revocation of the whole family, logout→session revoked), `sync/handler_integration_test.go` (push apply → stock -2 → replay (same command_id+payload, stock unchanged) → `command_conflict` on changed payload → dedupe on new command_id → `unknown_command` reject), `sales/handler_integration_test.go` (idempotent duplicate returns the same sale id with stock unchanged + 5-way concurrent same-key POST → exactly 1 sale row, 201 or 409 `idempotency_conflict`, stock decremented once), `saas/handler_integration_test.go` (analytics drill-down 200/404/400 + plan limits → 409 `plan_limit_exceeded` on users and products).
- **Backup/restore runbook (OPS-006)**: `scripts/backup.sh` (pg_dump | gzip, `--no-owner/--no-privileges`, retention `BACKUP_KEEP_DAYS`=14) + `scripts/restore.sh` (gunzip → psql `ON_ERROR_STOP`); runbook in `docs/11_OPERATIONS.md` (cron schedule, mandatory restore verification, app-only vs schema rollback).
- **Production security checklist (OPS-007)**: `docs/14_SECURITY_CHECKLIST.md` — go/no-go gates for secrets (JWT ≥32B, least-privilege DB role), TLS/Caddy + HSTS, request hardening (CORS allowlist, max body, rate limit, RLS FORCE), operations (verified backups, `/health`, metrics-collector-only `/metrics`), tenant data, deploy hygiene (forward-only migrations, pinned image, non-root `pos-api` user).

### Backend — tests (270 test functions; 248 pass + 22 skip offline, `go vet` clean)

| Package | Tests | Coverage |
|---------|-------|----------|
| `config` | 22 | env parsing, defaults, durations, overrides, `CASHIER_DISCOUNT_PCT` bounds/parse, `CORS_ALLOWED_ORIGINS` parse + wildcard default |
| `errors` | 7 | New/Wrap, error string, Unwrap, codes |
| `database` | 2 | nil pool, close without connect, `TestRLSIsolatesTenants` runs against a real Postgres when `DATABASE_URL` set (DB-010: proves cross-tenant reads are filtered/denied). Needs `.env` exported (skips otherwise) |
| `security` | 11 | issue/parse, expired, wrong issuer, short secret, role, passwords |
| `ratelimit` | 6 | memory + Redis limiter, window/attempts semantics |
| `auth` handler | 17 | 13 unit (login/refresh/logout happy paths, invalid JSON, missing fields, bad email, unavailable, route registration) + 4 DB-backed integration (AUTH-011: login happy path/wrong-password 401, refresh rotation + reuse revocation, logout session revocation) |
| `catalog` handler | 49 | CRUD, pagination, search, barcode, validation, PATCH, category PATCH/DELETE, product soft-delete, errors, 5 integration DB tests (run when `DATABASE_URL` set; soft-delete keeps `is_active=false` viewable — the sync-pull `delete` decode) |
| `customers` handler | 14 | routes, 401s, 403s (cashier write, guest read), 503 without DB, validation 400s (missing/blank name, bad email, long fields, bad uuid), invalid-body-vs-pool ordering |
| `dashboard` handler | 3 | route registration, auth required, unavailable without DB (summary math covered E2E) |
| `http` middleware | 25 | CORS (preflight, normal, allowlist echo, disallowed origin omitted, no-origin allowlist), SecurityHeaders, RateLimit, RequestID, Recovery, MaxBodySize, Claims, `HasPermission` RBAC matrix |
| `inventory` handler | 11 | route registration, auth required, unavailable without DB, invalid reason/zero-delta/bad-uuid/long-note 400, `inventory.adjust` 403 for cashier, `validReason` |
| `metrics` | 3 | exposition serves core family names + route/status_class/method labels, statusClass hundreds-digit bucketing, in-flight gauge returns to zero after a request |
| `registers` handler | 14 | current/open/close/list/detail happy paths + validation (starting cash, counted cash, balance on private balance board, conflict/404s), summary aggregation math, `pos.*` RBAC → 403 |
| `sales` handler | 31 | 29 unit (list/get/create, auth required, unavailable, pagination, validation, writeSale/writeError, normalizePayments/primaryPaymentMethod (split tender) + `validateDiscount` role caps + `saleCustomerID`, `loyaltyRateFromValue`, `pointsForTotal`; 6 skips: validation-order — 503-before-400 asserts that skip when the pool is nil) + 2 DB-backed integration (SALE-008: idempotent duplicate + 5-way concurrent → exactly 1 sale row, stock once) |
| `saas` handler | 7 | routes, auth required, 403 non-`saas_admin`, 503 without DB + 3 DB-backed integration (analytics drill-down 200/404/400, plan limit 409 on users, plan limit 409 on products) |
| `settings` handler | 14 | get/update, defaults, validation, auth required, unavailable |
| `sync` handler | 14 | 13 unit (pull: route registration, auth, unavailable, cursor validation, default 0 reaches pool check; push: route registration, auth, invalid body 400, unavailable, invalid command_id rejected, unknown operation, payload-hash determinism, status mapping) + 1 DB-backed integration (SYNC-007: push apply → replay → conflict → dedupe against real Postgres) |
| `users` handler | 20 | list/create, auth required, unavailable, pagination, validation (email, password, role, display_name), route registration |

### Flutter — features
- **Arabic-first l10n**: `AppStrings` (`lib/l10n/strings.dart`, no ARB) with `flutter_localizations` delegates; Arabic default, switchable from the login screen and POS settings sheet; language persisted via `SessionStore` (`app_language` key)
- **Egypt/EGP demo config**: login screen fetches `/v1/meta/countries`/`/v1/meta/currencies` (single-entry Egypt/EGP dropdowns); `Session` holds tenant `business_type`/`country_code`/`currency_code`/`default_language` + user `role`/`account_type`; money rendered with EGP symbol (`E£12.50`, RTL-aware)
- **SaaS control panel**: `ControlPanelScreen` (`lib/features/saas/`) for `saas_admin` users — `SaasSummary` stat cards, per-business-type breakdown, tenant list with user/product counts
- **Offline caching**: `cacheProducts()`/`cachedProducts()` in `LocalDatabase`, POS screen falls back to cache on API error
- **Categories**: `Category` model, `categories()` API method, filter chips in POS screen
- **Sale history**: `SaleSummary`/`SalesPage` models, `listSales()` API, `SaleHistoryScreen` with pagination
- **Device ID persistence**: `Session.deviceId`, `readDeviceId()`/`saveDeviceId()`, generated once in LoginScreen, reused across restarts
- **Typed errors**: `app_exception.dart` with `NetworkException`, `AuthException`, `ServerException`, `ValidationException`, `StockException` + `mapApiError()`
- **Remembered login**: `RememberedLogin` + `saveRememberedLogin`/`readRememberedLogin`/`clearRememberedLogin` in `SessionStore` (secure storage); "Remember logins" checkbox prefills tenant/email/password/device on the login screen
- **Stock badges**: per-card `stock_quantity` badge on the POS grid (red "Out of stock" when 0), l10n `outOfStock`
- **Offline sale queue**: checkout that fails with a network error is enqueued as a `create_sale` `PendingCommand` (client idempotency key) and replayed synchronously via the sync push endpoint on the next successful catalog load (batched ≤ 10, dropped once applied/replayed)
- **Split payments**: checkout opens a `PaymentSheet` bottom sheet (`lib/features/pos/pos_screen.dart`) — cash/card/mobile tender lines with the cash remainder auto-filled after card/mobile portions; wire model in `lib/core/payments.dart` (`PaymentMethod`, `PaymentInput`, `PaymentSplit.allocate`, `minorFromInput`); sale history shows a method chip
- **POS cash sessions (registers)**: one open session per tenant+device — `_SessionBar` in `pos_screen.dart` (open → starting-cash dialog with default 0, resume on relaunch, finish with optional counted cash, history). Report + history screens in `lib/features/pos/session_screen.dart` (`SessionReportScreen` Z-report with expected vs counted difference, `SessionHistoryScreen`); wire models in `lib/core/registers.dart` (`SessionSummary`, `RegisterSession`, `SessionListItem`, `SessionsPage`); offline `create_sale` queue now carries `session_id`
- **Login focus chain**: explicit `focusNode` + `textInputAction` (`next`/`done`) + `onFieldSubmitted` on all four login fields — fixes IME "Next" skipping Email under Arabic RTL (reading-order `nextFocus()` misbehavior); widget-tested
- **Dashboard screen (D1)**: `DashboardScreen` (`lib/features/dashboard/`) reachable via an insights `IconButton` in the POS app bar — today revenue/avg-sale/items/sales stat cards, payment mix, top products, per-cashier breakdown, recent sales, pull-to-refresh; wire models in `lib/core/dashboard.dart`, `ApiClient.dashboardSummary()`
- **Customers screen (C4)**: `CustomersScreen` (`lib/features/customers/`) reachable via a groups `IconButton` in the POS app bar — list with search, add-customer bottom sheet, loyalty points display. Models in `lib/core/customers.dart`; `ApiClient.listCustomers`/`createCustomer`. Strings: `customers`, `addCustomer`, `loyaltyPoints`.
- **Multi-device sync (B3)**: migration `014_sync_pull` adds `change_seq` + `bump_change_seq` triggers (register_sessions, inventory_adjustments, tenant_settings). `GET /v1/sync/pull?cursor=&limit=` streams tenant changes (categories, products, customers, sales, customer_loyalty_log, register_sessions, inventory_adjustments, tenant_settings) ordered by monotonic `change_seq`, bounded pages with `has_more` + next-`cursor`, soft-deleted catalog rows as `change_type: delete`, and `cursor_expired` (409) for cursors older than the retained window. `POST /v1/sync/push` (migration `015_sync_push_rls` FORCE-RLSs `sync_commands`) applies replay-safe commands in one tx: a `command_id`+payload pair is recorded then applied; retries with the same `command_id`+payload replay the stored outcome (`replayed: true`), a reused `command_id` with a different payload gives `status: conflict`/`command_conflict`, duplicate (device,operation,payload) dedupes, and per-command results carry `applied|replayed|conflict|rejected` + `error_code` (409 business conflicts like `insufficient_stock` → `conflict`). `sale.create` is the supported operation (shared `businesssales.CreateSale` extracted into `internal/transport/sales/sale.go`); idempotency key defaults to `sync:<command_id>`. Savepoints keep the tx usable after the unique-violation INSERT attempt. Module `internal/transport/sync/`. Flutter: `ApiClient.syncPull`/`syncPush` + `SyncRow`/`SyncPullPage`/`SyncPushCommand`/`SyncPushResult`/`SyncPushPage` models, `SessionStore.readSyncCursor`/`saveSyncCursor` (per-tenant key `sync_cursor_<tenant>`); the offline `create_sale` queue replays via `syncPush` (batched ≤ 10, payload carries client `idempotency_key`, removed on applied/replayed).

### Flutter — tests (106 passing, `flutter analyze` 0 issues)
| File | Tests | Coverage |
|------|-------|----------|
| `api_client_test.dart` | 55 | login, logout, products, categories, sales, refresh, createProduct, updateProduct, session, Product.fromJson, _message edge cases, Country/Currency parsing + meta fetch, SaasSummary/SaasTenant parsing, split-payment request, SaleResult/SaleSummary payment method, register session models current/open/close/sessions/session_id-on-createSale, DashboardSummary parse + fetch + error, Customer parse + listCustomers + createCustomer, syncPull fetch with cursor+limit, syncPull cursor_expired → ApiException, SyncRow parse (delete row), syncPush POST + applied/replayed/conflict mapping, SyncPushResult rejected defaults |
| `auth_test.dart` | 7 | session parse, SaaS/i18n fields, defaults, copyWith, deviceId, RememberedLogin, sync cursor persistence (per-tenant, defaults 0) |
| `local_database_test.dart` | 8 | open/close, cacheProducts/cachedProducts, pending command enqueue/order/dedupe/markComplete, split-payment payload round-trip |
| `app_exception_test.dart` | 9 | mapApiError for all HTTP codes, ApiException display |
| `strings_test.dart` | 5 | supported locales (ar default), Arabic/English labels (incl. remember/stock/offline/payment/session/dashboard/customers/loyalty), EGP money formatting, fallback, `methodLabel` |
| `payments_test.dart` | 17 | PaymentMethod wire mapping, PaymentInput json, PaymentSplit.allocate (auto cash remainder, splits, zero/negative/total guards), minorFromInput decimal parsing |
| `splash_screen_test.dart` | 3 | Arabic/English branding, entrance/glow animations, XAMLtech footer |
| `login_focus_test.dart` | 2 | Next/Done focus chain across the four login fields (Arabic RTL default) |

## Next phase

The authoritative roadmap is `Backend/go-pos-backend-implementation-ready/docs/13_ROADMAP.md`
(driven by an Odoo-POS feature study; includes a decision log for skipped/planned items).
Phase A ("close the checkout gap") is the active workstream. Short status:

### Phase A — checkout gap (active)
- [x] **Stock-on-hand badges** on POS product grid (backend already returns `stock_quantity`)
- [x] **Offline sale queue**: checkout queued in SQLite when API unreachable, replayed idempotently when online
- [ ] **Receipt generation** (A1) — SKIPPED for now: hardware-dependent (thermal printers)
- [x] **Payments**: `payment_method` + split payments on `POST /v1/sales` (migration `008_payments`) — verified on device (split card+cash sale, legacy cash default)
- [x] **Inventory adjustments** (A5): `POST/GET /v1/inventory/adjustments` (reasons damaged/restock/count), manager-only RBAC, curl E2E verified
- [x] **POS cash sessions (registers)**: open per terminal with starting cash → sales attach via `session_id` → finish with counted cash → Z-report (expected vs counted) + session history. One open session per tenant+device; offline `create_sale` replay carries `session_id`. Curl E2E verified; device walkthrough deferred (phone re-locked).

### Backlog (kept from prior plan; see roadmap for details & why items are skipped)
- **Manager PIN for refunds/discounts**: defer until refunds exist (discount **limits** + granular RBAC already shipped server-side — B1)
- **Cashier security / RBAC extension**: refunds (new `pos.refund`), per-tenant permission overrides
- **Multi-device sync**: pull + push shipped (B3 — `GET /v1/sync/pull` via `change_seq`, `POST /v1/sync/push` replay-safe commands); offline→online replay routed through `/v1/sync/push`; remaining: multi-entity write commands (only `sale.create`), UI conflict surfacing
- **Loyalty/v2**: buy-X-get-Y, coupons, promotion rules — needs new schema beyond the points rate already shipped (C4)
- **Restaurant/pharmacy/textile depth**: tables/KDS, expiry/lot, size-color variants (typed tenants)
- **SaaS control plane** (`Backend/saas/*.md`): tenants/subscriptions/plans/backups/restore — reference spec; out of current scope

### Tech debt
- **Structured logging**: tenant-scoped JSON logs (E2 — shipped, see decision log)
- **Password hashing**: bcrypt vs SaaS-kit's Argon2id — deviation recorded in roadmap decision log; Argon2id swap affects stored hashes (hardening backlog)
- **Rate limiting**: in-memory → Redis-backed for multi-instance (E1 — shipped, see decision log)
- **Integration tests**: Docker Compose env with real Postgres + Redis (E3 — shipped: `make integration-test`, DB-backed auth/sync/sales suites)
- **Flutter integration tests**: `flutter drive` for login → sale → history (E4 — blocked: no unlocked device/emulator in this environment)
- **OpenAPI spec**: auto-generate from Gin routes, publish for frontend codegen (E5 — shipped: `make openapi` → `docs/openapi.json`; schemas for write bodies are hand-augmented as the contract grows)