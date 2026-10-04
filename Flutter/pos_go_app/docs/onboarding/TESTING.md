# Testing Strategy

## Unit Tests

Test:

- Tour state machine.
- Step transitions.
- Eligibility.
- Completion conditions.
- Versioning.
- Trial reward calculation.
- Idempotency.

## Widget Tests

Test:

- Bubble.
- Spotlight.
- RTL.
- Pause.
- Skip.
- Close.
- Resume.

## Integration Tests

Test:

`Login → Store Setup → Product → Terminal → Cash Session → Sale`

## Security Tests

Verify:

- Cross-user isolation.
- Cross-store isolation.
- Fake completion rejection.
- Fake reward rejection.
- Duplicate reward protection.

## E2E

New trial user should be able to reach:

`Create account → Configure store → Add product → Configure POS → Open session → First sale → Activation → Reward`
