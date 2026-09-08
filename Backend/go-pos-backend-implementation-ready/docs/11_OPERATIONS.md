# Operations

## Development

PostgreSQL and Redis run through Docker Compose.

## Production topology

```text
Internet
   |
 TLS / Caddy
   |
 Go POS API
   |
 +---------+
 |         |
PostgreSQL Redis
```

PostgreSQL is authoritative.

Redis failure should not corrupt business data.

## Deployment

1. Build immutable application artifact.
2. Apply backward-compatible migrations.
3. Deploy application.
4. Check readiness.
5. Monitor error rate/latency.
6. Roll back application if required.
7. Never roll back destructive schema changes blindly.

## Backups

Backups must be automated and periodically restored into an isolated environment.

A backup that has never been restore-tested is not considered verified.

## Monitoring

Monitor:

- request rate;
- 4xx/5xx rate;
- latency p50/p95/p99;
- database pool exhaustion;
- database errors;
- Redis errors;
- authentication failures;
- sync failures;
- process restarts.

## Security

Production requires:

- TLS;
- restricted database network access;
- least-privilege DB credentials;
- secret management;
- explicit CORS;
- firewall policy;
- OS patching;
- log retention policy.
