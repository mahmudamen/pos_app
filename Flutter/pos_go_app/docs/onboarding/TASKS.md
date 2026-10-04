# Onboarding Engineering Tasks

## EPIC 1 — Architecture

- [ ] ONB-001 Create onboarding module.
- [ ] ONB-002 Create Tour domain.
- [ ] ONB-003 Create TourController.

## EPIC 2 — Runtime

- [ ] ONB-010 Tour registry.
- [ ] ONB-011 Target resolver.
- [ ] ONB-012 Navigation adapter.
- [ ] ONB-013 Spotlight overlay.

## EPIC 3 — UX

- [ ] ONB-020 Tour bubble.
- [ ] ONB-021 Arabic RTL.
- [ ] ONB-022 Pause/resume.
- [ ] ONB-023 Skip.

## EPIC 4 — Backend

- [ ] ONB-030 Database schema.
- [ ] ONB-031 Onboarding API.
- [ ] ONB-032 Authorization.

## EPIC 5 — Events

- [ ] ONB-040 Business event integration.
- [ ] ONB-041 Event idempotency.

## EPIC 6 — Store Setup

- [ ] ONB-050 Implement store_setup tour.

## EPIC 7 — First Product

- [ ] ONB-060 Implement first_product tour.

## EPIC 8 — POS Terminal

- [ ] ONB-070 Implement terminal_setup tour.

## EPIC 9 — Cash Session

- [ ] ONB-080 Implement cash_session tour.

## EPIC 10 — First Sale

- [ ] ONB-090 Implement first_sale tour.

## EPIC 11 — Trial Reward

- [ ] ONB-100 Implement eligibility.
- [ ] ONB-101 Implement idempotent reward.
- [ ] ONB-102 Implement Arabic reward UI.

## EPIC 12 — Onboarding Center

- [ ] ONB-110 Implement onboarding dashboard.

## EPIC 13 — Analytics

- [ ] ONB-120 Implement onboarding and activation events.

## EPIC 14 — Testing

- [ ] ONB-130 Unit tests.
- [ ] ONB-131 Widget tests.
- [ ] ONB-132 Integration tests.
- [ ] ONB-133 Security tests.

## EPIC 15 — Production

- [ ] ONB-140 Remove hard-coded API URLs.
- [ ] ONB-141 Remove token logging.
- [ ] ONB-142 Verify reward authority.
- [ ] ONB-143 Verify stable tour targets.
- [ ] ONB-144 Review Arabic strings.
- [ ] ONB-145 Test RTL.
- [ ] ONB-146 Test mobile/tablet/desktop.
- [ ] ONB-147 Test network failures.
- [ ] ONB-148 Test browser/app restart.
- [ ] ONB-149 Test complete first-sale flow.

## Definition of Done

A tour is production-ready only when it:

- Has one clearly defined goal.
- Has stable targets.
- Supports navigation.
- Supports pause/resume.
- Supports skip/close.
- Persists progress.
- Handles missing targets.
- Verifies real business state.
- Supports Arabic RTL.
- Has automated tests.
- Has analytics.
- Does not trust client completion.
