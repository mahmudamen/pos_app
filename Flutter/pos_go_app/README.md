# POS Go App

New Flutter POS client for the Go/PostgreSQL backend. This project intentionally has no Firebase dependencies.

## Current slice

- Go API base URL configuration
- Login and refresh-token client models
- Secure token storage
- Cashier POS screen with product search, cart, totals, and checkout placeholder
- Local cart state ready to connect to catalog and sales APIs

## Run

```bash
flutter pub get
flutter run --dart-define=API_BASE_URL=http://127.0.0.1:8080
```

The Go backend must be running before login can be used.
