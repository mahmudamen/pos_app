# POS SaaS Data Model

## Server entities

### Tenant
- `id`: UUID primary key
- `name`, `slug`: store identity; slug globally unique
- timestamps

### User
- `id`, `tenant_id`
- email, password hash, display name, active flag
- role/permission set
- last login timestamp

### Device
- `id`, `tenant_id`, client device ID, name
- last seen and revocation timestamps
- client device ID unique per tenant

### Session
- `id`, `tenant_id`, user ID, device ID
- hashed refresh token, expiry, last-used, revoked timestamps
- session family/replay metadata

### Category and Product
- tenant-scoped IDs
- category name/slug
- product name, SKU, optional barcode
- price and cost in integer minor units
- three-letter currency, stock quantity, active flag
- monotonic `change_seq`

### Customer
- tenant-scoped ID, name, optional email/phone
- monotonic `change_seq`

### Sale and SaleItem
- sale tenant, device, cashier, status, idempotency key
- subtotal, discount, tax, total as integer minor units
- currency and monotonic `change_seq`
- sale item product snapshot, quantity, unit price, discount, tax, total
- sale and items commit in one transaction

### SyncCommand
- tenant, device, stable client command ID
- operation/entity type, payload hash, state, error/result
- created/processed timestamps

### ChangeCursor
- tenant-scoped monotonic sequence position
- last acknowledged client cursor and expiry policy

## Client entities

### SecureSession
Access token, rotated refresh token, tenant ID, user ID, device ID, and display name stored in platform secure storage.

### CachedProduct
Product API representation cached locally for search and barcode lookup while offline.

### CartDraft
Local cart lines keyed by product ID with quantity and price snapshot for the active terminal.

### PendingCommand
Durable offline sale command with command ID, idempotency key, payload, retry count, and status.

### LocalSale
Receipt-ready local sale record linked to the pending command and eventual server sale ID.

## Invariants

- Every tenant-owned server query runs with transaction-local `app.current_tenant` context.
- Product SKU/barcode uniqueness is tenant-scoped.
- Sale totals are calculated from server product prices.
- Stock cannot become negative.
- Reusing an idempotency key with a different payload is a conflict.
- RLS policies apply to all tenant-owned tables.
