# Phase 2 — Authentication, Authorization, Sessions & Device Security

## Objective

Turn the Phase 1 foundation into a real production-oriented identity and
tenant-security core.

## Scope

### Authentication API

- Register user
- Login
- Logout current session
- Refresh access token
- Revoke session
- Revoke all sessions
- Change password
- Current user
- Current tenant membership

### Session security

- First-class session records
- Refresh-token rotation
- Refresh-token hashing
- Token-family reuse detection
- Session revocation
- Session expiry
- Device association
- IP/user-agent tracking
- Security events

### Authorization

- Tenant membership validation
- RBAC middleware
- Permission middleware
- Platform vs tenant scope
- Server-side authorization
- No client-supplied tenant trust

### Device security

- Device registration
- Device status
- Device revocation
- Device heartbeat
- Device/session association

### Abuse protection

- Redis distributed login rate limiting
- Account/IP throttling
- Sensitive-endpoint throttling
- Safe 429 responses

### API quality

- Request validation
- Request IDs
- Consistent errors
- Authentication middleware
- Tenant context middleware
- Authorization middleware

## Required security properties

1. Passwords use Argon2id.
2. Raw refresh tokens are never persisted.
3. Access tokens are short lived.
4. Refresh tokens rotate.
5. Refresh reuse revokes the token family/session.
6. Revoked sessions cannot refresh.
7. JWT issuer and audience are validated.
8. JWT signing algorithm is explicitly restricted.
9. Tenant IDs supplied by clients are never trusted for authorization.
10. Every tenant operation verifies membership.
11. Every permission check is server-side.
12. Login failures are rate limited.
13. Secrets never appear in logs or API responses.
14. Authentication events are auditable.
15. Security-sensitive mutations are idempotency-aware where appropriate.

## API

### Public

POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh

### Authenticated

POST /api/v1/auth/logout
GET  /api/v1/auth/me
GET  /api/v1/auth/sessions
DELETE /api/v1/auth/sessions/:id
POST /api/v1/auth/sessions/revoke-all
POST /api/v1/auth/change-password

### Tenant

GET /api/v1/tenants/:tenant_id
GET /api/v1/tenants/:tenant_id/members/me

### Device

POST /api/v1/devices
GET  /api/v1/devices
POST /api/v1/devices/:id/heartbeat
POST /api/v1/devices/:id/revoke

## Refresh flow

```text
Client
  |
  | refresh token
  v
API
  |
  +--> hash token
  |
  +--> find active session/family
  |
  +--> transaction
  |      |
  |      +--> verify current token
  |      +--> rotate token
  |      +--> replace stored hash
  |      +--> update last_used_at
  |      +--> commit
  |
  +--> issue new access token
  +--> issue new refresh token
```

Reuse detection:

```text
old refresh token presented
        |
        v
stored token no longer matches
        |
        v
detect reuse
        |
        +--> revoke token family/session
        +--> security event
        +--> reject request
```

## Device model

Phase 2 adds:

```text
devices
--------
id
tenant_id
branch_id
device_identifier
device_name
device_type
platform
app_version
status
credential_hash
created_at
last_seen_at
revoked_at
```

Never store a raw long-lived device secret.

## RBAC

Roles can be tenant-scoped.

Example permissions:

```text
tenant.read
tenant.update

users.read
users.create
users.update
users.delete

devices.read
devices.create
devices.revoke

audit.read
```

Do not give every authenticated user administrative permissions.

## Testing

Minimum security test suite:

- registration
- duplicate email
- invalid login
- successful login
- access-token validation
- issuer validation
- audience validation
- expiration
- invalid signature
- logout
- revoked session
- refresh rotation
- refresh replay/reuse
- revoke-all
- password change
- tenant membership
- cross-tenant access denial
- permission denial
- device registration
- device revocation
- device authentication
- rate limit
- concurrent refresh race
- concurrent session revocation

## Completion gate

```bash
gofmt -w .
go test ./...
go test -race ./...
go vet ./...
```

Do not mark Phase 2 complete while any security-critical test is failing.
