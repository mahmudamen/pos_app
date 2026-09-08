# POS Product Scope

## Product

This system is a multi-tenant, Android-first point-of-sale platform for small retail and food businesses. A terminal must continue selling when the network is unavailable, then synchronize safely when connectivity returns.

## Core workflows

1. An owner creates a tenant and staff users.
2. A cashier signs in on a registered terminal.
3. The terminal searches or scans tenant-owned products.
4. The cashier builds a cart, applies permitted discounts and taxes, and completes a sale.
5. The app prints or shares a receipt and records the sale locally.
6. The sync worker uploads pending commands and pulls catalog changes using a server cursor.
7. Managers review products, stock, customers, sales, and terminal sessions.

## Business invariants

- Tenant identity comes from authenticated token claims, never from a client-supplied tenant header.
- Product SKU and barcode uniqueness is tenant-scoped.
- Sale totals use integer minor currency units, not floating-point values.
- A sale and its stock changes commit atomically.
- Client retries are safe through tenant-scoped idempotency keys.
- Refresh tokens rotate; reuse revokes the related session family.
- PostgreSQL is the source of truth. Redis is for sessions, rate limits, and acceleration.
- Offline writes are durable and replayed in order with explicit conflict rules.

## First vertical slice

The backend implementation begins with the shared security foundation: password hashing plus issuer-, algorithm-, and token-type-checked JWTs carrying tenant, user, device, and session identity. The next slice is the login/refresh API, followed by the Flutter API client and local session storage.