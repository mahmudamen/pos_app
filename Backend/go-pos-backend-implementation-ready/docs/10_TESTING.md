# Testing Strategy

## Unit

Test:

- validators;
- money calculations;
- token services;
- conflict policies;
- error mapping.

## Integration

Use real PostgreSQL and Redis where behavior depends on them.

Required PostgreSQL tests:

- RLS cross-tenant isolation;
- unique constraints;
- transaction rollback;
- idempotency;
- concurrent sale creation;
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
