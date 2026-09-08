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
- Migrations are goose files in `internal/infrastructure/database/migrations/` (single-file `.sql`, numbered `001_extensions`, `003_core`, `004_permissions`, `005_sync`); apply with `make migrate`, roll back `make migrate-down`.
- API: everything under `/v1`; auth (`/auth/login|refresh|logout`), catalog (`/categories`, `/products`, `/products/barcode/:barcode`, PATCH `/products/:id`), sales (`POST /sales`). Money is integer minor units (`price_minor`, `subtotal_minor`).
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