# POS SaaS Quickstart

## Prerequisites

- Go 1.23+
- PostgreSQL 16+
- Redis 7 for readiness/session workflows
- Flutter SDK with Dart 3.3+

## Backend setup

```bash
cd Backend/go-pos-backend-implementation-ready
cp .env.example .env
# Create the local role/database as a PostgreSQL administrator:
psql -U postgres -d postgres -f scripts/bootstrap_local.sql
set -a; source .env; set +a
make migrate
make run
```

## Backend checks

```bash
go test ./...
go vet ./...
go build ./cmd/api
go run github.com/pressly/goose/v3/cmd/goose@v3.21.1 -dir internal/infrastructure/database/migrations validate
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
```

## Flutter setup

```bash
cd Flutter/pos_go_app
flutter pub get
flutter analyze
flutter test
flutter run --dart-define=API_BASE_URL=http://127.0.0.1:8080
```

## End-to-end smoke path

1. Seed a tenant, active user, device, and products in PostgreSQL.
2. Sign in from the Flutter client.
3. Confirm `GET /v1/products` returns only the authenticated tenant's products.
4. Add products to the cart and complete checkout.
5. Confirm `POST /v1/sales` creates one sale and decrements stock.
6. Repeat the same request with the same idempotency key and confirm the original sale is returned.
7. Disable the network, create a local pending command, restore connectivity, and confirm synchronization applies it once.
