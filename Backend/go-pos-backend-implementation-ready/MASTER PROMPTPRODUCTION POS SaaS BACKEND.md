You are a Principal Backend Engineer, Software Architect, Security Engineer,
Database Architect, DevOps Engineer, and QA Engineer.

You are working on a production-grade Multi-Tenant POS SaaS Backend.

Your task is to IMPLEMENT THE COMPLETE BACKEND CODEBASE, not just generate
documentation or examples.

The backend must be designed for a Flutter POS client and future web/mobile
clients.

============================================================
1. PROJECT BASELINE
============================================================

Use the existing repository as the source of truth.

The existing architecture specifies:

- Go 1.23+
- Gin HTTP framework
- PostgreSQL 16
- Redis 7
- pgx/v5
- JWT access tokens
- Rotating refresh tokens
- Goose migrations
- Prometheus metrics
- Structured JSON logging
- Docker Compose development environment
- systemd + Caddy production deployment

Existing documentation includes:

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

Also respect:

speckit.constitution
speckit.plan
speckit.tasks

DO NOT replace the existing architecture blindly.

First inspect the entire repository.

Understand existing code before modifying it.

Preserve good existing work.

Improve incomplete implementations.

Do not create duplicate infrastructure.

============================================================
2. PRIMARY OBJECTIVE
============================================================

Build the complete secure backend CORE for a SaaS POS platform.

The backend must provide:

1. SaaS tenant management
2. Organizations / businesses
3. Users
4. Authentication
5. Authorization
6. Sessions
7. Devices
8. Roles
9. Permissions
10. API keys where appropriate
11. OAuth2/OIDC-ready authentication architecture
12. JWT access tokens
13. Rotating refresh tokens
14. Session revocation
15. Device/session management
16. PostgreSQL persistence
17. Redis distributed state/cache
18. Audit logging
19. Security monitoring
20. Rate limiting
21. Idempotency
22. API versioning
23. Error handling
24. Observability
25. Background jobs
26. Offline POS synchronization
27. Conflict handling
28. Multi-database architecture where justified
29. Tenant isolation
30. POS domain foundation
31. Production deployment
32. Automated testing

The result must be production-quality software.

Do not produce pseudo-code.

Do not leave TODO placeholders for core functionality.

============================================================
3. ARCHITECTURE PRINCIPLES
============================================================

Use Clean Architecture / Hexagonal Architecture principles.

Separate:

HTTP/API
Application
Domain
Infrastructure
Persistence
Security
Background jobs

Recommended structure:

cmd/
    api/
    worker/
    migrate/

internal/
    domain/
    application/
    ports/
    adapters/
        http/
        postgres/
        redis/
        auth/
        messaging/
    middleware/
    security/
    jobs/
    observability/
    config/

pkg/

migrations/

docs/

tests/

Do not create giant files.

Prefer small cohesive packages.

Dependency direction must point inward.

Domain must not depend on Gin.

Domain must not depend directly on PostgreSQL.

Domain must not depend directly on Redis.

Use interfaces at appropriate architectural boundaries.

============================================================
4. MULTI-TENANCY
============================================================

The system is a TRUE multi-tenant SaaS application.

Design for:

Platform
    |
    +-- Tenant / Organization
          |
          +-- Branch
          |
          +-- Warehouse
          |
          +-- POS terminals
          |
          +-- Users
          |
          +-- Roles
          |
          +-- Products
          |
          +-- Inventory
          |
          +-- Orders
          |
          +-- Payments
          |
          +-- Customers
          |
          +-- Reports

Every tenant-owned record must contain tenant_id where appropriate.

NEVER trust tenant_id supplied by the client.

Tenant context must be established from authenticated identity,
membership, session, or authorized platform operation.

Every tenant-scoped query must enforce tenant isolation.

Implement defense in depth:

1. Application-level tenant filtering
2. Repository-level tenant enforcement
3. PostgreSQL Row Level Security where appropriate
4. Authorization checks

Prevent:

- IDOR
- cross-tenant access
- tenant enumeration
- insecure object references
- accidental unrestricted queries

Create reusable tenant context handling.

============================================================
5. DATABASE ARCHITECTURE
============================================================

PostgreSQL is the primary system of record.

Use pgx/v5.

Do not use an ORM unless the existing architecture explicitly requires one.

Prefer SQL-first repositories.

Use:

- transactions
- prepared statements where beneficial
- explicit indexes
- foreign keys
- unique constraints
- check constraints
- partial indexes
- proper timestamps
- UUID/UUIDv7 strategy where appropriate
- soft deletion only where justified

Every migration must be:

- deterministic
- reversible where practical
- safe
- documented

Do not modify an old migration after it has been released.

Create a new migration for schema changes.

============================================================
6. DATABASE SEPARATION
============================================================

Design the system so it can support more than one database when scale
requires it.

Recommended logical separation:

DATABASE A:
Identity / platform / tenant metadata

DATABASE B:
POS transactional data

DATABASE C:
Analytics / reporting

DATABASE D:
Audit/event data if required at scale

However:

DO NOT introduce unnecessary database complexity in the initial deployment.

The code must support separate PostgreSQL connection pools/configurations.

Example:

DB_IDENTITY_URL
DB_CORE_URL
DB_ANALYTICS_URL
DB_AUDIT_URL

If initially using one PostgreSQL server/database, keep logical boundaries
clean so they can later be separated.

Never distribute a transaction across independent databases unless a real
distributed transaction strategy exists.

Prefer transactional outbox/event patterns instead.

============================================================
7. AUTHENTICATION
============================================================

Implement production-grade authentication.

Required:

- registration
- login
- logout
- token refresh
- refresh token rotation
- session creation
- session revocation
- list active sessions
- revoke specific session
- revoke all sessions
- password change
- password reset architecture
- email verification architecture
- account lockout/risk controls
- authentication event logging

Passwords:

Use Argon2id.

Never store plaintext passwords.

Never log passwords.

Never return passwords.

Never return password hashes through APIs.

============================================================
8. JWT
============================================================

Implement short-lived JWT access tokens.

Access token should contain only necessary claims.

Example:

sub
sid
tenant_id
roles
permissions/version
iat
exp
iss
aud
jti

Do not put large user profiles inside JWTs.

Access token lifetime must be configurable.

Implement key management abstraction.

Do not hard-code signing secrets.

Support asymmetric signing architecture such as:

RS256 or EdDSA

where appropriate.

Include key rotation architecture.

Validate:

- issuer
- audience
- expiration
- not-before
- signature
- token type
- session state where applicable

============================================================
9. REFRESH TOKENS
============================================================

Refresh tokens must be:

- opaque/random where possible
- high entropy
- stored hashed in database
- associated with session/device
- rotated on every refresh

Implement refresh-token reuse detection.

Example:

Session:
    refresh_token_family_id
    current_refresh_token_hash
    previous token state
    revoked_at
    last_used_at

If token reuse is detected:

1. revoke the token family
2. revoke the associated session
3. log security event
4. optionally revoke all user sessions depending on policy

Never store raw refresh tokens in PostgreSQL.

============================================================
10. SESSION SYSTEM
============================================================

Sessions are first-class security objects.

Create:

user_sessions

Fields should include concepts such as:

id
user_id
tenant_id
device_id
refresh_token_family_id
ip_address
user_agent
device_name
created_at
last_seen_at
expires_at
revoked_at
revocation_reason

Provide APIs:

POST /api/v1/auth/login

POST /api/v1/auth/refresh

POST /api/v1/auth/logout

GET /api/v1/auth/sessions

DELETE /api/v1/auth/sessions/:id

POST /api/v1/auth/sessions/revoke-all

A user must be able to see and revoke sessions.

============================================================
11. OAUTH2 / OIDC
============================================================

Do NOT implement OAuth2 incorrectly.

Create a proper abstraction for external identity providers.

Architecture must support:

Google
Microsoft
Apple
GitHub
enterprise OIDC providers

without coupling the core authentication domain to one provider.

Separate:

Identity Provider
    |
OAuth/OIDC adapter
    |
Identity mapping
    |
Internal User
    |
Internal Session
    |
Internal Authorization

External OAuth identity must NEVER directly become an authorization system.

Authorization remains internal.

Store provider identities such as:

provider
provider_subject
user_id
email
created_at
last_login_at

Use OIDC discovery/JWKS validation where appropriate.

Validate:

issuer
audience
signature
nonce
state
redirect URI
PKCE

Do not store provider client secrets in source code.

============================================================
12. AUTHORIZATION
============================================================

Implement RBAC.

Entities:

User
Role
Permission
UserRole
RolePermission

Support tenant-specific roles.

Example permissions:

tenant.read
tenant.update

users.read
users.create
users.update
users.delete

products.read
products.create
products.update

inventory.read
inventory.adjust

orders.read
orders.create
orders.cancel

payments.read
payments.create
payments.refund

reports.read

settings.read
settings.update

audit.read

system.admin

Do not implement authorization only in the frontend.

Every protected endpoint must perform authorization server-side.

Implement middleware/helpers such as:

RequireAuthentication()
RequireTenant()
RequirePermission()
RequireRole()

Avoid scattered authorization logic.

============================================================
13. DEVICE MANAGEMENT
============================================================

POS systems use physical terminals.

Implement device registration.

Device model should support:

id
tenant_id
branch_id
device_identifier
device_name
device_type
platform
app_version
last_seen_at
status
created_at
revoked_at

Support:

device registration
device authentication
device revocation
device heartbeat
device/session association

Never trust arbitrary device identifiers without registration.

Design device credentials so they can be rotated/revoked.

============================================================
14. POS TERMINAL SECURITY
============================================================

A POS terminal must not behave exactly like a browser.

Support:

device identity
device authorization
terminal session
operator login
operator logout
offline capability
sync cursor
idempotency

Separate:

User authentication

from:

Device authentication

from:

POS operator authorization

============================================================
15. API DESIGN
============================================================

Use REST APIs.

Version:

/api/v1/

Use consistent resource naming.

Examples:

/api/v1/auth/login
/api/v1/auth/refresh

/api/v1/users
/api/v1/users/:id

/api/v1/tenants
/api/v1/tenants/:id

/api/v1/branches
/api/v1/devices

/api/v1/products
/api/v1/categories

/api/v1/inventory

/api/v1/orders
/api/v1/payments

/api/v1/sync

Use proper HTTP semantics.

============================================================
16. API RESPONSE FORMAT
============================================================

Use consistent response envelopes.

Success example:

{
  "data": {},
  "meta": {}
}

Error:

{
  "error": {
    "code": "AUTH_INVALID_CREDENTIALS",
    "message": "Invalid credentials",
    "request_id": "..."
  }
}

Do not expose:

SQL errors
stack traces
internal paths
database details
JWT internals
password information

Use machine-readable error codes.

============================================================
17. REQUEST ID / CORRELATION
============================================================

Every request must have:

request_id

Accept incoming request ID only according to a safe validation policy.

Otherwise generate one.

Return:

X-Request-ID

Include it in:

logs
errors
audit records where appropriate
tracing context

============================================================
18. RATE LIMITING
============================================================

Implement Redis-backed distributed rate limiting.

At minimum protect:

login
refresh
password reset
registration
OAuth endpoints
device registration
sensitive administrative APIs

Rate limit by combinations of:

IP
account/user
tenant
device

Do not allow one dimension to be bypassed trivially.

Return HTTP 429.

Include Retry-After where appropriate.

============================================================
19. BRUTE FORCE PROTECTION
============================================================

Login security must protect against:

credential stuffing
password spraying
brute force

Implement configurable controls.

Do not permanently lock users based only on a simplistic counter.

Use:

progressive delays
temporary lockouts
IP/account/device signals

Log security events.

============================================================
20. CSRF / CORS
============================================================

Because the primary client is Flutter:

Do not blindly enable wildcard CORS.

Make allowed origins configurable.

If cookie-based authentication is introduced for browser clients,
implement CSRF protection.

For mobile access-token APIs, use Authorization Bearer tokens.

Never accept tokens from arbitrary locations.

============================================================
21. SECURITY HEADERS
============================================================

Implement appropriate security headers for HTTP responses.

Examples:

Content-Security-Policy where applicable
X-Content-Type-Options
Referrer-Policy
Strict-Transport-Security in production
Cache-Control for sensitive responses

Do not apply browser-specific headers blindly to API behavior.

============================================================
22. INPUT VALIDATION
============================================================

Validate all external input.

Never trust:

JSON
query parameters
path parameters
headers
device IDs
tenant IDs
pagination
sorting
filter expressions

Use explicit DTO/request structures.

Do not bind arbitrary JSON directly into domain/database models.

Prevent:

SQL injection
mass assignment
parameter pollution
oversized requests
invalid UUIDs
invalid dates
invalid numeric ranges

============================================================
23. PAGINATION
============================================================

For large POS SaaS datasets prefer cursor/keyset pagination.

Avoid OFFSET pagination for large transactional tables.

Example:

GET /products?cursor=...&limit=50

Enforce maximum page size.

============================================================
24. IDEMPOTENCY
============================================================

POS clients frequently retry requests.

Implement idempotency support for critical mutation APIs.

Header:

Idempotency-Key

Especially:

orders
payments
refunds
inventory movements

Store:

tenant
endpoint
idempotency key
request fingerprint
status
response
created_at
expires_at

Same key + same request:

return original result.

Same key + different request:

return conflict.

============================================================
25. OFFLINE-FIRST SYNC
============================================================

The POS client must continue working during temporary connectivity loss.

Design synchronization APIs.

Example:

POST /api/v1/sync/push
GET  /api/v1/sync/pull

Support:

device_id
tenant_id
branch_id
client mutation ID
entity type
entity ID
operation
payload
client timestamp
server timestamp
version

Every mutation needs an idempotent client mutation identifier.

Implement:

push
pull
cursor
acknowledgement
retry
conflict detection
conflict resolution policy

Do not use client timestamps as the only source of truth.

Server controls authoritative ordering/versioning.

============================================================
26. VERSIONING / CONCURRENCY
============================================================

Use optimistic concurrency where appropriate.

Entities can have:

version
updated_at

For updates:

client sends expected version.

If stale:

return conflict.

Example:

409 CONFLICT

with machine-readable error code.

============================================================
27. AUDIT LOGGING
============================================================

Implement immutable audit logging.

Record:

actor
tenant
user
device
action
resource
resource_id
before
after
IP
user_agent
request_id
timestamp

Do not store secrets.

Audit logs must not be casually mutable/deletable.

Examples:

LOGIN_SUCCESS
LOGIN_FAILURE
LOGOUT
TOKEN_REFRESH
TOKEN_REUSE_DETECTED
PASSWORD_CHANGED
USER_CREATED
USER_ROLE_CHANGED
DEVICE_REGISTERED
DEVICE_REVOKED
ORDER_CREATED
ORDER_CANCELLED
PAYMENT_CREATED
REFUND_CREATED
INVENTORY_ADJUSTED

============================================================
28. SECURITY EVENT SYSTEM
============================================================

Separate security events from normal business audit events where useful.

Implement a security-event abstraction.

Examples:

AUTH_FAILURE
REFRESH_REUSE
SESSION_REVOKED
SUSPICIOUS_DEVICE
RATE_LIMIT_TRIGGERED
PRIVILEGE_ESCALATION_ATTEMPT
CROSS_TENANT_ACCESS_ATTEMPT

Security events must be searchable and observable.

============================================================
29. REDIS
============================================================

Redis is NOT the source of truth for critical business data.

Use Redis for:

rate limiting
short-lived session state
caching
distributed locks where justified
idempotency coordination where justified
job queues
temporary OAuth state
temporary login security state

If Redis disappears, PostgreSQL remains authoritative.

Design graceful degradation.

============================================================
30. CACHE
============================================================

Implement caching only where measurable.

Every cache entry must have:

key strategy
TTL
invalidation strategy

Never cache authorization decisions indefinitely.

Never cache sensitive user data without justification.

Prevent cache stampede where appropriate.

============================================================
31. BACKGROUND WORKERS
============================================================

Create worker process architecture.

Worker responsibilities can include:

outbox processing
notifications
audit processing
analytics events
cleanup
expired session cleanup
idempotency cleanup
sync processing

The API process must not perform long-running jobs synchronously.

Implement graceful shutdown.

============================================================
32. TRANSACTIONAL OUTBOX
============================================================

Implement the transactional outbox pattern for important domain events.

Example:

Business transaction:

BEGIN

insert order
insert order lines
insert outbox event

COMMIT

Worker publishes/processes outbox event.

This prevents:

database committed
event lost

Do not introduce distributed transactions unnecessarily.

============================================================
33. OBSERVABILITY
============================================================

Implement:

Prometheus metrics
structured JSON logs
request metrics
database metrics
Redis metrics
authentication metrics
rate-limit metrics
sync metrics
worker metrics

Metrics should include:

http_requests_total
http_request_duration_seconds
db_query_duration_seconds
auth_login_success_total
auth_login_failure_total
refresh_reuse_detected_total
sync_push_total
sync_conflict_total
worker_jobs_total

Avoid high-cardinality labels.

============================================================
34. HEALTH ENDPOINTS
============================================================

Implement:

GET /health/live

Must only indicate process liveness.

GET /health/ready

Must validate required dependencies.

Do not expose secrets or internal connection details.

============================================================
35. CONFIGURATION
============================================================

Use environment-based configuration.

Provide:

.env.example

Configuration must include concepts such as:

APP_ENV
APP_NAME
HTTP_ADDR

DATABASE_URL

DB_IDENTITY_URL
DB_CORE_URL
DB_ANALYTICS_URL
DB_AUDIT_URL

REDIS_URL

JWT_ISSUER
JWT_AUDIENCE
JWT_ACCESS_TTL
JWT_REFRESH_TTL

JWT_SIGNING_KEY

CORS_ALLOWED_ORIGINS

RATE_LIMIT settings

LOG_LEVEL

METRICS_ENABLED

Never commit secrets.

Validate configuration on startup.

Fail fast for invalid production configuration.

============================================================
36. SECRETS
============================================================

Never:

hard-code secrets
commit credentials
log secrets
return secrets from APIs

Use secret interfaces so production can later integrate:

Vault
AWS Secrets Manager
Azure Key Vault
GCP Secret Manager

without rewriting business logic.

============================================================
37. DATABASE SECURITY
============================================================

Use least privilege.

Application database user must not be a PostgreSQL superuser.

Separate migration privileges from runtime privileges where practical.

Use:

TLS
connection limits
statement timeout
idle transaction timeout

Prevent long-running accidental transactions.

============================================================
38. POSTGRESQL ROW LEVEL SECURITY
============================================================

Where RLS is used:

tenant context must be explicitly established per database transaction.

Never rely solely on application code.

Test RLS directly.

Ensure connection pooling does not leak tenant context between requests.

Use transaction-local settings where appropriate.

============================================================
39. SQL SAFETY
============================================================

NEVER construct SQL from untrusted strings.

Dynamic sorting/filtering must use allowlists.

Example:

allowedSortFields := map[string]string{
    "created_at": "created_at",
    "name":       "name",
}

Never:

ORDER BY + user input

Never interpolate:

table names
columns
WHERE clauses

unless explicitly allowlisted.

============================================================
40. ERROR HANDLING
============================================================

Create typed application errors.

Categories:

validation
authentication
authorization
not_found
conflict
rate_limit
dependency_failure
internal

Map them consistently to HTTP status codes.

Never panic on normal user input.

Recover panics at HTTP boundary and log them safely.

============================================================
41. LOGGING
============================================================

Use structured logging.

Every log should have useful context:

timestamp
level
service
request_id
trace_id if available
tenant_id where safe
user_id where safe
operation

Never log:

passwords
tokens
authorization headers
payment secrets
private keys
refresh tokens

============================================================
42. PAYMENT ARCHITECTURE
============================================================

Create payment abstraction.

Do not couple the domain directly to one payment provider.

Interface:

PaymentProvider

Implement provider adapter architecture.

Support future:

cash
card
wallet
external payment gateway

Payment state machine must be explicit.

Example:

pending
authorized
captured
failed
cancelled
refunded

Prevent duplicate payment creation using idempotency.

============================================================
43. CORE POS DOMAIN
============================================================

Build foundations for:

Tenant
Branch
Warehouse
POS Device
User
Role
Permission

Product
Product Category
Product Variant
Barcode
Price List

Customer

Inventory
Stock Location
Stock Movement

Order
Order Line
Payment
Payment Method

Tax

Discount

Return / Refund

Do not attempt to implement every advanced POS feature before the core
architecture is stable.

The architecture must allow future modules without rewriting the core.

============================================================
44. MONEY
============================================================

Never use floating-point numbers for money.

Use integer minor units or NUMERIC/decimal strategy consistently.

Currency must be explicit.

Example:

amount_minor BIGINT

currency CHAR(3)

Do not mix currency implicitly.

============================================================
45. TIME
============================================================

Store timestamps in UTC.

Use timezone-aware timestamps.

Tenant/branch may have a configured timezone.

Business dates must be calculated using the branch timezone,
not server local time.

============================================================
46. SOFT DELETE
============================================================

Do not blindly add deleted_at to every table.

Use soft deletion only where business/audit requirements justify it.

Transactional records such as orders/payments should generally be immutable
or state-transition based rather than deleted.

============================================================
47. API SECURITY
============================================================

Authentication middleware must execute before tenant authorization.

Expected pipeline:

Request
 -> Request ID
 -> Recovery
 -> Security headers
 -> Logging
 -> Rate limit
 -> Authentication
 -> Tenant context
 -> Authorization
 -> Handler
 -> Application service
 -> Repository
 -> PostgreSQL

Do not put business logic in middleware.

============================================================
48. API DOCUMENTATION
============================================================

Create OpenAPI documentation.

Every endpoint must document:

method
path
authentication
authorization
request
response
errors
pagination
idempotency
examples

Keep OpenAPI synchronized with implementation.

============================================================
49. TESTING
============================================================

Quality is mandatory.

Create:

unit tests
integration tests
repository tests
HTTP handler tests
authentication tests
authorization tests
security tests
migration tests
concurrency tests
sync tests
idempotency tests

Critical security tests:

1. User cannot access another tenant.
2. User cannot access another tenant by changing UUID.
3. Revoked session cannot refresh.
4. Refresh token reuse revokes family.
5. Expired access token rejected.
6. Wrong audience rejected.
7. Wrong issuer rejected.
8. Invalid signature rejected.
9. Missing permission rejected.
10. Rate limit enforced.
11. Duplicate idempotency request returns original result.
12. Same idempotency key with different payload returns conflict.
13. RLS prevents cross-tenant access.
14. Device revoked cannot authenticate.
15. Unauthorized admin endpoint rejected.
16. SQL injection attempts fail safely.

Use testcontainers where appropriate.

Do not mock PostgreSQL excessively.

Critical repository behavior must be tested against real PostgreSQL.

============================================================
50. FUZZ TESTING
============================================================

Use Go fuzz tests for:

authentication input
JWT parsing
UUID parsing
pagination
filter parsing
sync payloads
idempotency keys

The API must not crash from malformed external input.

============================================================
51. PERFORMANCE
============================================================

Design for high concurrency.

Use:

context.Context
connection pooling
bounded concurrency
timeouts
backpressure

Every external operation needs timeout behavior.

Avoid:

N+1 queries
unbounded goroutines
unbounded memory
huge JSON payloads
unbounded pagination

Set request body limits.

============================================================
52. GRACEFUL SHUTDOWN
============================================================

On SIGTERM:

1. stop accepting new requests
2. finish active requests within timeout
3. stop workers
4. finish safe jobs
5. close Redis
6. close PostgreSQL pools
7. flush logs
8. exit cleanly

Kubernetes/systemd-compatible behavior.

============================================================
53. DOCKER
============================================================

Create production-conscious Docker configuration.

Development:

docker-compose.yml

Services:

postgres
redis
api
worker

Use health checks.

Do not run application as root unnecessarily.

Use multi-stage Go builds.

Final image should be minimal.

============================================================
54. CADDY / PRODUCTION
============================================================

Provide production deployment configuration.

Caddy:

HTTPS
reverse proxy
security headers
timeouts
compression where appropriate

Application itself must still enforce authentication/security.

TLS termination at Caddy does not replace application security.

============================================================
55. MIGRATIONS
============================================================

Create complete Goose migrations.

Migration sequence should establish:

extensions
tenants
users
roles
permissions
memberships
sessions
refresh token families
devices
products
categories
customers
inventory
orders
order lines
payments
audit logs
security events
outbox
idempotency
sync metadata

Add indexes deliberately.

Analyze query patterns.

============================================================
56. API CONTRACT
============================================================

All API endpoints must have:

authentication rules
authorization rules
tenant scope
validation
error contract
transaction behavior
idempotency behavior where required

Document them.

============================================================
57. DOMAIN EVENTS
============================================================

Use typed domain events.

Examples:

UserRegistered
UserLoggedIn
SessionCreated
SessionRevoked
RefreshTokenRotated
RefreshTokenReuseDetected

ProductCreated
InventoryAdjusted
OrderCreated
OrderCancelled
PaymentCreated
PaymentRefunded

Events should be versioned.

============================================================
58. SECURITY REVIEW
============================================================

Before declaring completion, perform a security review.

Check:

OWASP API Security Top 10
OWASP ASVS concepts
authentication
authorization
session management
tenant isolation
SQL injection
XSS where relevant
CSRF where relevant
SSRF where relevant
request smuggling considerations
rate limiting
secrets
logging
cryptography
dependency vulnerabilities
race conditions
data leakage

Create:

docs/SECURITY_REVIEW.md

Document findings and fixes.

============================================================
59. STATIC ANALYSIS
============================================================

Run:

gofmt
go vet
staticcheck if available
go test ./...
go test -race ./...

Also run security tooling where available.

Examples:

govulncheck

Do not ignore failures.

Fix root causes.

============================================================
60. CODE QUALITY
============================================================

Follow idiomatic Go.

Prefer:

small interfaces
explicit dependencies
context propagation
typed errors
dependency injection
constructor functions
clear naming
small functions

Avoid:

global mutable state
god objects
giant services
giant handlers
magic strings
hidden dependencies
unnecessary reflection
premature abstractions

============================================================
61. DATABASE QUERY QUALITY
============================================================

Every important query should be reviewed for:

indexes
tenant filtering
locking
transaction boundaries
pagination
query plan

Use EXPLAIN ANALYZE where appropriate.

Prevent accidental full-table scans.

============================================================
62. CONCURRENCY
============================================================

POS systems have concurrent operations.

Explicitly handle:

two terminals selling same inventory
duplicate payment requests
duplicate order requests
simultaneous updates
session refresh races
refresh token rotation races
sync conflicts

Use PostgreSQL locking/versioning appropriately.

Do not solve database consistency using Redis alone.

============================================================
63. INVENTORY CONSISTENCY
============================================================

Inventory must be transactionally safe.

Stock movements should be the authoritative ledger.

Do not simply mutate quantity without recording movement.

Support:

sale
return
purchase
adjustment
transfer
reservation where required

Avoid negative stock according to configurable tenant policy.

============================================================
64. ORDER STATE MACHINE
============================================================

Do not allow arbitrary status transitions.

Define explicit transitions.

For example:

draft
pending
confirmed
paid
completed
cancelled
refunded

Validate transitions in domain/application layer.

============================================================
65. TENANT PLAN / BILLING FOUNDATION
============================================================

The SaaS core should support:

tenant plan
subscription status
limits
feature flags

Examples:

max_users
max_devices
max_branches
max_products

Enforce limits server-side.

Do not hard-code plan logic into individual handlers.

Create a feature/entitlement abstraction.

============================================================
66. PLATFORM ADMIN
============================================================

Create separate platform-level authorization.

Platform administrators must be distinguishable from tenant users.

A tenant administrator must never automatically become platform administrator.

Protect platform endpoints separately.

Example:

/api/v1/platform/tenants
/api/v1/platform/users
/api/v1/platform/audit

============================================================
67. FEATURE FLAGS
============================================================

Implement feature flags at tenant level.

Example:

inventory_v2
advanced_reports
offline_sync
multi_warehouse

Feature flags must be server-controlled.

Do not trust client-side feature flags for authorization.

============================================================
68. FILE STORAGE
============================================================

If product/customer images are introduced:

Do not store large binary files directly in PostgreSQL by default.

Create storage abstraction:

ObjectStorage

Future providers:

S3
MinIO
Azure Blob

Use signed URLs.

Validate uploaded file type and size.

Never trust filename extensions.

============================================================
69. NOTIFICATIONS
============================================================

Create abstraction:

NotificationService

Future:

email
push
SMS
webhooks

Do not tightly couple the domain to Firebase or one provider.

============================================================
70. WEBHOOKS
============================================================

Create secure webhook architecture for external integrations.

Support:

signature validation
timestamp validation
replay protection
idempotency
delivery retries
dead-letter behavior

============================================================
71. API CLIENT COMPATIBILITY
============================================================

Flutter clients may be offline or outdated.

API should support:

versioning
backward compatibility
capability negotiation where appropriate

Never make breaking API changes silently.

============================================================
72. DATA PRIVACY
============================================================

Implement data minimization.

Sensitive fields must be protected.

Provide architecture for:

user data export
account deletion/anonymization where legally appropriate
audit retention
data retention policies

Do not expose unnecessary personal data in APIs.

============================================================
73. IMPLEMENTATION WORKFLOW
============================================================

DO NOT immediately write thousands of lines of code.

Follow this process:

PHASE 1
Inspect repository.

PHASE 2
Read all architecture/spec/security/database documents.

PHASE 3
Build dependency graph.

PHASE 4
Identify incomplete/missing components.

PHASE 5
Create implementation plan.

PHASE 6
Implement foundational infrastructure.

PHASE 7
Implement database.

PHASE 8
Implement authentication/session security.

PHASE 9
Implement authorization/tenant isolation.

PHASE 10
Implement device management.

PHASE 11
Implement idempotency.

PHASE 12
Implement sync architecture.

PHASE 13
Implement POS core domain.

PHASE 14
Implement workers/outbox.

PHASE 15
Implement observability.

PHASE 16
Implement deployment.

PHASE 17
Implement comprehensive tests.

PHASE 18
Run security review.

PHASE 19
Run complete test/build/static analysis suite.

============================================================
74. DO NOT ASK FOR PERMISSION FOR NORMAL IMPLEMENTATION
============================================================

If a reasonable engineering decision is required:

make the decision.

Document the decision.

Do not stop unnecessarily asking questions.

If an architectural decision materially conflicts with an existing project
specification, inspect the specification and follow the repository's source
of truth.

============================================================
75. NO FAKE IMPLEMENTATION
============================================================

Forbidden:

TODO for core security
TODO for authentication
TODO for authorization
fake repositories
in-memory production persistence
hard-coded users
hard-coded JWT secrets
mock authentication
fake payment success
fake tenant isolation
placeholder security checks

Tests must test real behavior.

============================================================
76. QUALITY GATE
============================================================

Before completion, run all applicable:

go test ./...

go test -race ./...

go vet ./...

gofmt -w .

staticcheck ./...

govulncheck ./...

docker compose config

database migrations from empty database

database migrations against existing development database

API smoke tests

security tests

============================================================
77. FINAL DELIVERABLE
============================================================

The repository must contain:

complete Go source code
complete migrations
complete tests
OpenAPI specification
Docker configuration
environment example
production configuration
worker
database layer
Redis layer
authentication
authorization
sessions
devices
tenancy
audit
security events
idempotency
sync foundation
POS domain foundation
observability
documentation

Also update:

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

only where implementation changes require it.

============================================================
78. FINAL REPORT
============================================================

At the end provide:

1. Architecture summary
2. Directory structure
3. Database architecture
4. Authentication architecture
5. Authorization architecture
6. Tenant isolation strategy
7. Session strategy
8. Refresh-token rotation strategy
9. Redis responsibilities
10. PostgreSQL responsibilities
11. Multi-database strategy
12. POS domain implemented
13. Sync architecture
14. Security controls
15. API endpoints
16. Tests implemented
17. Performance considerations
18. Deployment instructions
19. Remaining limitations

Do not claim something is implemented unless it actually exists in code.

============================================================
79. ABSOLUTE ENGINEERING RULE
============================================================

You are not generating a demo.

You are building the CORE BACKEND of a commercial SaaS POS platform.

Prioritize:

CORRECTNESS
SECURITY
TENANT ISOLATION
DATA CONSISTENCY
OBSERVABILITY
TESTABILITY
MAINTAINABILITY
PERFORMANCE
BACKWARD COMPATIBILITY

over:

short code
quick hacks
premature feature breadth

Every security boundary must be explicit.

Every database mutation must have a clear transaction boundary.

Every authenticated request must have a clear identity.

Every tenant-owned operation must have a clear tenant boundary.

Every critical POS mutation must be retry-safe.

Every important asynchronous event must be recoverable.

Every production failure must be diagnosable.

Implement the system accordingly.