# Tester Test Cases — POS.Go (Android)

Build under test: `dist/pos_go_0.1.0+1_release.apk` (com.xamltech.pos_go, signed with the debug key — sideload via `adb install -r` or share the file directly).

Backend: must be reachable at `http://127.0.0.1:8080` (dev box) or the production URL. Login requires a running API. Demo tenant: `demo-restaurant` — `admin@demo-restaurant.com` / `admin` (manager) and `guest@demo-restaurant.com` / `admin` (guest).

Environment notes:
- All app strings are Arabic-first; you can switch to English from the login screen and inside the POS settings sheet.
- Money is EGP (`E£`).

---

## 1. Core checkout (smoke)

1. Launch the app; the login screen shows tenant/email/password + device fields.
2. Log in as `admin@demo-restaurant.com` / `admin`.
3. The product grid loads (Cappuccino and the rest of the seeded catalog under categories). If the catalog is empty, publish at least one product first (Section 5).
4. Tap a product twice to add 2 qty to the cart; the cart shows stock badges.
5. If "No open session on this terminal" is shown, open a register with starting cash and tap "Keep open".
6. "Complete sale" → the payment sheet appears (Payment method / cash field prefilled with the total).
7. Set a split tender: card 3.00, cash remainder; then "Confirm payment".
8. The success screen shows the sale id and the receipt preview button.

## 2. Sale history + receipt

1. Open the POS app-bar menu → Sale history.
2. The new sale is listed with a payment-method chip (Card / Cash / Mixed).
3. Tap the receipt icon → printer-style preview (dashed rules, totals, payments, E£ formatting).
4. Back. Pagination loads older sales as you scroll / tap "older".

## 3. Offline queue

1. Put the device in airplane mode (or pull the backend down) while logged in.
2. Add a product and "Complete sale" → confirm payment.
3. The sale is enqueued (a "queued" indicator should appear rather than a failure).
4. Restore network; pull-to-refresh the catalog so the queue replays via the sync endpoint.
5. The sale appears in sale history only once (no duplicates).

## 4. Register session (manager)

1. As manager, open a session (starting cash, e.g. 100).
2. Sell ~2 items, confirm.
3. "Finish" the session with counted cash ≠ expected → the Z-report shows the difference.
4. Session history lists the closed session with opening/closing cash.

## 5. Self-order QR publishing (new)

1. On the POS inventory screen, find a product and tap the publish switch (or the product tile switch) to enable "QR ordering" for it; a snackbar confirms it is on the menu.
2. Make sure that product has stock > 0 (out-of-stock items are hidden from the menu).
3. (Optional) In inventory, tap the tile switch again to unpublish; the snackbar notes it will be hidden until re-enabled.

## 6. QR sheet + customer ordering (new)

1. In the POS app bar, tap the **QR icon** (`qr_code_2`) — the self-order sheet opens.
2. Optional: type a table name (e.g. "T5"). The QR encodes `…/selforder?tenant=<slug>&table=T5`.
3. Tap **Regenerate** — a `&v=N` cache-buster is appended (visible in the text under the QR).
4. Scan the QR with another phone/camera → the web menu page opens for the tenant.
5. The menu shows only the products you published and that are in stock, under their category.
6. Add Cappuccino ×2 → the page shows a total → place the order.

## 7. Approve / cancel self-orders (new)

1. In the POS app bar, tap the **pending-orders icon** (`pending_actions`) → Self orders.
2. The new order appears with status `pending`; a badge shows the count.
3. **Approve** → confirm dialog → status becomes `approved`; stock decreases by the ordered qty; a `Sale` is created (visible in sale history).
4. Place a second order from the web page, then **Cancel** it → status `cancelled`; stock is not touched.
5. Approving an already-approved order is rejected server-side (409).

## 8. Product requests inbox (new)

> Backend note: the request-form bubble on the web page is served but the form submit is disabled when the API has no web-push VAPID key; verify via an approved web-push config or the API directly.

1. In the POS app bar, tap the **product-request icon** (`rate_review_outlined`).
2. A (seeded/interactive) request appears; open it → contact/note details.
3. **Fulfill** → status `fulfilled`; if you published+restocked the product meanwhile, an order is created and stock decreases.
4. Mark a second request **closed** → status `closed` (no stock effect).
5. Chips filter by Open / Fulfilled / Closed / All; the page paginates.

## 9. Discount policy (manager PIN)

1. As manager, edit your own user's security profile (or rely on defaults) so `discount_mode` = block/cap.
2. In the payment sheet, enter a discount that exceeds the cap → the inline hint explains (capped/warned/needs-PIN/prohibited).
3. Block mode: entering the manager PIN verifies (see wrong-PIN → "PIN locked" after 5 attempts for 5 minutes; a manager can unlock).
4. The final sale total honors the clamped/approved discount.

## 10. Roles

1. Log out; log in as `guest@demo-restaurant.com` / `admin`.
2. The POS hides manager-only actions (registers open/close, publish toggle, refunds, discount overrides) — expect 403-style behavior or hidden controls.
3. Log in again as the manager to restore full access.

---

## Pass criteria
- All checkouts that reach "Confirm payment" also appear in sale history (online and after offline replay) exactly once each.
- Stock never goes negative and the catalog grid matches server stock.
- Self-order lifecycle (place→approve/cancel) reflects stock and appears in sale history after approval.
- No layout overflows on a ~6.1" phone in portrait at the default text scale (Arabic RTL).

## Known backend prerequisites
- Register session must be open on the terminal the sale is made from (one open session per tenant+device).
- Self-order menu only shows published, in-stock products (`selforder_enabled = true`, `is_active = true`, `stock_quantity > 0`).