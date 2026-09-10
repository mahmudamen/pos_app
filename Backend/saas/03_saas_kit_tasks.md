# SaaS POS Kit — Engineering Tasks

Status:

    [ ] TODO
    [~] IN PROGRESS
    [x] DONE

---

# Phase 0 — Repository

- [ ] Create monorepo
- [ ] Create backend directory
- [ ] Create Flutter directory
- [ ] Create web admin directory
- [ ] Create infrastructure directory
- [ ] Create docs directory
- [ ] Configure Git
- [ ] Configure CI
- [ ] Configure formatting
- [ ] Configure linting

---

# Phase 1 — Go Backend

## Foundation

- [ ] Initialize Go module
- [ ] HTTP server
- [ ] Configuration system
- [ ] Environment loader
- [ ] PostgreSQL connection
- [ ] Redis connection
- [ ] Structured logger
- [ ] Request ID middleware
- [ ] Recovery middleware
- [ ] CORS configuration
- [ ] Rate limiting

## API

- [ ] API router
- [ ] /api/v1
- [ ] Error response format
- [ ] Validation
- [ ] Pagination
- [ ] Filtering
- [ ] Sorting

---

# Phase 2 — Control Database

- [ ] Create saas_control database
- [ ] Migration framework
- [ ] tenants table
- [ ] plans table
- [ ] plan_features table
- [ ] plan_limits table
- [ ] subscriptions table
- [ ] users table
- [ ] sessions table
- [ ] audit_logs table
- [ ] jobs table
- [ ] backups table

---

# Phase 3 — SaaS Authentication

- [ ] Admin login
- [ ] Tenant login
- [ ] Argon2id password hashing
- [ ] JWT access token
- [ ] Refresh token
- [ ] Refresh token rotation
- [ ] Session revocation
- [ ] Logout
- [ ] Password reset
- [ ] Account suspension

---

# Phase 4 — Tenant Manager

- [ ] Tenant repository
- [ ] Tenant service
- [ ] Tenant API
- [ ] Tenant state machine
- [ ] Service state machine
- [ ] Tenant creation
- [ ] Tenant activation
- [ ] Tenant pause
- [ ] Tenant stop
- [ ] Tenant suspend
- [ ] Tenant resume

---

# Phase 5 — PostgreSQL Database Manager

- [ ] PostgreSQL admin connection
- [ ] Create database
- [ ] Database existence check
- [ ] Database size
- [ ] Database health
- [ ] Database connection test
- [ ] Database migration runner
- [ ] Database template
- [ ] Database deletion
- [ ] Safe database lifecycle

---

# Phase 6 — Provisioning

- [ ] Provisioning job
- [ ] Create tenant DB
- [ ] Run migrations
- [ ] Seed roles
- [ ] Seed permissions
- [ ] Seed POS configuration
- [ ] Create owner
- [ ] Validate database
- [ ] Mark tenant active
- [ ] Failure recovery
- [ ] Retry support
- [ ] Idempotency

---

# Phase 7 — Tenant POS Database

Create migrations for:

- [ ] users
- [ ] roles
- [ ] permissions
- [ ] products
- [ ] categories
- [ ] units
- [ ] product prices
- [ ] customers
- [ ] suppliers
- [ ] sales
- [ ] sale_lines
- [ ] purchases
- [ ] purchase_lines
- [ ] payments
- [ ] cash_sessions
- [ ] inventory
- [ ] stock_movements
- [ ] warehouses
- [ ] audit_logs

---

# Phase 8 — RBAC

- [ ] OWNER role
- [ ] MANAGER role
- [ ] CASHIER role
- [ ] Permission table
- [ ] Role permission table
- [ ] User role assignment
- [ ] Permission middleware
- [ ] Tenant authorization middleware

---

# Phase 9 — Plan Engine

- [ ] Plan CRUD
- [ ] Feature CRUD
- [ ] Limit CRUD
- [ ] Plan-feature mapping
- [ ] Plan-limit mapping
- [ ] Feature middleware
- [ ] Limit middleware
- [ ] Usage calculation
- [ ] Usage API

---

# Phase 10 — User Limits

Implement:

- [ ] Maximum users
- [ ] Maximum managers
- [ ] Maximum cashiers
- [ ] User creation validation
- [ ] Manager creation validation
- [ ] Cashier creation validation
- [ ] User deactivation
- [ ] Usage recalculation

Error codes:

    USER_LIMIT_REACHED
    MANAGER_LIMIT_REACHED
    CASHIER_LIMIT_REACHED

---

# Phase 11 — Invitations

- [ ] Invitation model
- [ ] Invitation token
- [ ] Expiration
- [ ] Role assignment
- [ ] Accept invitation
- [ ] Revoke invitation
- [ ] Resend invitation
- [ ] Invitation audit

---

# Phase 12 — Backup

- [ ] Backup service
- [ ] pg_dump wrapper
- [ ] Temporary backup file
- [ ] Checksum
- [ ] Backup metadata
- [ ] Backup job
- [ ] Backup progress
- [ ] Backup failure handling
- [ ] Backup cleanup

---

# Phase 13 — Backup Storage

- [ ] Storage interface
- [ ] Local implementation
- [ ] MinIO implementation
- [ ] S3 implementation
- [ ] Object encryption
- [ ] Private bucket
- [ ] Signed download URL
- [ ] Storage deletion

---

# Phase 14 — Backup Scheduler

- [ ] Plan backup policy
- [ ] Daily backup
- [ ] Weekly backup
- [ ] Retention
- [ ] Automatic cleanup
- [ ] Failed backup alert
- [ ] Backup health status

---

# Phase 15 — Restore

- [ ] Restore job
- [ ] Backup validation
- [ ] Checksum validation
- [ ] Temporary database
- [ ] pg_restore
- [ ] Migration validation
- [ ] Database health check
- [ ] Database switch
- [ ] Service restart
- [ ] Restore audit
- [ ] Restore rollback

---

# Phase 16 — Job System

Jobs:

- [ ] ProvisionTenant
- [ ] BackupTenant
- [ ] RestoreTenant
- [ ] DeleteBackup
- [ ] SuspendTenant
- [ ] ResumeTenant
- [ ] CleanupTenant

Job fields:

- [ ] ID
- [ ] type
- [ ] tenant_id
- [ ] status
- [ ] progress
- [ ] attempts
- [ ] error
- [ ] timestamps

---

# Phase 17 — SaaS Admin APIs

Implement:

    GET    /api/v1/admin/tenants
    POST   /api/v1/admin/tenants
    GET    /api/v1/admin/tenants/:id
    POST   /api/v1/admin/tenants/:id/start
    POST   /api/v1/admin/tenants/:id/pause
    POST   /api/v1/admin/tenants/:id/stop
    POST   /api/v1/admin/tenants/:id/suspend
    POST   /api/v1/admin/tenants/:id/resume
    POST   /api/v1/admin/tenants/:id/backup
    POST   /api/v1/admin/tenants/:id/restore
    DELETE /api/v1/admin/tenants/:id

---

# Phase 18 — Plan APIs

    GET    /api/v1/admin/plans
    POST   /api/v1/admin/plans
    PUT    /api/v1/admin/plans/:id
    DELETE /api/v1/admin/plans/:id

Features:

    GET
    POST
    PUT
    DELETE

Limits:

    GET
    POST
    PUT
    DELETE

---

# Phase 19 — Tenant APIs

    GET /api/v1/me
    GET /api/v1/me/plan
    GET /api/v1/me/capabilities
    GET /api/v1/me/usage
    GET /api/v1/me/service-status

---

# Phase 20 — Flutter Foundation

- [ ] Flutter project
- [ ] Architecture
- [ ] Dependency injection
- [ ] API client
- [ ] Secure token storage
- [ ] Routing
- [ ] Error handling
- [ ] Localization
- [ ] Arabic RTL
- [ ] Theme
- [ ] Logging

---

# Phase 21 — Flutter Authentication

- [ ] Login screen
- [ ] Logout
- [ ] Refresh token
- [ ] Session expiration
- [ ] Password reset
- [ ] User profile

---

# Phase 22 — Flutter POS

- [ ] Product search
- [ ] Barcode scanner
- [ ] Product details
- [ ] Cart
- [ ] Quantity
- [ ] Unit
- [ ] Price
- [ ] Discount
- [ ] Tax
- [ ] Payment
- [ ] Receipt
- [ ] Sale completion

---

# Phase 23 — Flutter Inventory

- [ ] Product list
- [ ] Categories
- [ ] Units
- [ ] Stock
- [ ] Warehouse
- [ ] Stock movement
- [ ] Stock adjustment
- [ ] Inventory search

---

# Phase 24 — Flutter Purchase

- [ ] Supplier
- [ ] Purchase order
- [ ] Purchase lines
- [ ] Purchase price
- [ ] Receiving
- [ ] Stock update

---

# Phase 25 — Flutter User Management

Manager:

- [ ] Create cashier
- [ ] Disable cashier
- [ ] Reset cashier
- [ ] Assign permissions where allowed

Display:

    Users: 3 / 5
    Managers: 1 / 1
    Cashiers: 2 / 4

---

# Phase 26 — Flutter Plan Features

- [ ] Load capabilities
- [ ] Feature guards
- [ ] Limit guards
- [ ] Upgrade screen
- [ ] Plan information
- [ ] Usage information
- [ ] Subscription state

Do not rely on client-side guards for security.

---

# Phase 27 — Flutter Offline

- [ ] SQLite
- [ ] Local product cache
- [ ] Local customer cache
- [ ] Local POS configuration
- [ ] Offline sale queue
- [ ] Sync engine
- [ ] Retry
- [ ] Conflict detection
- [ ] Sync status UI

---

# Phase 28 — Web Control Panel

## Dashboard

- [ ] Tenant count
- [ ] Active tenants
- [ ] Paused tenants
- [ ] Suspended tenants
- [ ] Backup health
- [ ] Job health
- [ ] Storage usage

## Tenant Page

- [ ] Tenant profile
- [ ] Plan
- [ ] Subscription
- [ ] Database
- [ ] Database size
- [ ] Users
- [ ] Usage
- [ ] Features
- [ ] Backups
- [ ] Jobs
- [ ] Audit

Actions:

- [ ] Start
- [ ] Pause
- [ ] Stop
- [ ] Suspend
- [ ] Resume
- [ ] Backup
- [ ] Restore

---

# Phase 29 — Web Plan Management

- [ ] Create plan
- [ ] Edit plan
- [ ] Set price
- [ ] Set billing period
- [ ] Configure features
- [ ] Configure user limits
- [ ] Configure backup retention
- [ ] Activate/deactivate plan

---

# Phase 30 — Subscription

- [ ] Trial
- [ ] Active
- [ ] Grace period
- [ ] Expired
- [ ] Suspended
- [ ] Cancelled
- [ ] Resume
- [ ] Upgrade
- [ ] Downgrade

---

# Phase 31 — Monitoring

- [ ] Health endpoint
- [ ] Readiness endpoint
- [ ] Prometheus metrics
- [ ] Grafana dashboards
- [ ] Error monitoring
- [ ] Job metrics
- [ ] Backup metrics
- [ ] Database metrics

---

# Phase 32 — Security Testing

- [ ] Authentication tests
- [ ] Authorization tests
- [ ] Tenant isolation tests
- [ ] IDOR tests
- [ ] SQL injection tests
- [ ] JWT tests
- [ ] Refresh token tests
- [ ] Role escalation tests
- [ ] Plan bypass tests
- [ ] Backup access tests
- [ ] Restore authorization tests

---

# Phase 33 — Backup Testing

- [ ] Create backup
- [ ] Verify checksum
- [ ] Download backup
- [ ] Restore backup
- [ ] Verify products
- [ ] Verify sales
- [ ] Verify inventory
- [ ] Verify users
- [ ] Verify permissions
- [ ] Verify POS operation

---

# Phase 34 — Load Testing

Test:

    10 tenants
    100 tenants
    1,000 tenants

Measure:

- [ ] API latency
- [ ] Database connections
- [ ] Worker throughput
- [ ] Backup throughput
- [ ] Synchronization throughput

---

# Phase 35 — Production

- [ ] Docker images
- [ ] Docker Compose
- [ ] Reverse proxy
- [ ] TLS
- [ ] PostgreSQL
- [ ] Redis
- [ ] MinIO/S3
- [ ] Secrets
- [ ] Monitoring
- [ ] Log rotation
- [ ] Database backup
- [ ] Disaster recovery

---

# Phase 36 — Final Acceptance

- [ ] Create tenant
- [ ] Automatically create database
- [ ] Assign plan
- [ ] Create manager
- [ ] Invite cashier
- [ ] Enforce cashier limit
- [ ] Login from Flutter
- [ ] Create product
- [ ] Make sale
- [ ] Update inventory
- [ ] Create purchase
- [ ] Create backup
- [ ] Verify backup
- [ ] Restore backup
- [ ] Pause tenant
- [ ] Verify POS blocked
- [ ] Resume tenant
- [ ] Verify POS works
- [ ] Stop tenant
- [ ] Verify database preserved
- [ ] Start tenant
- [ ] Verify data preserved
- [ ] Suspend subscription
- [ ] Verify service restriction
- [ ] Upgrade plan
- [ ] Verify new features
- [ ] Downgrade plan
- [ ] Verify usage validation
- [ ] Verify complete audit trail
