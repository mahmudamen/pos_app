# 25 — Launch readiness, and the last phase before it

Status: active. This is the gate list, not a feature list. Anything in
**Blockers** stops a release; anything in **Should** stops only if it is cheap.

Written after the defect sweep of 2026-09-28, in which four defects were found
that all shared one shape: **a feature was advertised, built or assumed, and
nothing enforced or even enabled it.** Three of the four are fixed in the tree
and land with the next deploy. The fourth is a decision, not a defect.

---

## 1. What was fixed, and what it cost

| # | Defect | Where | Fix |
|---|--------|-------|-----|
| 1 | Invoice capture shipped in the app but **disabled on production** — `OCR_ENABLED` defaults `false` and `docker-compose.prod.yml` never forwarded it, so `POST /v1/purchases/ocr` answered `503 ocr_unavailable` behind a manager-visible button | `docker-compose.prod.yml`, `.env.prod.example` | Forward the flag **and** the metering windows, with non-zero production defaults |
| 2 | The public site advertised a **Copilot that does not exist** — "always on", "24/7", "grounded in your store's real data" | `site_strings.go`, `site_strings_ar.go` | Rewritten to describe only what ships: on-device Tesseract, Arabic + English, human-confirmed |
| 3 | `plans.features` **gated nothing** — serialised into SaaS responses and rendered in `/pricing`, never read by a single branch, so a sold feature had no way to say no | `internal/plans` (new), `purchases/handler.go`, migrations `049` + `050` | A real entitlement check, enforced on `POST /v1/purchases/ocr` |
| 4 | OCR **audit trail discarded** — the backend persisted `match_score` and `row_text` for exactly this purpose; the Flutter client sent `0` and `''`, and never showed the merchant a confidence score | `purchases.dart`, `ocr_review_screen.dart`, scan-line JSON | Round-trip completed; sub-60 matches now flagged in the review UI |

### Why defect 3 was not fixed the obvious way

The obvious fix — add `ocr_capture` to only the paid plans — would have
satisfied the "gate it properly" instinct and broken the product. The metering
windows (`ocr_usage`, `ocr_credits_remaining`, migration `036`) are the intended
control surface and already answer `402` on exhaustion. Entitlement says *may
you*, metering says *how often*; enforcing both would mean a starter shop is
refused a scan it has paid for, and a bug in the meter locks a shop out of
stock-taking entirely.

So migration `049` grants `ocr_capture` to **every** seeded plan and the handler
gates on entitlement alone.

That decision is what exposed the `050` defect, and it is worth being explicit
about the ordering: granting rather than gating was the conservative choice, but
gating is what turned a dangling plan code into a customer-facing 402. The
enforcement was not wrong — the data was. If OCR becomes a price differentiator, the fix is
one `DELETE` from `starter`/`trial` in a new migration, with no handler change.
Recorded here because the next engineer will otherwise read the migration and
assume it was an oversight.

---

## 2. Blockers — a release does not ship until these clear

### B1. The Google Play listing 404s

`https://play.google.com/store/apps/details?id=com.xamltech.pos_go` returns
**404**. Two consequences, and they are independent:

- The `Get it on Google Play` link in the **live footer** of `posgo.xamltech.com`
  is a dead link on a production site.
- `CHANGELOG.md` in the public repo describes `1.0.0` as the first public
  release, which cannot be verified while the listing is not publicly listed.

Most likely uploaded and in review, or on a closed track. Google also varies by
region, so confirm from a phone in Egypt before acting. The footer already
carries a working `Download APK` link beside the broken one, so the honest
fallback is to lead with the APK rather than to build a new page.

*Needs a human answer: live, in review, or never uploaded.*

### B2. The Play upload key has no off-box backup

The single highest-severity item here. A lost upload key is a release-blocking
incident with no user-facing workaround: you cannot ship an update to an
existing install without it, and it does not expire. Every other line in this
document is a day of work. This is an afternoon of work, and it is the one that
protects all of them.

Three things were wrong with the instructions as written, and each would have
failed at the moment they were needed:

1. They told you to upload a `.zip.age` file that no procedure ever creates.
   The box has `gpg` and no `age`, so the archive is `.gpg`. Corrected
   everywhere, including the `offsite` staging notes.
2. The keystore **file** checksum was labelled the keystore "fingerprint". It is
   a `sha256sum` of the `.jks`; `keytool` never prints it, so anyone comparing
   it against certificate output would conclude the backup was corrupt. The
   three values — certificate fingerprint (public, in every APK), keystore file
   checksum, and archive checksum — are now named separately.
3. Nothing ever checked the backup. A backup that has never been restored is a
   guess.

`Flutter/pos_go_app/play_store/verify_backup.sh` now exists for that last point.
It needs no passphrase to check the live key, and with the archive it proves the
archive decrypts to *this* keystore:

```bash
Flutter/pos_go_app/play_store/verify_backup.sh                      # local key
GPG_PASSPHRASE='…' …/verify_backup.sh .secrets/play/offsite/pos-go-play-keys.zip.gpg
```

All paths through it are tested: correct passphrase, wrong passphrase, a
missing archive, and the one that matters — an archive that decrypts to a
*different* keystore, which it flags loudly rather than reporting success. The
keystore file checksum is read from the gitignored `credentials.txt`
(`JKS_SHA256:`) rather than hardcoded in the tracked script, so a value derived
from a secret never enters git history.

**What still cannot be done from here:** `.secrets/play/offsite/` is a copy on
the same NVMe — same device id, different inode. It survives an accidental
`rm`, and nothing else: not disk failure, not theft, not ransomware. One
encrypted copy has to physically leave this machine, into storage you control.
Until that has happened, treat this item as open.

### B3. Eight commits are unpushed

`pos` is one commit ahead, `pos-go` is five. Nothing in this document is
visible to the world until both are pushed.

### B4. Migrations `049` and `050` — `049` verified, `050` mandatory

Both are forward-only and additive. `049` rewrites the `plans.features` JSONB
array; `050` inserts two catalog rows. **Neither adds a table, so neither needs
a `grants_prod.sql` change.**

`049` is **applied and verified** against a real Postgres: 36/36 plan rows now
carry `ocr_capture`, the `Down` removes it from all 36, and a re-`up` restores
all 36. Dedupe-guarded, so both directions are idempotent.

`050` is the one that matters, and it exists because `049`'s enforcement landed
on a latent data defect:

> `tenants.plan` has defaulted to the literal `'standard'` since migration
> `016`, but **no row with code `standard` was ever inserted into `plans`.**
> 3,187 rows in the dev database hold it. Because features were display-only
> until now, nothing noticed. The moment `/v1/purchases/ocr` started resolving
> entitlements with a `LEFT JOIN` and failing closed, **every existing customer
> would have been refused invoice capture** with a 402 that read like a genuine
> plan gap.

`050` seeds `standard` and `test` with `is_active = FALSE`, so they resolve for
entitlements while staying off the public pricing page (which selects
`WHERE is_active`). Verified: all 3,786 tenant plan codes now resolve, and the
active plan set is unchanged.

**Apply `050` before the API that reads it.** On a database where the only
plans are the seeded ones, expect roughly a 3,000-row `UPDATE`. Confirm with:

```sql
-- must return zero rows
SELECT DISTINCT t.plan FROM tenants t
  LEFT JOIN plans p ON p.code = t.plan WHERE p.code IS NULL;

-- must return only sellable plans
SELECT code FROM plans WHERE is_active ORDER BY price_minor, code;
```

Three DB-backed invariants in
`internal/transport/purchases/plan_entitlement_integration_test.go` now hold
this in place: every tenant plan code resolves, the legacy tiers stay
unsellable, and the default tier is entitled to invoice capture. The first was
verified non-vacuous by deleting the `standard` row inside a rolled-back
transaction and watching the predicate report it.

### A correction to this document

An earlier draft of this file said the catalog seeds "4 plans". It seeds 6
sellable ones in the dev database — `starter`, `trial`, `tiny`, `business`,
`enterprise`, and `cafe` — plus 30 throwaway `cafe18d…` rows from load
testing, and `AGENTS.md` claims 21. None of those numbers is load-bearing, but
the discrepancy is itself worth a look before launch: it suggests plan rows and
tenant plan codes have drifted apart for a long time, which is exactly the
condition `050` repairs.

---

## 3. Verify after deploy — not before

These cannot be proven from the dev box, and each has bitten before.

- **OCR actually works on production.** Needs a real tenant token; the
  unauthenticated probe only proves the route is routed.
  `docker exec pos-prod-api tesseract --list-langs` must print **both** `ara`
  and `eng` — the image installs `tesseract-ocr-data-ara` explicitly and relies
  on `tesseract-ocr` pulling in the English pack. If `eng` is absent, every
  English invoice silently degrades. Do not assume it.
- **Metering is not unbounded.** Confirm `OCR_DAY_LIMIT` etc. resolved to
  `100/500/2000` in the running container. Each scan is synchronous on the
  request goroutine, so an unlimited tenant can saturate a core and stall the
  API for every other merchant.
- **The site no longer claims a Copilot.** Fetch `/` and `/?lang=ar`, and
  confirm the section describes invoice capture, not an assistant. The Arabic
  copy is a separate string table and has no test coverage.
- **The footer GitHub link is live** (`18ccaa2`).
- **The App Bundle is re-uploaded** from the fixed source, since the signing
  key and the OCR entitlement both changed.

---

## 4. Should — cheap, and they are the difference between a launch and a demo

- **Delete the dead `oc/capture` display labels.** `site_plans.go` defines
  `pos.advanced`, `e_invoice`, `refunds`, `discount_policy`, `billing` and
  `discount_policy` on top of `ocr_capture`; no seeded plan carries them and no
  code reads them. They render nowhere today, but they are a list of promises
  with no implementation behind them, which is the exact failure this document
  exists to prevent.
- **Surface the entitlement in the API response** so the client can say "not in
  your plan" instead of showing a raw `402`. `plans.Missing` is written and
  unused; wiring it into the error body is a few lines.
- **Reconcile the plan catalog with reality.** The catalog, the tenant plan
  codes and `AGENTS.md` all disagree (see the correction below). `050` repairs
  the dangling codes; the remaining `cafe18d…` load-test rows should be swept
  so nobody reads a 40-row catalog as a product decision.
- **Add `plan_limit_exceeded` to the Flutter map** so it becomes a human
  sentence rather than a status code.
- **Reconcile the two public narratives.** The site now describes invoice
  capture honestly and `pos-go/ROADMAP.md` says the assistant is "not started".
  Consistent as of this commit — keep it that way, because the site is the one
  that gets read by customers and the roadmap is the one that gets read by
  engineers.

---

## 5. The last phase after launch: a Copilot that can be true

Scope deliberately narrow, and **not** a chat box.

**Facts come from SQL. Language comes from a model. Never the reverse.** A
number about a shop's money must never be generated — it must be queried. This
is the same conclusion `Backend/local_ai_pos_ocr_rag_agent_architecture.md` §31
reaches, and it is the only design that fits the constraints this product
actually has:

- **Works offline.** Tesseract runs in-container with no egress
  (`Dockerfile:14`). An LLM call is a network dependency, which the product
  has otherwise engineered entirely out of itself.
- **No per-query cost.** Margins here do not support per-token billing for an
  SME till.
- **Never hallucinates a number.** The failure mode of an LLM over a
  financial system is not a bad sentence, it is a confidently wrong total.
- **Arabic-first, and genuinely so.** Not a translation layer over an English
  answer.

First three questions, all answerable from tables that already exist
(`register_sessions`, `sale_payments`, `sale_items`, `products`,
`inventory_adjustments`):

1. How much cash should be in the drawer right now, and why does it differ?
2. What sold badly this month, and is that because it was out of stock?
3. What did this cashier discount, and under which policy?

Preconditions, in order: `internal/plans` must gain a `copilot.*` feature key
(it now exists to be extended), `permissions.go` a `copilot.read` grant, and
`settings.go` a `copilot.enabled` key. All three are one-line additions to
tables that already exist, which is why this is achievable and why it was not
achievable before this commit.

A model may be added to *phrase* the answer, with a deterministic no-model
fallback so the feature degrades instead of dying. The fallback is not a
simplification; it is the offline path.

### Explicitly out of scope

- Per-tenant white-labelling of the app. The Belgian comparator sells this as
  its central promise and it is a genuine gap, but a theming layer on every
  screen plus a receipt re-test is not a launch-week change.
- Anything that makes credits **purchasable**. Google Play Billing becomes
  mandatory the instant an app can spend real money; mobile wallets have to be
  reconciled and it is a billing project, not a feature.
- RAG over documents. The 1826-line architecture document is a design, not
  evidence, and nothing in it is built except §6/§9/§10/§11.
