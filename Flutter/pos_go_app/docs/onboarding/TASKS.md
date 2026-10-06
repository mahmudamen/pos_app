# Onboarding Engineering Tasks

## Status (2026-10-07)

- Backend slice done & green: migration `054_onboarding.sql`, `internal/transport/onboarding/`
  (handler + readiness + 15 DB-backed/unit tests), wired in `server/router.go`.
- Flutter slice done & green: data layer (`lib/core/onboarding.dart`), API client wiring,
  tour engine (`lib/core/tour/`), spotlight overlay, onboarding center screen + notification
  dot on the POS grid; `flutter analyze` 0 issues; full suite **615 tests pass**
  (onboarding: api 15 + engine 17 + overlay 5 + center 5).
- Implementable tours: `cash_session`, `first_sale`, and **`terminal_setup`** (added
  2026-10-07 — targets the printers screen: app-bar printer button → "Printing enabled"
  switch → "Print test page"). `store_setup` and `first_product` are **not implementable**
  (no store-editor / product-editor UIs) — shipped as empty-step placeholders
  ("Not available in this build").
- Not yet done: trial reward UIs (EPIC 11), analytics aggregation (EPIC 13), on-device
  E2E (ONB-132), security tests (ONB-133), production gate review (EPIC 15), and prod
  migration `054` + grants application.
- Decision (2026-10-07): EPIC 11 trial **stays 15 days**; the 30-day spec assumption is
  a documented deviation, not a bug to fix.
- Known spec drift / quirk: the server verifies `terminal_configured` as "any
  `register_sessions` row exists" (readiness.go `goalSatisfied`), i.e. the printer tour's
  Finish reports `goal_not_met` until the user also opens a register session. Client-side
  printer config is local-only and invisible to the backend, so this proxy stands.

## EPIC 1 — Architecture

- [x] ONB-001 Create onboarding module. (`lib/core/onboarding.dart`,
      `lib/features/onboarding/`)
- [x] ONB-002 Create Tour domain. (`lib/core/tour/tour_target.dart`,
      `tour_definitions.dart`, `TourCatalog`, `TourPhase`)
- [x] ONB-003 Create TourController. (`lib/core/tour/tour_controller.dart` — next/skip/close,
      pause/resume, goal verification)

## EPIC 2 — Runtime

- [x] ONB-010 Tour registry. (`tour_target_registry.dart`)
- [x] ONB-011 Target resolver. (`tour_overlay.dart` `_targetRect` via `localToGlobal` +
      `hasSize` guard, post-frame `_resolveTargetAfterLayout`)
- [x] ONB-012 Navigation adapter. (step navigation lives in `TourController`
      — no separate hop adapter; goals are step-embedded)
- [x] ONB-013 Spotlight overlay. (`tour_overlay.dart` — spotlight hole + bubble + fallback card)

## EPIC 3 — UX

- [x] ONB-020 Tour bubble. (positioned `_PositionedBubble`, action Rows as `Wrap`)
- [x] ONB-021 Arabic RTL. (bubble anchored via `MediaQuery.sizeOf`, inherits text direction;
      overlay tests exercise Arabic-first strings)
- [x] ONB-022 Pause/resume. (`setTourStep` pauses/runs; `pause` keeps progress)
- [x] ONB-023 Skip. (`skip()` triggers `SKIPPED` with success + analytics)

## EPIC 4 — Backend

- [x] ONB-030 Database schema. (`054_onboarding.sql`)
- [x] ONB-031 Onboarding API. (`internal/transport/onboarding/handler.go`,
      `readiness.go`; `GET /v1/onboarding/tours/:tour/state`)
- [x] ONB-032 Authorization. (role/permission gating in handler + grants)

## EPIC 5 — Events

- [x] ONB-040 Business event integration. (started / step / goal_not_met / completed /
      version_mismatch persisted per tenant+user and surfaced in state)
- [x] ONB-041 Event idempotency. (409 `goal_not_met` / `tour_version_mismatch` /
      `tour_completed`; repeated completes don't double-apply)

## EPIC 6 — Store Setup

- [ ] ONB-050 Implement store_setup tour. **Not implementable — no store-editor UI.**

## EPIC 7 — First Product

- [ ] ONB-060 Implement first_product tour. **Not implementable — no product-editor UI.**

## EPIC 8 — POS Terminal

- [x] ONB-070 Implement terminal_setup tour. (steps intro → enable → verify target the
      printers screen: app-bar printer button, "Printing enabled" switch, "Print test page"
      button; goal `terminal_configured` verified server-side)

## EPIC 9 — Cash Session

- [x] ONB-080 Implement cash_session tour. (steps intro → start → starting_cash → verify;
      `start` maps to the session bar, `starting_cash` to the dialog, `verify` to goal
      check against the register session)

## EPIC 10 — First Sale

- [x] ONB-090 Implement first_sale tour. (steps intro → add_item → checkout → first_sale;
      goal verified from the real sales feed)

## EPIC 11 — Trial Reward

- [ ] ONB-100 Implement eligibility. (**Decision 2026-10-07: keep the 15-day trial; the
      30-day spec assumption is a documented deviation.** Reward UI otherwise not shipped.)
- [ ] ONB-101 Implement idempotent reward. (defer until reward ships)
- [ ] ONB-102 Implement Arabic reward UI. (not shipped)

## EPIC 12 — Onboarding Center

- [x] ONB-110 Implement onboarding dashboard. (`onboarding_center_screen.dart` — card grid,
      goal/phase chips, completion checkmarks, help/close per tour)

## EPIC 13 — Analytics

- [ ] ONB-120 Implement onboarding and activation events. (per-tour event rows are
      persisted; dedicated aggregation is not built)

## EPIC 14 — Testing

- [x] ONB-130 Unit tests. (`onboarding_api_test.dart` 15, `tour_engine_test.dart` 16)
- [x] ONB-131 Widget tests. (`tour_overlay_test.dart` 5, `onboarding_center_screen_test.dart` 5;
      injectable `_CenterApi`/`_TourApi` fakes)
- [ ] ONB-132 Integration tests. (backend DB-backed integration suite exists; no on-device
      Flutter E2E has been run)
- [ ] ONB-133 Security tests. (not written)

## EPIC 15 — Production

- [ ] ONB-140 Remove hard-coded API URLs. (client uses `API_BASE_URL` default)
- [ ] ONB-141 Remove token logging. (no token logged)
- [ ] ONB-142 Verify reward authority.
- [ ] ONB-143 Verify stable tour targets. (overlay target-resolution bugs fixed; verify on device)
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