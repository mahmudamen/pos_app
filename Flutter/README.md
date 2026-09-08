# Flutter POS SaaS System

A progressive spec-kit for building a POS SaaS Android app using Flutter, with a C# ASP.NET Core + PostgreSQL backend.  
This repository evolves through **five versions (v1–v5)**, each adding complexity and optimizations.

## Features
- Barcode & QR scanning
- Fast product search with caching
- Multi-API price handling
- Offline-first architecture
- Modular UI for retail workflows

## Versions
- v1: Basic UI + product list
- v2: Barcode/QR integration
- v3: API-driven pricing + caching
- v4: Offline-first sync + advanced search
- v5: Enterprise SaaS polish (multi-tenant, analytics)

## Development
- CLI-first workflow with VS Code
- Dart + Flutter
- PostgreSQL backend via REST APIs
- Modular architecture with Riverpod/BLoC

## Skills Constitution
See `skills_tasks_spec.md` for detailed breakdown.
### Goals
- Basic product list UI
- Simple navigation
- Static product data

### Tasks
- Scaffold Flutter project
- Implement `ListView.builder` for products
- Add basic theming (Material Design)
### Goals
- Integrate barcode/QR scanning
- Link scanned codes to product list

### Tasks
- Add `flutter_barcode_scanner` package
- Implement scan button in POS UI
- Map scanned code → product lookup
### Goals
- Connect to C# backend APIs
- Handle multi-API pricing
- Cache product data locally

### Tasks
- Integrate `http` + `dio` for API calls
- Use Hive/SQLite for local caching
- Implement price aggregation logic
### Goals
- Offline-first sync with PostgreSQL backend
- Indexed product search
- Background sync tasks

### Tasks
- Implement `moor`/`drift` for local DB
- Add search indexing for fast queries
- Background sync with isolates
### Goals
- Multi-tenant support
- Analytics dashboard
- Role-based access

### Tasks
- Tenant-aware API calls
- Add charts with `fl_chart`
- Implement role-based UI flows

