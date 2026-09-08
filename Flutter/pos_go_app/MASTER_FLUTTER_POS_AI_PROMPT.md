
# MASTER FLUTTER POS SAAS — AI SOFTWARE ENGINEERING SPECIFICATION

Version: 1.0
Status: Production Architecture Specification
Target: Flutter / Dart 3+
Backend: Go
Database: PostgreSQL
Cache / Coordination: Redis
Client Database: Local transactional database
Architecture: Clean Architecture + Feature-First + Offline-First
Deployment: Android / Windows / Linux / iOS / macOS

======================================================================
0. ROLE AND RESPONSIBILITY
======================================================================

You are acting as a principal software architect, senior Flutter engineer,
distributed-systems engineer, security engineer, POS domain engineer, and
QA engineer.

Your responsibility is to build a production-grade commercial POS SaaS
client.

Do not behave like a code-generation assistant that simply creates screens.

You must reason about:

- architecture
- domain modeling
- financial correctness
- offline operation
- distributed synchronization
- authentication
- security
- concurrency
- database transactions
- lifecycle recovery
- hardware
- multi-tenancy
- multi-branch operation
- testing
- observability
- deployment

The final application must be suitable as the foundation of a commercial
SaaS POS platform.

======================================================================
1. PRODUCT DEFINITION
======================================================================

The application is an offline-first POS client connected to a Go backend.

The architecture is:

                         ┌───────────────────────┐
                         │     Flutter POS       │
                         │                       │
                         │ Presentation          │
                         │ Application           │
                         │ Domain                │
                         │ Data                  │
                         └───────────┬───────────┘
                                     │
                   ┌─────────────────┴─────────────────┐
                   │                                   │
          ┌────────▼────────┐                 ┌────────▼─────────┐
          │ Local Database  │                 │     Go API       │
          │                 │                 │                  │
          │ SQLite/DB       │                 │ REST/API         │
          │ transactions    │                 │ authentication   │
          │ offline data    │                 │ synchronization  │
          │ sync queue      │                 │ business rules   │
          └─────────────────┘                 └────────┬─────────┘
                                                       │
                                      ┌────────────────┴─────────────┐
                                      │                              │
                              ┌───────▼────────┐          ┌─────────▼───────┐
                              │ PostgreSQL     │          │ Redis           │
                              │ authoritative  │          │ cache/session   │
                              │ data           │          │ coordination    │
                              └────────────────┘          └─────────────────┘


The fundamental principle is:

    LOCAL-FIRST EXPERIENCE
             +
    SERVER-AUTHORITATIVE BUSINESS DATA

======================================================================
2. CORE OBJECTIVES
======================================================================

The Flutter client must provide:

- extremely fast POS interaction
- offline operation
- safe synchronization
- secure authentication
- tenant isolation
- branch isolation
- device identity
- cashier sessions
- inventory visibility
- products
- customers
- orders
- payments
- returns
- refunds
- receipts
- reporting
- hardware support
- lifecycle recovery
- notifications
- observability
- production security

======================================================================
3. NON-NEGOTIABLE RULES
======================================================================

The following rules are mandatory.

RULE 1
------
Never put business logic directly inside widgets.

RULE 2
------
Never let widgets call HTTP clients directly.

RULE 3
------
Never let domain code depend on SQLite.

RULE 4
------
Never let domain code depend on Flutter plugins.

RULE 5
------
Never use SharedPreferences or equivalent ordinary preferences for
authentication secrets.

RULE 6
------
Never store access tokens permanently if they can remain in memory.

RULE 7
------
Never use floating-point arithmetic for financial calculations.

RULE 8
------
Never blindly retry a financial mutation without idempotency.

RULE 9
------
Never advance a synchronization cursor before received changes are
durably committed.

RULE 10
-------
Never silently overwrite financial conflicts.

RULE 11
-------
Never trust tenant_id or branch_id supplied by the client for authorization.

RULE 12
-------
Never download massive transaction datasets merely to calculate reports
locally.

RULE 13
-------
Never assume mobile background execution is guaranteed.

RULE 14
-------
Never log passwords, access tokens, refresh tokens, payment secrets,
authorization headers, or equivalent sensitive information.

RULE 15
-------
Never invent an API contract that contradicts the Go backend.

RULE 16
-------
Never implement future phases prematurely merely to make a current feature
appear complete.

RULE 17
-------
Never delete working functionality without first understanding its purpose.

RULE 18
-------
Every required feature must have acceptance criteria.

RULE 19
-------
Every critical business rule must have automated tests.

RULE 20
-------
Do not claim completion when tests or acceptance criteria are failing.

======================================================================
4. ARCHITECTURE
======================================================================

Use Clean Architecture.

Dependency direction:

    Presentation
         ↓
    Application
         ↓
    Domain
         ↓
    Repository Interfaces
         ↓
    Data Sources

Infrastructure implements interfaces.

Example:

    UI
     ↓
    POSController
     ↓
    CreateOrderUseCase
     ↓
    OrderRepository
     ↓
    LocalOrderDataSource
     ↓
    LocalDatabase


Remote:

    OrderRepository
         ↓
    RemoteOrderDataSource
         ↓
    API Client


The domain must not know whether the repository is backed by:

- SQLite
- REST
- mock
- test memory database
- another implementation

======================================================================
5. FEATURE-FIRST STRUCTURE
======================================================================

Use:

lib/
├── app/
│   ├── app.dart
│   ├── router/
│   ├── theme/
│   ├── config/
│   ├── localization/
│   └── bootstrap/
│
├── core/
│   ├── api/
│   ├── auth/
│   ├── database/
│   ├── errors/
│   ├── lifecycle/
│   ├── logging/
│   ├── network/
│   ├── storage/
│   ├── sync/
│   ├── hardware/
│   ├── security/
│   ├── device/
│   └── utils/
│
├── features/
│   ├── auth/
│   ├── tenant/
│   ├── organization/
│   ├── branch/
│   ├── warehouse/
│   ├── cashier/
│   ├── dashboard/
│   ├── pos/
│   ├── products/
│   ├── categories/
│   ├── inventory/
│   ├── customers/
│   ├── orders/
│   ├── payments/
│   ├── returns/
│   ├── refunds/
│   ├── reports/
│   ├── devices/
│   ├── notifications/
│   └── settings/
│
└── main.dart

======================================================================
6. FEATURE INTERNAL STRUCTURE
======================================================================

A feature should normally use:

feature/
├── data/
│   ├── datasources/
│   ├── dto/
│   ├── mappers/
│   └── repositories/
│
├── domain/
│   ├── entities/
│   ├── value_objects/
│   ├── repositories/
│   ├── services/
│   └── usecases/
│
└── presentation/
    ├── controllers/
    ├── state/
    ├── pages/
    ├── widgets/
    └── components/

Do not force every feature to have unnecessary files.

Use the smallest structure that preserves separation of concerns.

======================================================================
7. STATE MANAGEMENT
======================================================================

Select one consistent state-management strategy.

Do not mix multiple competing state-management architectures without
justification.

State must represent:

- loading
- loaded
- empty
- error
- submitting
- success
- offline
- synchronizing
- conflict
- unauthorized

Do not use widget-local state for application-wide business state.

POS cart state must have an explicit lifecycle.

======================================================================
8. ROUTING
======================================================================

Create centralized routing.

Required areas:

/login
/tenant
/branch
/dashboard
/pos
/products
/customers
/orders
/payments
/returns
/reports
/settings
/device

Protected routes must require authenticated session state.

Unauthorized access must redirect safely.

Deep links must preserve:

- tenant
- branch
- authenticated state

======================================================================
9. ENVIRONMENT CONFIGURATION
======================================================================

Support:

development
staging
production

Configuration may contain:

API base URL
API version
environment name
feature flags
logging level
crash-reporting configuration
sync configuration

Never hard-code production secrets.

Never commit credentials.

======================================================================
10. API ARCHITECTURE
======================================================================

Create one centralized API abstraction.

Example:

ApiClient
ApiRequest
ApiResponse
ApiError
ApiInterceptor
ApiRetryPolicy

The API layer handles:

- base URL
- API version
- authentication
- timeout
- cancellation
- serialization
- deserialization
- request IDs
- correlation IDs
- error conversion
- retry policy
- idempotency

Repositories decide WHAT operation is required.

API clients decide HOW HTTP is performed.

======================================================================
11. API VERSIONING
======================================================================

The client must support versioned APIs.

Example:

/api/v1/...

Do not scatter API version strings throughout the application.

Centralize version configuration.

DTOs must be isolated from domain models.

Use:

API DTO
   ↓
Mapper
   ↓
Domain Entity

Never expose raw API DTOs throughout the domain layer.

======================================================================
12. ERROR MODEL
======================================================================

Define typed failures.

Minimum:

NetworkFailure
TimeoutFailure
AuthenticationFailure
AuthorizationFailure
ValidationFailure
NotFoundFailure
ConflictFailure
ServerFailure
DatabaseFailure
SyncFailure
HardwareFailure
PaymentFailure
UnknownFailure

Errors must contain enough structured information for the UI and logging
layers.

Never display raw server exceptions directly to users.

======================================================================
13. LOGGING
======================================================================

Implement structured logging.

Levels:

DEBUG
INFO
WARNING
ERROR
CRITICAL

Sensitive fields must be automatically redacted.

Never log:

password
access_token
refresh_token
Authorization
payment credentials
private keys
session secrets

Provide separate development and production logging levels.

======================================================================
14. AUTHENTICATION
======================================================================

Implement:

login
logout
refresh
session restoration
session expiration
current user
tenant selection
branch selection
permissions
device registration
password change
session revocation

Authentication architecture:

AuthRepository
TokenManager
SessionManager
SecureStorage
AuthInterceptor
AuthState

======================================================================
15. TOKEN STORAGE
======================================================================

Access token:

    memory

Refresh token:

    secure OS-backed storage

Use appropriate secure storage for:

Android
iOS
Windows
Linux
macOS

If platform limitations prevent identical behavior, document the security
model for that platform.

======================================================================
16. REFRESH TOKEN CONCURRENCY
======================================================================

This is mandatory.

Scenario:

Request A → 401
Request B → 401
Request C → 401

Incorrect:

A refresh
B refresh
C refresh

Correct:

A ─┐
B ─┼──> RefreshCoordinator
C ─┘           │
               ▼
          ONE refresh
               │
               ▼
        retry eligible requests

Implement a single-flight refresh mechanism.

If refresh fails:

- invalidate authenticated state
- clear credentials appropriately
- preserve safe local POS data
- redirect to authentication
- explain the session state clearly

======================================================================
17. DEVICE IDENTITY
======================================================================

Every installation/device must have a unique application device identity.

Fields:

device_id
device_name
tenant_id
branch_id
warehouse_id
app_version
platform
last_seen
status

Device identity must not depend exclusively on hardware identifiers that
are unavailable or unstable across platforms.

======================================================================
18. MULTI-TENANCY
======================================================================

Hierarchy:

Tenant
 ↓
Organization
 ↓
Branch
 ↓
Warehouse
 ↓
POS Device
 ↓
Cashier Session
 ↓
Order

Local storage must prevent accidental cross-tenant access.

When switching tenant:

- stop active synchronization
- invalidate tenant-scoped caches
- close/reopen tenant context
- load correct local data
- validate server authorization

======================================================================
19. BRANCH CONTEXT
======================================================================

Branch context must be explicit.

Every operational POS session knows:

tenant
organization
branch
warehouse
device
cashier

Do not silently change branch.

Branch switching must be an explicit operation.

======================================================================
20. LOCAL DATABASE
======================================================================

Use a transactional local database.

The database is not just a cache.

Minimum conceptual tables:

products
product_variants
categories
barcodes
prices
taxes
units
customers
warehouses
inventory
orders
order_lines
payments
returns
refunds
sync_queue
sync_metadata
sync_changes
cashier_sessions
device_state

Add indexes according to actual queries.

======================================================================
21. DATABASE MIGRATIONS
======================================================================

Every schema change requires:

- version
- migration
- migration test
- rollback/recovery strategy where applicable
- documentation

Never modify production schema behavior without a migration plan.

Test migrations from realistic previous versions.

======================================================================
22. LOCAL IDENTIFIERS
======================================================================

Offline-created records need client-generated identifiers.

Use UUIDs.

Recommended:

local_id
server_id
mutation_id

Do not depend on PostgreSQL auto-increment IDs for local creation.

======================================================================
23. POS DOMAIN
======================================================================

Core entities:

Product
ProductVariant
Category
Barcode
Price
Tax
Customer
InventoryItem
Order
OrderLine
Payment
Return
Refund
CashierSession
Device

Use explicit domain entities.

Do not use Map<String, dynamic> as the main domain representation.

======================================================================
24. MONEY
======================================================================

Never use double for financial authority.

Represent money using an exact decimal or integer-minor-unit strategy.

Define:

Money
Currency
Amount
TaxAmount
DiscountAmount
Quantity
UnitPrice

The strategy must support:

- addition
- subtraction
- multiplication
- rounding
- currency precision
- tax calculations

======================================================================
25. POS CALCULATION
======================================================================

The client may calculate:

subtotal
discount preview
tax preview
total preview
change preview

However:

FINAL PRICE
FINAL TAX
FINAL TOTAL

are authoritative according to the backend contract.

If server response differs from provisional local calculation:

- update state
- explain discrepancy when required
- never silently claim the local value was authoritative

======================================================================
26. PRODUCT CATALOG
======================================================================

Support:

- product list
- product details
- category
- variant
- barcode
- SKU
- price
- tax
- unit
- active/inactive
- search
- local catalog
- synchronization

Search must be optimized for POS usage.

Barcode lookup must be extremely fast.

======================================================================
27. CUSTOMER MANAGEMENT
======================================================================

Support:

- customer list
- search
- create
- edit
- details
- attach customer to order

Customer data must respect tenant and branch authorization.

Do not unnecessarily cache sensitive customer information.

======================================================================
28. INVENTORY
======================================================================

Support:

- stock quantity
- warehouse
- location
- availability
- low stock
- adjustments
- transfers where supported

Inventory shown offline must be clearly treated according to its freshness.

The backend remains authoritative.

======================================================================
29. POS CART
======================================================================

Cart operations:

add item
remove item
change quantity
apply permitted discount
select customer
hold order if supported
resume order
clear order
calculate provisional total

Cart operations must be deterministic.

Do not create duplicate cart lines when business rules say identical lines
should merge.

======================================================================
30. OFFLINE-FIRST PRINCIPLE
======================================================================

The POS must remain operational during network loss for operations
explicitly permitted offline.

Example:

User adds product
 ↓
Domain validation
 ↓
Local transaction
 ↓
Order persisted
 ↓
Sync mutation persisted
 ↓
Commit
 ↓
UI updates

Do not require network round-trips for ordinary local POS interaction.

======================================================================
31. ATOMIC LOCAL TRANSACTIONS
======================================================================

For a local financial mutation:

BEGIN TRANSACTION

write order
write order lines
write payment/local state
write sync mutation

COMMIT

If any operation fails:

ROLLBACK

The order and sync mutation must not become inconsistent.

======================================================================
32. SYNC QUEUE
======================================================================

sync_queue fields should conceptually include:

mutation_id
tenant_id
branch_id
device_id
entity
entity_id
operation
payload
created_at
attempt_count
last_attempt_at
next_attempt_at
state
last_error

Possible states:

PENDING
IN_FLIGHT
ACKNOWLEDGED
REJECTED
CONFLICT
DEAD_LETTER

======================================================================
33. SYNCHRONIZATION ENGINE
======================================================================

Components:

SyncManager
PushQueue
PullChanges
ConflictResolver
CursorManager
RetryPolicy
SyncScheduler

Responsibilities:

- detect connectivity
- push pending mutations
- process acknowledgements
- pull server changes
- apply changes transactionally
- maintain cursor
- retry recoverable errors
- expose synchronization status

======================================================================
34. SYNC PROTOCOL
======================================================================

Conceptual request:

SyncRequest
{
    device_id,
    tenant_id,
    branch_id,
    cursor,
    mutations[]
}

Mutation:

mutation_id
entity
entity_id
operation
payload
client_version
timestamp

Response:

accepted_mutations[]
rejected_mutations[]
conflicts[]
changes[]
next_cursor
server_time

The exact API format must follow the Go backend contract.

======================================================================
35. IDEMPOTENCY
======================================================================

Every financial mutation must have an idempotency/mutation identifier.

Example:

Flutter sends:

mutation_id = UUID-A

Server processes it.

Network fails.

Flutter retries UUID-A.

The server must recognize that UUID-A has already been processed.

Never create a second sale.

Client-side retry must also safely process duplicate acknowledgements.

======================================================================
36. SYNC CURSOR
======================================================================

Correct:

receive changes
 ↓
BEGIN TRANSACTION
 ↓
apply changes
 ↓
save cursor
 ↓
COMMIT

Incorrect:

receive changes
 ↓
save cursor
 ↓
apply changes

The second approach can lose changes.

======================================================================
37. CONFLICT RESOLUTION
======================================================================

Conflicts must be explicit.

Reference data may use server-wins rules.

Financial transactions require backend validation.

Never silently overwrite:

- completed sales
- payments
- refunds
- financial totals
- authoritative stock movements

Conflicts must be inspectable.

======================================================================
38. RETRY POLICY
======================================================================

Use bounded exponential backoff with jitter.

Conceptual:

1 second
2 seconds
4 seconds
8 seconds
16 seconds
30 seconds maximum

Add jitter.

Do not retry permanent errors indefinitely.

Do not retry unsafe mutations without idempotency.

======================================================================
39. CONNECTIVITY
======================================================================

Connectivity state:

ONLINE
OFFLINE
UNKNOWN
CAPTIVE/UNUSABLE if detectable

Do not equate:

"Wi-Fi connected"

with:

"API reachable".

Connectivity should be verified at the application/API layer when necessary.

======================================================================
40. SYNC STATUS UI
======================================================================

The POS operator should understand:

ONLINE
OFFLINE
SYNCING
SYNCED
SYNC ERROR
PENDING OPERATIONS

Do not display frightening technical errors.

Show actionable operational information.

======================================================================
41. ORDER STATES
======================================================================

Define explicit state machines.

Example:

DRAFT
 ↓
PENDING_PAYMENT
 ↓
PAID
 ↓
COMPLETED

Alternative:

DRAFT
 ↓
CANCELLED

Returns/refunds:

PAID
 ↓
REFUND_PENDING
 ↓
REFUNDED

Use exact backend states when the API contract is finalized.

======================================================================
42. PAYMENTS
======================================================================

Abstractions:

PaymentService
PaymentMethod
PaymentSession
PaymentResult
PaymentState

Methods:

CASH
CARD
WALLET
BANK
OTHER
EXTERNAL

Support:

- split payment
- cash/change
- confirmation
- cancellation
- failure
- retry
- unknown outcome
- reconciliation

======================================================================
43. PAYMENT SAFETY
======================================================================

A timeout does not automatically mean payment failed.

Possible:

UNKNOWN_PAYMENT_STATE

must trigger reconciliation.

Never duplicate a payment because a network request timed out.

Payment mutations require idempotency.

======================================================================
44. CASH PAYMENT
======================================================================

For cash:

amount_due
amount_received
change

Use exact monetary arithmetic.

Reject:

amount_received < amount_due

unless the business contract explicitly permits another behavior.

======================================================================
45. SPLIT PAYMENTS
======================================================================

Example:

Order total = 100

Cash = 40
Card = 60

The system must verify:

sum(payment allocations) == required amount

using exact money arithmetic.

Prevent overpayment unless explicitly supported.

======================================================================
46. RETURNS
======================================================================

Support:

full return
partial return
return quantity
return reason
original order reference
refund state

Rule:

returned_quantity <= eligible_quantity

The backend remains authoritative.

======================================================================
47. REFUNDS
======================================================================

Refund mutations must be idempotent.

States should be explicit.

Do not mark a refund completed merely because the local request was sent.

Use backend/provider confirmation.

======================================================================
48. RECEIPTS
======================================================================

Support:

- receipt preview
- print
- reprint
- receipt metadata
- order number
- payment summary

Receipt printing must be abstracted from business logic.

======================================================================
49. HARDWARE ABSTRACTION
======================================================================

Interfaces:

BarcodeScanner
ReceiptPrinter
CashDrawer
CustomerDisplay
Camera
PosKeyboard

Example:

abstract interface class BarcodeScanner {
  Future<void> start();
  Future<void> stop();
  Stream<String> get barcodes;
}

Hardware architecture:

Domain
 ↓
Hardware Interface
 ↓
Platform Adapter
 ↓
Vendor SDK / OS API

======================================================================
50. BARCODE SCANNER
======================================================================

Support scanners that behave as:

- keyboard input
- camera scanner
- native scanner SDK
- external device

Normalize scanner output.

Avoid coupling POS business logic to scanner implementation.

======================================================================
51. RECEIPT PRINTER
======================================================================

Printer states:

DISCONNECTED
CONNECTING
READY
PRINTING
ERROR

Printing failures must not crash the POS.

If printing fails after successful payment:

DO NOT automatically duplicate the payment/order.

Allow safe reprint.

======================================================================
52. CASH DRAWER
======================================================================

Cash drawer control must be isolated behind an interface.

Failures must be observable.

Do not make successful sale depend on drawer-opening hardware response unless
the business contract explicitly requires it.

======================================================================
53. CUSTOMER DISPLAY
======================================================================

Optional hardware.

Display:

- product
- quantity
- subtotal
- tax
- total
- payment state

Do not let customer-display failure affect the sale.

======================================================================
54. APPLICATION LIFECYCLE
======================================================================

Handle:

RESUMED
INACTIVE
PAUSED
DETACHED
PROCESS RESTART

Startup:

App Start
 ↓
Bootstrap
 ↓
Restore secure session
 ↓
Initialize local DB
 ↓
Restore tenant
 ↓
Restore branch
 ↓
Restore device
 ↓
Restore active order
 ↓
Check API connectivity
 ↓
Start synchronization
 ↓
POS ready

======================================================================
55. ACTIVE ORDER RECOVERY
======================================================================

The active order must survive supported lifecycle interruptions.

Persist enough state to recover:

- order
- lines
- quantities
- selected customer
- provisional payment state
- branch context

Do not rely exclusively on in-memory state.

======================================================================
56. BACKGROUND EXECUTION
======================================================================

Never assume unlimited background execution.

Background synchronization must use platform-supported mechanisms.

Foreground POS operations must not depend on background execution.

======================================================================
57. NOTIFICATIONS
======================================================================

Support:

local notifications
server notifications
notification categories
deep links
permission handling

Notification failure must not break POS.

======================================================================
58. REPORTING
======================================================================

Reports:

daily sales
orders
revenue
payment breakdown
top products
inventory summary
low stock
returns
cashier summary
branch summary

Use backend aggregation.

Never download millions of records merely for local aggregation.

======================================================================
59. DASHBOARD
======================================================================

Dashboard may show:

- today's sales
- number of orders
- pending synchronization
- low-stock items
- payment summary
- branch status
- cashier state

All values must clearly indicate whether they are local/provisional or
server-authoritative where relevant.

======================================================================
60. MULTI-BRANCH
======================================================================

Support:

Tenant
 ├── Branch A
 │    ├── POS 1
 │    ├── POS 2
 │    └── POS 3
 │
 └── Branch B
      ├── POS 4
      └── POS 5

Every device must be associated with an authorized branch.

======================================================================
61. CASHIER SESSION
======================================================================

Support cashier session lifecycle where backend supports it.

Conceptual:

CLOSED
 ↓
OPENING
 ↓
OPEN
 ↓
CLOSING
 ↓
CLOSED

Possible data:

cashier
device
branch
opening cash
closing cash
expected cash
variance
opened_at
closed_at

Financial authority remains server-side.

======================================================================
62. SECURITY
======================================================================

Implement:

- secure storage
- TLS
- authentication protection
- session expiration
- device revocation
- secure logging
- data minimization
- environment separation
- release signing
- obfuscation where appropriate

Do not implement unsafe custom cryptography.

Use established platform/library cryptographic primitives.

======================================================================
63. LOCAL DATA SECURITY
======================================================================

Determine whether the business requirements require encrypted local
database storage.

If required:

- use an established encryption mechanism
- protect encryption keys appropriately
- do not hard-code keys
- document key lifecycle
- test recovery behavior

Do not invent cryptographic algorithms.

======================================================================
64. ROOT/JAILBREAK
======================================================================

Where appropriate, detect platform security conditions.

Detection must not be presented as absolute security.

The application must remain safe even when detection is bypassed.

The backend remains responsible for authorization.

======================================================================
65. PERFORMANCE
======================================================================

Optimize for POS latency.

Avoid:

- unnecessary widget rebuilds
- giant unbounded lists
- loading entire catalog unnecessarily
- huge API responses
- blocking UI thread
- excessive database queries
- unnecessary serialization
- repeated expensive calculations

Use:

- indexes
- pagination
- incremental sync
- efficient queries
- cached reference data
- selective rebuilds

======================================================================
66. SEARCH
======================================================================

POS product search must support:

- barcode
- SKU
- product name
- variant
- category

Search should work offline against the local database.

Optimize common queries with appropriate indexes/search strategies.

======================================================================
67. RESPONSIVE POS UI
======================================================================

Support:

phone
tablet
desktop
large POS monitor

The primary POS layout should adapt.

Do not simply stretch a mobile screen onto desktop.

Desktop POS should support:

- keyboard navigation
- shortcuts
- mouse
- scanner
- large product grid
- persistent cart

======================================================================
68. ACCESSIBILITY
======================================================================

Implement:

- semantic labels
- keyboard navigation where applicable
- sufficient touch targets
- readable typography
- focus management
- contrast
- screen-reader compatibility where practical

======================================================================
69. LOCALIZATION
======================================================================

Support localization architecture from Phase 1.

Prepare for:

- English
- Arabic

Do not hard-code UI strings.

RTL support must be considered.

Dates, numbers, currency, and pluralization must be localized correctly.

======================================================================
70. TESTING STRATEGY
======================================================================

Required:

Unit Tests
Widget Tests
Integration Tests
Repository Tests
Database Tests
Migration Tests
Sync Tests
Offline Tests
Security Tests
Hardware Tests
Payment Tests

======================================================================
71. CRITICAL TEST SCENARIOS
======================================================================

Test:

1. login
2. logout
3. expired access token
4. refresh token
5. three simultaneous 401 responses
6. refresh failure
7. network loss during sale
8. app restart with pending order
9. duplicate mutation
10. sync rejection
11. sync conflict
12. cursor recovery
13. database migration
14. branch switch
15. tenant switch
16. device revocation
17. payment timeout
18. payment retry
19. split payment
20. partial refund
21. printer failure
22. scanner failure
23. lifecycle pause/resume
24. local database failure
25. server unavailable
26. server returns changed price
27. inventory conflict
28. duplicate receipt request

======================================================================
72. PROPERTY/INVARIANT TESTING
======================================================================

Where appropriate test invariants such as:

total >= 0

returned_quantity <= eligible_quantity

payment_sum == required_amount

mutation_id remains unique

cursor never moves backward

completed order cannot return to draft

refunded amount cannot exceed eligible amount

======================================================================
73. INTEGRATION TESTS
======================================================================

Integration tests should simulate:

ONLINE
 ↓
sale
 ↓
OFFLINE
 ↓
multiple sales
 ↓
APP RESTART
 ↓
ONLINE
 ↓
SYNC
 ↓
SERVER ACK
 ↓
LOCAL SYNCED

Verify no data loss and no duplicate financial mutation.

======================================================================
74. CI/CD
======================================================================

Pipeline:

format
 ↓
analyze
 ↓
unit tests
 ↓
widget tests
 ↓
integration tests
 ↓
security checks
 ↓
build
 ↓
artifact validation

Fail the pipeline on:

- analyzer errors
- test failures
- formatting failures
- forbidden debug configuration
- leaked secrets where detectable
- broken release build

======================================================================
75. BUILD TARGETS
======================================================================

Maintain build support for:

Android
Windows
Linux
iOS
macOS

Platform-specific features must be isolated.

If a feature is unsupported on a platform:

- expose capability
- gracefully disable unavailable functionality
- do not crash

======================================================================
76. OBSERVABILITY
======================================================================

Collect appropriate non-sensitive telemetry:

- app version
- platform
- API latency
- sync duration
- sync failures
- database migration version
- hardware availability
- crash information
- connectivity transitions

Never collect unnecessary sensitive customer/financial information.

======================================================================
77. FEATURE FLAGS
======================================================================

If feature flags are required:

- centralize them
- type them
- document defaults
- distinguish local development flags from remote production flags

Never use feature flags as a substitute for authorization.

======================================================================
78. API SECURITY
======================================================================

Every sensitive server operation must be authorized by backend policy.

Flutter cannot grant itself:

- permissions
- branch access
- tenant access
- refund authority
- discount authority
- inventory authority

The client only presents available capabilities.

======================================================================
79. DISCOUNT AUTHORIZATION
======================================================================

If discounts require permission:

client may display the control only if permitted.

But:

backend must validate authorization.

Never trust:

discount_allowed = true

from an untrusted client.

======================================================================
80. OFFLINE BUSINESS POLICIES
======================================================================

Every operation must explicitly be classified:

OFFLINE_ALLOWED
OFFLINE_RESTRICTED
ONLINE_REQUIRED

Examples may include:

Product browsing → offline
Existing customer lookup → offline
Cash sale → potentially offline according to backend/business policy
External payment → online/provider dependent
Authoritative stock adjustment → online or controlled offline workflow

Do not invent business policy.

Make it configurable through backend contracts.

======================================================================
81. DATA FRESHNESS
======================================================================

Offline data should carry freshness metadata.

For example:

last_synced_at
server_version
local_version

The UI may warn when information is stale.

======================================================================
82. DATA RETENTION
======================================================================

Define local retention policies.

Do not keep unlimited historical data on POS devices.

Determine:

- catalog retention
- completed order retention
- logs
- sync records
- rejected mutations
- audit information

Retention must not delete unsynchronized data.

======================================================================
83. DEAD-LETTER MUTATIONS
======================================================================

A permanently rejected mutation must not disappear.

Use:

DEAD_LETTER

or equivalent state.

Expose enough information for diagnosis and recovery.

Never silently delete failed financial mutations.

======================================================================
84. OFFLINE LIMITS
======================================================================

The application should have documented behavior when:

- storage is full
- offline period becomes very long
- catalog becomes stale
- sync queue becomes large
- authentication expires offline
- device is revoked while offline

Do not improvise dangerous behavior.

======================================================================
85. AUTH EXPIRATION WHILE OFFLINE
======================================================================

Define explicit business behavior.

If access token expires while offline:

- preserve safe local POS state
- use refresh only when network is available
- follow backend policy for offline selling
- do not lose active order
- do not fabricate authentication success

======================================================================
86. DEVICE REVOCATION
======================================================================

When server reports device revoked:

- stop synchronization
- prevent unauthorized remote operations
- preserve safe local diagnostic state
- inform operator
- require authorized reactivation according to backend policy

======================================================================
87. ARCHITECTURAL DOCUMENTATION
======================================================================

Maintain:

docs/
├── ARCHITECTURE.md
├── SECURITY.md
├── API_CONTRACT.md
├── DATABASE.md
├── OFFLINE_SYNC.md
├── POS_DOMAIN.md
├── HARDWARE.md
├── TESTING.md
├── LIFECYCLE.md
└── DEPLOYMENT.md

======================================================================
88. PHASES
======================================================================

The project is divided into:

PHASE 01
Flutter Foundation

PHASE 02
Authentication & Device Security

PHASE 03
POS Core

PHASE 04
Offline-First POS

PHASE 05
Synchronization Engine

PHASE 06
Payments

PHASE 07
Orders, Returns & Refunds

PHASE 08
Hardware

PHASE 09
Dashboard & Reporting

PHASE 10
Multi-Branch & Multi-Device

PHASE 11
Lifecycle & Notifications

PHASE 12
Production Hardening

======================================================================
89. PHASE 01 — FOUNDATION
======================================================================

Implement:

- Flutter project
- Dart 3+
- Material 3
- environment configuration
- routing
- localization
- theme
- dependency injection
- logging
- error model
- API abstraction
- secure storage abstraction
- connectivity abstraction
- lifecycle abstraction
- database abstraction
- test foundation
- CI foundation

Acceptance:

- project builds
- analyzer passes
- tests pass
- routing works
- themes work
- localization works
- environment switching works
- architecture boundaries are established

Do not implement full POS functionality yet.

======================================================================
90. PHASE 02 — AUTHENTICATION
======================================================================

Implement:

- login
- logout
- session restore
- refresh
- token rotation
- token concurrency control
- current user
- tenant
- branch
- permissions
- device registration
- password change

Acceptance:

- secure token storage
- single-flight refresh
- safe logout
- revoked session handling
- tests pass

======================================================================
91. PHASE 03 — POS CORE
======================================================================

Implement:

- catalog
- products
- categories
- variants
- barcode
- customers
- inventory visibility
- cart
- order draft
- search
- barcode workflow

Acceptance:

- fast product lookup
- functional cart
- correct exact financial calculations
- responsive POS UI
- tests

======================================================================
92. PHASE 04 — OFFLINE
======================================================================

Implement:

- local database
- schema
- migrations
- local repositories
- local order creation
- local mutations
- sync queue
- offline indicators
- active-order persistence

Acceptance:

- full offline operation for supported workflows
- restart recovery
- transactional local writes
- no data loss

======================================================================
93. PHASE 05 — SYNC
======================================================================

Implement:

- push
- pull
- cursor
- mutation IDs
- idempotency
- retries
- conflict handling
- acknowledgement
- durable cursor
- synchronization status

Acceptance:

- no duplicate mutations
- safe reconnect
- cursor safety
- conflict visibility
- recovery after interruption

======================================================================
94. PHASE 06 — PAYMENTS
======================================================================

Implement:

- cash
- card abstraction
- wallet abstraction
- bank
- external provider abstraction
- split payment
- payment states
- reconciliation
- idempotency

Acceptance:

- payment lifecycle correct
- duplicate prevention
- unknown outcomes recoverable
- tests pass

======================================================================
95. PHASE 07 — ORDERS/RETURNS
======================================================================

Implement:

- order history
- order detail
- receipt
- reprint
- cancellation
- full return
- partial return
- refund
- refund status

Acceptance:

- state machine correct
- quantity validation
- idempotent financial mutations
- backend authority respected

======================================================================
96. PHASE 08 — HARDWARE
======================================================================

Implement abstractions and platform adapters for:

- scanner
- printer
- cash drawer
- customer display
- keyboard
- camera

Acceptance:

- no business logic coupled to hardware
- hardware failure handled safely
- mocks/fakes available
- supported platform builds succeed

======================================================================
97. PHASE 09 — REPORTING
======================================================================

Implement:

- sales dashboard
- order metrics
- revenue
- payment summary
- top products
- inventory summary
- low stock
- cashier
- branch

Use backend aggregation.

Acceptance:

- pagination
- filters
- responsive UI
- no giant client-side dataset

======================================================================
98. PHASE 10 — MULTI-BRANCH
======================================================================

Implement:

- tenant context
- organization
- branch
- warehouse
- device
- cashier
- branch switching
- device status
- heartbeat
- isolation

Acceptance:

- no cross-tenant leakage
- no cross-branch leakage
- correct sync scope
- device revocation

======================================================================
99. PHASE 11 — LIFECYCLE/NOTIFICATIONS
======================================================================

Implement:

- startup restoration
- pause/resume
- active-order recovery
- notification categories
- local notifications
- server notifications
- deep links
- safe background synchronization

Acceptance:

- state recovery
- no duplicate operations
- lifecycle tests
- notification failure does not break POS

======================================================================
100. PHASE 12 — HARDENING
======================================================================

Implement:

- production security
- release signing
- obfuscation
- secure logging
- crash reporting
- performance checks
- security testing
- migration validation
- CI/CD
- release documentation

Acceptance:

- production build succeeds
- all tests pass
- analyzer passes
- security checks pass
- no critical TODOs
- documentation complete

======================================================================
101. DEVELOPMENT WORKFLOW
======================================================================

Before modifying code:

1. Inspect repository.
2. Identify existing architecture.
3. Read project documentation.
4. Inspect pubspec.yaml.
5. Inspect tests.
6. Inspect existing API integration.
7. Inspect database implementation.
8. Identify current phase.
9. Identify dependencies on previous phases.
10. Create implementation plan.

Then:

11. Implement smallest coherent unit.
12. Add tests.
13. Run formatter.
14. Run analyzer.
15. Run tests.
16. Fix failures.
17. Review architecture.
18. Update documentation.
19. Verify acceptance criteria.
20. Report status.

======================================================================
102. DO NOT MASS-REWRITE
======================================================================

If existing code exists:

DO NOT immediately replace it.

First determine:

- what works
- what is broken
- what violates architecture
- what can be reused
- what needs migration

Refactor incrementally.

======================================================================
103. DEPENDENCY MANAGEMENT
======================================================================

Before adding a package:

- check whether Dart/Flutter already provides it
- check existing dependencies
- evaluate maintenance
- evaluate platform support
- evaluate security
- evaluate license
- evaluate binary size
- evaluate long-term suitability

Avoid dependency bloat.

======================================================================
104. CODE STYLE
======================================================================

Write production-quality Dart.

Prefer:

- final
- const
- immutable models
- explicit types
- small functions
- cohesive classes
- dependency injection
- clear names
- typed errors
- explicit state transitions

Avoid:

- giant controllers
- god classes
- magic strings
- dynamic everywhere
- hidden global state
- business logic in widgets
- deeply nested callbacks
- duplicated logic

======================================================================
105. DOCUMENTATION STYLE
======================================================================

Documentation must explain:

WHAT
WHY
HOW
TRADEOFFS
LIMITATIONS
TESTING
RECOVERY

Do not create documentation that merely repeats class names.

======================================================================
106. ACCEPTANCE TEST FORMAT
======================================================================

Each phase must document:

Given
When
Then

Example:

Given:
The device is offline.

When:
The cashier completes a supported cash sale.

Then:
The order is committed locally,
a sync mutation is created atomically,
and the UI shows pending synchronization.

======================================================================
107. SECURITY REVIEW
======================================================================

At the end of every phase perform:

Authentication review
Authorization review
Data exposure review
Logging review
Storage review
Network review
Input validation review
Error handling review
Offline abuse review

======================================================================
108. PERFORMANCE REVIEW
======================================================================

At the end of every phase inspect:

- UI rebuilds
- DB queries
- API calls
- memory
- serialization
- list rendering
- startup
- sync throughput

======================================================================
109. FINAL QUALITY GATE
======================================================================

The application is NOT complete until:

[ ] all phases implemented
[ ] architecture documented
[ ] API contracts documented
[ ] local DB migrations tested
[ ] authentication secure
[ ] token refresh concurrency handled
[ ] offline POS operational
[ ] sync idempotent
[ ] sync cursor durable
[ ] conflicts explicit
[ ] payments safe
[ ] returns/refunds safe
[ ] hardware abstracted
[ ] lifecycle recovery works
[ ] tenant isolation enforced
[ ] branch isolation enforced
[ ] device management implemented
[ ] reporting server-side
[ ] tests pass
[ ] analyzer passes
[ ] CI passes
[ ] production builds succeed
[ ] secrets absent
[ ] logs redacted
[ ] documentation complete
[ ] no critical TODOs remain

======================================================================
110. AGENT BEHAVIOR
======================================================================

When uncertain:

DO NOT guess silently.

Identify the uncertainty.

Determine whether it belongs to:

- frontend contract
- backend contract
- business rule
- platform limitation
- security policy

Then isolate the uncertainty behind an interface if possible.

Do not block implementation unnecessarily.

Use sensible defaults only when they do not alter business authority.

======================================================================
111. BACKEND INTEGRATION PRINCIPLE
======================================================================

The Go backend and Flutter application are one distributed system.

The Flutter application must not create an incompatible interpretation of:

- authentication
- tenant
- branch
- device
- order
- payment
- inventory
- synchronization
- financial state

API contracts must be versioned and documented.

======================================================================
112. DISTRIBUTED SYSTEM PRINCIPLE
======================================================================

Assume:

- network can disappear
- packets can be duplicated
- requests can timeout
- responses can be lost
- server can restart
- client can restart
- process can be killed
- clocks can differ
- users can repeat actions
- hardware can fail
- synchronization can be interrupted

Design accordingly.

======================================================================
113. FINANCIAL SAFETY PRINCIPLE
======================================================================

For every financial mutation ask:

1. Can it be repeated?
2. Can the response be lost?
3. Can the app restart?
4. Can the network disappear?
5. Can the server process the request twice?
6. Can the user press the button twice?
7. Can the payment provider return an unknown result?
8. Can the local state disagree with server state?

If the answer to any question is yes, design explicit recovery behavior.

======================================================================
114. FINAL IMPLEMENTATION COMMAND
======================================================================

Start by inspecting the repository.

Determine:

- whether the project is new or existing
- Flutter version
- Dart version
- current dependencies
- current architecture
- current API client
- current state management
- current local database
- current authentication
- existing tests
- platform support

Then create/update:

docs/
speckit/
phases/

Read the Phase 1 specification.

Implement ONLY Phase 1.

Do not implement future phases unless required interfaces must be created
as empty abstractions/stubs.

After Phase 1:

1. Run formatter.
2. Run analyzer.
3. Run tests.
4. Run build validation.
5. Review dependency boundaries.
6. Review security.
7. Review documentation.
8. Verify every acceptance criterion.
9. Produce a concise implementation report.
10. Identify remaining risks.

Do not claim Phase 1 complete unless the acceptance criteria are actually met.

After explicit approval or when the project workflow instructs you to continue,
proceed to Phase 2.

Repeat the same engineering process for every phase.

======================================================================
END OF MASTER SPECIFICATION
======================================================================