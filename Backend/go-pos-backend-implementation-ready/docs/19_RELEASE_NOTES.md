# Release Notes — POS Go v1.0

**Date:** 2026-09-15
**Scope:** First public release, deployed to production on the VPS.

---

## Merchant onboarding (new)
- First-use flow: welcome → pick a business vertical (7: coffee_shop,
  restaurant, retail, book_store, mobile_shop, computer_shop, grocery) →
  signup → **immediate auto-login** with remember-me.
- New store is provisioned instantly: owner account (`owner`/`standard`),
  tenant on a **15-day trial** (`plan=trial`, `trial_ends_at`), demo catalog
  seeded with **10 real products per vertical** — EGP prices, curated images,
  product descriptions, and check-digit-valid **EAN-13 barcodes**
  (`622` prefix + vertical class + running number), plus matching categories.
- Cross-tenant email uniqueness via the `store_emails` table, enforced
  transactionally in the same commit — duplicate registrations return
  **409 `email_taken`**. Unsupported business types return 400.
- **Trial expiry enforcement**: logins for a tenant past `trial_ends_at` are
  blocked with **403 `trial_expired`** and a renewal message. Admin can extend
  the trial by updating the tenant row.

## Checkout & operations (already shipped, surfaced in this release)
- Full POS loop validated on-device: register open → split payments
  (cash/card/mobile) → sale (idempotent) → receipt (JSON + ESC/POS print).
- Offline sale queue revisited via `/v1/sync/push` (replay-safe, batching ≤10).
- Refunds, discount caps w/ manager PIN, loyalty accrual, register Z-report.

## Catalog data fix
- Product responses now include `description` on **all** read paths
  (list, barcode lookup, create, get, patch) — previously seeded but not
  returned. Flutter `Product` model parses it.
- Demo catalogs regenerated with valid 13-digit EAN-13 barcodes.

## Platform / deployment (VPS 197.44.6.42)
- Docker Compose stack: `pos-prod-api`, `pos-prod-postgres` (16-alpine),
  `pos-prod-redis` (7-alpine); API now bound to **127.0.0.1:8080** only.
- nginx reverse proxy serves the API over **HTTPS (443)** with HTTP→HTTPS
  redirect; **trusted Let's Encrypt certificate installed + auto-renew**
  (Cloudflare DNS `api.xamltech.com → 197.44.6.42` now live).
  Security headers come from the Go middleware only; nginx adds none.
- ufw default-deny; direct `:8080` exposure removed; fail2ban active.
- DB migrations to **v26**: `store_emails` (+ least-privilege grants to
  `pos_app_rls` so the app role can enforce signup uniqueness).
- Bug fixed during rollout: signup failed under the least-privilege DB role
  (`permission denied for table store_emails`) — resolved by granting
  INSERT/SELECT to `pos_app_rls` in `scripts/grants_prod.sql`.

## Backend quality gates for this release
- `gofmt` clean, `go vet clean`, full Go test suite green
  (DB-backed integration incl. auth/sales/sync/saas).
- Flutter: `flutter analyze` 0 issues, 135 unit tests passing.
- On-device integration E2E green against **production** through an SSH tunnel:
  login → browse → sell → receipt → logout, plus onboarding →
  trial store → catalog → relogin → sale.

## Known limitations / next
- Public HTTPS is live (`https://api.xamltech.com`, trusted cert).
- Android app currently built for debug distribution; release signing pending.
- Billing / plan upgrades are API stubs (plan limits enforced); subscription
  billing integration is the next phase (see `docs/FLUTTER_WEB_SAAS_CONTROL_PLAN.md`).