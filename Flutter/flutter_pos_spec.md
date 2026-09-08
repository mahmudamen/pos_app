# Flutter POS SaaS — Frontend Specification
## Version: v1.0 | Target: Android POS Terminals

---

## 1. Architecture Overview

### 1.1 Stack
| Layer | Technology |
|-------|-----------|
| Framework | Flutter 3.24+ (Dart 3.5+) |
| State Management | Riverpod 2.x (AsyncNotifier + StateNotifier) |
| Local DB | Drift (SQLite) — offline-first |
| Network | Dio 5.x + Retrofit |
| Auth | OAuth2 + JWT (access/refresh tokens) |
| DI | Riverpod (built-in) |
| Serialization | Freezed + JSON Serializable |

### 1.2 Project Structure
```
lib/
├── main.dart
├── app.dart                    # MaterialApp + ProviderScope
├── core/
│   ├── constants/
│   │   ├── api_constants.dart      # Base URLs, endpoints, timeouts
│   │   ├── app_constants.dart      # App name, version, build
│   │   └── storage_keys.dart       # SharedPreferences/Hive keys
│   ├── di/
│   │   └── providers.dart          # All Riverpod providers
│   ├── exceptions/
│   │   ├── api_exception.dart
│   │   └── auth_exception.dart
│   ├── extensions/
│   │   ├── context_ext.dart
│   │   └── string_ext.dart
│   ├── network/
│   │   ├── dio_client.dart         # Configured Dio instance
│   │   ├── api_interceptor.dart    # Auth header injection + token refresh
│   │   ├── error_interceptor.dart  # Global error handling
│   │   └── network_monitor.dart    # Connectivity check
│   ├── router/
│   │   └── app_router.dart         # GoRouter declarative routing
│   ├── theme/
│   │   ├── app_theme.dart
│   │   └── app_colors.dart
│   └── utils/
│       ├── logger.dart
│       └── validators.dart
├── data/
│   ├── datasources/
│   │   ├── local/
│   │   │   ├── drift_database.dart      # Generated Drift DB
│   │   │   ├── tables/
│   │   │   │   ├── products_table.dart
│   │   │   │   ├── sales_table.dart
│   │   │   │   ├── sync_queue_table.dart
│   │   │   │   └── tenants_table.dart
│   │   │   └── daos/
│   │   │       ├── product_dao.dart
│   │   │       └── sale_dao.dart
│   │   └── remote/
│   │       ├── auth_api.dart            # Retrofit: /auth/*
│   │       ├── product_api.dart         # Retrofit: /products/*
│   │       ├── sale_api.dart            # Retrofit: /sales/*
│   │       └── sync_api.dart            # Retrofit: /sync/*
│   ├── models/
│   │   ├── auth/
│   │   │   ├── login_request.dart
│   │   │   ├── login_response.dart
│   │   │   ├── refresh_request.dart
│   │   │   └── user_model.dart
│   │   ├── product/
│   │   │   ├── product_model.dart
│   │   │   └── price_model.dart
│   │   ├── sale/
│   │   │   ├── sale_model.dart
│   │   │   └── sale_item_model.dart
│   │   └── sync/
│   │       ├── sync_request.dart
│   │       └── sync_response.dart
│   └── repositories/
│       ├── auth_repository.dart
│       ├── product_repository.dart
│       ├── sale_repository.dart
│       └── sync_repository.dart
├── domain/
│   ├── entities/
│   │   ├── product_entity.dart
│   │   ├── sale_entity.dart
│   │   └── user_entity.dart
│   └── usecases/
│       ├── login_usecase.dart
│       ├── scan_product_usecase.dart
│       └── create_sale_usecase.dart
└── presentation/
    ├── screens/
    │   ├── splash_screen.dart
    │   ├── login_screen.dart
    │   ├── pos_screen.dart              # Main POS UI
    │   ├── product_search_screen.dart
    │   ├── cart_screen.dart
    │   ├── receipt_screen.dart
    │   └── settings_screen.dart
    ├── widgets/
    │   ├── product_card.dart
    │   ├── cart_item_tile.dart
    │   ├── numeric_keypad.dart
    │   └── barcode_scanner.dart
    └── providers/
        ├── auth_provider.dart
        ├── cart_provider.dart
        ├── product_provider.dart
        └── sync_provider.dart
```

---

## 2. Authentication & Session Management

### 2.1 Auth Flow (OAuth2 + JWT)

```
┌─────────────┐     POST /auth/login      ┌─────────────┐
│   Flutter   │ ────────────────────────> │   Go API    │
│   Client    │  {email, password,        │   Ubuntu    │
│             │   tenant_id, device_id}   │  PostgreSQL │
└─────────────┘                           └─────────────┘
     ^                                          │
     │     200 OK {access_token, refresh_token, │
     │     expires_in, user, tenant}            │
     └──────────────────────────────────────────┘
```

### 2.2 Token Storage (Secure)
| Token | Storage | Key |
|-------|---------|-----|
| Access Token | `flutter_secure_storage` | `access_token` |
| Refresh Token | `flutter_secure_storage` | `refresh_token` |
| Tenant ID | `flutter_secure_storage` | `tenant_id` |
| Device ID | `flutter_secure_storage` | `device_id` |
| Session Fingerprint | `flutter_secure_storage` | `session_fp` |

### 2.3 Token Refresh Logic
```dart
// api_interceptor.dart — Automatic refresh on 401
onError: (DioException err, handler) async {
  if (err.response?.statusCode == 401) {
    final refreshToken = await secureStorage.read(key: 'refresh_token');
    if (refreshToken != null) {
      try {
        final newTokens = await authApi.refresh(RefreshRequest(refreshToken));
        await secureStorage.write(key: 'access_token', value: newTokens.accessToken);
        // Retry original request with new token
        err.requestOptions.headers['Authorization'] = 'Bearer ${newTokens.accessToken}';
        final response = await dio.fetch(err.requestOptions);
        return handler.resolve(response);
      } catch (e) {
        // Refresh failed → force logout
        ref.read(authProvider.notifier).logout();
      }
    }
  }
  return handler.next(err);
}
```

### 2.4 Session Security Rules
- **Access Token TTL**: 15 minutes
- **Refresh Token TTL**: 7 days (rotated on every use)
- **Device Binding**: Each token pair is bound to `device_id` + `session_fp` (hash of device info)
- **Concurrent Session Limit**: Max 3 active sessions per user per tenant
- **Force Logout**: Backend can invalidate all sessions via `/auth/logout-all`
- **Biometric Lock**: Optional fingerprint/PIN after 5 minutes idle

---

## 3. API Contract (Aligned with Go Backend)

### 3.1 Base Configuration
```dart
class ApiConstants {
  static const String baseUrl = 'https://api.yourpos.com/v1';
  static const Duration connectTimeout = Duration(seconds: 10);
  static const Duration receiveTimeout = Duration(seconds: 30);
  static const Map<String, String> defaultHeaders = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
    'X-Client-Version': '1.0.0',
    'X-Platform': 'android',
  };
}
```

### 3.2 Auth Endpoints

#### POST `/auth/login`
```dart
// Request
class LoginRequest {
  final String email;
  final String password;
  final String tenantId;        // "acme-corp" or UUID
  final String deviceId;        // Generated once, stored securely
  final String deviceName;      // "POS-Terminal-01"
  final String? fcmToken;       // For push notifications
}

// Response 200
class LoginResponse {
  final String accessToken;     // JWT, 15min
  final String refreshToken;    // JWT, 7days
  final int expiresIn;          // 900 seconds
  final UserModel user;
  final TenantModel tenant;
  final List<String> permissions;
}

// Response 401
{"error": "invalid_credentials", "message": "Email or password incorrect"}
// Response 403
{"error": "tenant_inactive", "message": "Your subscription has expired"}
```

#### POST `/auth/refresh`
```dart
// Request
class RefreshRequest {
  final String refreshToken;
  final String deviceId;
}

// Response 200
class RefreshResponse {
  final String accessToken;
  final String refreshToken;    // Rotated
  final int expiresIn;
}

// Response 401
{"error": "invalid_refresh", "message": "Session expired, please login again"}
```

#### POST `/auth/logout`
```dart
// Request — Bearer token in header
// Response 204 No Content
// Backend invalidates the refresh token for this device
```

### 3.3 Product Endpoints

#### GET `/products`
```dart
// Headers: Authorization: Bearer <token>
//          X-Tenant-ID: <tenant_id>
// Query Params:
//   ?search=coke&category_id=123&page=1&limit=50&updated_after=2024-01-01T00:00:00Z

// Response 200
class ProductListResponse {
  final List<ProductModel> items;
  final PaginationMeta meta;
}

class ProductModel {
  final String id;              // UUID
  final String sku;             // Barcode/QR code
  final String name;
  final String? description;
  final double price;           // Base price
  final double? salePrice;      // Promotional
  final String currency;        // "USD", "SAR"
  final String? categoryId;
  final String? imageUrl;
  final double stockQuantity;
  final bool isActive;
  final DateTime updatedAt;
  final Map<String, dynamic>? metadata;  // Extra fields per tenant
}
```

#### GET `/products/{sku}`
```dart
// For barcode scan lookup — must be <200ms response time
// Response 200: ProductModel
// Response 404: {"error": "product_not_found"}
```

### 3.4 Sale Endpoints

#### POST `/sales`
```dart
// Request
class SaleRequest {
  final String id;                    // Client-generated UUID (idempotency)
  final List<SaleItemRequest> items;
  final double subtotal;
  final double taxAmount;
  final double discountAmount;
  final double total;
  final String currency;
  final String paymentMethod;         // "cash", "card", "mada"
  final String? customerId;
  final String? notes;
  final DateTime createdAt;           // Client timestamp
  final String deviceId;
  final String? receiptNumber;        // Local receipt ID
}

class SaleItemRequest {
  final String productId;
  final String productName;           // Denormalized for audit
  final String sku;
  final double quantity;
  final double unitPrice;
  final double totalPrice;
}

// Response 201
class SaleResponse {
  final String id;
  final String receiptNumber;
  final String status;                // "confirmed", "pending_sync"
  final DateTime serverTimestamp;
}

// Response 409 (Idempotency conflict)
{"error": "sale_exists", "existing_id": "uuid-here"}
```

#### GET `/sales`
```dart
// Query: ?start_date=2024-01-01&end_date=2024-01-31&page=1&limit=50
// Response: Paginated list of SaleModel
```

### 3.5 Sync Endpoints

#### POST `/sync/push`
```dart
// Batch upload offline sales
class SyncPushRequest {
  final String deviceId;
  final String lastSyncToken;         // From last successful sync
  final List<SaleRequest> sales;
  final List<StockAdjustment> stockAdjustments;
}

// Response 200
class SyncPushResponse {
  final List<String> acceptedIds;
  final List<SyncConflict> conflicts;
  final String newSyncToken;
  final DateTime serverTimestamp;
}
```

#### GET `/sync/pull`
```dart
// Query: ?last_sync_token=xyz&entity_types=products,prices,customers
// Response: Delta changes since last sync
class SyncPullResponse {
  final List<ProductModel> products;
  final List<PriceModel> prices;
  final List<CustomerModel> customers;
  final String newSyncToken;
  final bool hasMore;
  final DateTime serverTimestamp;
}
```

---

## 4. Security Specification

### 4.1 Transport Security
- **TLS 1.3** mandatory for all API calls
- **Certificate Pinning**: Pin the backend certificate in `dio_client.dart`
- **SSL Error Handling**: Show blocking dialog on cert mismatch (MITM protection)

### 4.2 Request Security Headers
```dart
final secureHeaders = {
  'Authorization': 'Bearer $accessToken',
  'X-Tenant-ID': tenantId,
  'X-Device-ID': deviceId,
  'X-Request-ID': Uuid().v4(),        // For request tracing
  'X-Session-Fingerprint': sessionFp,   // Hash of device + app signature
  'X-Timestamp': DateTime.now().toUtc().toIso8601String(),
};
```

### 4.3 Local Data Security
| Data | Protection |
|------|-----------|
| Tokens | `flutter_secure_storage` (Android Keystore / iOS Keychain) |
| SQLite DB | SQLCipher encryption with key from secure storage |
| Receipt cache | AES-256-GCM, key derived from device binding |
| Logs | No PII in logs; rotate every 7 days |

### 4.4 Input Validation
```dart
class Validators {
  static String? email(String? v) =>
    v != null && RegExp(r'^[^@]+@[^@]+\.[^@]+').hasMatch(v) ? null : 'Invalid email';

  static String? password(String? v) =>
    v != null && v.length >= 8 ? null : 'Min 8 characters';

  static String? tenantId(String? v) =>
    v != null && RegExp(r'^[a-z0-9-]+$').hasMatch(v) ? null : 'Invalid tenant ID';
}
```

### 4.5 Anti-Tampering
- **App Signature Check**: Verify APK signature hash on startup
- **Root Detection**: Warn/block on rooted devices (optional per tenant policy)
- **Screenshot Prevention**: `FLAG_SECURE` on login and POS screens
- **Obfuscation**: ProGuard/R8 enabled; no hardcoded secrets

---

## 5. Offline-First Architecture

### 5.1 Sync Strategy: "Local-First, Sync Later"
```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  POS Scan   │ --> │ Local SQLite │ --> │ Background  │
│  Sale       │     │ (Drift)      │     │ Sync Queue  │
└─────────────┘     └──────────────┘     └──────┬──────┘
                                                │
                    ┌───────────────────────────┘
                    ▼
            ┌─────────────┐     ┌─────────────┐
            │  Retry with │ --> │  Go API     │
            │  Exponential│     │  /sync/push │
            │  Backoff    │     └─────────────┘
            └─────────────┘
```

### 5.2 Sync Queue Table
```dart
class SyncQueueItems extends Table {
  TextColumn get id => text().clientDefault(() => const Uuid().v4())();
  TextColumn get entityType => text()();  // "sale", "stock_adjust"
  TextColumn get payload => text()();      // JSON
  TextColumn get status => text()();      // "pending", "syncing", "failed", "resolved"
  IntColumn get retryCount => integer().withDefault(const Constant(0))();
  DateTimeColumn get createdAt => dateTime().withDefault(currentDateAndTime)();
  DateTimeColumn get lastAttempt => dateTime().nullable()();
  TextColumn get errorMessage => text().nullable()();

  @override
  Set<Column> get primaryKey => {id};
}
```

### 5.3 Conflict Resolution
- **Sales**: Client-generated UUID prevents duplicates; server accepts first, rejects duplicates with 409
- **Stock**: Server is source of truth; client gets server state on next pull
- **Prices**: Server wins; client updates local cache on pull

### 5.4 Background Sync
```dart
// Using WorkManager for periodic sync (every 15 min minimum)
@pragma('vm:entry-point')
void callbackDispatcher() {
  Workmanager().executeTask((task, inputData) async {
    final container = ProviderContainer();
    await container.read(syncRepositoryProvider).syncAll();
    container.dispose();
    return Future.value(true);
  });
}
```

---

## 6. State Management (Riverpod)

### 6.1 Auth State
```dart
@riverpod
class AuthNotifier extends _$AuthNotifier {
  @override
  Future<AuthState> build() async {
    final token = await ref.read(secureStorageProvider).read(key: 'access_token');
    if (token == null) return const AuthState.unauthenticated();
    return _validateToken(token);
  }

  Future<void> login(LoginRequest request) async {
    state = const AsyncValue.loading();
    state = await AsyncValue.guard(() async {
      final response = await ref.read(authRepositoryProvider).login(request);
      await _persistTokens(response);
      return AuthState.authenticated(response.user, response.tenant);
    });
  }

  Future<void> logout() async {
    await ref.read(authRepositoryProvider).logout();
    await _clearTokens();
    state = const AuthState.unauthenticated();
  }
}
```

### 6.2 Cart State (POS Core)
```dart
@riverpod
class CartNotifier extends _$CartNotifier {
  @override
  List<CartItem> build() => [];

  void addProduct(ProductModel product) {
    final existing = state.indexWhere((i) => i.product.id == product.id);
    if (existing >= 0) {
      state = [
        ...state.sublist(0, existing),
        state[existing].copyWith(quantity: state[existing].quantity + 1),
        ...state.sublist(existing + 1),
      ];
    } else {
      state = [...state, CartItem(product: product, quantity: 1)];
    }
  }

  double get subtotal => state.fold(0, (sum, i) => sum + (i.product.price * i.quantity));
}
```

---

## 7. Error Handling & UX

### 7.1 API Error Mapping
| HTTP Code | User Message | Action |
|-----------|-------------|--------|
| 400 | "Invalid request" | Show field errors |
| 401 | "Session expired" | Auto-refresh or redirect login |
| 403 | "Access denied" | Show permission dialog |
| 404 | "Product not found" | Show scan retry UI |
| 409 | "Already processed" | Mark as synced locally |
| 422 | "Validation failed" | Show server errors per field |
| 429 | "Too many requests" | Exponential backoff + toast |
| 500+ | "Server error" | Queue for retry + notify admin |
| Network | "Offline mode" | Switch to local DB, queue sync |

### 7.2 Global Error Handler
```dart
class ErrorHandler {
  static void handle(Object error, StackTrace stack) {
    if (error is DioException) {
      switch (error.type) {
        case DioExceptionType.connectionTimeout:
        case DioExceptionType.receiveTimeout:
          _showToast('Connection slow. Using offline mode.');
          break;
        case DioExceptionType.connectionError:
          _showToast('No internet. Sale saved locally.');
          break;
        default:
          _logError(error, stack);
      }
    }
  }
}
```

---

## 8. Barcode & QR Integration

### 8.1 Scanner Implementation
```yaml
# pubspec.yaml dependencies
flutter_barcode_scanner: ^2.0.0
mobile_scanner: ^3.5.0  # Alternative with better performance
```

```dart
class BarcodeScanner {
  static Future<String?> scan() async {
    final result = await FlutterBarcodeScanner.scanBarcode(
      '#FF0000',      // Line color
      'Cancel',
      true,           // Flash
      ScanMode.BARCODE,
    );
    return result == '-1' ? null : result;  // -1 = cancelled
  }
}

// In POS screen
Future<void> onScanPressed() async {
  final code = await BarcodeScanner.scan();
  if (code != null) {
    final product = await ref.read(productRepositoryProvider).findBySku(code);
    if (product != null) {
      ref.read(cartNotifierProvider.notifier).addProduct(product);
    } else {
      _showNotFoundDialog(code);
    }
  }
}
```

---

## 9. Build & Deployment

### 9.1 Flavors
```bash
# Development
flutter run --flavor dev -t lib/main_dev.dart
# Production
flutter build apk --flavor prod --release -t lib/main_prod.dart
```

### 9.2 Environment Config
```dart
class Environment {
  static const String baseUrl = String.fromEnvironment(
    'BASE_URL',
    defaultValue: 'https://api-dev.yourpos.com/v1',
  );
  static const bool enableLogging = bool.fromEnvironment('ENABLE_LOGGING', defaultValue: false);
}
```

### 9.3 CI/CD Pipeline (GitHub Actions)
```yaml
name: Flutter Build
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: subosito/flutter-action@v2
        with:
          flutter-version: '3.24.0'
      - run: flutter pub get
      - run: flutter analyze
      - run: flutter test
      - run: flutter build apk --release
```

---

## 10. Dependencies (pubspec.yaml)

```yaml
dependencies:
  flutter:
    sdk: flutter

  # State Management
  flutter_riverpod: ^2.5.0
  riverpod_annotation: ^2.3.0

  # Network
  dio: ^5.5.0
  retrofit: ^4.1.0

  # Local DB
  drift: ^2.19.0
  drift_flutter: ^0.1.0
  sqlite3_flutter_libs: ^0.5.0

  # Security
  flutter_secure_storage: ^9.2.0

  # Serialization
  freezed_annotation: ^2.4.0
  json_annotation: ^4.9.0
  uuid: ^4.4.0

  # UI
  go_router: ^14.2.0
  fl_chart: ^0.68.0
  intl: ^0.19.0

  # Scanner
  mobile_scanner: ^3.5.0

  # Background
  workmanager: ^0.5.0
  connectivity_plus: ^6.0.0

dev_dependencies:
  build_runner: ^2.4.0
  riverpod_generator: ^2.4.0
  freezed: ^2.5.0
  json_serializable: ^6.8.0
  retrofit_generator: ^8.1.0
  drift_dev: ^2.19.0
```

---

## 11. Testing Strategy

| Type | Tool | Coverage Target |
|------|------|-----------------|
| Unit | `flutter_test` | 80% business logic |
| Widget | `flutter_test` + `golden_toolkit` | All screens |
| Integration | `integration_test` | Critical flows (login → scan → sale → sync) |
| API Mock | `dio_mock_adapter` | All repository tests |
| E2E | Firebase Test Lab | Smoke tests on real devices |

---

*Document Version: 1.0 | Last Updated: 2026-09-04*
*Aligned with: `backend_spec_go.md` v1.0*
