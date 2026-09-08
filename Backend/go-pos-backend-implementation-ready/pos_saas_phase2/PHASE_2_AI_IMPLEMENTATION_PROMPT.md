# Phase 2 AI Coding-Agent Prompt

You are continuing the Go POS SaaS backend from Phase 1.

DO NOT rebuild Phase 1 from scratch.

First inspect the complete repository and preserve its architecture.

Your mission is to implement Phase 2 as production-quality code.

============================================================
PHASE 2
============================================================

Implement:

AUTHENTICATION
AUTHORIZATION
SESSIONS
REFRESH TOKEN ROTATION
TENANT CONTEXT
RBAC
DEVICE SECURITY
REDIS RATE LIMITING
AUDIT EVENTS
SECURITY EVENTS
INTEGRATION TESTS

============================================================
NON-NEGOTIABLE
============================================================

No fake implementations.

No TODOs for security-critical behavior.

No in-memory authentication in production paths.

No plaintext passwords.

No plaintext refresh tokens in PostgreSQL.

No client-controlled authorization.

No trusted client tenant_id.

No wildcard authorization.

No SQL string interpolation.

No sensitive data in logs.

No swallowed errors.

No panic-based business logic.

============================================================
1. INSPECT FIRST
============================================================

Read:

README.md
docs/00_CONSTITUTION.md
docs/01_REQUIREMENTS.md
docs/02_ARCHITECTURE.md
docs/03_PLAN.md
docs/04_TASKS.md
docs/05_CODING_STANDARDS.md
docs/06_SECURITY.md
docs/07_API_CONTRACT.md
docs/08_DATABASE.md
docs/09_SYNC.md
docs/10_TESTING.md
docs/11_OPERATIONS.md

Read all existing Go files.

Build a mental dependency map.

Do not duplicate existing functionality.

============================================================
2. ARCHITECTURE
============================================================

Use:

HTTP handler
    ->
application service
    ->
domain
    ->
repository interface
    ->
PostgreSQL adapter

Security:

HTTP
 ->
authentication
 ->
identity
 ->
tenant context
 ->
authorization
 ->
application service

Redis must remain infrastructure.

Do not place SQL in handlers.

Do not place business rules in Gin middleware.

============================================================
3. REGISTRATION
============================================================

POST /api/v1/auth/register

Validate:

email
password
tenant/business name where required

Normalize email consistently.

Hash password with Argon2id.

Create:

tenant
user
membership
initial tenant role

Use one PostgreSQL transaction.

Never return password hash.

Prevent duplicate account creation safely using database constraints.

Do not reveal unnecessary account-existence information.

============================================================
4. LOGIN
============================================================

POST /api/v1/auth/login

Flow:

validate request
rate-limit
normalize email
lookup user
verify password
verify account status
resolve tenant/membership
create session
generate access token
generate refresh token
hash refresh token
persist session
audit success
return tokens

Failure:

use generic authentication error.

Do not reveal:

"email exists"
"password wrong"
"user disabled"

unless the project security policy explicitly permits it.

Always record useful security telemetry without leaking secrets.

============================================================
5. REFRESH TOKENS
============================================================

Implement secure rotating refresh tokens.

Generate cryptographically random opaque token.

Persist only a cryptographic hash.

Associate token with:

session
user
tenant
token family

Refresh must execute atomically.

Protect against concurrent refresh.

Use row-level locking or equivalent transactional strategy.

The same valid refresh token must not successfully refresh twice.

If reuse is detected:

revoke family/session
write security event
reject refresh

Return a safe authentication error.

============================================================
6. ACCESS TOKENS
============================================================

Access JWT must include only required claims:

sub
sid
tenant_id
jti
iss
aud
iat
nbf
exp

Validate:

signature
algorithm
issuer
audience
expiration
not-before

Never accept "none".

Do not accept arbitrary signing algorithms.

Keep access tokens short-lived.

============================================================
7. LOGOUT
============================================================

POST /api/v1/auth/logout

Revoke current session.

Do not require the client to delete the token for server-side revocation.

Write audit/security event.

============================================================
8. SESSION APIs
============================================================

Implement:

GET /api/v1/auth/sessions

DELETE /api/v1/auth/sessions/:id

POST /api/v1/auth/sessions/revoke-all

A user can only manage their own sessions.

Tenant administrators must not automatically gain access to every user's
sessions unless an explicit privileged security permission exists.

Never allow:

DELETE /sessions/{other-user-session}

============================================================
9. PASSWORD CHANGE
============================================================

POST /api/v1/auth/change-password

Require:

current password
new password

Verify current password.

Hash new password with Argon2id.

After successful password change:

revoke existing refresh-token families according to security policy.

Require reauthentication for sensitive operations where appropriate.

Write audit/security event.

============================================================
10. AUTHENTICATION MIDDLEWARE
============================================================

Implement:

RequireAuthentication

Extract:

Authorization: Bearer <token>

Reject malformed/missing tokens.

Never accept access tokens from query strings.

Attach authenticated principal to context.

Principal should contain:

user ID
session ID
tenant ID

Do not place raw token in context.

============================================================
11. TENANT CONTEXT
============================================================

Implement tenant context after authentication.

The tenant ID in JWT is an identity hint, not sufficient authorization.

Verify active membership against PostgreSQL.

For tenant resources:

authenticated user
+
active membership
+
required permission

must all succeed.

Never use:

tenant_id := request.body.tenant_id

as an authorization mechanism.

============================================================
12. RBAC
============================================================

Implement repository/service resolution of:

membership
roles
permissions

Implement reusable authorization:

RequirePermission("users.read")
RequirePermission("devices.create")

Permission checks must be server-side.

Cache permission data carefully if desired, but authorization correctness
must not depend on stale Redis state.

============================================================
13. DEVICE SECURITY
============================================================

Create migration for devices.

Implement:

POST /api/v1/devices
GET /api/v1/devices
POST /api/v1/devices/:id/heartbeat
POST /api/v1/devices/:id/revoke

Device credentials must be high entropy.

Persist only a hash of long-lived credentials.

Associate device with tenant.

Do not permit cross-tenant device lookup.

Revoked devices cannot authenticate.

============================================================
14. REDIS RATE LIMITING
============================================================

Implement a distributed limiter.

Use Redis atomic operations.

Protect:

register
login
refresh
change-password
device registration

Use multiple dimensions where appropriate:

IP
account
tenant
device

Avoid unbounded Redis keys.

Use TTLs.

Return 429.

Do not expose internal Redis errors.

============================================================
15. AUDIT EVENTS
============================================================

Implement application-level audit service.

At minimum:

USER_REGISTERED
LOGIN_SUCCESS
LOGIN_FAILURE
LOGOUT
TOKEN_REFRESH
TOKEN_REUSE_DETECTED
SESSION_REVOKED
SESSIONS_REVOKED_ALL
PASSWORD_CHANGED
DEVICE_REGISTERED
DEVICE_REVOKED

Do not write passwords/tokens into metadata.

============================================================
16. SECURITY EVENTS
============================================================

Implement security event service.

Capture:

authentication failures
refresh reuse
rate limit violations
revoked-device attempts
authorization failures where useful
cross-tenant access attempts

Avoid turning logs into a data-exfiltration source.

============================================================
17. DATABASE TRANSACTIONS
============================================================

Registration transaction:

tenant
user
membership
role assignment
audit event

Login transaction:

session creation
audit event

Refresh transaction:

lock session
verify token
rotate token
update session
audit event

Use PostgreSQL transactions.

============================================================
18. CONCURRENCY
============================================================

Explicitly test:

two simultaneous refresh calls
two simultaneous session revocations
duplicate registration
multiple device registration attempts

The refresh-token race is critical.

Exactly one concurrent refresh should win for a single token.

============================================================
19. API ERRORS
============================================================

Use stable codes.

Examples:

AUTH_INVALID_CREDENTIALS
AUTH_TOKEN_INVALID
AUTH_TOKEN_EXPIRED
AUTH_SESSION_REVOKED
AUTH_REFRESH_REUSED
AUTH_FORBIDDEN
AUTH_TENANT_ACCESS_DENIED
AUTH_DEVICE_REVOKED
RATE_LIMIT_EXCEEDED
VALIDATION_ERROR

Do not expose internal error details.

============================================================
20. TESTS
============================================================

Use real PostgreSQL for integration tests.

Test:

register
login
logout
refresh
refresh rotation
refresh reuse
session listing
session revocation
revoke-all
password change
RBAC
tenant isolation
device registration
device revocation
rate limiting

Security tests must attempt malicious tenant switching.

Example:

authenticated user from tenant A

request:

GET /api/v1/tenants/<tenant-B-id>/...

must return authorization failure.

============================================================
21. Fuzzing
============================================================

Add fuzz tests for:

JWT parsing
authorization header parsing
refresh token parsing
registration validation
login validation

Malformed input must never crash the server.

============================================================
22. QUALITY
============================================================

Run:

gofmt -w .
go test ./...
go test -race ./...
go vet ./...

If available:

staticcheck ./...
govulncheck ./...

Fix failures.

Do not disable tests merely to make the build pass.

============================================================
23. SECURITY REVIEW
============================================================

Before completion inspect:

OWASP API Security Top 10
authentication
authorization
session management
IDOR
tenant isolation
rate limiting
credential stuffing
token theft
refresh replay
race conditions
secret leakage
SQL injection
logging leakage

Update:

docs/06_SECURITY.md
docs/07_API_CONTRACT.md
docs/08_DATABASE.md
docs/10_TESTING.md

where implementation requires it.

============================================================
24. FINAL REPORT
============================================================

Report:

files created
files changed
migrations
API endpoints
authentication flow
refresh-token flow
session model
tenant isolation mechanism
RBAC mechanism
device security
Redis rate limits
audit/security events
tests
security findings
remaining limitations

Never claim a feature is complete unless it is actually implemented and tested.
