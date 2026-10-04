# Implementation Plan

## Phase 1 — Foundation

- Create onboarding module.
- Create Tour, TourStep, Goal, Status and Progress.
- Create TourController.
- Create target resolver.
- Create navigation adapter.
- Create overlay renderer.
- Add persistence.

## Phase 2 — Backend

- Create onboarding persistence.
- Implement onboarding API.
- Implement authorization.

## Phase 3 — Business Events

Connect:

- store.created
- product.created
- terminal.configured
- cash_session.opened
- sale.created
- payment.completed

## Phase 4 — Store Setup

Goal: store is ready.

## Phase 5 — First Product

Goal: at least one sellable product exists.

## Phase 6 — POS Setup

Goal: terminal is ready.

## Phase 7 — First Sale

Goal: `sale.created AND payment.completed`.

## Phase 8 — Trial Reward

Implement server-side eligibility and idempotent extension to 30 total days.

## Phase 9 — Onboarding Center

Implement Arabic dashboard with progress, recommended goal and resume action.

## Phase 10 — Analytics

Implement onboarding and activation metrics.

## Phase 11 — Quality

Unit, widget, integration, E2E, security, RTL and responsive testing.

## Phase 12 — Production

Remove debug URLs, token logging and client-side reward authority. Validate all targets and translations.
