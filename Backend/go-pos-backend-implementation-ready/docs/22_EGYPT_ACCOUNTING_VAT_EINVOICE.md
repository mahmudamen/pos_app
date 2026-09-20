# Egypt VAT + Accounting Core + ETA e-Invoice / e-Receipt Integration — Plan

Status: plan (design) — Phase 0, no code shipped for this workstream yet.
Owner: backend team. Mirrors `docs/13_ROADMAP.md` Phase E ("Egypt tax & accounting");

This document is the authoritative plan for adding (1) an accounting/GL core,
(2) a VAT engine (per Law 67/2016), and (3) direct integration with the Egyptian
Tax Authority (ETA) central clearance platform for **e-invoices** (B2B/B2G) and
**e-receipts** (B2C), to the Go backend that already powers the POS/Flutter SaaS.

## 1. Legal & regulatory context (verified 2026)

- **VAT**: Value-Added Tax Law No. 67/2016. Standard rate **14%**. EGP only.
- **E-invoicing (B2B/B2G)**: all VAT-registered businesses must push each sales
  invoice to the ETA central platform **in real time (same day)**, digitally
  signed, before it is legally valid. ETA clears it and returns a **UUID + QR**.
  Paper invoices no longer support input-VAT deduction since July 2023
  (Law 206/2020 + executive decrees).
- **E-receipts (B2C)**: a **separate track** on the same platform. Issued from
  registered **POS devices** (each device gets an ETA-registered serial + device
  PIN). Reported within **24–72 hours** of the sale. Since Jan 2026 enforcements:
  every printed e-receipt must show a **QR** linking to the validated record.
- **Threshold**: mandatory registration dropped from EGP 500k → **EGP 250k**
  gross annual revenue (Resolution 281/2025; deadline was **31 Mar 2026**).
  Above it, registration is compulsory. Below it the merchant can run the POS
  without ETA (this must be a first-class "not registered yet" mode).
- **Penalties (2026 enforcement stage)**: EGP 20,000 fine for missing
  registration + EGP 1,000/day; tiered regime up to suspending the right to issue
  valid documents; QR-less printed receipts are themselves a violation.
- **Product coding**: every invoice/receipt line needs **GS1 / EGS / GPC code**
  for the item. Missing codes are the #1 technical blocker at go-live.
- **eSeal**: X.509 certificate issued by a licensed CA (Egypt Trust) or a cloud
  signing service; used to sign every document. No eSeal → no submission.
- **Retention**: minimum 5 (advised 7) years of the signed documents + records.

The two tracks are **operationally separate**; a retailer selling B2B and B2C
must do both. Plan supports both with shared machinery.

## 2. Scope

### In scope
- Accounting sub-ledger (GL) core: double-entry journal + minimal COA, running
  balances per tenant, GL account types, posting from sales/purchases/payments.
  No full ERP: we are a POS SaaS, not QuickBooks. "Accounting core" here = the
  books a POS needs to (a) prove VAT, (b) reconcile with ETA, (c) produce the
  monthly VAT return inputs, (d) track cash.
- VAT engine: per-line VAT classification (standard 14%, zero, exempt, other
  ETA subtypes), discount pro-ration, rounding, tax-inclusive street prices,
  per-sale tax breakdown, credit notes on refunds.
- ETA integration: identity (OAuth client-credentials session token), eSeal
  signing (canonical JSON + RSA-SHA256), document build (Invoice v1.2/v1.3 +
  Receipt v1.2), submission (e-invoice clearance + e-receipt 24–72h), chunking,
  retry/outbox, status polling, notifications webhook, document cancel/credit
  note, receipt lookup + anonymous QR verification.
- Merchant onboarding of ETA profile + TRN/activity/commercial register +
  branch/device registration + GS1/EGS catalog mapping.
- Monitoring/reporting: VAT sales register, VAT purchase register, monthly VAT
  return inputs, submission dashboard (pending → submitted → accepted/rejected),
  recon report (POS sales vs ETA-accepted docs).
- Config surface repurposed from the existing `tenant_settings` JSON.

### Out of scope (Phase 0)
- Full ERP / payroll / fixed assets / audit trail UI.
- Cross-border / reverse-charge / multi-currency exports (imported purchases are
  downstream; note in decision log).
- E-archive (e.g. document archival API) — later phase; we keep signed payloads
  locally which covers the retention duty.

## 3. Current-state gap analysis

What exists today in `Backend/go-pos-backend-implementation-ready`:

- `products` (003_core): no VAT fields, no EGS/GS1 codes, no unit taxation flag.
- `sales`/`sale_items`: a dormant `tax_minor BIGINT DEFAULT 0` column exists but
  is never written (today `total = subtotal − discount`, no taxes). No taxable
  amount, no tax-rate split.
- `tenants`: `country_code`, `currency_code`, `address`; **no TRN / activity
  code / commercial registration / VAT flags / branch/device ETA identity**.
- `devices` table exists (tenant, id, uuid) but no ETA device serial / PIN.
- `customers`: name/email/phone only — **no TRN for B2B buyers**.
- Self-order + product requests + webpush infra exist (already shipped) — the
  same merchant printer flow is the natural home for the receipt QR.
- Purchases vertical (migration `035_purchases`): supplier invoices with OCR
  scanner — natural input for the **purchase ledger + input-VAT register**.
- All money is integer minor units; `currency CHAR(3)` defaults `'EGP'` for
  Egypt tenants — good fit for exact tax math.

**Consequence**: this workstream needs real migrations; it is the biggest schema
change since RLS. All new tables follow the house rules (RLS + FORCE, `change_seq`
bump triggers where in the sync feed, grants in `scripts/grants_prod.sql`).
E-invoice documents are **observability/business state**: receipts and bookkeeping
rows DO stream to sync pull; ETA submission envelopes are transactional records
kept out of the sync feed (like `client_events`) — decision log item.

## 4. Target architecture

```
 POS sale (CreateSale)
      │  already-tx
      ▼
 [VAT engine] ── taxes per line + per sale (pure, unit-tested)
      │  enriched sale rows (sales.tax_minor, sale_items.tax_*)
      ▼
 [Document builder]  sale → ETA JSON (Invoice v1.2/1.3 | Receipt v1.2) + QR data
      │
      ▼
 [eSeal signer]  canonical JSON → RSA-SHA256 (base64)  (crypto, no hardware)
      │
      ▼
 [Submission outbox]  eta_documents (pending → submitted → accepted|rejected)
      │   chunked POST (≤100 docs)  +  async polling  +  webhook receiver
      ▼
 [ETA central platform]  (preprod then prod) ──► UUID/QR/status files back
      │
      ▼
 [Accounting core]  journal entries (GL) + VAT registers + monthly return inputs
      │
      ▼
 [Reports + admin UI]  submission dashboard, recon, VAT return worksheet
```

Key invariants:
- The POS sale always succeeds offline (registers/sync model unchanged).
  ETA compliance is **eventual** via the outbox: e-invoice "same-day" and
  e-receipt "24–72h" are satisfied by a DB job that drains the outbox; a
  night/periodic job catches the 72h limit with alerting.
- Tax is computed **once**, at sale time, by the pure VAT engine — never by the
  client. The client only sees `prices_are_inclusive` flag + totals + the receipt
  QR.
- The outbox is the single source of truth for submission status; the ETA
  server-assigned UUID/QR get captured and echoed on receipts.

## 5. Data model (new migrations 040+)

`040_eta_taxonomy.sql`
- `tax_categories(id, code, subtype, name_ar, name_en, rate_bp INT, is_vat,
  is_recoverable, exempt_reason, seq)` — seeds: `standard-14` (T1, V001, 1400),
  `zero` (V002), `exempt` (V003 + reason), `reduced/other` placeholders.
- `products`: add `vat_code_id FK tax_categories`, `egs_code TEXT`,
  `gs1_code TEXT`, `tax_included_price BOOL NOT NULL DEFAULT TRUE`.
- `customers`: add `tax_id TEXT` (TRN, for B2B) + `customer_type 'consumer'|'business'`.

`041_eta_identity.sql`
- `tenants`: add `tax_id TEXT`, `commercial_reg TEXT`, `activity_code TEXT`,
  `vat_registered BOOL NOT NULL DEFAULT FALSE`, `eta_enabled BOOL DEFAULT FALSE`,
  `default_vat_code`, `eta_client_id TEXT`, `eta_client_secret_enc TEXT`,
  `eta_branch_code TEXT` (branchCode), `eta_legal_type TEXT`.
- `devices`: add `eta_device_serial TEXT`, `eta_device_code TEXT`,
  `eta_pos_pin_enc TEXT` (device PIN assigned when ETA registers the POS).
- Secrets stored **encrypted at rest** (AES-GCM with a key in env
  `ETA_SECRET_ENC_KEY`), never plaintext in DB (see `docs/06_SECURITY.md`).

`042_eta_documents.sql` (submission outbox)
- `eta_documents(id, tenant_id, sale_id FK, document_type 'invoice'|'receipt'|
  'credit_note'|'debit_note', type_version, branch_code, device_id,
  receipt_number/seq, status 'pending'|'submitted'|'accepted'|'rejected'|
  'cancelled', payload JSONB (unsigned), signature TEXT, canonical_string TEXT,
  eta_uuid TEXT, submission_id TEXT, qr_value TEXT, error_code, error_message,
  attempts INT, next_attempt_at, submitted_at, accepted_at, created_at,
  UNIQUE(tenant_id, sale_id, document_type))`. RLS FORCE, tenant policy,
  index `(tenant_id, status, next_attempt_at)`. **Excluded from sync pull feed**.

`043_gl_accounting.sql` (accounting core)
- `gl_accounts(id, tenant_id, code TEXT, name TEXT, type
  asset|liability|equity|income|expense, is_active, parent_id)` + seeds
  (cash, sales revenue, sales discount, input VAT, output VAT payable, COGS,
  inventory, purchases, accounts receivable, bank…).
- `gl_journals(id, tenant_id, ref_type 'sale'|'refund'|'purchase'|'register'|'adj',
  ref_id, currency, posted_at, description, created_by)` +
  `gl_journal_lines(id, tenant_id, journal_id, account_id, debit_minor,
  credit_minor, created_at)`; debits=credits enforced in-app + CHECK-free
  (no `DO $$` in migrations — violation adds to `scripts/grants_prod.sql`).

Migration policy: numbered single-file goose, RLS + FORCE on every tenant table,
grants for `pos_app` (dev) and `pos_app_rls` (prod) via `scripts/grants_prod.sql`,
no `DO $$…$$` blocks.

## 6. VAT engine (pure Go, `internal/accounting/vat.go`)

- Prices are **VAT-inclusive street prices** (Egypt retail norm). The engine
  splits each line into `exclusive_amount`, `vat_amount` and per-tax breakdown.
- Rate lookup from `tax_categories.rate_bp`; default standard 14%.
- Discounts: pro-rate across lines by pre-discount subtotal, and mutually
  exclusively within a line (discount cannot exceed the taxable base).
- Rounding: EGP minor-units, **per-line round-half-up then sum** (decision log:
  sum-then-round can differ by 1 minor unit and fails ETA line/totals equality —
  we validate with the ETA preprod and pin the choice in contract tests).
- Refunds → credit note / receipt cancel: reverse lines at original rates.
- Outputs: `line.Breakdown{code, subtype, rate_bp, taxable_minor, tax_minor}` and
  `Sale.Totals{taxable_minor, tax_minor, exempt_minor, gross_minor}`.
- Surface on API responses (`GET /v1/sales/:id`, receipts, dashboard) without
  breaking existing clients: additive fields only.

## 7. ETA integration (`internal/eta/`)

### 7.1 Identity & session
- OAuth 2.0 client-credentials to obtain short-lived **access token**
  (`POST /api/v1.0/auth/token`); store/refresh, Redis-cache the token, refresh on
  401. Config from tenant ETA profile; **preprod** base
  `https://api.preprod.invoicing.eta.gov.eg` vs prod `https://api.invoicing.eta.gov.eg`.

### 7.2 Signing (eSeal)
- `.pfx`/`.p12` certificate (or cloud-signing key ref) + password from env /
  tenant secret store. Canonical form per ETA SDK ("canonical string",
  deterministic key order, nested `key.subkey`, arrays `key[index]`, trimmed
  values) → UTF-8 → **RSA-SHA256** → base64 → document `signatures[0].value`.
- Keep the canonical-string + signature builder **byte-level unit tested**
  against ETA's official sample documents.

### 7.3 Document builder
- Invoice/credit/debit (1.2/1.3): issuer (TRN `rin`, name, branchCode, activity,
  address), B2B buyer (TRN mandatory), lines with `internalCode`+`itemsCode`
  (EGS/GS1/GPC), taxableItem (taxType/subType/rate/amount), discounts, totals
  (taxable, tax, gross, net), `extraDiscountAmount`.
- Receipt (1.2, JSON): `receiptType 's'`, `typeVersion '1.2'`, `receiptNumber`
  per branch, `dateTimeIssued` UTC, `issuer{rin, companyTradeName, branchCode,
  branchAddress, deviceSerialNumber, activityCode}`, `receiver` (simplified),
  items + taxes, `documentTotals`, `paymentMethod` codes, and the signature +
  `customerName` etc. as per v1.2 requirements.
- Receipt numbering: per-branch sequence from `eta_documents` (max+1) — unique.
- **QR**: UTF-8 QR payload → rendered already by the ESC/POS builder in
  `internal/transport/receipts/escp.go` (extend payload to include the ETA QR
  string; JSON/print receipts echo it).

### 7.4 Submission flows
- Chunked submission: `POST /api/v1.0/documentsubmissions` with ≤100 docs →
  returns `submissionId`; poll `GET /api/v1.0/documentsubmissions/{id}` or the
  documents endpoint until per-doc acceptance; accept async (a background worker
  drains the outbox).
- Status transitions: pending → submitted → accepted (store `eta_uuid` + QR) /
  rejected (store `error_code`+`message`, surface to merchant, allow
  fix-and-resubmit as a **new document** if the draft is correctable, credit
  + reissue otherwise).
- Backoff/retry: exponential with `next_attempt_at`, cap attempts, alert on
  doomed ones (e.g. a 409 conflict) — the 72h e-receipt deadline is a business
  alert (SaaS dashboards), not a code failure.
- **Webhook receiver**: optional `ETA_NOTIFICATIONS_URL` endpoint that receives
  ETA callbacks; the merchant registers the callback in ETA portal; updates the
  outbox rows. (Same pattern as the existing sales/sync idempotency: replay-safe.)
- **Cancel/replace**: refunds → e-credit note for e-invoice sales; e-receipt
  cancel API for voided B2C receipts (flag: only within allowed window,
  else a new corrective receipt — decision log).

### 7.5 gRPC/structure
- `internal/eta/` modules: identity.go, signer.go, builder.go, submit.go,
  outbox.go, webhook.go, qr.go — mirrors existing package layout
  (`internal/transport/*`, `internal/infrastructure/*`) and keeps HTTP transport
  (Gin) thin.

## 8. Accounting core (GL) — minimal viable books

- Posting rules (pure, unit-tested): sale → Debit Cash/AR, Credit Sales Revenue,
  Credit Output VAT Payable, Debit Sales Discount (if any); refund → reverse;
  cash register close → reconcile; stock move on sale → inventory → COGS journal
  (with inventory adjustments from `012_inventory_adjustments`); purchases +
  input VAT from `035_purchases` (OCR-verified supplier invoice fields feed the
  VAT purchase register).
- VAT files: `GetVATSalesRegister(from,to)` and `GetVATPurchasesRegister(...)`
  querying the outbox + GL; monthly **VAT return worksheet** endpoint consuming
  both registers (output VAT minus input VAT = payable).
- Reports hook into the existing dashboard/reporting handlers.

## 9. Admin UI + merchant onboarding (Flutter + web_admin)

- Store settings: TRN, activity code, commercial register, VAT flags, default
  VAT rate, branch code, device ETA registration (serial/PIN).
- Catalog: per-product VAT code + EGS/GS1 code fields (bulk CSV upload for
  big catalogs); a "missing codes" report drives the pre-go-live mapping.
- Submission dashboard (mirrors SaaS control panel style): counts per status,
  rejected rows with ETA messages + resubmit/correct actions.
- VAT registers + monthly return worksheet screens; recon view POS-vs-ETA.
- Receipt print: ETA QR mandatory once `eta_enabled`.
- $ strings: Arabic-first l10n as everywhere else in the app.

## 10. Workstreams & phases (mapped to roadmap Phases E1–E7)

| Phase | Deliverable | Key outputs |
|---|---|---|
| E0 | Tax/VAT schema + engine | migrations 040/041/042, `internal/accounting` VAT engine, product/customer/tenant fields, tests |
| E1 | Document builder + signer | Invoice/Receipt/Credit builders, canonicalizer + RSA-SHA256, byte-level contract tests vs ETA samples |
| E2 | Outbox + submitter + webhook | 042 table + worker + chunked submit + status polling + notifications receiver + backoff |
| E3 | Accounting core | 043 schema, posting engine, journal API, registers + VAT return worksheet |
| E4 | Merchant onboarding + catalog mapping | tenant/device ETA setup endpoints + Flutter settings + EGS/GS1 bulk import |
| E5 | Reports + dashboard + recon | submission dashboard, VAT registers UI, recon report |
| E6 | Receipt QR + printing | ETA QR in ESC/POS + JSON receipts; verification link printing |
| E7 | Go-live + ops | preprod 2-week soak (decision log), prod cut-over, alerts for 72h window, runbook in `docs/11_OPERATIONS.md` |

Phases E0–E2 are the critical path to "we are compliant enough to submit";
E3–E6 are the accountancy value layer; E7 is compliance ops. Suggest shipping
E0–E2 + E4 together (a merchant must be able to register devices and map codes
to submit anything).

## 11. Testing & compliance

- `go test` stays DB-light: VAT engine, builders, signer, outbox transitions,
  posting rules are all pure/unit + DB-backed suites guarded by
  `TEST_DATABASE_URL` (existing pattern).
- Contract tests: ETA official sample docs (invoice + receipt) signed + parsed
  byte-for-byte; run against preprod for 2 weeks before production (per ETA
  guidance). Flutter widget tests for settings screens.
- Acceptance: a seeded Egypt tenant can register a device, publish a price,
  sell B2B (→ cleared e-invoice with UUID/QR) and B2C (→ e-receipt + QR), refund
  it (→ credit note / cancel), and generate a monthly VAT worksheet whose numbers
  equal ETA-accepted totals.

## 12. Configuration & operations

- Env: `ETA_BASE_URL` (default preprod), `ETA_SECRET_ENC_KEY`, VAPID already
  ships; per-tenant OAuth + cert come from tenant config (encrypted).
- Backups: `scripts/backup.sh` already dumps everything — the outbox + signed
  payloads are in the DB, so restore keeps us compliant (retention).
- Runbook additions: onboarding checklist, rejected-batch triage, 72h deadline
  alerts, year-round catalog code review cadence.

## 13. Decision log

- D1 Tax-inclusive street prices are the model; VAT shown split on documents
  only (retail norm). Revisit for wholesale mode later.
- D2 Per-line round-half-up then sum; verify against ETA preprod and pin it.
- D3 ETA state is eventual via outbox; POS remains offline-first (consistent
  with the existing sync philosophy).
- D4 `eta_documents`/GL rows: transactional records — **not** in the sync pull
  feed (mirrors `client_events` exclusion); they get a private staff API.
- D5 eSeal via `.pfx` in env-first; cloud-signing (OrchidaTax-style) is a
  pluggable signer interface for merchants who prefer no hardware.
- D6 Receipt-number collision handling: claim sequence inside the same tx that
  inserts the outbox row.
- D7 Not in scope yet: E-archive API, foreign invoices, reverse charge, payroll.
- D8 Cancellation windows: e-receipts cancelled only while allowed by ETA; else
  corrective document (credit + reissue) — pending ETA guidance pin-down.
- D9 The existing dormant `sales.tax_minor` becomes the engine-written column;
  sale_items gain `taxable_minor`/`tax_minor` (additive, non-breaking).

## 14. Risks

- ETA spec churn (schema versions bump periodically) → isolate parsing/signing
  behind interfaces, version the document types, pin acceptance tests.
- GS1/EGS catalog mapping is a manual effort per merchant → bulk import + a
  suggestions endpoint (match by barcode prefix) in E4.
- Secrets/certs compromise → encrypted at rest + rotation runbook; never expose
  the private key through the API.
- 72h e-receipt deadline pressure on flaky VPS → the outbox/worker pattern with
  alerting, plus chunked submissions and webhook-driven success to keep the
  window safe.