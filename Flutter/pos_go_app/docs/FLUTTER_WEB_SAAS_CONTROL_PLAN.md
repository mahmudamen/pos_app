# Plan — Flutter Web Control for SaaS + User Subscription

A management web console built with **Flutter (web target)** that reuses the
existing `pos_go_app` codebase (same models, `ApiClient`, Arabic-first l10n).
It backs the SaaS control plane APIs already shipped on the backend
(`GET /v1/saas/summary`, `GET /v1/saas/tenants`,
`GET /v1/saas/tenants/:id/analytics`, plan limits, onboarding).

## Goals
1. SaaS staff (role `saas_admin`) manage tenants: audit, analytics drill-down,
   plan changes, trial extension, suspend/resume, and (future) billing.
2. Merchant self-service: subscription status, upgrade/downgrade, invoices,
   billing card management.
3. Everything hits the same `/v1/*` API behind nginx TLS — no new backend
   surface needed for v1 (billing endpoints are stubbed platform work).

## Delivery phases

### Phase 1 — Control panel (admin) v1.0
- **Flutter web entry**: add `web/` target (already supported by the codebase),
  serve as static assets behind nginx `admin.xamltech.com` with basic-auth or
  backend-gated `saas_admin` login.
- **Reuse**: `ControlPanelScreen` (mobile) promotes to a responsive shell:
  - Dashboard: `SaasSummary` stat cards + per-business-type breakdown.
  - Tenant list (search/filter by slug/business_type/plan/trial state).
  - Tenant drill-down: analytics endpoint (plan/resources/users/products,
    today stats, 7-day trend, top products, recent sales).
- **Actions (new, backed by small admin routes — see backend work)**: extend
  trial (`trial_ends_at`), change plan (`max_users`/`max_products`), suspend/
  resume tenant (login block), reset merchant password.
- **Security**: web app uses same JWT; `saas_admin` role required; strict CSP;
  CORS allowlist add `https://admin.xamltech.com` (already config-driven).

### Phase 2 — Tenant self-service (subscription management)
- Routes (backend): `GET /v1/saas/me` (own subscription + usage),
  `PATCH /v1/saas/me/plan` (upgrade/downgrade within approved matrix),
  `GET /v1/saas/me/invoices`, `POST /v1/saas/me/checkout` (Stripe-style intent).
- Credit-card **not** collected on device: web console hosts checkout;
  POS app only reflects current plan + suspensions.
- Plan tiers: `trial` → `starter` → `grow` → `pro`, each with
  `max_users`/`max_products` (already enforced server-side via 409
  `plan_limit_exceeded`).
- Trial expiry UX: dashboard banner on web AND a deep link from the mobile app
  (`403 trial_expired` already prompts renewal).

### Phase 3 — Billing hooks (backend)
- Webhook receiver for the payment provider (idempotent, verified signature).
- `subscriptions` + `invoices` tables (migration), ledger of plan changes.
- Automatic trial expiry extension on first successful payment; dunning
  (grace period) before suspension.

## Architecture
```
Flutter web (pos_go_app, -d web)   ── https://admin.xamltech.com
        │  JWT (saas_admin)
        ▼
nginx (TLS, CSP, CORS allowlist)  ── proxy /v1/* ──► pos-api (127.0.0.1:8080)
                                                          │
                                                    postgres (RLS FORCE)
                                                          │
                                                    Stripe webhook ──► /v1/saas/billing
```

## Out of scope for v1
- Multi-tenant hostname routing (still `tenant_id` in the login payload).
- Native billing on Android (web is the billing surface).
- Fintech payments SDKs (cash/card/mobile tender stays in the POS app).

## Pointers
- Backend: `internal/transport/saas/` (summary/tenants/analytics/plan limits),
  `scripts/grants_prod.sql`, `migrations` for future `subscriptions`.
- Flutter: `lib/features/saas/control_panel_screen.dart`,
  `lib/core/api_client.dart` (add `adminTenantAction`/`selfService` methods),
  `lib/l10n/strings.dart` (Arabic-first).
- Deploy: nginx `admin.xamltech.com` site (static web build + API proxy),
  same certbot flow as `api.xamltech.com`.