# Domain Model

## Tour

- id
- version
- goal
- status
- steps

## TourProgress

- user_id
- store_id
- tour_id
- tour_version
- current_step
- status
- started_at
- paused_at
- completed_at
- updated_at

## Status

`NOT_STARTED`, `RUNNING`, `PAUSED`, `COMPLETED`, `SKIPPED`

## Goals

Examples:

- store_configured
- first_category
- first_product
- terminal_configured
- cash_session_opened
- first_sale

## Store Activation

- store_configured
- product_ready
- terminal_ready
- cash_session_opened
- first_sale_completed
