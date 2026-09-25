# Data safety — draft answers for Play Console

App: POS.Go · Package `com.xamltech.pos_go` · Version 1.0.0 (code 3)

These answers are derived from what the shipped code actually does. Every line
marked **[verify]** has to be confirmed by a human before the form is submitted,
because a wrong Data safety declaration is a policy violation, not a cosmetic
error.

## Summary

| Question | Answer | Why |
|---|---|---|
| Does your app collect or share any of the required user data types? | **Yes** | Accounts, purchases and diagnostics are sent to the POS backend the merchant controls |
| Is all of the user data collected by your app encrypted in transit? | **Yes** | Release builds set `android:usesCleartextTraffic="false"`; every host is HTTPS |
| Do you provide a way for users to request that their data is deleted? | **Yes — by email** | Deletion is handled manually on request to support@xamltech.com **[verify]** |
| Is the data shared with third parties? | **No** | The app talks only to the merchant's own POS server and XAMLtech infrastructure |
| Is the data collected for ads or tracking? | **No** | No advertising or tracking SDKs are bundled (no Firebase, no ad SDK) |
| Is the app a "news"-like or kids app? | **No** | Business tool, audience 18+ |

## Declared data types

| Data type | Collected | Shared | Purpose | Optional? | Encrypted in transit | Can be deleted |
|---|---|---|---|---|---|---|
| Name | Yes | No | App functionality (staff and customer records) | No | Yes | On request |
| Email address | Yes | No | App functionality (sign-in, staff accounts) | No | Yes | On request |
| Phone number | Yes | No | App functionality (customer records) | Yes (staff may skip it) | Yes | On request |
| Purchase or transaction history | Yes | No | App functionality (sales, invoices, refunds) | No | Yes | On request |
| Photos | Yes | No | App functionality (supplier invoice scanning) | Yes (only when the user scans) | Yes | Retained server-side with the purchase **[verify]** |
| Files or docs | Yes | No | App functionality (supplier invoices) | Yes | Yes | **[verify]** |
| App interactions (crash reports, screen names, app version) | Yes | No | Diagnostics | No | Yes | On request |
| Device or other IDs | Yes | No | App functionality (device-bound sessions, per-terminal cash sessions) | No | Yes | Cleared on sign-out / reinstall **[verify]** |

## Not declared, and why

- **Location**: not requested by the app.
- **Contacts**: not read; staff and customers are typed in by hand.
- **Camera/microphone**: the invoice scanner uses the system camera or photo
  picker. The app declares no `CAMERA` permission, so the OS picker governs
  access and nothing is recorded in the background.
- **Financial info**: no card numbers are collected. Card and mobile-money
  tenders are amounts only, recorded by the merchant.
- **Health and fitness, sensitive info, messages, calendar, web browsing**:
  none.

## Permissions shipped in the release build

Verified from the built artifact with `aapt2 dump badging`:

```
android.permission.INTERNET
android.permission.WAKE_LOCK
com.xamltech.pos_go.DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION  (Flutter-generated, app-private)
```

`ACCESS_NOTIFICATION_POLICY` was template boilerplate, is never used by the app,
and was removed for this release.

## Account deletion

- In-app: no self-serve deletion. **[verify]** whether to add one or keep the
  email channel.
- Email channel: support@xamltech.com
- If the app is later published with per-user cloud accounts, Google requires a
  web URL for deletion as well; the product owner has to decide whether
  `xamltech.com/delete-account` needs to exist before the next update. **[verify]**
