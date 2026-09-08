# Security Requirements

## Authentication

- Validate JWT signature, algorithm, issuer, expiration, token type.
- Use separate access and refresh secrets.
- Keep access tokens short-lived.
- Rotate refresh tokens.
- Hash refresh tokens at rest.
- Detect refresh-token reuse.
- Revoke the relevant session/token family on reuse.

## Tenant isolation

Tenant ID comes from trusted authenticated context.

RLS must be enabled and forced where appropriate for tenant-owned tables. Policies must cover SELECT/INSERT/UPDATE/DELETE according to the actual table semantics.

RLS context is transaction-local.

## Sessions

When enforcing a session limit, order by an explicit timestamp/sequence and revoke the oldest eligible session. Do not use unordered Redis set removal.

## Rate limiting

A fixed-window implementation must set its expiration only when the counter is first created, e.g. an atomic Redis script. Do not refresh the TTL on every request unless sliding-window semantics are explicitly intended.

## CORS

Use an explicit allow-list in production.

## Request hardening

- request body size limit;
- read/write/idle timeouts;
- maximum pagination size;
- validation of uploaded/encoded payloads;
- graceful handling of malformed JWTs.

## Secrets

Generate high-entropy secrets outside source control. `.env.example` contains placeholders only.

## Logging

Never log:

- passwords;
- raw JWTs;
- refresh tokens;
- Authorization headers;
- payment credentials;
- unnecessary sensitive customer data.
