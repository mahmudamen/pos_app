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

```text
GET  /v1/sync/pull?cursor=...
POST /v1/sync/push
```

## Idempotency

Write commands that can create business state accept a tenant-scoped idempotency key.

A retry with the same key and semantically identical payload returns the original logical result.

A reused key with a different payload returns a conflict.

Client-supplied tenant headers are not an authorization mechanism.
