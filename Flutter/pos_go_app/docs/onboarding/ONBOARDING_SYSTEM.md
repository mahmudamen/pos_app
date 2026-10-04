# POS Go Onboarding System

## Vision

Turn POS Go onboarding from passive documentation into an interactive business workflow.

The user should always understand:

1. What should I do now?
2. Where do I do it?
3. What should I enter?
4. How do I know it worked?
5. What should I do next?

## Activation Journey

```text
NEW USER
  ↓
Store Setup
  ↓
First Product
  ↓
POS Terminal
  ↓
Cash Session
  ↓
First Sale
  ↓
ACTIVATED
  ↓
Trial Reward
  ↓
Up to 30 Total Days
```

## Golden Rule

Do not measure onboarding by the number of tooltips viewed.

Measure successful business outcomes.

## Engineering Rule

```text
Flutter
  └── Guidance

Go
  ├── Business State
  ├── Completion Verification
  ├── Trial Eligibility
  └── Trial Reward

PostgreSQL
  └── Durable State
```

## Product Rule

Every tour must answer:

> What successful action should the user be able to perform after this tour?

If a tour has multiple unrelated goals, split it.
