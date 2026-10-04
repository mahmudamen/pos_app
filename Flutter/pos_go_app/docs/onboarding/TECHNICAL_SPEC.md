# Technical Specification

## Tour Definition

A tour contains:

- id
- version
- goal
- audience
- steps
- completion condition

Example:

```yaml
id: first_product
version: 1
goal: create_first_product
steps:
  - id: open_products
    route: /products
    target: products_menu
  - id: click_add
    target: add_product_button
  - id: enter_name
    target: product_name
  - id: save
    target: save_product
completion:
  event: product.created
```

## Actions

`NAVIGATE`, `CLICK`, `INPUT`, `SELECT`, `SUBMIT`, `OBSERVE`, `WAIT`

## Stable Targets

Prefer stable identifiers such as:

`data-tour="add-product"`

Do not use brittle selectors such as `button:nth-child(3)`.

## State

`NOT_STARTED → RUNNING → PAUSED → RUNNING → COMPLETED`

Alternative terminal state:

`RUNNING → SKIPPED`

## Persistence

The backend is authoritative for completion and rewards.

## Versioning

Tours are versioned, e.g. `first_product:v1`.
