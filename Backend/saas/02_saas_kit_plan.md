# SaaS POS Kit — Master Implementation Plan

## Phase 0 — Repository Foundation

Create a monorepo:

    saas-pos/
        backend/
        flutter_pos/
        web_admin/
        infrastructure/
        docs/

Backend:

    Go

Flutter:

    Dart / Flutter

Web Admin:

    React / TypeScript

Database:

    PostgreSQL

Infrastructure:

    Docker Compose initially
    Kubernetes-ready later

---

# Phase 1 — Control Plane

## Goal

Create the SaaS management core.

Implement:

- SaaS admin authentication
- Customer
- Tenant
- Plan
- Subscription
- Tenant state
- Service state
- Audit log

Database:

    saas_control

Tables:

    users
    tenants
    plans
    plan_features
    plan_limits
    subscriptions
    audit_logs

---

# Phase 2 — PostgreSQL Tenant Database Manager

## Goal

Allow the Go backend to create and manage isolated PostgreSQL databases.

Implement:

    CreateDatabase
    DropDatabase
    DatabaseExists
    DatabaseSize
    DatabaseHealth
    RunMigrations
    GetConnectionInfo

Tenant database naming:

    tenant_<numeric_id>

Example:

    tenant_10001

Never expose database credentials to Flutter.

---

# Phase 3 — Tenant Provisioning Engine

## Goal

Automatically create a complete tenant environment.

Flow:

    SaaS Admin
       ↓
    Create Tenant
       ↓
    Assign Plan
       ↓
    Provision Job
       ↓
    Create PostgreSQL DB
       ↓
    Run migrations
       ↓
    Seed data
       ↓
    Create owner
       ↓
    Activate tenant

Provisioning must be resumable.

---

# Phase 4 — Tenant POS API

## Goal

Implement the business backend.

Modules:

- Products
- Categories
- Customers
- Suppliers
- Sales
- Sale lines
- Purchases
- Purchase lines
- Inventory
- Stock movements
- Payments
- POS sessions
- Cash sessions
- Users
- Roles

All APIs must operate against the authenticated tenant database.

---

# Phase 5 — Authentication and Authorization

Implement:

- Login
- Refresh token
- Logout
- Session revocation
- Password reset
- User activation
- User suspension
- Role permissions
- Tenant isolation

Roles:

    OWNER
    MANAGER
    CASHIER

Permission system:

    resource.action

Examples:

    sales.create
    sales.read
    products.create
    inventory.adjust

---

# Phase 6 — Plan Engine

## Goal

Make pricing plans enforceable by the backend.

Plan contains:

    price
    billing_period
    features
    limits

Limits:

    users
    managers
    cashiers
    branches
    warehouses
    products
    monthly_transactions
    storage
    backup_retention_days

Backend middleware:

    RequireFeature()
    RequireLimit()
    RequireUsage()

Example:

    RequireFeature("inventory.advanced")

---

# Phase 7 — Subscription Engine

Implement:

    trial
    active
    grace_period
    past_due
    suspended
    cancelled

Subscription events:

    created
    renewed
    upgraded
    downgraded
    expired
    suspended
    resumed
    cancelled

Subscription state must affect service access.

---

# Phase 8 — User Invitation System

Tenant owner/manager can invite users.

Invitation:

    email
    tenant
    role
    expires_at
    status

Statuses:

    pending
    accepted
    expired
    revoked

Example plan:

    Manager: 1
    Cashiers: 4

If cashiers = 4:

    fifth cashier → rejected

Backend response:

    USER_LIMIT_REACHED

---

# Phase 9 — PostgreSQL Backup Engine

## Goal

Create reliable tenant backups.

Backup:

    pg_dump

Preferred format:

    custom format

Example:

    tenant_10001_20260911.dump

Process:

    create job
    ↓
    lock/coordinate tenant state
    ↓
    pg_dump
    ↓
    checksum
    ↓
    upload storage
    ↓
    verify
    ↓
    metadata
    ↓
    completed

---

# Phase 10 — Backup Storage

Support abstraction:

    BackupStorage

Implementations:

    LocalStorage
    S3Storage
    MinioStorage

Interface:

    Put()
    Get()
    Delete()
    Exists()
    GenerateDownloadURL()

The application must not depend directly on S3.

---

# Phase 11 — Automated Backups

Plan configuration:

    backup_enabled
    backup_frequency
    retention_days

Examples:

Starter:

    daily
    7 days

Business:

    daily
    30 days

Enterprise:

    daily
    90 days

The scheduler creates backup jobs.

---

# Phase 12 — Restore Engine

Restore must be a job.

Flow:

    select backup
        ↓
    validate checksum
        ↓
    stop tenant
        ↓
    create temporary DB
        ↓
    pg_restore
        ↓
    migrate
        ↓
    health check
        ↓
    switch database
        ↓
    activate tenant

Keep the old database temporarily where possible.

---

# Phase 13 — Service Control

Implement API:

    POST /admin/tenants/{id}/start
    POST /admin/tenants/{id}/pause
    POST /admin/tenants/{id}/stop
    POST /admin/tenants/{id}/suspend
    POST /admin/tenants/{id}/resume

Rules:

START:

    service = STARTED

PAUSE:

    database preserved
    API blocked

STOP:

    API blocked
    database preserved

SUSPEND:

    API blocked
    subscription inactive

---

# Phase 14 — Web SaaS Control Panel

Create:

    Dashboard
    Tenants
    Tenant Details
    Plans
    Subscriptions
    Users
    Backups
    Restore Jobs
    Provisioning Jobs
    Audit Logs
    System Health

Tenant detail page:

    Customer
    Plan
    Subscription
    Database
    Database size
    Service state
    Last backup
    Backup status
    Users
    Usage
    Features

Actions:

    Start
    Pause
    Stop
    Backup
    Restore
    Suspend
    Resume

---

# Phase 15 — Flutter POS

Implement:

    Authentication
    Tenant configuration
    Products
    Customers
    POS
    Cart
    Payment
    Sales
    Purchase
    Inventory
    Users
    Settings
    Offline mode
    Synchronization

Flutter receives:

    plan
    features
    limits
    usage

The Flutter application dynamically enables features.

---

# Phase 16 — Flutter Offline Synchronization

Implement:

    local database

Recommended:

    SQLite

Offline queue:

    sync_operations

Fields:

    id
    operation
    entity
    local_id
    payload
    created_at
    retry_count
    status

States:

    pending
    syncing
    synced
    failed
    conflict

---

# Phase 17 — POS Transaction Integrity

Every sale must use:

    transaction

Example:

    BEGIN

    create sale
    create sale lines
    create payment
    create stock movement
    update stock

    COMMIT

Any failure:

    ROLLBACK

Never create a partially completed sale.

---

# Phase 18 — Pricing and Usage

Track:

    current_users
    current_managers
    current_cashiers
    product_count
    monthly_sales
    storage_usage
    backup_storage

Expose:

    /api/v1/tenant/usage

Flutter can display:

    Users 3 / 5
    Cashiers 2 / 4
    Storage 1.2 GB / 5 GB

---

# Phase 19 — Upgrade/Downgrade

Upgrade:

    immediately increase capabilities

Downgrade:

    check current usage

Example:

Plan limit:

    5 users

Current:

    7 users

Downgrade must not silently disable users.

Return:

    PLAN_DOWNGRADE_BLOCKED

with required remediation.

---

# Phase 20 — Billing Integration

Billing abstraction:

    BillingProvider

Possible providers:

    Stripe
    Paymob
    Fawry
    Paddle
    manual invoice

Do not couple SaaS core directly to a payment provider.

---

# Phase 21 — Monitoring

Implement:

    /health
    /ready
    /metrics

Monitor:

    API latency
    PostgreSQL connections
    tenant count
    active jobs
    failed jobs
    backup failures
    restore failures
    storage usage

---

# Phase 22 — Security Audit

Test:

- tenant escape
- IDOR
- JWT abuse
- refresh-token reuse
- role escalation
- SQL injection
- mass assignment
- API rate limits
- backup access
- restore authorization
- admin authorization
- expired subscription bypass
- plan-limit bypass

---

# Phase 23 — Production Infrastructure

Docker services:

    postgres
    redis
    backend
    worker
    web_admin
    minio

Production:

    reverse proxy
    TLS
    monitoring
    backup storage
    secrets management

---

# Phase 24 — Disaster Recovery

Document:

    RPO
    RTO

Implement:

    database backup
    backup verification
    offsite storage
    restore testing
    disaster recovery procedure

A backup that has never been restored/tested must not be considered fully trusted.

---

# Phase 25 — Production Release

Before release:

- integration tests
- security tests
- load tests
- migration tests
- backup/restore tests
- Flutter offline tests
- subscription tests
- tenant isolation tests
- disaster recovery test

Release only after all critical tests pass.
