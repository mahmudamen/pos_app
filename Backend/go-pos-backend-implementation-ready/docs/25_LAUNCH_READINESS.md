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
| 3 | `plans.features` **gated nothing** — serialised into SaaS responses and rendered in `/pricing`, never read by a single branch, so a sold feature had no way to say no | `internal/plans` (new), `purchases/handler.go`, migration `049` | A real entitlement check, enforced on `POST /v1/purchases/ocr` |
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
gates on entitlement alone. If OCR becomes a price differentiator, the fix is
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

### B3. Eight commits are unpushed

`pos` is one commit ahead, `pos-go` is five. Nothing in this document is
visible to the world until both are pushed.

### B4. Migration `049` has never been applied

Forward-only, additive, no new tables, so **no `grants_prod.sql` change is
needed** — it only rewrites the `plans.features` JSONB array. Run it before the
API that reads it, and confirm with:

```sql
SELECT code, features FROM plans ORDER BY code;
```

Every row should contain `ocr_capture`. A tenant whose row does not get
`402 plan_limit_exceeded` on invoice capture.

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
- **Fix the AGENTS.md plan count.** It claims 21 plans; migrations seed 4.
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
