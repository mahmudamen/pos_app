# Coding Standards

## Go

- Follow idiomatic Go.
- Keep packages small and cohesive.
- Constructors validate required dependencies.
- Context is the first parameter for I/O.
- Wrap errors with `%w`.
- Use `errors.Is` / `errors.As`.
- Avoid unnecessary interfaces.
- Prefer explicit dependencies over globals.

## HTTP

Handlers should:

1. parse input;
2. validate transport-level requirements;
3. call a use case;
4. map typed errors to HTTP responses;
5. return the standard envelope.

Handlers must not contain business transactions.

## SQL

- Parameterized queries only.
- Explicit columns.
- No `SELECT *` in production repositories.
- Index every important query path.
- Review query plans for hot paths.
- Keep tenant predicates/RLS assumptions explicit.

## Errors

Stable application error codes should be machine-readable:

```json
{
  "error": {
    "code": "validation_error",
    "message": "invalid request",
    "request_id": "..."
  }
}
```

Never return database error strings directly to clients.

## Tests

Prefer table-driven unit tests. Integration tests must exercise real PostgreSQL behavior for RLS, constraints, transactions, and query semantics.
