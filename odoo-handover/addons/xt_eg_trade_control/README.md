# Trade Control

`xt_eg_trade_control` is a new Odoo 19 addon for simple sales, inventory, collection, and Egypt tax document follow-up.

The module is intentionally focused on daily operators:

- One clean dashboard for sales, collections, deliveries, stock warnings, and tax document states.
- Easy sale screen with customer, warehouse, lines, totals, and one-step confirm/invoice action.
- Egypt trade fields on customers and products.
- Tax document queue with states, retry counter, UUID field, payload preview, and source document link.
- Lightweight extensions for invoices, sales orders, and warehouse pickings.

This first version does not submit documents to ETA. It prepares the internal control layer that can later be connected to ETA signing and APIs after tax/legal review.

## Install

Add the repository to `addons_path`, restart Odoo, update Apps, then install **Trade Control**.

Required Odoo apps: Sales, Inventory, Accounting/Invoicing, Contacts, Mail, and Web.
