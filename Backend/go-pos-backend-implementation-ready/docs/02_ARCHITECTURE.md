# Architecture

## Request flow

```text
Flutter Client
    |
    v
Caddy / TLS
    |
    v
Gin Router
    |
    +--> middleware: request ID, recovery, auth, tenant, rate limit
    |
    v
HTTP Handler / DTO
    |
    v
Use Case / Service
    |
    +--> transaction
    |       |
    |       v
    |    Repository
    |       |
    |       v
    |    PostgreSQL
    |
    +--> Redis for sessions/cache/rate limits
```

## Package layout

```text
cmd/api/
internal/
  app/
  config/
  domain/
  application/
  transport/http/
  infrastructure/database/
  infrastructure/redis/
  observability/
pkg/
migrations/
```

## Tenant RLS

Tenant identity must be set on the same database connection/transaction that executes tenant-scoped SQL.

Preferred pattern:

```sql
SELECT set_config('app.current_tenant', $1, true);
```

The `true` flag makes the setting transaction-local.

Do not acquire a connection, set a session variable, release it, and then assume a later operation uses that same connection.

## Identity model

Separate:

- server primary key: UUID `devices.id`
- client-provided stable device identifier: `devices.client_device_id`

Do not use two unrelated identifiers interchangeably.

## Money

Use integer minor units:

```text
amount_minor BIGINT
currency CHAR(3)
```

All calculations must use integer arithmetic or a decimal library where required by the domain.

## Transaction ownership

The application service/use-case owns transaction boundaries. Repositories accept a transaction-aware database interface so related writes execute atomically.
