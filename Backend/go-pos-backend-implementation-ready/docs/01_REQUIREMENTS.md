# Requirements

## Functional

### Authentication
- User login with tenant context.
- Access token validation.
- Refresh token rotation.
- Device/session registration.
- Session revocation.
- Maximum active sessions per user.

### Tenancy
- Every protected business operation resolves tenant identity from trusted authentication context.
- Tenant boundaries are enforced in application authorization and PostgreSQL RLS.

### Catalog
- Categories.
- Products.
- Barcode lookup.
- Product activation/deactivation.
- Tenant-scoped uniqueness for business identifiers.

### Sales
- Sale creation.
- Sale items.
- Totals calculated from exact monetary values.
- Atomic stock and sale operations where stock exists.
- Idempotent sale creation.
- Sale retrieval.

### Offline sync
- Pull changes using a monotonic server cursor.
- Push client commands with stable command IDs.
- Replay-safe processing.
- Explicit conflict policy per entity.

## Non-functional

- Context-aware cancellation.
- Structured logs.
- Health/readiness endpoints.
- Prometheus metrics.
- Automated migrations.
- Integration tests against PostgreSQL.
- Race detector in CI.
- Static analysis and security scanning.
- Graceful shutdown.

## Performance

Performance values are benchmark targets, not universal network SLOs.

- Indexed barcode lookup: target single-digit milliseconds at repository level under representative load.
- Sale creation: benchmark p50/p95 under representative concurrency.
- Sync pull: bounded page sizes and indexed cursor access.

## Reliability

- Transactions for multi-write business operations.
- Idempotency for retryable commands.
- Database backup and restore procedure.
- Redis is an optimization/session/rate-limit dependency, not the business source of truth.

## Security

- Strong secrets.
- TLS at the edge.
- Password hashing with bcrypt.
- Short-lived access tokens.
- Rotating refresh tokens.
- RLS policies for tenant-scoped tables.
- Least-privilege database role.
- Request body and timeout limits.
