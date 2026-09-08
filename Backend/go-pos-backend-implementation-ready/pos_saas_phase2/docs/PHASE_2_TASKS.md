# Phase 2 Task Backlog

## A — Authentication

- [ ] Register endpoint
- [ ] Login endpoint
- [ ] Logout endpoint
- [ ] Refresh endpoint
- [ ] Current-user endpoint
- [ ] Change-password endpoint
- [ ] Generic authentication error responses
- [ ] Password policy validation

## B — Sessions

- [ ] Session repository
- [ ] Session creation
- [ ] Session lookup
- [ ] Session revocation
- [ ] Revoke-all
- [ ] Refresh-token generation
- [ ] Refresh-token hashing
- [ ] Rotation transaction
- [ ] Reuse detection
- [ ] Token-family revocation
- [ ] Expired-session cleanup

## C — JWT

- [ ] Access token service
- [ ] Explicit algorithm validation
- [ ] issuer validation
- [ ] audience validation
- [ ] expiration validation
- [ ] subject validation
- [ ] session ID claim
- [ ] tenant ID claim
- [ ] key abstraction

## D — Authorization

- [ ] Authentication middleware
- [ ] Tenant context middleware
- [ ] Membership lookup
- [ ] Role lookup
- [ ] Permission lookup
- [ ] RequirePermission helper
- [ ] RequireRole helper
- [ ] Platform authorization boundary
- [ ] Cross-tenant security tests

## E — Devices

- [ ] Device migration
- [ ] Device repository
- [ ] Register device
- [ ] List devices
- [ ] Heartbeat
- [ ] Revoke
- [ ] Credential hashing
- [ ] Device authentication
- [ ] Device/session binding

## F — Redis Security

- [ ] Distributed login rate limiter
- [ ] Refresh endpoint rate limiter
- [ ] Password-change limiter
- [ ] Device-registration limiter
- [ ] Abuse counters
- [ ] TTL enforcement

## G — Audit/Security

- [ ] Audit repository
- [ ] Security-event repository
- [ ] Login success event
- [ ] Login failure event
- [ ] Session revoked event
- [ ] Refresh reuse event
- [ ] Password changed event
- [ ] Device registered event
- [ ] Device revoked event

## H — Testing

- [ ] Authentication integration tests
- [ ] Session integration tests
- [ ] Authorization tests
- [ ] Tenant isolation tests
- [ ] Redis rate-limit tests
- [ ] Refresh race test
- [ ] Fuzz token parser
- [ ] API error contract tests
