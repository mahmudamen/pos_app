# API Contract

Base path:

```text
/v1
```

## Success envelope

```json
{
  "data": {},
  "meta": {
    "request_id": "..."
  }
}
```

## Error envelope

```json
{
  "error": {
    "code": "validation_error",
    "message": "invalid request",
    "request_id": "..."
  }
}
```

## Health

```text
GET /health/live
GET /health/ready
```

## Authentication

```text
POST /v1/auth/login
POST /v1/auth/refresh
POST /v1/auth/logout
```

## Catalog

```text
GET    /v1/categories
POST   /v1/categories
GET    /v1/products
POST   /v1/products
GET    /v1/products/:id
PATCH  /v1/products/:id
GET    /v1/products/barcode/:barcode
```

## Sales

```text
POST /v1/sales
GET  /v1/sales/:id
```

## Sync

Pull endpoint streams tenant changes ordered by the monotonic `change_seq`.
Entities: categories, products, customers, sales, customer_loyalty_log,
register_sessions, inventory_adjustments, tenant_settings. Users, devices and
sync_commands are intentionally excluded (sensitive/internal).

```text
GET /v1/sync/pull?cursor=<int>&limit=<int>
```

Query params: `cursor` (default 0, opaque — pass back the `cursor` you
received), `limit` (default 100, clamped to 1..500).

Response `data.items[]`:
- `entity` — one of the syncable entities
- `id` — entity id as text (`tenant_settings` uses the settings `key`)
- `change_seq` — monotonically increasing across entities
- `change_type` — `upsert`, or `delete` for soft-deleted catalog rows
- `data` — full tenant row as JSON

Stream is ordered by `change_seq` ASC; `data.has_more` true when another page
exists; `data.cursor` is the last seen `change_seq` (the next request's
`cursor`). A cursor older than the retained window returns 409
`{error:{code:"cursor_expired"}}` and requires a controlled full resync
(`cursor=0`).

`POST /v1/sync/push` is part of the roadmap backlog (the offline `create_sale`
replay via client idempotency key is the current write path).

## Idempotency

Write commands that can create business state accept a tenant-scoped idempotency key.

A retry with the same key and semantically identical payload returns the original logical result.

A reused key with a different payload returns a conflict.

Client-supplied tenant headers are not an authorization mechanism.
