# Architecture

```text
                    POS Go
                       |
              +--------+--------+
              |                 |
          POS Client        Go Backend
              |                 |
        Tour Runtime      Onboarding Service
              |                 |
       +------+-------+     PostgreSQL
       |              |
   Tour Registry   UI Adapter
       |              |
       +------ Tour Engine
                  |
          +-------+-------+
          |               |
      Navigation       Verifier
```

## Client Responsibilities

- Render tour UI.
- Highlight targets.
- Navigate.
- Collect interaction.
- Persist temporary UI state.
- Send progress/events.
- Render Arabic RTL content.

## Backend Responsibilities

- Authoritative onboarding state.
- Business-state verification.
- Trial eligibility.
- Trial extension.
- Idempotency.
- Authorization.

## Separation

`Tour Definition != Tour Runtime != Business Domain != Trial Billing`

The tour engine must never implement pricing, tax, inventory, sale or payment business rules.
