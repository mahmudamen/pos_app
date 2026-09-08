# POS API Contract

Base URL: `/v1`

## Envelope

Success:
```json
{"data": {}, "meta": {"request_id": "..."}}
```

Error:
```json
{"error": {"code": "...", "message": "...", "request_id": "..."}}
```

## Authentication

### `POST /auth/login`

Request:
```json
{"tenant_id":"store-slug","email":"cashier@example.com","password":"...","device_id":"stable-device-id","device_name":"Counter 1"}
```

Response data contains `access_token`, `refresh_token`, `expires_in`, `user`, and `tenant`.

### `POST /auth/refresh`

Request: `{"refresh_token":"..."}`

Response data contains a new access token and rotated refresh token.

## Catalog

- `GET /products?search=...` requires `Authorization: Bearer <access-token>`.
- `GET /products/barcode/:barcode` requires bearer authorization.
- Product data uses `price_minor`, `currency`, and `stock_quantity`.

## Sales

### `POST /sales`

Headers:
- `Authorization: Bearer <access-token>`
- `Idempotency-Key: <tenant-scoped-stable-key>`

Request:
```json
{"items":[{"product_id":"uuid","quantity":2}]}
```

The server owns price and total calculation. A successful response is `201` and returns sale ID, subtotal, total, and currency. Insufficient stock returns `409`. Reusing a key with a different payload returns `409`.

## Health

- `GET /health/live`
- `GET /health/ready`

## Client UI contract

The Flutter app must expose login, searchable catalog, cart, checkout, explicit loading/error/offline states, secure session storage, and a durable pending-command boundary. Payment, receipt, and sync screens must not bypass the API/data repositories.
