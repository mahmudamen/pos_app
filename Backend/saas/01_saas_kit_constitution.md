# SaaS POS Platform Constitution

## 1. Project Identity

Name:
SaaS POS Kit

Architecture:
- Go backend
- PostgreSQL
- Flutter POS
- Web SaaS Control Panel
- REST/JSON API
- JWT + Refresh Token authentication
- PostgreSQL backup/restore infrastructure
- Multi-tenant SaaS

Primary Goal:

Build a production-grade SaaS platform that allows an administrator to create,
configure, activate, pause, stop, backup, restore and manage independent POS
client environments.

Each SaaS customer must have isolated business data and must only access
resources belonging to their tenant.

---

# 2. Core Architecture

The platform contains two major planes.

## Control Plane

Responsible for SaaS management.

Responsibilities:

- Tenants
- Customers
- Subscriptions
- Pricing plans
- Feature flags
- User limits
- Role limits
- Tenant lifecycle
- Database provisioning
- Database status
- Backups
- Restore jobs
- Billing state
- Service suspension
- Service activation
- Admin users
- Audit logs
- System monitoring

The control plane owns the SaaS metadata.

It must NOT expose arbitrary customer business data.

---

## Data Plane

Each customer POS environment contains:

- Products
- Categories
- Customers
- Suppliers
- Sales
- Sale lines
- Purchases
- Purchase lines
- Payments
- Cash sessions
- Inventory
- Stock movements
- Users
- Roles
- POS configuration
- Settings
- Audit information

The data plane belongs to one tenant.

---

# 3. Database Isolation

Database isolation is mandatory.

Recommended production architecture:

Control Database:

    saas_control

Tenant Databases:

    tenant_000001
    tenant_000002
    tenant_000003

The control database stores:

- tenant ID
- database name
- database host
- database status
- plan ID
- subscription status
- service state
- backup metadata
- provisioning status

Tenant databases store business data.

Never use a single unrestricted shared database connection for all tenants.

---

# 4. Tenant Isolation

Every request must resolve:

    authenticated_user
        ↓
    tenant
        ↓
    tenant_database
        ↓
    tenant_resource

A user must never be able to select another tenant by modifying:

    tenant_id
    database_id
    customer_id
    user_id

Tenant ownership must be derived from authentication/session context.

---

# 5. Tenant Lifecycle

Tenant states:

    pending
    provisioning
    active
    paused
    stopped
    suspended
    deleting
    deleted
    failed

State transitions must be validated.

Example:

    pending
        ↓
    provisioning
        ↓
    active

Active:

    active → paused
    active → stopped
    active → suspended

Paused:

    paused → active
    paused → stopped

Stopped:

    stopped → active

Suspended:

    suspended → active

Invalid transitions must return an API error.

---

# 6. Service States

The SaaS service state is different from the database state.

Service:

    STARTED
    PAUSED
    STOPPED

Database:

    CREATING
    READY
    BACKUP
    RESTORING
    ERROR

A tenant may have:

    service = PAUSED
    database = READY

The database remains intact while API access is blocked.

---

# 7. Pricing Plans

Plans define platform capabilities.

Example:

## Starter

Users:

    2

Roles:

    1 manager
    1 cashier

Features:

- POS
- Products
- Inventory
- Sales
- Basic reports
- Backup

---

## Business

Users:

    5

Roles:

    1 manager
    4 cashiers

Features:

- POS
- Products
- Inventory
- Purchasing
- Sales
- Reports
- Multiple POS sessions
- Backup
- Restore
- Advanced permissions

---

## Enterprise

Users:

    configurable

Features:

- Multiple branches
- Advanced inventory
- Advanced reports
- API
- Integrations
- Multiple managers
- Multiple warehouses
- Advanced backup retention

Plans must be configurable by the SaaS administrator.

Never hard-code plan limits into Flutter.

---

# 8. Feature Enforcement

Features must be represented as capabilities.

Example:

    pos.sales
    pos.purchase
    inventory.basic
    inventory.advanced
    reports.basic
    reports.advanced
    backup.create
    backup.restore
    multi_branch
    api.access
    user_management
    advanced_roles

Backend is the final authority.

Flutter only displays the capabilities.

A hidden Flutter button is NOT security.

---

# 9. User Model

Each tenant can have users.

Minimum supported roles:

    OWNER
    MANAGER
    CASHIER

Default plan:

    1 OWNER/MANAGER
    N CASHIERS

The owner/manager can create cashiers only within plan limits.

The SaaS administrator can override limits.

---

# 10. Role Security

Permissions must be explicit.

Example:

MANAGER:

- products.read
- products.create
- products.update
- inventory.read
- inventory.adjust
- sales.read
- sales.create
- purchases.read
- purchases.create
- reports.read
- users.read
- users.create
- users.disable

CASHIER:

- products.read
- sales.read
- sales.create
- payments.create
- pos.open
- pos.close

Cashier must not:

- modify plan
- manage subscription
- create managers
- delete products
- restore database
- access SaaS administration

---

# 11. No Client-Controlled Authorization

Never trust:

    role
    plan
    feature
    tenant_id
    permissions

sent by Flutter.

The server determines:

    tenant
    role
    permissions
    plan
    limits
    subscription state

from trusted backend state.

---

# 12. Authentication

Authentication must support:

- Login
- Logout
- Access token
- Refresh token
- Token rotation
- Session management
- Password hashing
- Account lockout
- Device/session tracking
- Revocation

Passwords must never be stored in plaintext.

Recommended password hashing:

    Argon2id

---

# 13. API Security

All production API traffic must use HTTPS.

Security requirements:

- TLS
- Secure headers
- Rate limiting
- Request validation
- SQL parameterization
- Audit logging
- Authentication middleware
- Authorization middleware
- Tenant middleware
- Request IDs
- Structured logs

---

# 14. Backup Constitution

Every tenant database must support:

    manual backup
    scheduled backup
    retention policy
    backup verification
    backup metadata
    restore operation

Backups must be PostgreSQL-native.

Preferred:

    pg_dump
    pg_restore

Backup files must not be stored permanently inside the application container.

Recommended storage:

    S3-compatible object storage

Examples:

    MinIO
    S3
    Cloudflare R2
    Backblaze B2

---

# 15. Backup Metadata

Control database stores:

    backup_id
    tenant_id
    database_name
    storage_key
    size
    checksum
    created_at
    completed_at
    status
    retention_until
    initiated_by

Possible states:

    QUEUED
    RUNNING
    COMPLETED
    FAILED
    DELETED

---

# 16. Backup Security

Backup files are sensitive.

Requirements:

- Private storage
- Encryption at rest
- HTTPS
- Access authorization
- Short-lived download URLs
- Checksum validation
- Retention policy
- Audit logging

Never expose the physical storage path to clients.

---

# 17. Restore

Restore must be an asynchronous operation.

Flow:

    ADMIN
      ↓
    select tenant
      ↓
    select backup
      ↓
    validate backup
      ↓
    create restore job
      ↓
    stop tenant service
      ↓
    restore PostgreSQL database
      ↓
    validate database
      ↓
    mark database READY
      ↓
    start service
      ↓
    audit event

Restore must never overwrite a production database without explicit authorization.

Recommended:

    restore to temporary database
    validate
    swap/replace

---

# 18. Tenant Provisioning

Creating a tenant must be transactional from the control-plane perspective.

Flow:

    create customer
        ↓
    create tenant
        ↓
    assign plan
        ↓
    create database job
        ↓
    create PostgreSQL database
        ↓
    run migrations
        ↓
    seed initial data
        ↓
    create owner
        ↓
    mark tenant READY
        ↓
    activate service

Provisioning failures must be recoverable.

---

# 19. Background Jobs

Long-running operations must NOT run inside HTTP handlers.

Jobs include:

- Database creation
- Database migration
- Backup
- Restore
- Backup verification
- Backup deletion
- Tenant suspension
- Tenant activation
- Scheduled maintenance

Use a persistent job queue.

Recommended:

    Redis + worker

or:

    PostgreSQL-backed job queue

Every job must have:

    job_id
    type
    tenant_id
    status
    progress
    error
    created_at
    started_at
    finished_at

---

# 20. Idempotency

Critical operations must be idempotent.

Examples:

    create tenant
    provision database
    create backup
    restore backup
    activate tenant
    pause tenant

Repeated requests must not create duplicated resources.

---

# 21. Audit

Every administrative action must generate an audit event.

Example:

    actor
    action
    tenant
    resource
    resource_id
    old_value
    new_value
    IP
    user_agent
    timestamp

Audit records must be append-only.

---

# 22. Delete Policy

Business records must not be physically deleted unless explicitly allowed.

Prefer:

    active = false
    archived_at
    deleted_at

Administrative destructive actions require elevated permission.

Database deletion requires explicit confirmation.

---

# 23. Flutter Principle

Flutter is a client.

Flutter must:

- authenticate
- cache configuration
- display features
- enforce UX restrictions
- queue offline operations
- synchronize data
- handle service status

Flutter must NOT become the source of truth for:

- prices
- permissions
- stock
- plans
- user limits
- accounting values

The backend is authoritative.

---

# 24. Offline POS

The POS must support controlled offline operation.

Offline data:

- Products
- Customers
- Prices
- Tax configuration
- POS configuration
- User permissions
- Pending sales

Offline transactions receive:

    local_transaction_id

Synchronization creates:

    server_transaction_id

Conflict resolution must be deterministic.

---

# 25. Pricing Enforcement

The backend must expose:

    plan
    features
    limits
    usage

Example:

GET /api/v1/me/capabilities

Response:

    {
      "plan": "business",
      "features": [...],
      "limits": {
        "users": 5,
        "cashiers": 4,
        "managers": 1
      },
      "usage": {
        "users": 3,
        "cashiers": 2
      }
    }

Flutter uses this response to configure the UI.

---

# 26. Service Pause

Pause means:

- Preserve database
- Preserve backups
- Block normal tenant API
- Prevent POS transactions
- Keep control-plane access available

The tenant owner can see:

    Service paused

---

# 27. Service Stop

Stop means:

- Tenant API disabled
- POS unavailable
- Database preserved
- Scheduled jobs stopped where appropriate
- Backup retention remains active

Stopping must not delete tenant data.

---

# 28. Subscription Suspension

When subscription expires:

    active
      ↓
    grace_period
      ↓
    suspended

Suspended tenant:

- Cannot create sales
- Cannot create purchases
- Cannot synchronize POS
- Cannot create users

Owner can still access:

- subscription page
- billing page
- support
- restore information
- account settings

---

# 29. Control Plane Web Application

The web application must provide:

Dashboard

    tenants
    active tenants
    paused tenants
    suspended tenants
    database health
    backup health
    job health

Tenant Management

    create
    provision
    activate
    pause
    stop
    suspend
    restore
    backup
    delete

Plan Management

    create plan
    edit plan
    enable feature
    disable feature
    configure limits
    configure pricing

User Management

    create
    disable
    reset password
    assign role
    revoke session

Backup Management

    list
    create
    download
    verify
    restore
    delete according to retention rules

---

# 30. Observability

System must expose:

- health endpoint
- readiness endpoint
- metrics
- structured logs
- job monitoring
- database connection health
- backup health

Recommended:

    Prometheus
    Grafana

---

# 31. API Versioning

Use:

    /api/v1/

Future:

    /api/v2/

Never break existing mobile clients without versioning.

---

# 32. Architecture Rules

Go:

    clean architecture

Suggested:

    cmd/
    internal/
        auth/
        tenant/
        subscription/
        plan/
        user/
        role/
        database/
        backup/
        restore/
        jobs/
        audit/
        billing/
        api/
    migrations/
    pkg/

Flutter:

    feature-first architecture

Suggested:

    lib/
      core/
      auth/
      tenant/
      pos/
      products/
      inventory/
      sales/
      purchases/
      users/
      settings/
      sync/

---

# 33. Quality Requirements

Code must prioritize:

- correctness
- security
- testability
- maintainability
- observability
- deterministic behavior
- explicit error handling
- database transaction integrity

No giant files.

No god services.

No hidden global state.

No business logic inside HTTP handlers.

No SQL concatenation.

No authorization logic in Flutter only.

---

# 34. Definition of Done

A feature is complete only when:

- backend implementation exists
- database migration exists
- API exists
- authorization exists
- tenant isolation is tested
- unit tests exist
- integration tests exist
- Flutter integration exists where required
- error states are handled
- audit events exist where appropriate
- documentation exists
- logs are useful
- metrics exist for operationally important jobs
