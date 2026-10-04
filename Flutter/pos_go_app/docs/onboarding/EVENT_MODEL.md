# Event Model

## Business Events

- user.created
- store.created
- store.updated
- category.created
- product.created
- product.updated
- terminal.created
- terminal.configured
- cash_session.opened
- cash_session.closed
- sale.created
- payment.completed
- receipt.configured
- customer.created
- employee.created

Each event requires a unique `event_id`.

Duplicate events must not duplicate progress or rewards.
