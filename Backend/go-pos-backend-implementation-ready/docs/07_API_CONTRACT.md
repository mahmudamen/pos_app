# API Contract

Base path:

```text
/v1
```

## OpenAPI

`docs/openapi.json` is generated from the Gin route table by
`go run ./cmd/openapi` (Makefile: `make openapi`) — the routes are the source
of truth, so the spec cannot drift from what the server serves. The generator
assembles the engine through the same `internal/transport/server.Register`
path the API binary uses, with a nil pool and no rate limiter, so it runs
offline with no database. Path items carry tags, operationIds and the shared
data/error envelope; add request/response schemas for new write endpoints as
the contract grows.

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

### Variants (C3 — fashion/textile)

```text
GET    /v1/products/:id/variants          (authenticated)
POST   /v1/products/:id/variants         (catalog.write)
PATCH  /v1/variants/:id                  (catalog.write)
DELETE /v1/variants/:id                  (catalog.write, soft-delete is_active=false)
```

`variant` fields: `id`, `product_id`, `name`, `sku`, `barcode`, `price_minor`,
`stock_quantity`, `is_active`. Variant `price_minor`/`stock_quantity` are
authoritative at POS: KDS/checkout locks the variant row, decrements **variant**
stock, and does not touch `products.stock_quantity` for variant lines.

### Lots (C2 — pharmacy/expiry)

```text
GET    /v1/products/:id/lots               (catalog.read; ?include_empty=)
POST   /v1/products/:id/lots               (catalog.write)
PATCH  /v1/lots/:id                        (catalog.write)
```

`lot` fields: `id`, `product_id`, `lot_number`, `expiry_date`, `serial_number`,
`quantity_remaining`, `is_active`. Products with `track_lots=true` consume
sales FEFO (`ORDER BY expiry_date ASC NULLS LAST` — un-dated lots are shelf-life
unlimited and used last), writing one `sale_items` row per lot.

## Sales

```text
POST /v1/sales
GET  /v1/sales
GET  /v1/sales/:id
GET  /v1/sales/:id/receipt        (JSON receipt payload)
GET  /v1/sales/:id/receipt/print  (application/vnd.escpos byte stream)
```

Restaurant (C1) — `POST /v1/sales`:
- `table_id` — optional; must reference an open (`status != closed`) table,
  404 `table_not_found` / 409 `table_not_available` otherwise; the table is
  marked occupied on completion.
- `split` — optional list of `{ "sale_id": "<source sale id>", "lines": [ { "sale_item_id": "<uuid>", "quantity": n } ] }`
  for split bills: items are attracted from the source sale, the group's totals
  sum across member sales, and each member keeps its own payments.
- `tip_minor` — optional gratuity, `>= 0`, summed into the response as
  `tips_minor` (visible on item-less totals, payments, and the receipt).

Pharmacy (C2) — sale line gains optional `lot_id`; the server picks lots FEFO
when omitted and echoes `lot_id`/`lot_number` per `sale_items` row.

Fashion (C3) — sale line gains optional `variant_id`; server locks and decrements
the variant, echoes `variant_id`/`variant_name`.

Receipt payload (JSON at `/receipt`): `tenant_name`, `tenant_address`,
`sale_id`, `status`, `created_at`, `cashier`, `device`, `floor_name`,
`table_name`, `customer_name`, `currency`, `items[]` (`name`, `sku`,
`quantity`, `unit_price_minor`, `total_minor`, `variant_name`, `lot_number`),
`subtotal_minor`, `discount_minor`, `tips_minor`, `total_minor`,
`payments[]` (`method`, `amount_minor`, `tip_minor`), `loyalty_points_earned`.
The `/print` variant returns the same content as ESC/POS bytes (32-col thermal
layout, QR, cut) for direct printer handoff.

## Sync

Pull endpoint streams tenant changes ordered by the monotonic `change_seq`.
Entities: categories, products, product_variants, product_lots, customers,
sales, customer_loyalty_log, register_sessions, inventory_adjustments,
floors, restaurant_tables, tenant_settings. Users, devices and sync_commands
are intentionally excluded (sensitive/internal).

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

`POST /v1/sync/push` applies replay-safe commands in one transaction.

```text
POST /v1/sync/push
{ "commands": [ { "command_id": "<uuid>", "operation": "sale.create", "payload": { ... } } ] }
```

- `command_id` — client-supplied UUID that identifies the intent; must be
  stable across retries.
- `operation` — currently `sale.create` only.
- `payload` — operation payload, forwarded verbatim to the handler.

Per-command `data.results[]`:
- `command_id` — echoed back for correlation
- `status` — `applied` (freshly executed), `replayed` (same `command_id` +
  same payload seen before; stored outcome returned, nothing re-applied),
  `conflict` (same `command_id` reused with a different payload →
  `command_conflict`, or a business conflict like `insufficient_stock`),
  or `rejected` (invalid/unrecognized, e.g. `unknown_command`).
- `replayed` — true when the result came from the stored record
- `result` / `error_code` / `error_detail` — operation outcome

Dedupe: an identical (device, operation, payload) seen under a fresh
`command_id` replays the stored outcome instead of applying twice. Batch cap:
100 commands per request (400 `validation_error` beyond that). All commands in
 a batch commit atomically.

## SaaS platform (operation-aware role)

All `/v1/saas/*` endpoints require an access token whose role is exactly
`saas_admin` (403 `forbidden` otherwise).

`GET /v1/saas/summary` returns platform-wide totals:

```text
{ "data": { "tenants": <n>, "users": <n>, "products": <n>, "revenue_minor": <n>,
            "by_business_type": { "<type>": { "tenants": <n>, "users": <n>, "revenue_minor": <n> } } } }
```

Tenant lists expose plan fields (`plan`, `max_users`, `max_products`) plus live
user/product counts:

```text
GET /v1/saas/tenants?page=1&limit=20
{ "data": [ { "id", "name", "slug", "business_type", "country_code", "currency_code",
              "default_language", "plan", "max_users", "max_products",
              "users": <n>, "products": <n> } ], "meta": { ...paginated... } }
```

Per-tenant drill-down (D2) runs inside one transaction with the tenant RLS
context set, so the counts/stats are scoped to that tenant:

```text
GET /v1/saas/tenants/:id/analytics
{ "data": {
    "tenant": { "id", "name", "slug", "business_type", "country_code",
                "currency_code", "default_language", "plan", "max_users", "max_products" },
    "counts": { "users": <n>, "products": <n> },
    "today": { "date", "revenue_minor", "sales_count", "avg_sale_minor", "items_sold" },
    "revenue_trend": [ { "day", "revenue_minor" } ],                        // last 7 days
    "top_products": [ { "product_name", "sku", "quantity", "revenue_minor" } ], // top 5 (7d)
    "recent_sales": [ { "id", "status", "total_minor", "currency", "payment_method", "cashier", "created_at" } ] } }
```

Errors: 400 `invalid_tenant_id` (malformed id), 404 `tenant_not_found`.

## Tenant plan limits

Migration `016_tenant_plan` gives every tenant `plan` (default `standard`)
plus `max_users` and `max_products`; a value of `0` (the default) means
unlimited. `POST /v1/users` and `POST /v1/products` enforce the caps *inside*
the tenant transaction (RLS-scoped count) and respond 409:

```text
{ "error": { "code": "plan_limit_exceeded", "message": "tenant user limit reached", ... } }
```

Billing / plan-change webhook ingestion is not yet implemented (external
provider is a backlog item); plan columns are currently updated directly by a
platform admin, e.g. `UPDATE tenants SET max_users = 5, max_products = 500 WHERE slug = 'demo-store';`

## Idempotency

Write commands that can create business state accept a tenant-scoped idempotency key.

A retry with the same key and semantically identical payload returns the original logical result.

A reused key with a different payload returns a conflict.

Client-supplied tenant headers are not an authorization mechanism.
