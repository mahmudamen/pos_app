# Security

## Principles

Onboarding must never bypass authorization.

The backend validates:

- User
- Store
- Role
- Permissions
- Business state
- Trial eligibility
- Reward eligibility

Never trust client-provided:

- tour_completed
- trial_extended
- store_ready

Never store or log:

- Passwords
- Access tokens
- Refresh tokens
- API keys
- Payment credentials

Trial rewards must be server-side, idempotent and auditable.
