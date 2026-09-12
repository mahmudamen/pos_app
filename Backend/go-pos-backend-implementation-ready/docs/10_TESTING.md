# Testing Strategy

## Unit

Test:

- validators;
- money calculations;
- token services;
- conflict policies;
- error mapping.

## Integration

Use real PostgreSQL and Redis where behavior depends on them. The DB-backed
suites (auth, catalog, sales, sync, database) skip unless
`TEST_DATABASE_URL`/`DATABASE_URL` is configured. To run them against an
ephemeral Docker Compose stack (real Postgres + Redis, E3):

```bash
./scripts/integration-test.sh               # go test ./... with TEST_DATABASE_URL set
RACE=1 ./scripts/integration-test.sh        # same, under go test -race
# or: make integration-test
```

Coverage currently includes:

- RLS cross-tenant isolation (`TestRLSIsolatesTenants`);
- auth login / refresh rotation / replay revocation / logout (AUTH-011);
- sync push apply → replay → conflict → dedupe (SYNC-007);
- idempotent + concurrent sale creation (SALE-008);
- catalog CRUD and soft-delete.

Required PostgreSQL tests also include:

- unique constraints;
- transaction rollback;
- change sequence ordering.

## API

Use `httptest` against the Gin router.

Verify:

- status codes;
- response envelope;
- validation;
- authentication;
- authorization;
- request IDs;
- error sanitization.

## Security

Test:

- invalid JWT algorithm;
- wrong issuer;
- expired tokens;
- wrong token type;
- refresh replay;
- cross-tenant access;
- rate-limit behavior;
- CORS policy.

## Performance

Benchmark:

- barcode lookup;
- sale creation;
- sync pull.

Use representative data and concurrency. Do not present a local benchmark as a universal production latency guarantee.

## Required quality gates

```bash
gofmt
go vet ./...
go test ./...
go test -race ./...
golangci-lint run ./...
gosec ./...
```
