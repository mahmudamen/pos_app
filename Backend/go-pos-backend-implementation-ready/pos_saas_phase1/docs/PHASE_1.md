# Phase 1 — Foundation & Security Core

## Goal

Establish a secure production-oriented backend foundation before implementing
the complete POS domain.

## Included

- configuration
- PostgreSQL connection pool
- Redis connection
- HTTP server
- liveness/readiness
- Argon2id password hashing
- JWT access-token service
- initial identity/tenant schema
- memberships/RBAC schema
- session/refresh-token schema
- audit/security-event schema
- idempotency schema
- transactional outbox schema
- Docker development services

## Next phase

Phase 2 should implement actual authentication APIs, tenant context,
authorization middleware, session creation/revocation, refresh-token rotation,
rate limiting, device management, and their integration tests.

## Security note

The JWT implementation in this phase is a foundation only. Production
deployment should move signing to an asymmetric key architecture with key
rotation and a secure secret/key-management mechanism.
