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
- Migrations are goose files in `internal/infrastructure/database/migrations/` (single-file `.sql`, numbered `001_extensions`, `003_core`, `004_permissions`, `005_sync`, `006_tenant_type`, `007_localization`, `008_payments`); apply with `make migrate`, roll back `make migrate-down`.
- API: everything under `/v1`; auth (`/auth/login|refresh|logout`), catalog (`/categories`, `/categories/:id`, `/products`, `/products/:id`, `/products/barcode/:barcode`, PATCH `/products/:id`), sales (`GET /v1/sales` list, `GET /v1/sales/:id` detail with items + payments, `POST /v1/sales` with idempotency, stock locking, RLS and split tender), users (`GET /users`, `POST /users`), meta (`/meta/countries`, `/meta/currencies` — public), platform (`/saas/summary`, `/saas/tenants` — require `saas_admin` role exactly). Money is integer minor units (`price_minor`, `subtotal_minor`, `amount_minor`).
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
- **Meta + SaaS endpoints**: public `GET /v1/meta/countries`, `GET /v1/meta/currencies`; `GET /v1/saas/summary` and `GET /v1/saas/tenants` behind `saas_admin` role (cross-tenant aggregates loop tenants inside one tx setting `app.current_tenant` per tenant — RLS-safe)
- **CORS middleware**: wildcard origin, OPTIONS preflight, wired in `cmd/api/main.go`
- **SecurityHeaders middleware**: HSTS, X-Frame-Options DENY, nosniff, no-store, XSS protection
- **LoginRateLimit middleware**: in-memory, configurable attempts/window per IP, wired to `/auth/login`
- **JWT**: `TokenManager` with typed access/refresh tokens, `IssueWithRole`, short-secret rejection
- **Inventory adjustments (A5)**: migration `012_inventory_adjustments` (reasons `damaged|restock|count`, `quantity_delta <> 0`, optional note ≤255, RLS FORCE); `POST/GET /v1/inventory/adjustments`; `created_by` echoes actor display name; stock cannot go negative
- **Dashboard/analytics (D1)**: `GET /v1/dashboard/summary` — `today` (revenue/sales/avg/items), `top_products`, `recent_sales`, `per_cashier`, `payment_mix`; one tx with per-tenant RLS context
- **Granular RBAC (B1)**: static `resource.action` matrix in `internal/transport/http/permissions.go` (`HasPermission`) — `pos.read|open|close|sale|discount`, `inventory.adjust` (manager/owner/saas_admin), `dashboard.read`, `saas.admin`; enforced on registers (read/open/close → 403 `permission_denied` otherwise) and inventory create
- **Server-enforced discount limits (B1)**: `POST /v1/sales` accepts optional `discount_minor`; total = subtotal − discount and payments must sum to the discounted total; non-zero discounts require `pos.discount`; cashiers capped at `CASHIER_DISCOUNT_PCT` (config, default 5% of subtotal), managers/owners uncapped; pure helpers `validateDiscount` (`internal/transport/sales/discounts.go`)

### Backend — tests (225 passing, `go vet` clean)
| Package | Tests | Coverage |
|---------|-------|----------|
| `config` | 17 | env parsing, defaults, durations, overrides, `CASHIER_DISCOUNT_PCT` bounds/parse |
| `errors` | 7 | New/Wrap, error string, Unwrap, codes |
| `database` | 2 | nil pool, close without connect, tenant-context check (RLS integration SKIPs without `TEST_DATABASE_URL`) |
| `security` | 11 | issue/parse, expired, wrong issuer, short secret, role, passwords |
| `auth` handler | 13 | login/refresh/logout happy paths, invalid JSON, missing fields, bad email, unavailable, route registration |
| `catalog` handler | 49 | CRUD, pagination, search, barcode, validation, PATCH, category PATCH/DELETE, product soft-delete, errors |
| `http` middleware | 21 | CORS, SecurityHeaders, RateLimit, RequestID, Recovery, MaxBodySize, Claims, `HasPermission` RBAC matrix |
| `registers` handler | 20 | current/open/close/list/detail happy paths + validation (starting cash, counted cash, balance on private balance board, conflict/404s), summary aggregation math, `pos.*` RBAC → 403 |
| `sales` handler | 25 | list/get/create, auth required, unavailable, pagination, validation, writeSale/writeError, normalizePayments/primaryPaymentMethod (split tender) + `validateDiscount` role caps (discount >0 gated/required, cashier cap, manager uncapped, negative/exceeds/forbidden) |
| `settings` handler | 14 | get/update, defaults, validation, auth required, unavailable |
| `users` handler | 20 | list/create, auth required, unavailable, pagination, validation (email, password, role, display_name), route registration |
| `dashboard` handler | 3 | route registration, auth required, unavailable without DB (summary math covered E2E) |
| `inventory` handler | 11 | route registration, auth required, unavailable without DB, invalid reason/zero-delta/bad-uuid/long-note 400, `inventory.adjust` 403 for cashier, `validReason` |

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
- **Offline sale queue**: checkout that fails with a network error is enqueued as a `create_sale` `PendingCommand` (client idempotency key) and replayed on the next successful catalog load
- **Split payments**: checkout opens a `PaymentSheet` bottom sheet (`lib/features/pos/pos_screen.dart`) — cash/card/mobile tender lines with the cash remainder auto-filled after card/mobile portions; wire model in `lib/core/payments.dart` (`PaymentMethod`, `PaymentInput`, `PaymentSplit.allocate`, `minorFromInput`); sale history shows a method chip
- **POS cash sessions (registers)**: one open session per tenant+device — `_SessionBar` in `pos_screen.dart` (open → starting-cash dialog with default 0, resume on relaunch, finish with optional counted cash, history). Report + history screens in `lib/features/pos/session_screen.dart` (`SessionReportScreen` Z-report with expected vs counted difference, `SessionHistoryScreen`); wire models in `lib/core/registers.dart` (`SessionSummary`, `RegisterSession`, `SessionListItem`, `SessionsPage`); offline `create_sale` queue now carries `session_id`
- **Login focus chain**: explicit `focusNode` + `textInputAction` (`next`/`done`) + `onFieldSubmitted` on all four login fields — fixes IME "Next" skipping Email under Arabic RTL (reading-order `nextFocus()` misbehavior); widget-tested
- **Dashboard screen (D1)**: `DashboardScreen` (`lib/features/dashboard/`) reachable via an insights `IconButton` in the POS app bar — today revenue/avg-sale/items/sales stat cards, payment mix, top products, per-cashier breakdown, recent sales, pull-to-refresh; wire models in `lib/core/dashboard.dart`, `ApiClient.dashboardSummary()`

### Flutter — tests (97 passing, `flutter analyze` 0 issues)
| File | Tests | Coverage |
|------|-------|----------|
| `api_client_test.dart` | 47 | login, logout, products, categories, sales, refresh, createProduct, updateProduct, session, Product.fromJson, _message edge cases, Country/Currency parsing + meta fetch, SaasSummary/SaasTenant parsing, split-payment request, SaleResult/SaleSummary payment method, register session models current/open/close/sessions/session_id-on-createSale, DashboardSummary parse + fetch + error |
| `auth_test.dart` | 6 | session parse, SaaS/i18n fields, defaults, copyWith, deviceId, RememberedLogin |
| `local_database_test.dart` | 8 | open/close, cacheProducts/cachedProducts, pending command enqueue/order/dedupe/markComplete, split-payment payload round-trip |
| `app_exception_test.dart` | 9 | mapApiError for all HTTP codes, ApiException display |
| `strings_test.dart` | 7 | supported locales (ar default), Arabic/English labels (incl. remember/stock/offline/payment/session/dashboard), EGP money formatting, fallback, `methodLabel` |
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
- **Multi-device sync**: full offline→online replay + conflict resolution (idempotency already in place)
- **Loyalty/v1**: points, buy-X-get-Y — needs `customers` table
- **Restaurant/pharmacy/textile depth**: tables/KDS, expiry/lot, size-color variants (typed tenants)
- **SaaS control plane** (`Backend/saas/*.md`): tenants/subscriptions/plans/backups/restore — reference spec; out of current scope

### Tech debt
- **Rate limiting**: in-memory → Redis-backed for multi-instance
- **Structured logging**: tenant-scoped JSON logs
- **Password hashing**: bcrypt vs SaaS-kit's Argon2id — deviation recorded in roadmap decision log; Argon2id swap affects stored hashes (hardening backlog)
- **Integration tests**: Docker Compose env with real Postgres + Redis
- **Flutter integration tests**: `flutter drive` for login → sale → history
- **OpenAPI spec**: auto-generate from Gin routes, publish for frontend codegen