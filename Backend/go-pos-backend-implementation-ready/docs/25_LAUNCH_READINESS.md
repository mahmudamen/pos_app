# 25 — Launch readiness, and the last phase before it

Status: **blockers B4 cleared, all five "Should" items done, three new defects
found by the post-deploy checks and fixed.** What is left is what a machine
cannot do: two human answers (B1, B2) and a push (B3).

This is the gate list, not a feature list. Anything in
**Blockers** stops a release; anything in **Should** stops only if it is cheap.

Written after the defect sweep of 2026-09-28, in which four defects were found
that all shared one shape: **a feature was advertised, built or assumed, and
nothing enforced or even enabled it.** Three of the four are fixed in the tree
and land with the next deploy. The fourth is a decision, not a defect.

---

## 0. What the post-deploy pass found (2026-09-28, all fixed)

The checks in §3 exist because "deployed" and "working" are different claims.
Running them for real found three more instances of the exact same shape as the
original sweep — assumed, not verified — and one of them was a crash on the
primary-language path.

| # | Finding | Evidence | Fix |
|---|---------|----------|-----|
| 5 | **`tesseract` had no English pack.** `docker exec pos-prod-api tesseract --list-langs` printed `ara` alone. The Dockerfile installed `tesseract-ocr-data-ara` and relied on `tesseract-ocr` to bring the English data along, which on alpine it does not. Every English invoice degraded while Arabic scanned fine | live container, pre-fix | `Dockerfile` installs `tesseract-ocr-data-eng` explicitly and **asserts both `.traineddata` files exist at build time**, so this cannot regress silently again. Verified live: `ara` + `eng`, and a real English invoice scan returns 200 with parsed lines (`Cement 50kg bag 20 x 185.00` → qty 20, 18500 minor, score 100) |
| 6 | **Arabic invoice scans returned 500.** `POST /v1/purchases/ocr` panicked: `runtime error: index out of range [-1]` at `internal/ocr/parser.go:215`. `qtyPos` and `totalPos` were both guarded with `>= 0`; `pricePos` was not, and either multiply branch can leave it at `-1` on a row where OCR mangles the digits around `×`. The Recovery middleware turned the panic into a bare `internal server error` | prod logs + reproduced | `parser.go` refuses to build an item with no price and reports the row verbatim in `Discarded` for human review — a fabricated price is worse than a flagged row. `TestParseInvoiceNeverPanicsOnUnpriceableRows` covers ten shapes and **panics again if the guard is removed** |
| 7 | **Production had no backups at all.** The runbook claimed a daily cron; there was no cron entry, no systemd timer and an empty backup directory, on the box that had just taken two migrations. `scripts/backup.sh` could not have worked there anyway: the database is on a private docker network with no published port | live VPS | `/opt/pos/backup_daily.sh` + `/etc/cron.d/pos-backup` (03:17), dumping **inside** the postgres container, with a `gzip -t` integrity gate so a truncated dump is never rotated in. Run once by hand: 116K, integrity OK. `docs/11_OPERATIONS.md` now documents why the script cannot run on that host |

**A fourth finding is *not* fixed, because nothing can fix it from here:** the
app requests `POST_NOTIFICATIONS` and registers an FCM token on first launch,
but the backend has **no send path** — `grep -r "firebase\|fcm\|messaging/send"`
over `internal/` returns nothing. The merchant is asked to consent to
notifications that can never arrive. Either ship a sender or stop asking; the
permission and the Data safety form should not imply a feature that does not
exist. Recorded because the Play listing upload is the moment it becomes
visible to Google.

Also fixed while in the same file: the live `/pricing` page was leaking the raw
key `pos.subdomain` in five places, because migration `041` grants that feature
to every plan and `siteTranslations` had no entry for it. Now labelled
"Your own store subdomain" / "نطاق فرعي خاص بمتجرك", and the six unshipped
labels the Should list flagged (`pos.advanced`, `e_invoice`, `self_order`,
`refunds`, `discount_policy`, `billing`) are gone.

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

### B3. Commits are unpushed — blocked on a key, not on the work

`pos` is ahead of `origin` and `pos-go` is five commits ahead. Nothing in this
document is visible to the world until both are pushed.

The blocker is authentication, not anything in the tree: this box's only SSH
key is the VPS deploy key, which GitHub does not know, and `gh` is logged out.

```bash
gh auth login && gh auth setup-git      # or add a GitHub-authorized key
git fetch origin && git rebase origin/origin   # remote has an equal-but-differently-hashed twin
git push origin origin
```

### B4. Migrations `049` and `050` — **CLEARED on production 2026-09-28**

Production is at goose **50**. `ocr_capture` is on 12/12 plans, the sellable
catalog is unchanged (`starter, trial, business, enterprise`), and the dangling
`standard` code that would have 402'd every existing customer is resolved. The
history below is kept because the ordering constraint is the part worth
keeping.

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

Run 2026-09-28 against the deploy that took migrations to 50. Every line is a
check that was actually executed, not an intention.

- [x] **OCR actually works on production.** Logged in as a real tenant
      (`admin@demo-restaurant`) and POSTed a generated invoice to
      `/v1/purchases/ocr`. English: **200**, rows parsed with prices and a
      match score. Arabic: **200** after fix 6, previously 500.
- [x] **Both tesseract languages are present.** `ara` and `eng` — after fix 5.
      *This is the check that would have caught the missing English pack.*
- [x] **Metering is not unbounded.** The running container reports
      `OCR_ENABLED=true`, `OCR_DAY_LIMIT=100`, `OCR_WEEK_LIMIT=500`,
      `OCR_MONTH_LIMIT=2000`.
- [x] **The site no longer claims a Copilot.** `/` and `/?lang=ar` contain no
      "always on", "24/7" or "grounded in your store's real data". The Arabic
      copy explicitly names the drawer/Z-report assistant as *the next release*.
      The remaining "copilot" hits are CSS class names.
- [x] **The footer GitHub link is live** (`18ccaa2`) →
      `https://github.com/mahmudamen/pos-go`.
- [x] **Migrations to 50**, dangling plan codes: none, sellable catalog
      unchanged (`starter, trial, business, enterprise`), `ocr_capture` on
      12/12 plans, `standard` resolves again.
- [x] **`scripts/verify_prod.sh` green** — both vhosts, TLS, the HTML pages and
      a real login, 13/13.
- [ ] **The App Bundle is re-uploaded** from this source, since the signing key
      and the OCR entitlement both changed. *Still needs a human at the console.*
- [ ] **OCR reviewed on a real phone.** The curl scan proves the server; it does
      not prove the on-device camera → blur check → review flow. Needs the
      emulator or a handset.

---

## 4. Should — cheap, and they are the difference between a launch and a demo

All five are done (2026-09-28).

- [x] **Delete the dead display labels.** `site_plans.go` no longer carries
      `pos.advanced`, `e_invoice`, `self_order`, `refunds`, `discount_policy`
      or `billing`: no seeded plan grants them and no code reads them. The
      reverse mistake was fixed at the same time — `pos.subdomain` *is*
      granted to every plan and had no label, so the live pricing page printed
      the raw key.
- [x] **Surface the entitlement in the API response.** The 402 body now carries
      `missing_features`, so the client can say "not in your plan" instead of
      showing a status code. `plans.Missing` is no longer dead code.
- [x] **Reconcile the plan catalog with reality.** All 34 `cafe…` load-test
      rows are swept from the dev database (`cafe18d…` plus the `cafe` plan
      itself — `grep` confirms no migration seeds either; the only `cafe` in
      `migrations/` is the Nescafe product row). Production never had them: 10
      rows, 4 sellable. Dev is now 7 rows / 5 sellable, matching the catalog.
- [x] **`plan_limit_exceeded` in the Flutter map.** `mapApiError` is
      code-aware and 402s become a typed `PlanLimitException`; `ApiException`
      gained `isPlanLimit` / `isScanLimit`; the purchases screen shows a
      localized sentence (Arabic included) and offers the credit top-up when
      the meter is the thing that ran out.
- [x] **Reconcile the two public narratives.** The site describes invoice
      capture honestly, in both languages, and names the assistant as the next
      release. Verified live, not just in the source.

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
