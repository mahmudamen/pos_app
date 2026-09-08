# Database Design

## Core entities

```text
tenants
users
devices
sessions
refresh_tokens
categories
products
customers
sales
sale_items
```

## Tenant identity

Every tenant-owned row contains `tenant_id`.

Business identifiers such as barcode, SKU, and client idempotency keys are tenant-scoped unless the protocol explicitly requires global uniqueness.

## Device identity

`devices.id` is the server primary key.

`devices.client_device_id` is the client-controlled stable identifier and is unique within a tenant.

## Sales

A sale and its items are created in one transaction.

Money uses exact integer minor units.

Recommended fields:

```text
subtotal_minor BIGINT
discount_minor BIGINT
tax_minor BIGINT
total_minor BIGINT
currency CHAR(3)
```

## RLS

Example policy context:

```sql
current_setting('app.current_tenant', true)
```

Application code must set the value transaction-locally before tenant SQL executes.

## Sync sequence

Maintain a monotonic server-side sequence for changes.

Clients store the latest acknowledged cursor and request changes after that sequence.

The server sequence is authoritative; client clocks are not used for ordering.
