# Engineering Constitution

## Non-negotiable principles

1. PostgreSQL is the source of truth for business state.
2. Tenant isolation is defense-in-depth: application authorization plus database RLS.
3. Never use `float64` for money. Store integer minor units or another exact representation.
4. Business invariants that span multiple writes must be enforced inside a database transaction.
5. Write APIs must support idempotency where retries can create duplicate business effects.
6. Secrets never belong in source control.
7. Authentication identifies a principal; authorization determines what that principal may do.
8. Refresh tokens are rotated and replay/reuse detection is mandatory.
9. API errors are typed, stable, sanitized, and observable.
10. HTTP handlers remain thin. Business rules live in use cases/services.
11. Repositories own SQL; use cases own business transactions.
12. Every I/O operation accepts and respects `context.Context`.
13. Database connections must never be stored in request context.
14. No external network call is made while a database transaction is open.
15. Logs must not contain passwords, tokens, payment secrets, or unnecessary personal data.
16. Normal operational errors must return errors, not panic.
17. Migrations are explicit and never destructive by default.
18. Compatibility with the Flutter client is treated as a contract.
19. Security-sensitive behavior receives automated tests.
20. A change is not complete until its tests, documentation, and operational impact are addressed.

## Forbidden patterns

- raw SQL interpolation
- trusting a tenant ID supplied by an untrusted client
- global uniqueness where the business identity is tenant-scoped
- storing a `*pgx.Conn` in `context.Context`
- using Redis `SPOP` as an approximation of oldest-session eviction
- fixed-window rate limiting implemented by refreshing the TTL on every request
- obsolete `X-XSS-Protection` as a security control
- wildcard production CORS
- hard-coded JWT secrets
- logging Authorization headers
