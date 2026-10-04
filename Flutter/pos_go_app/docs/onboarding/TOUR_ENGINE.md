# Tour Engine

## Responsibilities

- Load tour.
- Validate audience.
- Resolve current state.
- Navigate.
- Resolve target.
- Render bubble.
- Wait for action.
- Verify business result.
- Advance.
- Persist state.

## Lifecycle

`LOAD → ELIGIBILITY → START → RESOLVE STEP → NAVIGATE → WAIT TARGET → SHOW GUIDANCE → WAIT ACTION → VERIFY → NEXT → COMPLETE`

## Missing Target

The engine must never crash the application if a target is missing.

Recommended behavior:

`retry → recalculate → wait → fallback`

After timeout:

`Try again | Skip step | Exit tour`

## Dynamic UI

Support lazy-loaded pages, modals, dropdowns, API rendering and responsive layouts.
