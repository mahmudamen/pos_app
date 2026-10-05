# 26 — Demo & testing accounts

Reference for the seeded demo shops used in demos, screenshots and manual QA.
**This file contains no passwords.** The plaintext credential sheet lives outside
git at `.secrets/testing/DEMO_CREDENTIALS.txt` (chmod 600, `.gitignore`d).

## Tenants

Five business verticals plus the SaaS platform tenant. Each demo tenant is
seeded with a small base catalogue (`scripts/seed_demo.sql`) *and* a rich
Egyptian-market catalogue of 184 products with real names, prices, EAN-13
barcodes and stock (`.secrets/testing/demo_products_seed.sql`).

| Tenant slug | Name | Vertical | Users | Products | Categories |
|---|---|---|---|---|---|
| `demo-restaurant` | Demo Restaurant | restaurant | 5 | 64 | 10 |
| `demo-grocery` | Demo Grocery | grocery | 5 | 59 | 7 |
| `demo-book-store` | Demo Book Store | book_store | 5 | 48 | 10 |
| `demo-computer-shop` | Demo Computer Shop | computer_shop | 5 | 39 | 6 |
| `demo-mobile-shop` | Demo Mobile Shop | mobile_shop | 5 | 35 | 8 |
| `saas` | SaaS | platform | 4 | 0 | 0 |

Tenant UUIDs are **not** stable — `scripts/demo_testing_reset.sql` drops and
recreates them. Read them from the database rather than hardcoding:

```sql
SELECT slug, id FROM tenants WHERE slug LIKE 'demo-%' OR slug = 'saas';
```

The Flutter login screen and every API call need `tenant_id` alongside the
email, so fetch it first:

```bash
curl -s -X POST https://posgo.xamltech.com/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"tenant_id":"<uuid>","email":"demo-restaurant-manager@xamltech.com",
       "password":"<see .secrets/testing/DEMO_CREDENTIALS.txt>",
       "device_id":"curl","device_name":"curl"}'
```

## Accounts

Five roles per demo tenant, all with `account_type=demo` except the guest. The
`guest` account carries `role=cashier` **and** `account_type=guest`, which is
what the RBAC checks key off — so it is not just a renamed cashier.

| Tenant | Email | Role | Account type | Display name |
|---|---|---|---|---|
| `demo-book-store` | `demo-book-store-cashier@xamltech.com` | cashier | demo | Senior Cashier |
| `demo-book-store` | `demo-book-store-guest@xamltech.com` | cashier | guest | Guest Cashier |
| `demo-book-store` | `demo-book-store-manager@xamltech.com` | manager | demo | Store Manager |
| `demo-book-store` | `demo-book-store-manager2@xamltech.com` | manager | demo | Second Manager |
| `demo-book-store` | `demo-book-store-owner@xamltech.com` | owner | demo | Store Owner |
| `demo-computer-shop` | `demo-computer-shop-cashier@xamltech.com` | cashier | demo | Senior Cashier |
| `demo-computer-shop` | `demo-computer-shop-guest@xamltech.com` | cashier | guest | Guest Cashier |
| `demo-computer-shop` | `demo-computer-shop-manager@xamltech.com` | manager | demo | Store Manager |
| `demo-computer-shop` | `demo-computer-shop-manager2@xamltech.com` | manager | demo | Second Manager |
| `demo-computer-shop` | `demo-computer-shop-owner@xamltech.com` | owner | demo | Store Owner |
| `demo-grocery` | `demo-grocery-cashier@xamltech.com` | cashier | demo | Senior Cashier |
| `demo-grocery` | `demo-grocery-guest@xamltech.com` | cashier | guest | Guest Cashier |
| `demo-grocery` | `demo-grocery-manager@xamltech.com` | manager | demo | Store Manager |
| `demo-grocery` | `demo-grocery-manager2@xamltech.com` | manager | demo | Second Manager |
| `demo-grocery` | `demo-grocery-owner@xamltech.com` | owner | demo | Store Owner |
| `demo-mobile-shop` | `demo-mobile-shop-cashier@xamltech.com` | cashier | demo | Senior Cashier |
| `demo-mobile-shop` | `demo-mobile-shop-guest@xamltech.com` | cashier | guest | Guest Cashier |
| `demo-mobile-shop` | `demo-mobile-shop-manager@xamltech.com` | manager | demo | Store Manager |
| `demo-mobile-shop` | `demo-mobile-shop-manager2@xamltech.com` | manager | demo | Second Manager |
| `demo-mobile-shop` | `demo-mobile-shop-owner@xamltech.com` | owner | demo | Store Owner |
| `demo-restaurant` | `demo-restaurant-cashier@xamltech.com` | cashier | demo | Senior Cashier |
| `demo-restaurant` | `demo-restaurant-guest@xamltech.com` | cashier | guest | Guest Cashier |
| `demo-restaurant` | `demo-restaurant-manager@xamltech.com` | manager | demo | Store Manager |
| `demo-restaurant` | `demo-restaurant-manager2@xamltech.com` | manager | demo | Second Manager |
| `demo-restaurant` | `demo-restaurant-owner@xamltech.com` | owner | demo | Store Owner |
| `saas` | `mahmudamen@xamltech.com` | saas_admin | standard | Mahmoud Amen (Owner) |
| `saas` | `osama@xamltech.com` | saas_admin | standard | Osama (Manager) |
## Databases

| | Local (dev box) | Production (VPS `197.44.6.42`) |
|---|---|---|
| Engine | native PostgreSQL on `127.0.0.1:5432` | `pos-prod-postgres` container (private network, no published port) |
| Database | `pos` | `pos` |
| Seed/migrate role | `pos_app` (owner) from `.env` | `pos_app` (owner); the app itself runs as least-privilege `pos_app_rls` |
| Goose version | 50 | 50 |
| Reachable from | the dev box | `docker exec pos-prod-postgres psql -U pos_app -d pos` |

Everything the API writes is `FORCE ROW LEVEL SECURITY`, so **a plain
aggregate count returns 0 rows** — even for the owner — until tenant context is
set. This bites when verifying a seed:

```sql
-- wrong: always 0
SELECT count(*) FROM users;

-- right: per-tenant, inside one session
SELECT set_config('app.current_tenant', '<tenant-uuid>', false);
SELECT count(*) FROM users;
```

## Reseeding

Order matters: the reset drops the tenants, so the seeds must follow it.

```bash
set -a; . ./.env; set +a          # DATABASE_URL is not auto-loaded
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/demo_testing_reset.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/seed_demo.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f ../.secrets/testing/demo_users_seed.sql
psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f ../.secrets/testing/demo_products_seed.sql
```

The reset is **deliberately scoped**: it only removes `demo-*` tenants and the
throwaway `*probe*` / `*walk*` tenants left behind by browser walkthroughs.
Real customer tenants are never touched. It clears `floors`, `product_lots`,
`product_variants` and `restaurant_tables` explicitly because those four FKs
to `tenants` are `NO ACTION` (every other child table cascades) — miss them and
the tenant delete is refused.

Preview it without writing anything by turning the final `COMMIT` into a
`ROLLBACK`:

```bash
sed 's/^COMMIT;$/ROLLBACK;/' scripts/demo_testing_reset.sql \
  | psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f -
```

## Regenerating credentials

```bash
python3 tool/gen_demo_testing_users.py    # -> .secrets/testing/demo_users_seed.sql + csv/json
python3 tool/gen_demo_products.py         # -> .secrets/testing/demo_products_seed.sql
```

Both outputs are idempotent upserts, so re-running them against an already
seeded database refreshes the passwords rather than duplicating rows.
