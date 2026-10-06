import 'dart:convert';
import 'dart:typed_data';
import 'dart:ui' show Locale;

import 'package:http/http.dart' as http;
import 'package:uuid/uuid.dart';

import 'app_exception.dart';
import 'customers.dart';
import 'billing.dart';
import 'community.dart';
import 'dashboard.dart';
import 'inventory.dart';
import 'onboarding.dart';
import 'payments.dart';
import 'purchases.dart';
import 'receipts.dart';
import 'restaurants.dart';
import 'registers.dart';
import 'saas.dart';
import 'security.dart';
import 'selforder.dart';
import 'session_store.dart';
import 'users.dart';

/// SaaS control-plane models live in saas.dart (exported for convenience).
export 'saas.dart' show SaasSummary, SaasTenant, SaasTenantsPage;

class ApiClient {
  ApiClient({http.Client? client, this.onSessionRefreshed})
      : _client = client ?? http.Client();

  final http.Client _client;
  final Future<void> Function(Session session)? onSessionRefreshed;

  /// The store API. Production serves it on its own hostname; the console
  /// domain refuses these paths (and refuses store accounts at sign-in).
  final String baseUrl = const String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'https://posgo.xamltech.com',
  );

  /// The control plane lives on the console domain. A build that ships the
  /// platform console points this at the same host as [baseUrl]; the default
  /// keeps `/v1/saas` and `/v1/platform` off the POS domain even when a
  /// developer overrides `API_BASE_URL` for a local backend.
  final String saasBaseUrl = const String.fromEnvironment(
    'SAAS_API_BASE_URL',
    defaultValue: 'https://api.xamltech.com',
  );

  /// Newest session known to be valid. Its refresh token is the only one that
  /// has not yet been rotated, so a caller holding a stale `Session` snapshot
  /// never replays a consumed token (which would trip the backend's AUTH-007
  /// reuse detection and revoke the whole session family).
  Session? _latestSession;

  /// The refresh currently in flight. Concurrent 401s (e.g. the parallel
  /// catalog/settings/register boot load) join this one rotation instead of
  /// submitting the same refresh token twice and racing the rotation.
  Future<Session>? _refreshInFlight;

  Future<Session> login({
    required String tenantId,
    required String email,
    required String password,
    required String deviceId,
    required String deviceName,
  }) async {
    _resetAuthState();
    final response = await _client.post(
      Uri.parse('$baseUrl/v1/auth/login'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'tenant_id': tenantId,
        'email': email,
        'password': password,
        'device_id': deviceId,
        'device_name': deviceName,
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    final data = jsonDecode(response.body)['data'] as Map<String, dynamic>;
    final session = Session.fromJson(data);
    _latestSession = session;
    return session;
  }

  Future<Session> register({
    required String storeName,
    required String businessType,
    required String email,
    required String password,
    required String displayName,
    required String deviceId,
    required String deviceName,
    String countryCode = 'EG',
    String currencyCode = 'EGP',
    String language = 'ar',
    List<String> interests = const [],
    String plan = 'trial',
  }) async {
    final response = await _client.post(
      Uri.parse('$baseUrl/v1/auth/register'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'store_name': storeName,
        'business_type': businessType,
        'email': email,
        'password': password,
        'display_name': displayName,
        'country_code': countryCode,
        'currency_code': currencyCode,
        'language': language,
        'default_language': language,
        'device_id': deviceId,
        'device_name': deviceName,
        'interests': interests,
        'plan': plan,
      }),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    final data = jsonDecode(response.body)['data'] as Map<String, dynamic>;
    final session = Session.fromJson(data);
    _latestSession = session;
    return session;
  }

  Future<void> logout(Session session) async {
    _resetAuthState();
    final response = await _client.post(
      Uri.parse('$baseUrl/v1/auth/logout'),
      headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer ${session.accessToken}',
      },
    );
    if (response.statusCode != 204) {
      throw ApiException(_message(response));
    }
  }

  /// Reports a client-observability event (crash, error, screen/action) to the
  /// backend. Best-effort: callers are expected to swallow failures so
  /// telemetry can never interrupt the POS flow.
  Future<void> postClientEvent(
    Session session,
    String event, {
    String? appVersion,
    String? screen,
    String? stackTrace,
    Map<String, dynamic> payload = const {},
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/client/events'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'event': event,
          if (appVersion != null) 'app_version': appVersion,
          if (screen != null) 'screen': screen,
          if (stackTrace != null) 'stack_trace': stackTrace,
          'payload': payload,
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
  }

  /// Verifies the acting user's manager PIN. Returns the result including the
  /// remaining attempts; a 423 `pin_locked` yields a locked result so the UI
  /// can show the lockout rather than a generic failure.
  Future<PinVerifyResult> verifyPin(
    Session session, {
    required String pin,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/auth/verify-pin'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'pin': pin}),
      ),
    );
    if (response.statusCode == 423) return const PinVerifyResult.locked();
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    return PinVerifyResult.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  /// Clears another user's PIN lockout (managers may only unlock cashiers).
  Future<void> unlockPin(
    Session session, {
    required String userId,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/auth/unlock-pin'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'user_id': userId}),
      ),
    );
    if (response.statusCode != 204) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
  }

  Future<Session> refresh(Session session) => _refresh(session);

  /// Drops tracked auth state so a fresh login/register or a sign-out can never
  /// re-use the previous user's session family.
  void _resetAuthState() {
    _latestSession = null;
    _refreshInFlight = null;
  }

  Future<List<Category>> categories(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/categories'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as List<dynamic>;
    return data
        .map((item) => Category.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<SalesPage> listSales(
    Session session, {
    int page = 1,
    int limit = 50,
  }) async {
    final uri = Uri.parse('$baseUrl/v1/sales').replace(
      queryParameters: {'page': '$page', 'limit': '$limit'},
    );
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(uri, headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final data = body['data'] as List<dynamic>;
    final meta = body['meta'] as Map<String, dynamic>;
    return SalesPage(
      sales: data
          .map((item) => SaleSummary.fromJson(item as Map<String, dynamic>))
          .toList(),
      total: (meta['total'] as num).toInt(),
      page: (meta['page'] as num).toInt(),
      limit: (meta['limit'] as num).toInt(),
    );
  }

  /// Aggregated register activity for `[from, to)` — same bound syntax as
  /// [sessions]; an absent bound leaves that side of the window open.
  Future<RangeSummary> sessionSummary(
    Session session, {
    String? from,
    String? to,
  }) async {
    final uri = Uri.parse('$baseUrl/v1/registers/summary').replace(
      queryParameters: {
        if (from != null && from.isNotEmpty) 'from': from,
        if (to != null && to.isNotEmpty) 'to': to,
      },
    );
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(uri, headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    return RangeSummary.fromJson(body['data'] as Map<String, dynamic>);
  }

  Future<List<Product>> products(Session session, {String search = ''}) async {
    final uri = Uri.parse('$baseUrl/v1/products').replace(
      queryParameters: search.trim().isEmpty ? null : {'search': search.trim()},
    );
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(uri, headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as List<dynamic>;
    return data
        .map((item) => Product.fromJson(item as Map<String, dynamic>))
        .toList();
  }


  Future<Product> productByBarcode(Session session, String barcode) async {
    final uri = Uri.parse('$baseUrl/v1/products/barcode/${Uri.encodeComponent(barcode)}');
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(uri, headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    return Product.fromJson(body['data'] as Map<String, dynamic>);
  }

  Future<List<ProductLot>> productLots(Session session, String productId,
      {bool includeEmpty = false}) async {
    final uri = Uri.parse('$baseUrl/v1/products/${Uri.encodeComponent(productId)}/lots')
        .replace(queryParameters: {'include_empty': includeEmpty ? 'true' : 'false'});
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(uri, headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as List<dynamic>;
    return data.map((e) => ProductLot.fromJson(e as Map<String, dynamic>)).toList();
  }

  Future<List<ProductVariant>> productVariants(Session session, String productId) async {
    final uri =
        Uri.parse('$baseUrl/v1/products/${Uri.encodeComponent(productId)}/variants');
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(uri, headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as List<dynamic>;
    return data.map((e) => ProductVariant.fromJson(e as Map<String, dynamic>)).toList();
  }
  Future<ProductLot> createProductLot(
    Session session,
    String productId, {
    required String lotNumber,
    String expiryDate = '',
    required int quantity,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/products/${Uri.encodeComponent(productId)}/lots'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'lot_number': lotNumber,
          if (expiryDate.isNotEmpty) 'expiry_date': expiryDate,
          'quantity': quantity,
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final data = body['data'] as Map<String, dynamic>;
    return ProductLot(
      id: data['id'] as String,
      productId: data['product_id'] as String,
      lotNumber: data['lot_number'] as String,
      expiryDate: data['expiry_date'] as String?,
      serialNumber: null,
      quantityRemaining: (data['quantity'] as num).toInt(),
    );
  }

  Future<Map<String, dynamic>> updateProductLot(
    Session session,
    String lotId, {
    int? quantity,
    String? expiryDate,
  }) async {
    final bodyMap = <String, dynamic>{};
    if (quantity != null) bodyMap['quantity'] = quantity;
    if (expiryDate != null) bodyMap['expiry_date'] = expiryDate;
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse('$baseUrl/v1/lots/${Uri.encodeComponent(lotId)}'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode(bodyMap),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    return body['data'] as Map<String, dynamic>;
  }

  Future<ProductVariant> createProductVariant(
    Session session,
    String productId, {
    required String name,
    required String sku,
    required int priceMinor,
    int stockQuantity = 0,
    String size = '',
    String color = '',
    String barcode = '',
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/products/${Uri.encodeComponent(productId)}/variants'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'name': name,
          'sku': sku,
          'price_minor': priceMinor,
          if (stockQuantity != 0) 'stock_quantity': stockQuantity,
          if (size.isNotEmpty) 'size': size,
          if (color.isNotEmpty) 'color': color,
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final data = body['data'] as Map<String, dynamic>;
    return ProductVariant.fromJson(data);
  }

  Future<ProductVariant> updateProductVariant(
    Session session,
    String variantId, {
    String? name,
    String? sku,
    int? priceMinor,
    int? stockQuantity,
    String? size,
    String? color,
    bool? isActive,
  }) async {
    final bodyMap = <String, dynamic>{};
    if (name != null) bodyMap['name'] = name;
    if (sku != null) bodyMap['sku'] = sku;
    if (priceMinor != null) bodyMap['price_minor'] = priceMinor;
    if (stockQuantity != null) bodyMap['stock_quantity'] = stockQuantity;
    if (size != null) bodyMap['size'] = size;
    if (color != null) bodyMap['color'] = color;
    if (isActive != null) bodyMap['is_active'] = isActive;
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse('$baseUrl/v1/variants/${Uri.encodeComponent(variantId)}'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode(bodyMap),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final data = body['data'] as Map<String, dynamic>;
    return ProductVariant.fromJson(data);
  }

  Future<void> deleteProductVariant(
    Session session,
    String variantId,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.delete(
        Uri.parse('$baseUrl/v1/variants/${Uri.encodeComponent(variantId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
  }
  Future<Product> createProduct(
    Session session, {
    required String name,
    required String sku,
    required int priceMinor,
    required String currency,
    int costMinor = 0,
    int stockQuantity = 0,
    String? barcode,
    String? categoryId,
    String? imageUrl,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/products'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'name': name,
          'sku': sku,
          'price_minor': priceMinor,
          'cost_minor': costMinor,
          'currency': currency,
          'stock_quantity': stockQuantity,
          if (barcode != null) 'barcode': barcode,
          if (categoryId != null) 'category_id': categoryId,
          if (imageUrl != null) 'image_url': imageUrl,
        }),
      ),
    );
    if (response.statusCode != 201) throw ApiException(_message(response));
    return Product.fromJson(jsonDecode(response.body)['data']);
  }

  Future<Product> updateProduct(
    Session session,
    String productId, {
    String? name,
    String? sku,
    int? priceMinor,
    int? costMinor,
    String? currency,
    int? stockQuantity,
    String? barcode,
    String? categoryId,
    String? imageUrl,
    bool? isActive,
    bool? selforderEnabled,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse('$baseUrl/v1/products/$productId'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (name != null) 'name': name,
          if (sku != null) 'sku': sku,
          if (priceMinor != null) 'price_minor': priceMinor,
          if (costMinor != null) 'cost_minor': costMinor,
          if (currency != null) 'currency': currency,
          if (stockQuantity != null) 'stock_quantity': stockQuantity,
          if (barcode != null) 'barcode': barcode,
          if (categoryId != null) 'category_id': categoryId,
          if (imageUrl != null) 'image_url': imageUrl,
          if (isActive != null) 'is_active': isActive,
          if (selforderEnabled != null) 'selforder_enabled': selforderEnabled,
        }),
      ),
    );
    if (response.statusCode != 200) throw ApiException(_message(response));
    return Product.fromJson(jsonDecode(response.body)['data']);
  }

  Future<SaleResult> createSale(
    Session session,
    List<SaleItemInput> items, {
    String? idempotencyKey,
    List<PaymentInput>? payments,
    String? sessionId,
    String? tableId,
    int discountMinor = 0,
    String managerPin = '',
    int roundingMinor = 0,
  }) async {
    final requestIdempotencyKey = idempotencyKey ?? const Uuid().v4();
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/sales'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
          'Idempotency-Key': requestIdempotencyKey,
        },
        body: jsonEncode({
          'items': items
              .map((item) => {
                    'product_id': item.productId,
                    'quantity': item.quantity,
                  })
              .toList(),
          if (payments != null && payments.isNotEmpty)
            'payments': payments.map((p) => p.toJson()).toList(),
          if (sessionId != null) 'session_id': sessionId,
          if (tableId != null && tableId.isNotEmpty) 'table_id': tableId,
          if (discountMinor > 0) 'discount_minor': discountMinor,
          if (managerPin.trim().isNotEmpty) 'manager_pin': managerPin.trim(),
          if (roundingMinor > 0) 'rounding_minor': roundingMinor,
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as Map<String, dynamic>;
    return SaleResult.fromJson(data);
  }

  Future<List<Country>> countries() async {
    final response = await _client.get(
      Uri.parse('$baseUrl/v1/meta/countries'),
      headers: const {'Accept': 'application/json'},
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as List<dynamic>;
    return data
        .map((item) => Country.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<List<Currency>> currencies() async {
    final response = await _client.get(
      Uri.parse('$baseUrl/v1/meta/currencies'),
      headers: const {'Accept': 'application/json'},
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as List<dynamic>;
    return data
        .map((item) => Currency.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<SaasSummary> saasSummary(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$saasBaseUrl/v1/saas/summary'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return SaasSummary.fromJson(jsonDecode(response.body)['data']);
  }

  Future<SaasTenantsPage> saasTenants(
    Session session, {
    int page = 1,
    int limit = 50,
    String? q,
    String? businessType,
    String? plan,
    String? status,
    bool includeInternal = false,
  }) async {
    final uri = Uri.parse('$saasBaseUrl/v1/saas/tenants').replace(
      queryParameters: {
        'page': '$page',
        'limit': '$limit',
        if (q != null && q.isNotEmpty) 'q': q,
        if (businessType != null && businessType.isNotEmpty)
          'business_type': businessType,
        if (plan != null && plan.isNotEmpty) 'plan': plan,
        if (status != null && status.isNotEmpty) 'status': status,
        if (includeInternal) 'include_internal': '1',
      },
    );
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(uri, headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final meta = body['meta'] as Map<String, dynamic>;
    return SaasTenantsPage(
      tenants: (body['data'] as List<dynamic>)
          .map((item) => SaasTenant.fromJson(item as Map<String, dynamic>))
          .toList(),
      total: (meta['total'] as num?)?.toInt() ?? 0,
      page: (meta['page'] as num?)?.toInt() ?? page,
      limit: (meta['limit'] as num?)?.toInt() ?? limit,
    );
  }

  Future<DashboardSummary> dashboardSummary(
    Session session, {
    String vatMode = 'exclusive',
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
            '$baseUrl/v1/dashboard/summary?vat=${Uri.encodeQueryComponent(vatMode)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return DashboardSummary.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<TenantAnalytics> saasTenantAnalytics(
    Session session,
    String tenantId,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$saasBaseUrl/v1/saas/tenants/$tenantId/analytics'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return TenantAnalytics.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<BillingSummaryData> billingSummary(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$saasBaseUrl/v1/saas/billing/summary'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return BillingSummaryData.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<List<BillingPlan>> billingPlans(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$saasBaseUrl/v1/saas/plans'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return (jsonDecode(response.body)['data'] as List<dynamic>)
        .map((e) => BillingPlan.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<List<PaymentProvider>> paymentProviders(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$saasBaseUrl/v1/saas/payment-providers'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return (jsonDecode(response.body)['data'] as List<dynamic>)
        .map((e) => PaymentProvider.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<BillingSubscriptionsPage> billingSubscriptions(
    Session session, {
    String? status,
    int page = 1,
    int limit = 50,
  }) async {
    final uri = Uri.parse('$saasBaseUrl/v1/saas/subscriptions').replace(
      queryParameters: {
        'page': '$page',
        'limit': '$limit',
        if (status != null && status.isNotEmpty) 'status': status,
      },
    );
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(uri, headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return _subscriptionsPage(response, page, limit);
  }

  Future<BillingSubscription> tenantSubscription(
    Session session,
    String tenantId,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$saasBaseUrl/v1/saas/tenants/$tenantId/subscription'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return BillingSubscription.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<BillingSubscription> assignSubscription(
    Session session,
    String tenantId, {
    required String planCode,
    String? status,
    int? trialDays,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/saas/tenants/$tenantId/subscription'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'plan_code': planCode,
          if (status != null && status.isNotEmpty) 'status': status,
          if (trialDays != null) 'trial_days': trialDays,
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    return BillingSubscription.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<BillingSubscription> setSubscriptionStatus(
    Session session,
    String subscriptionId,
    String status,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/saas/subscriptions/$subscriptionId/status'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'status': status}),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return BillingSubscription.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<BillingSubscription> changeSubscriptionPlan(
    Session session,
    String subscriptionId,
    String planCode,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/saas/subscriptions/$subscriptionId/change-plan'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'plan_code': planCode}),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return BillingSubscription.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<BillingInvoicesPage> billingInvoices(
    Session session, {
    String? status,
    int page = 1,
    int limit = 50,
  }) async {
    final uri = Uri.parse('$saasBaseUrl/v1/saas/invoices').replace(
      queryParameters: {
        'page': '$page',
        'limit': '$limit',
        if (status != null && status.isNotEmpty) 'status': status,
      },
    );
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(uri, headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final meta = body['meta'] as Map<String, dynamic>;
    return BillingInvoicesPage(
      invoices: (body['data'] as List<dynamic>)
          .map((e) => BillingInvoice.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: (meta['total'] as num?)?.toInt() ?? 0,
      page: (meta['page'] as num?)?.toInt() ?? page,
      limit: (meta['limit'] as num?)?.toInt() ?? limit,
    );
  }

  Future<BillingInvoice> createInvoice(
    Session session,
    String tenantId, {
    required int amountMinor,
    required String description,
    String currency = 'EGP',
    String? dueAt,
    String? provider,
    String? subscriptionId,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/saas/tenants/$tenantId/invoices'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'amount_minor': amountMinor,
          'currency': currency,
          'description': description,
          if (dueAt != null && dueAt.isNotEmpty) 'due_at': dueAt,
          if (provider != null && provider.isNotEmpty) 'provider': provider,
          if (subscriptionId != null && subscriptionId.isNotEmpty)
            'subscription_id': subscriptionId,
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    return BillingInvoice.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<BillingInvoice> payInvoice(
    Session session,
    String invoiceId, {
    String? provider,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/saas/invoices/$invoiceId/pay'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (provider != null && provider.isNotEmpty) 'provider': provider,
        }),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return BillingInvoice.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<BillingInvoice> voidInvoice(
    Session session,
    String invoiceId,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/saas/invoices/$invoiceId/void'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode(const {}),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return BillingInvoice.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<BillingInvoice> refundInvoice(
    Session session,
    String invoiceId,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/saas/invoices/$invoiceId/refund'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode(const {}),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return BillingInvoice.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<SaasUsersPage> saasUsers(
    Session session, {
    String? tenantId,
    int page = 1,
    int limit = 50,
  }) async {
    final uri = Uri.parse('$saasBaseUrl/v1/saas/users').replace(
      queryParameters: {
        'page': '$page',
        'limit': '$limit',
        if (tenantId != null && tenantId.isNotEmpty) 'tenant_id': tenantId,
      },
    );
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(uri, headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final meta = body['meta'] as Map<String, dynamic>;
    return SaasUsersPage(
      users: (body['data'] as List<dynamic>)
          .map((e) => SaasUser.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: (meta['total'] as num?)?.toInt() ?? 0,
      page: (meta['page'] as num?)?.toInt() ?? page,
      limit: (meta['limit'] as num?)?.toInt() ?? limit,
    );
  }

  Future<SaasUser> createSaasUser(
    Session session,
    String tenantId, {
    required String email,
    required String displayName,
    required String password,
    required String role,
    String? accountType,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/saas/tenants/$tenantId/users'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'email': email,
          'display_name': displayName,
          'password': password,
          'role': role,
          if (accountType != null && accountType.isNotEmpty)
            'account_type': accountType,
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    return SaasUser.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  BillingSubscriptionsPage _subscriptionsPage(
      http.Response response, int page, int limit) {
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final meta = body['meta'] as Map<String, dynamic>;
    return BillingSubscriptionsPage(
      subscriptions: (body['data'] as List<dynamic>)
          .map((e) => BillingSubscription.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: (meta['total'] as num?)?.toInt() ?? 0,
      page: (meta['page'] as num?)?.toInt() ?? page,
      limit: (meta['limit'] as num?)?.toInt() ?? limit,
    );
  }

  Future<List<TrialEntitlement>> platformTrialEntitlements(
    Session session, {
    int limit = 100,
    int offset = 0,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
            '$saasBaseUrl/v1/platform/trial/entitlements?limit=$limit&offset=$offset'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as Map<String, dynamic>;
    return (data['entitlements'] as List<dynamic>? ?? [])
        .map((e) => TrialEntitlement.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<TrialEntitlement> trialChange(
    Session session,
    String entitlementId,
    String action, {
    int? extraDays,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse(
            '$saasBaseUrl/v1/platform/trial/entitlements/$entitlementId/$action'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (extraDays != null) 'extra_days': extraDays,
        }),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return TrialEntitlement.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<List<AuditEntry>> platformAudit(
    Session session, {
    int limit = 100,
    int offset = 0,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$saasBaseUrl/v1/platform/audit?limit=$limit&offset=$offset'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return AuditPage.fromJson(jsonDecode(response.body) as Map<String, dynamic>)
        .entries;
  }

  Future<TrialPolicy> platformTrialSettings(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$saasBaseUrl/v1/platform/trial/settings'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return TrialPolicy.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<TrialPolicy> platformUpdateTrialSettings(
    Session session,
    TrialPolicy policy,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.put(
        Uri.parse('$saasBaseUrl/v1/platform/trial/settings'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode(policy.toJson()),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return TrialPolicy.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<void> platformSuspendTenant(
    Session session,
    String tenantId, {
    String reason = '',
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/platform/tenants/$tenantId/suspend'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (reason.isNotEmpty) 'reason': reason,
        }),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
  }

  Future<void> platformActivateTenant(
    Session session,
    String tenantId,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/platform/tenants/$tenantId/activate'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: '',
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
  }

  /// Permanently disables a tenant (status `disabled`) and cancels its live
  /// subscription. Distinct from suspend: a stopped store can no longer sign
  /// in at all (403 `organization_stopped`).
  Future<void> platformStopTenant(
    Session session,
    String tenantId, {
    String reason = '',
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/platform/tenants/$tenantId/stop'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (reason.isNotEmpty) 'reason': reason,
        }),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
  }

  /// Streams the per-tenant, RLS-scoped JSON backup (`application/json`). The
  /// body is a single document: tenant_id/slug/exported_at + a key per
  /// tenant-scoped table holding its rows (never platform tables).
  Future<String?> platformBackupTenant(
    Session session,
    String tenantId,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$saasBaseUrl/v1/platform/tenants/$tenantId/backup'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: '',
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return response.body;
  }
  // IDENTITY
  Future<Map<String, dynamic>> identitySummary(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/identity'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    return body['data'] as Map<String, dynamic>;
  }

  Future<Map<String, dynamic>> identityEmailVerify(Session session,
      {required String code}) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/identity/email/verify'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'code': code}),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    return body['data'] as Map<String, dynamic>;
  }

  Future<void> identityEmailResend(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/identity/email/resend'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: '',
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
  }
  // NOTIFICATIONS
  Future<List<Map<String, dynamic>>> listNotifications(Session session,
      {int limit = 50}) async {
    final uri =
        Uri.parse('$baseUrl/v1/notifications').replace(queryParameters: {
      'limit': limit.toString(),
    });
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        uri,
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final data = body['data'] as List<dynamic>;
    return data.map((e) => e as Map<String, dynamic>).toList();
  }

  Future<int> notificationsCount(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/notifications/count'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final data = body['data'] as Map<String, dynamic>;
    return (data['count'] as num).toInt();
  }

  Future<CustomersPage> listCustomers(
    Session session, {
    String query = '',
    int page = 1,
    int limit = 50,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
          '$baseUrl/v1/customers?page=$page&limit=$limit${query.isEmpty ? '' : '&q=${Uri.encodeQueryComponent(query)}'}',
        ),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    final data = jsonDecode(respond.body);
    final meta = (data['meta'] as Map<String, dynamic>?) ?? const {};
    return CustomersPage(
      customers: (data['data'] as List<dynamic>? ?? [])
          .map((e) => Customer.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: _toIntValue(meta['total']),
      page: _toIntValue(meta['page']) == 0 ? page : _toIntValue(meta['page']),
      limit:
          _toIntValue(meta['limit']) == 0 ? limit : _toIntValue(meta['limit']),
    );
  }

  Future<Customer> createCustomer(
    Session session, {
    required String name,
    String email = '',
    String phone = '',
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/customers'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'name': name,
          if (email.isNotEmpty) 'email': email,
          if (phone.isNotEmpty) 'phone': phone,
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond));
    }
    return Customer.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<TenantUsersPage> listUsers(
    Session session, {
    int page = 1,
    int limit = 50,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/users?page=$page&limit=$limit'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    final data = jsonDecode(response.body) as Map<String, dynamic>;
    final meta = (data['meta'] as Map<String, dynamic>?) ?? const {};
    return TenantUsersPage(
      users: (data['data'] as List<dynamic>? ?? [])
          .map((e) => TenantUser.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: _toIntValue(meta['total']),
      page: _toIntValue(meta['page']) == 0 ? page : _toIntValue(meta['page']),
      limit:
          _toIntValue(meta['limit']) == 0 ? limit : _toIntValue(meta['limit']),
    );
  }

  Future<TenantUser> createUser(
    Session session, {
    required String email,
    required String displayName,
    required String password,
    required String role,
    String accountType = 'standard',
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/users'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'email': email,
          'display_name': displayName,
          'password': password,
          'role': role,
          'account_type': accountType,
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    return TenantUser.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  /// Saves a user's raw POS security profile (`PUT /users/:id/security`).
  /// Returns the RESOLVED permissions the backend computed from the saved row.
  Future<UserPermissions> updateUserSecurity(
    Session session, {
    required String userId,
    required UserSecurityProfile security,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.put(
        Uri.parse('$baseUrl/v1/users/$userId/security'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode(security.toJson()),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    final data = jsonDecode(response.body) as Map<String, dynamic>;
    final payload = data['data'] as Map<String, dynamic>? ?? const {};
    return UserPermissions.fromJson(
        payload['permissions'] as Map<String, dynamic>? ?? const {});
  }

  Future<ProfilesPage> listProfiles(
    Session session, {
    String query = '',
    String? role,
    int page = 1,
    int limit = 50,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
          '$baseUrl/v1/community/profiles?page=$page&limit=$limit${query.isEmpty ? '' : '&q=${Uri.encodeQueryComponent(query)}'}${role == null || role.isEmpty ? '' : '&role=$role'}',
        ),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final data = jsonDecode(respond.body);
    final meta = (data['meta'] as Map<String, dynamic>?) ?? const {};
    return ProfilesPage(
      profiles: (data['data'] as List<dynamic>? ?? [])
          .map((e) => StaffProfile.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: _toIntValue(meta['total']),
      page: _toIntValue(meta['page']) == 0 ? page : _toIntValue(meta['page']),
      limit:
          _toIntValue(meta['limit']) == 0 ? limit : _toIntValue(meta['limit']),
    );
  }

  Future<StaffProfile> myProfile(Session session) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/community/profiles/me'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return StaffProfile.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<StaffProfile> getProfile(Session session, String userId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
            '$baseUrl/v1/community/profiles/${Uri.encodeComponent(userId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final decoded = jsonDecode(respond.body) as Map<String, dynamic>;
    final data =
        (decoded['data'] as Map<String, dynamic>?) ?? <String, dynamic>{};
    final memberships = (decoded['memberships'] as List<dynamic>? ?? [])
        .cast<Map<String, dynamic>>();
    if (memberships.isNotEmpty) {
      data['memberships'] = memberships;
    }
    return StaffProfile.fromJson(data);
  }

  Future<StaffProfile> updateMyProfile(
    Session session, {
    required String headline,
    required String bio,
    required String location,
    required int yearsExperience,
    required bool isChief,
    required List<String> skills,
    Map<String, dynamic>? resume,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.put(
        Uri.parse('$baseUrl/v1/community/profiles/me'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'headline': headline,
          'bio': bio,
          'location': location,
          'years_experience': yearsExperience,
          'is_chief': isChief,
          'skills': skills,
          if (resume != null) 'resume': resume,
        }),
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return StaffProfile.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<CompaniesPage> listCompanies(
    Session session, {
    String query = '',
    int page = 1,
    int limit = 50,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
          '$baseUrl/v1/community/companies?page=$page&limit=$limit${query.isEmpty ? '' : '&q=${Uri.encodeQueryComponent(query)}'}',
        ),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final data = jsonDecode(respond.body);
    final meta = (data['meta'] as Map<String, dynamic>?) ?? const {};
    return CompaniesPage(
      companies: (data['data'] as List<dynamic>? ?? [])
          .map((e) => Company.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: _toIntValue(meta['total']),
      page: _toIntValue(meta['page']) == 0 ? page : _toIntValue(meta['page']),
      limit:
          _toIntValue(meta['limit']) == 0 ? limit : _toIntValue(meta['limit']),
    );
  }

  Future<Company> getCompany(Session session, String companyId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
            '$baseUrl/v1/community/companies/${Uri.encodeComponent(companyId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return Company.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<Company> createCompany(
    Session session, {
    required String name,
    String slug = '',
    String description = '',
    String industry = '',
    String website = '',
    String city = '',
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/community/companies'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'name': name,
          if (slug.isNotEmpty) 'slug': slug,
          if (description.isNotEmpty) 'description': description,
          if (industry.isNotEmpty) 'industry': industry,
          if (website.isNotEmpty) 'website': website,
          if (city.isNotEmpty) 'city': city,
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return Company.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<Company> updateCompany(
    Session session,
    String companyId, {
    String? name,
    String? slug,
    String? description,
    String? industry,
    String? website,
    String? city,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse(
            '$baseUrl/v1/community/companies/${Uri.encodeComponent(companyId)}'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (name != null) 'name': name,
          if (slug != null) 'slug': slug,
          if (description != null) 'description': description,
          if (industry != null) 'industry': industry,
          if (website != null) 'website': website,
          if (city != null) 'city': city,
        }),
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return Company.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<void> deleteCompany(Session session, String companyId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.delete(
        Uri.parse(
            '$baseUrl/v1/community/companies/${Uri.encodeComponent(companyId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
  }

  Future<CompanyMember> addCompanyMember(
    Session session,
    String companyId, {
    required String userId,
    String role = 'staff',
    String title = '',
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse(
            '$baseUrl/v1/community/companies/${Uri.encodeComponent(companyId)}/members'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'user_id': userId,
          'role': role,
          if (title.isNotEmpty) 'title': title,
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return CompanyMember.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<void> removeCompanyMember(
      Session session, String companyId, String userId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.delete(
        Uri.parse(
            '$baseUrl/v1/community/companies/${Uri.encodeComponent(companyId)}/members/${Uri.encodeComponent(userId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
  }

  /// The caller's own national-community member row, or `null` when not a
  /// member yet (`404 not_a_member`).
  Future<NationalMember?> nationalMe(Session session) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/community/me'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode == 404) return null;
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return NationalMember.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  /// Join the national community with an invitation code (any casing and the
  /// optional `EG-` display prefix are normalized before sending).
  Future<NationalMember> nationalJoin(Session session, String code) async {
    var normalized =
        code.trim().toUpperCase().replaceAll(RegExp('[^A-Z0-9]'), '');
    if (normalized.startsWith('EG')) {
      normalized = normalized.substring(2);
    }
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/community/join'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'code': normalized}),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return NationalMember.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<List<NationalInvitation>> listNationalInvitations(
      Session session) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/community/invitations'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return (jsonDecode(respond.body)['data'] as List<dynamic>? ?? [])
        .map((e) => NationalInvitation.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<NationalInvitation> createNationalInvitation(
    Session session, {
    String email = '',
    String note = '',
    int maxUses = 1,
    String? expiresAt,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/community/invitations'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (email.isNotEmpty) 'email': email,
          if (note.isNotEmpty) 'note': note,
          'max_uses': maxUses,
          if (expiresAt != null && expiresAt.isNotEmpty)
            'expires_at': expiresAt,
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return NationalInvitation.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<void> revokeNationalInvitation(
      Session session, String invitationId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.delete(
        Uri.parse(
            '$baseUrl/v1/community/invitations/${Uri.encodeComponent(invitationId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
  }

  Future<NationalMembersPage> listNationalMembers(
    Session session, {
    String query = '',
    String? level,
    String? role,
    int page = 1,
    int limit = 50,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
          '$baseUrl/v1/community/members'
          '?page=$page&limit=$limit'
          '${query.isEmpty ? '' : '&q=${Uri.encodeQueryComponent(query)}'}'
          '${level == null || level.isEmpty ? '' : '&level=$level'}'
          '${role == null || role.isEmpty ? '' : '&role=$role'}',
        ),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final data = jsonDecode(respond.body);
    final meta = (data['meta'] as Map<String, dynamic>?) ?? const {};
    return NationalMembersPage(
      members: (data['data'] as List<dynamic>? ?? [])
          .map((e) => NationalMember.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: _toIntValue(meta['total']),
      page: _toIntValue(meta['page']) == 0 ? page : _toIntValue(meta['page']),
      limit:
          _toIntValue(meta['limit']) == 0 ? limit : _toIntValue(meta['limit']),
    );
  }

  Future<NationalMember> getNationalMember(
      Session session, String userId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
            '$baseUrl/v1/community/members/${Uri.encodeComponent(userId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return NationalMember.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<NationalMember> updateNationalMember(
    Session session,
    String userId, {
    String? status,
    String? role,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse(
            '$baseUrl/v1/community/members/${Uri.encodeComponent(userId)}'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (status != null && status.isNotEmpty) 'status': status,
          if (role != null && role.isNotEmpty) 'role': role,
        }),
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return NationalMember.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<JobsPage> listJobs(
    Session session, {
    String query = '',
    String companyId = '',
    String type = '',
    bool mine = false,
    int page = 1,
    int limit = 50,
  }) async {
    final buffer = StringBuffer('/v1/community/jobs?page=$page&limit=$limit');
    if (query.isNotEmpty) {
      buffer.write('&q=${Uri.encodeQueryComponent(query)}');
    }
    if (companyId.isNotEmpty) {
      buffer.write('&company_id=${Uri.encodeComponent(companyId)}');
    }
    if (type.isNotEmpty) {
      buffer.write('&type=${Uri.encodeComponent(type)}');
    }
    if (mine) {
      buffer.write('&mine=1');
    }
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl$buffer'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final data = jsonDecode(respond.body);
    final meta = (data['meta'] as Map<String, dynamic>?) ?? const {};
    return JobsPage(
      jobs: (data['data'] as List<dynamic>? ?? [])
          .map((e) => JobOffer.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: _toIntValue(meta['total']),
      page: _toIntValue(meta['page']) == 0 ? page : _toIntValue(meta['page']),
      limit:
          _toIntValue(meta['limit']) == 0 ? limit : _toIntValue(meta['limit']),
    );
  }

  Future<JobOffer> getJob(Session session, String jobId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/community/jobs/${Uri.encodeComponent(jobId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return JobOffer.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<JobOffer> createJob(
    Session session, {
    required String title,
    String description = '',
    String employmentType = 'full_time',
    String location = '',
    int salaryMinor = 0,
    String salaryCurrency = 'EGP',
    List<String> skillTags = const [],
    String companyId = '',
    String closesAt = '',
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/community/jobs'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'title': title,
          if (description.isNotEmpty) 'description': description,
          'employment_type': employmentType,
          if (location.isNotEmpty) 'location': location,
          if (salaryMinor > 0) 'salary_minor': salaryMinor,
          'salary_currency': salaryCurrency,
          if (skillTags.isNotEmpty) 'skill_tags': skillTags,
          if (companyId.isNotEmpty) 'company_id': companyId,
          if (closesAt.isNotEmpty) 'closes_at': closesAt,
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return JobOffer.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<JobOffer> updateJob(
    Session session,
    String jobId, {
    String? title,
    String? description,
    String? employmentType,
    String? location,
    int? salaryMinor,
    String? salaryCurrency,
    List<String>? skillTags,
    String? companyId,
    String? closesAt,
    bool? isActive,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse('$baseUrl/v1/community/jobs/${Uri.encodeComponent(jobId)}'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (title != null && title.isNotEmpty) 'title': title,
          if (description != null) 'description': description,
          if (employmentType != null && employmentType.isNotEmpty)
            'employment_type': employmentType,
          if (location != null) 'location': location,
          if (salaryMinor != null) 'salary_minor': salaryMinor,
          if (salaryCurrency != null && salaryCurrency.isNotEmpty)
            'salary_currency': salaryCurrency,
          if (skillTags != null) 'skill_tags': skillTags,
          if (companyId != null) 'company_id': companyId,
          if (closesAt != null) 'closes_at': closesAt,
          if (isActive != null) 'is_active': isActive,
        }),
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return JobOffer.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<void> deleteJob(Session session, String jobId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.delete(
        Uri.parse('$baseUrl/v1/community/jobs/${Uri.encodeComponent(jobId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
  }

  Future<JobApplication> applyToJob(
    Session session,
    String jobId, {
    String coverNote = '',
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse(
            '$baseUrl/v1/community/jobs/${Uri.encodeComponent(jobId)}/apply'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'cover_note': coverNote}),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return JobApplication.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<JobApplicationsPage> listJobApplications(
      Session session, String jobId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
            '$baseUrl/v1/community/jobs/${Uri.encodeComponent(jobId)}/applications'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final data = jsonDecode(respond.body);
    final meta = (data['meta'] as Map<String, dynamic>?) ?? const {};
    return JobApplicationsPage(
      applications: (data['data'] as List<dynamic>? ?? [])
          .map((e) => JobApplication.fromJson(e as Map<String, dynamic>))
          .toList(),
      appliedTotal: _toIntValue(meta['applied_total']),
    );
  }

  Future<void> updateApplicationStatus(
      Session session, String applicationId, String status) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse(
            '$baseUrl/v1/community/applications/${Uri.encodeComponent(applicationId)}'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'status': status}),
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
  }

  Future<List<BadgeDefinition>> listBadges(
    Session session, {
    String query = '',
    bool includeInactive = false,
    int page = 1,
    int limit = 100,
  }) async {
    final parameters = <String, String>{
      'page': '$page',
      'limit': '$limit',
      if (query.isNotEmpty) 'q': query,
      if (includeInactive) 'all': '1',
    };
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/community/badges')
            .replace(queryParameters: parameters),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return (jsonDecode(respond.body)['data'] as List<dynamic>? ?? [])
        .map((e) => BadgeDefinition.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<BadgeDefinition> createBadge(
    Session session, {
    required String code,
    required String name,
    String description = '',
    String icon = 'military_tech',
    Map<String, dynamic> criteria = const {},
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/community/badges'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'code': code,
          'name': name,
          if (description.isNotEmpty) 'description': description,
          'icon': icon,
          if (criteria.isNotEmpty) 'criteria': criteria,
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return BadgeDefinition.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<BadgeDefinition> updateBadge(
    Session session,
    String badgeId, {
    String? name,
    String? description,
    String? icon,
    bool? isActive,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse('$baseUrl/v1/community/badges/${Uri.encodeComponent(badgeId)}'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (name != null) 'name': name,
          if (description != null) 'description': description,
          if (icon != null) 'icon': icon,
          if (isActive != null) 'is_active': isActive,
        }),
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return BadgeDefinition.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<List<UserBadge>> listProfileBadges(
      Session session, String userId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
            '$baseUrl/v1/community/profiles/${Uri.encodeComponent(userId)}/badges'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return (jsonDecode(respond.body)['data'] as List<dynamic>? ?? [])
        .map((e) => UserBadge.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<UserBadge> awardProfileBadge(
    Session session,
    String userId, {
    required String badgeId,
    String note = '',
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse(
            '$baseUrl/v1/community/profiles/${Uri.encodeComponent(userId)}/badges'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'badge_id': badgeId,
          if (note.isNotEmpty) 'note': note,
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return UserBadge.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<void> revokeProfileBadge(
      Session session, String userId, String badgeId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.delete(
        Uri.parse(
            '$baseUrl/v1/community/profiles/${Uri.encodeComponent(userId)}/badges/${Uri.encodeComponent(badgeId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
  }

  Future<List<EndorsementGroup>> listProfileEndorsements(
      Session session, String userId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
            '$baseUrl/v1/community/profiles/${Uri.encodeComponent(userId)}/endorsements'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return (jsonDecode(respond.body)['data'] as List<dynamic>? ?? [])
        .map((e) => EndorsementGroup.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  Future<Endorsement> endorsePeer(
    Session session, {
    required String receiverId,
    required String skill,
    String note = '',
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/community/endorsements'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'receiver_id': receiverId,
          'skill': skill,
          if (note.isNotEmpty) 'note': note,
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return Endorsement.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<void> withdrawEndorsement(Session session, String endorsementId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.delete(
        Uri.parse(
            '$baseUrl/v1/community/endorsements/${Uri.encodeComponent(endorsementId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
  }

  Future<ForumCategoriesPage> listForumCategories(
    Session session, {
    bool includeInactive = false,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
            '$baseUrl/v1/community/forum/categories${includeInactive ? '?all=1' : ''}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final data = jsonDecode(respond.body);
    final meta = (data['meta'] as Map<String, dynamic>?) ?? const {};
    return ForumCategoriesPage(
      categories: (data['data'] as List<dynamic>? ?? [])
          .map((e) => ForumCategory.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: _toIntValue(meta['total']),
    );
  }

  Future<ForumCategory> createForumCategory(
    Session session, {
    required String name,
    String slug = '',
    String description = '',
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/community/forum/categories'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'name': name,
          if (slug.isNotEmpty) 'slug': slug,
          if (description.isNotEmpty) 'description': description,
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return ForumCategory.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<ForumCategory> updateForumCategory(
    Session session,
    String categoryId, {
    String? name,
    String? slug,
    String? description,
    bool? isActive,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse(
            '$baseUrl/v1/community/forum/categories/${Uri.encodeComponent(categoryId)}'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (name != null) 'name': name,
          if (slug != null) 'slug': slug,
          if (description != null) 'description': description,
          if (isActive != null) 'is_active': isActive,
        }),
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return ForumCategory.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<ForumPostsPage> listForumPosts(
    Session session, {
    String query = '',
    String categoryId = '',
    String sort = 'latest',
    int page = 1,
    int limit = 50,
  }) async {
    final buffer = StringBuffer('/v1/community/forum/posts?page=$page&limit=$limit');
    if (query.isNotEmpty) {
      buffer.write('&q=${Uri.encodeQueryComponent(query)}');
    }
    if (categoryId.isNotEmpty) {
      buffer.write('&category_id=${Uri.encodeComponent(categoryId)}');
    }
    if (sort.isNotEmpty) {
      buffer.write('&sort=$sort');
    }
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl$buffer'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final data = jsonDecode(respond.body);
    final meta = (data['meta'] as Map<String, dynamic>?) ?? const {};
    return ForumPostsPage(
      posts: (data['data'] as List<dynamic>? ?? [])
          .map((e) => ForumPost.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: _toIntValue(meta['total']),
      page: _toIntValue(meta['page']) == 0 ? page : _toIntValue(meta['page']),
      limit:
          _toIntValue(meta['limit']) == 0 ? limit : _toIntValue(meta['limit']),
    );
  }

  Future<ForumPost> createForumPost(
    Session session, {
    required String categoryId,
    required String title,
    String body = '',
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/community/forum/posts'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'category_id': categoryId,
          'title': title,
          if (body.isNotEmpty) 'body': body,
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return ForumPost.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<ForumPostDetail> getForumPost(Session session, String postId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
            '$baseUrl/v1/community/forum/posts/${Uri.encodeComponent(postId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final data = jsonDecode(respond.body) as Map<String, dynamic>;
    return ForumPostDetail(
      post: ForumPost.fromJson(data['data'] as Map<String, dynamic>),
      comments: (data['comments'] as List<dynamic>? ?? [])
          .map((e) => ForumComment.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  Future<ForumPost> updateForumPost(
    Session session,
    String postId, {
    String? title,
    String? body,
    bool? isPinned,
    bool? isClosed,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse(
            '$baseUrl/v1/community/forum/posts/${Uri.encodeComponent(postId)}'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (title != null) 'title': title,
          if (body != null) 'body': body,
          if (isPinned != null) 'is_pinned': isPinned,
          if (isClosed != null) 'is_closed': isClosed,
        }),
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return ForumPost.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<void> deleteForumPost(Session session, String postId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.delete(
        Uri.parse(
            '$baseUrl/v1/community/forum/posts/${Uri.encodeComponent(postId)}'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
  }

  Future<ForumComment> createForumComment(
    Session session,
    String postId, {
    required String body,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse(
            '$baseUrl/v1/community/forum/posts/${Uri.encodeComponent(postId)}/comments'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'body': body}),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return ForumComment.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<ForumLikeResult> toggleForumLike(Session session, String postId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse(
            '$baseUrl/v1/community/forum/posts/${Uri.encodeComponent(postId)}/like'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final data =
        jsonDecode(respond.body)['data'] as Map<String, dynamic>? ?? const {};
    return ForumLikeResult(
      liked: data['liked'] as bool? ?? false,
      likeCount: _toIntValue(data['like_count']),
    );
  }

  Future<AdjustmentResult> createInventoryAdjustment(
    Session session, {
    required String productId,
    required String reason,
    required int quantityDelta,
    String note = '',
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/inventory/adjustments'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'product_id': productId,
          'reason': reason,
          'quantity_delta': quantityDelta,
          if (note.isNotEmpty) 'note': note,
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond));
    }
    return AdjustmentResult.fromJson(
        jsonDecode(respond.body) as Map<String, dynamic>);
  }

  Future<InventoryAdjustmentPage> listInventoryAdjustments(
    Session session, {
    int page = 1,
    int limit = 50,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/inventory/adjustments?page=$page&limit=$limit'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    return InventoryAdjustmentPage.fromJson(
        jsonDecode(respond.body) as Map<String, dynamic>);
  }

  /// Uploads an invoice photo and returns the OCR-proposed purchase lines.
  /// The merchant reviews them (edit qty/price, map to products, set sale
  /// prices) before [applyPurchase] commits anything. A tenant past its OCR
  /// window limits with no credits left receives 402 `ocr_window_limit_reached`.
  Future<OcrScanResult> ocrScanInvoice(
    Session session,
    Uint8List imageBytes, {
    String filename = 'invoice.jpg',
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) async {
        final request = http.MultipartRequest(
          'POST',
          Uri.parse('$baseUrl/v1/purchases/ocr'),
        );
        request.headers['Accept'] = 'application/json';
        request.headers['Authorization'] = 'Bearer $accessToken';
        request.files.add(http.MultipartFile.fromBytes(
          'file',
          imageBytes,
          filename: filename,
        ));
        final streamed = await request.send();
        return http.Response.fromStream(streamed);
      },
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final data = jsonDecode(respond.body)['data'] as Map<String, dynamic>;
    return OcrScanResult.fromJson(data);
  }

  /// Commits a reviewed purchase: for each line the backend matches/adds the
  /// product, adds stock, sets cost to the invoice unit price, and syncs the
  /// product unit. New products are created with sale price = cost; the client
  /// patches a desired sale price afterwards via [updateProduct].
  Future<ApplyPurchaseResult> applyPurchase(
    Session session, {
    String supplier = '',
    String invoiceNo = '',
    String currency = 'EGP',
    int taxMinor = 0,
    String ocrText = '',
    required List<PurchaseLineInput> items,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/purchases'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'supplier': supplier,
          'invoice_no': invoiceNo,
          'currency': currency,
          'tax_minor': taxMinor,
          'ocr_text': ocrText,
          'items': items.map((item) => item.toJson()).toList(),
        }),
      ),
    );
    if (respond.statusCode != 201) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    return ApplyPurchaseResult.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  /// Paginated purchase ledger (newest first).
  Future<PurchasesPage> listPurchases(
    Session session, {
    int page = 1,
    int limit = 50,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/purchases?page=$page&limit=$limit'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    return PurchasesPage.fromJson(
        jsonDecode(respond.body) as Map<String, dynamic>);
  }

  /// Current OCR meter state (scans used per window, configured caps, credit
  /// balance) so the client can show limits and a top-up affordance.
  Future<OcrMeterState> ocrUsage(Session session) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/purchases/ocr/usage'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    return OcrMeterState.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  /// Adds OCR scan credits to the tenant (manager/owner only; cashiers get
  /// 403). Returns the new balance.
  Future<int> ocrTopup(Session session, int points) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/purchases/ocr/topup'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'points': points}),
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond), code: _errorCode(respond));
    }
    final data = jsonDecode(respond.body)['data'] as Map<String, dynamic>;
    return (data['credits_remaining'] as num?)?.toInt() ?? 0;
  }

  Future<SyncPullPage> syncPull(
    Session session, {
    int cursor = 0,
    int limit = 100,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/sync/pull?cursor=$cursor&limit=$limit'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode == 409) {
      throw ApiException(_message(respond));
    }
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    final data = jsonDecode(respond.body)['data'] as Map<String, dynamic>;
    final items = (data['items'] as List<dynamic>? ?? [])
        .map((e) => SyncRow.fromJson(e as Map<String, dynamic>))
        .toList();
    return SyncPullPage(
      items: items,
      cursor: _toIntValue(data['cursor']),
      hasMore: data['has_more'] as bool? ?? false,
    );
  }

  Future<SyncPushPage> syncPush(
    Session session,
    List<SyncPushCommand> commands,
  ) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/sync/push'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'commands': commands
              .map((c) => {
                    'command_id': c.commandId,
                    'operation': c.operation,
                    'payload': c.payload,
                  })
              .toList(),
        }),
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    final data = jsonDecode(respond.body)['data'] as Map<String, dynamic>;
    final results = (data['results'] as List<dynamic>? ?? [])
        .map((e) => SyncPushResult.fromJson(e as Map<String, dynamic>))
        .toList();
    return SyncPushPage(results: results);
  }

  /// Fetches the printable receipt payload for a sale as JSON. Clients can
  /// either render their own preview or hit /v1/sales/:id/receipt/print for
  /// the ESC/POS byte stream.
  Future<RefundResult> refundSale(
    Session session,
    String saleId, {
    String reason = '',
    String managerPin = '',
    String? idempotencyKey,
  }) async {
    final key = idempotencyKey ?? const Uuid().v4();
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/sales/$saleId/refund'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
          'Idempotency-Key': key,
        },
        body: jsonEncode({
          if (reason.trim().isNotEmpty) 'reason': reason.trim(),
          if (managerPin.trim().isNotEmpty) 'manager_pin': managerPin.trim(),
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as Map<String, dynamic>;
    return RefundResult.fromJson(data);
  }

  Future<SaleReceipt> fetchReceipt(Session session, String saleId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/sales/$saleId/receipt'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    final data = jsonDecode(respond.body)['data'] as Map<String, dynamic>;
    return SaleReceipt.fromJson(data);
  }

  /// Fetches the ESC/POS byte stream for a sale's receipt. [cols] selects the
  /// backend layout width (24 for 58mm, 32 for 80mm); [cut] toggles the
  /// trailing paper-cut command (multi-copy jobs disable it and cut once);
  /// [compact] requests the denser narrow-paper layout.
  Future<Uint8List> fetchReceiptPrint(
    Session session,
    String saleId, {
    int cols = 32,
    bool cut = true,
    bool compact = false,
  }) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse(
            '$baseUrl/v1/sales/$saleId/receipt/print?cols=$cols&cut=${cut ? 1 : 0}&compact=${compact ? 1 : 0}'),
        headers: {
          'Accept': 'application/vnd.escpos',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    return respond.bodyBytes;
  }

  Future<SaleDetail> saleDetail(Session session, String saleId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/sales/$saleId'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    final data = jsonDecode(respond.body)['data'] as Map<String, dynamic>;
    return SaleDetail.fromJson(data);
  }

  /// Cashier-side self-orders awaiting / done on the cashier side.
  Future<SelfOrdersPage> selfOrders(
    Session session, {
    String? status,
    int page = 1,
    int limit = 50,
  }) async {
    final uri = Uri.parse('$baseUrl/v1/self-orders').replace(
      queryParameters: {
        if (status != null) 'status': status,
        'page': '$page',
        'limit': '$limit',
      },
    );
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        uri,
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    return SelfOrdersPage.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<SelfOrderApproval> approveSelfOrder(
      Session session, String orderId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/self-orders/$orderId/approve'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    return SelfOrderApproval.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<void> cancelSelfOrder(Session session, String orderId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/self-orders/$orderId/cancel'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
  }

  Future<ProductRequestsPage> productRequests(
    Session session, {
    String? status,
    int page = 1,
    int limit = 50,
  }) async {
    final uri = Uri.parse('$baseUrl/v1/product-requests').replace(
      queryParameters: {
        if (status != null) 'status': status,
        'page': '$page',
        'limit': '$limit',
      },
    );
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        uri,
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
    return ProductRequestsPage.fromJson(
        jsonDecode(respond.body)['data'] as Map<String, dynamic>);
  }

  Future<void> fulfillProductRequest(Session session, String requestId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/product-requests/$requestId/fulfill'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
  }

  Future<void> closeProductRequest(Session session, String requestId) async {
    final respond = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/product-requests/$requestId/close'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (respond.statusCode != 200) {
      throw ApiException(_message(respond));
    }
  }

  Future<TenantSettings> settings(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/settings'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return TenantSettings.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<TenantSettings> updateSettings(
    Session session,
    TenantSettings settings,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.put(
        Uri.parse('$baseUrl/v1/settings'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'settings': settings.toUpdateMap()}),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return TenantSettings.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  /// The terminal's open register session, or null when the terminal has none.
  Future<RegisterSession?> currentSession(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/registers/current'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode == 404) return null;
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return RegisterSession.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<RegisterSession> openSession(
    Session session, {
    required int openingCashMinor,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/registers/open'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'opening_cash_minor': openingCashMinor}),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    return RegisterSession.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<RegisterSession> closeSession(
    Session session,
    String sessionId, {
    int? closingCashMinor,
    String managerPin = '',
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/registers/$sessionId/close'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (closingCashMinor != null) 'closing_cash_minor': closingCashMinor,
          if (managerPin.trim().isNotEmpty) 'manager_pin': managerPin.trim(),
        }),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return RegisterSession.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<SessionsPage> sessions(
    Session session, {
    int page = 1,
    int limit = 50,
    String? status,
    String? deviceId,
    String? from,
    String? to,
  }) async {
    final uri = Uri.parse('$baseUrl/v1/registers').replace(
      queryParameters: {
        'page': '$page',
        'limit': '$limit',
        if (status != null && status.isNotEmpty) 'status': status,
        if (deviceId != null && deviceId.isNotEmpty) 'device_id': deviceId,
        if (from != null && from.isNotEmpty) 'from': from,
        if (to != null && to.isNotEmpty) 'to': to,
      },
    );
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(uri, headers: {
        'Accept': 'application/json',
        'Authorization': 'Bearer $accessToken',
      }),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final meta = body['meta'] as Map<String, dynamic>;
    return SessionsPage(
      sessions: (body['data'] as List<dynamic>)
          .map((item) => SessionListItem.fromJson(item as Map<String, dynamic>))
          .toList(),
      total: (meta['total'] as num).toInt(),
      page: (meta['page'] as num).toInt(),
      limit: (meta['limit'] as num).toInt(),
    );
  }

  Future<RegisterSession> sessionDetail(
      Session session, String sessionId) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/registers/$sessionId'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    return RegisterSession.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<List<Floor>> floors(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/floors'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as List<dynamic>;
    return data
        .map((item) => Floor.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<List<RestaurantTable>> tables(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/tables'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as List<dynamic>;
    return data
        .map((item) => RestaurantTable.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  Future<void> updateTable(
    Session session,
    String tableId, {
    String? status,
    int? seats,
    String? name,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse('$baseUrl/v1/tables/$tableId'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          if (status != null) 'status': status,
          if (seats != null) 'seats': seats,
          if (name != null) 'name': name,
        }),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
  }

  Future<Floor> createFloor(
    Session session, {
    required String name,
    int sortOrder = 0,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/floors'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'name': name,
          if (sortOrder != 0) 'sort_order': sortOrder,
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    return Floor.fromJson(body['data'] as Map<String, dynamic>);
  }

  Future<RestaurantTable> createTable(
    Session session, {
    required String floorId,
    required String name,
    int seats = 0,
    String status = 'free',
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/tables'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'floor_id': floorId,
          'name': name,
          if (seats != 0) 'seats': seats,
          if (status != 'free') 'status': status,
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    return RestaurantTable.fromJson(body['data'] as Map<String, dynamic>);
  }

  Future<SplitBillResult> splitSale(
    Session session,
    String saleId,
    List<List<SplitBillLine>> children,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/sales/$saleId/split'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
          'Idempotency-Key': 'split:${const Uuid().v4()}',
        },
        body: jsonEncode({
          'children': children
              .map((child) => {
                    'lines': child.map((line) => line.toJson()).toList(),
                  })
              .toList(),
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    return SplitBillResult.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<http.Response> _authenticatedRequest(
    Session session,
    Future<http.Response> Function(String accessToken) request,
  ) async {
    final response = await _guardTransport(() => request(session.accessToken));
    if (response.statusCode != 401) return response;

    final refreshed = await _refresh(session);
    final retry = await _guardTransport(() => request(refreshed.accessToken));
    return retry;
  }

  /// Turns a dead link into a [NetworkException] so callers can tell "the
  /// network cut" from "the server said no".
  ///
  /// Without this the raw [SocketException] escapes and every caller has to
  /// re-derive the distinction. It matters most at checkout, where only a
  /// transport failure may hand the sale to the offline queue: a server
  /// rejection has to surface, and a bug must never be queued.
  Future<http.Response> _guardTransport(
    Future<http.Response> Function() send,
  ) async {
    try {
      return await send();
    } on NetworkException {
      rethrow;
    } catch (error) {
      if (isTransportFailure(error)) {
        throw NetworkException('network unreachable', original: error);
      }
      rethrow;
    }
  }

  Future<Session> _refresh(Session session) async {
    final existing = _refreshInFlight;
    if (existing != null) return existing;

    final future = _doRefresh(_refreshBase(session));
    _refreshInFlight = future;
    try {
      final refreshed = await future;
      _latestSession = refreshed;
      await onSessionRefreshed?.call(refreshed);
      return refreshed;
    } finally {
      if (identical(_refreshInFlight, future)) {
        _refreshInFlight = null;
      }
    }
  }

  /// Base for a rotation: prefer the newest session known to the client so a
  /// caller holding a pre-rotation snapshot can't replay a consumed token.
  Session _refreshBase(Session requested) => _latestSession ?? requested;

  Future<Session> _doRefresh(Session base) async {
    final response = await _client.post(
      Uri.parse('$baseUrl/v1/auth/refresh'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'refresh_token': base.refreshToken}),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as Map<String, dynamic>;
    return base.copyWith(
      accessToken: data['access_token'] as String,
      refreshToken: data['refresh_token'] as String,
    );
  }

  String _message(http.Response response) {
    try {
      final body = jsonDecode(response.body) as Map<String, dynamic>;
      return body['error']?['message'] as String? ?? 'Request failed';
    } catch (_) {
      return 'Request failed (${response.statusCode})';
    }
  }

  String? _errorCode(http.Response response) {
    try {
      final body = jsonDecode(response.body) as Map<String, dynamic>;
      return body['error']?['code'] as String?;
    } catch (_) {
      return null;
    }
  }

  Future<OnboardingState> onboardingState(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/onboarding'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    return OnboardingState.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<OnboardingReadiness> onboardingReadiness(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/onboarding/readiness'),
        headers: {
          'Accept': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    return OnboardingReadiness.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<TourActionResult> startTour(
    Session session,
    String tourId, {
    int? version,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/onboarding/tours/$tourId/start'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({if (version != null) 'tour_version': version}),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    return TourActionResult.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<TourActionResult> setTourStep(
    Session session,
    String tourId,
    String currentStep,
  ) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.patch(
        Uri.parse('$baseUrl/v1/onboarding/tours/$tourId'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({'current_step': currentStep}),
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    return TourActionResult.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<TourActionResult> pauseTour(Session session, String tourId) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/onboarding/tours/$tourId/pause'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    return TourActionResult.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<TourActionResult> resumeTour(Session session, String tourId) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/onboarding/tours/$tourId/resume'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    return TourActionResult.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<TourActionResult> skipTour(Session session, String tourId) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/onboarding/tours/$tourId/skip'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    return TourActionResult.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  /// Completing a tour is a server-side business-goal verification: the API
  /// answers 409 `goal_not_met` (surfaced as [ApiException.isGoalNotMet])
  /// until the real business fact exists, so the tour never lies.
  Future<TourActionResult> completeTour(Session session, String tourId) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/onboarding/tours/$tourId/complete'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
      ),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    return TourActionResult.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }

  Future<OnboardingEventResult> reportOnboardingEvent(
    Session session, {
    required String eventId,
    required String event,
    Map<String, dynamic>? payload,
  }) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.post(
        Uri.parse('$baseUrl/v1/onboarding/events'),
        headers: {
          'Accept': 'application/json',
          'Content-Type': 'application/json',
          'Authorization': 'Bearer $accessToken',
        },
        body: jsonEncode({
          'event_id': eventId,
          'event': event,
          if (payload != null && payload.isNotEmpty) 'payload': payload,
        }),
      ),
    );
    if (response.statusCode != 200 && response.statusCode != 201) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    return OnboardingEventResult.fromJson(
        jsonDecode(response.body)['data'] as Map<String, dynamic>);
  }
}

class Product {
  const Product({
    required this.id,
    required this.name,
    required this.sku,
    required this.barcode,
    required this.priceMinor,
    required this.currency,
    required this.stockQuantity,
    this.categoryId = '',
    this.costMinor = 0,
    this.imageUrl = '',
    this.description = '',
    this.nameAr = '',
    this.descriptionAr = '',
    this.unit = 'piece',
    this.isActive = true,
    this.selforderEnabled = false,
    this.createdAt,
  });

  factory Product.fromJson(Map<String, dynamic> json) => Product(
        id: json['id'] as String,
        name: json['name'] as String,
        sku: json['sku'] as String,
        barcode: json['barcode'] as String? ?? '',
        priceMinor: (json['price_minor'] as num).toInt(),
        currency: json['currency'] as String,
        stockQuantity: (json['stock_quantity'] as num).toInt(),
        categoryId: json['category_id'] as String? ?? '',
        costMinor: (json['cost_minor'] as num?)?.toInt() ?? 0,
        imageUrl: json['image_url'] as String? ?? '',
        description: json['description'] as String? ?? '',
        nameAr: json['name_ar'] as String? ?? '',
        descriptionAr: json['description_ar'] as String? ?? '',
        unit: json['unit'] as String? ?? 'piece',
        isActive: json['is_active'] as bool? ?? true,
        selforderEnabled: json['selforder_enabled'] as bool? ?? false,
        createdAt: json['created_at'] == null
            ? null
            : DateTime.tryParse(json['created_at'] as String),
      );

  final String id;
  final String name;
  final String sku;
  final String barcode;
  final int priceMinor;
  final String currency;
  final int stockQuantity;
  final String categoryId;
  final int costMinor;
  final String imageUrl;
  final String description;
  final String nameAr;
  final String descriptionAr;
  final String unit;
  final bool isActive;
  final bool selforderEnabled;
  final DateTime? createdAt;

  /// Arabic product name when the device language is Arabic and the store
  /// catalog provides it; otherwise the primary (English) name.
  String displayName(Locale locale) =>
      locale.languageCode == 'ar' && nameAr.isNotEmpty ? nameAr : name;

  /// True when the product was published to the self-order menu (visible on
  /// the customer QR page while in stock).
  bool get publishedForSelfOrder => selforderEnabled && stockQuantity > 0;

  /// True when the product was created within [within] of [now]. Products
  /// without a created_at (older backend, cached rows) are never "new".
  bool isRecentlyAdded(DateTime now,
      {Duration within = const Duration(days: 7)}) {
    final created = createdAt;
    if (created == null) return false;
    final diff = now.difference(created);
    return !diff.isNegative && diff <= within;
  }
}

class SaleItemInput {
  const SaleItemInput({required this.productId, required this.quantity});

  final String productId;
  final int quantity;
}

class SaleResult {
  const SaleResult({
    required this.id,
    required this.subtotalMinor,
    required this.totalMinor,
    required this.currency,
    required this.paymentMethod,
    this.discountCapped = false,
    this.discountWarning = '',
  });

  factory SaleResult.fromJson(Map<String, dynamic> json) => SaleResult(
        id: json['id'] as String,
        subtotalMinor: (json['subtotal_minor'] as num).toInt(),
        totalMinor: (json['total_minor'] as num).toInt(),
        currency: json['currency'] as String,
        paymentMethod:
            PaymentMethod.fromWire(json['payment_method'] as String?),
        discountCapped: json['discount_capped'] as bool? ?? false,
        discountWarning: json['discount_warning'] as String? ?? '',
      );

  final String id;
  final int subtotalMinor;
  final int totalMinor;
  final String currency;
  final PaymentMethod paymentMethod;
  final bool discountCapped;
  final String discountWarning;
}

class RefundResult {
  const RefundResult({
    required this.id,
    required this.saleId,
    required this.refundMinor,
    required this.status,
    this.reason = '',
    this.createdBy = '',
    this.createdAt = '',
  });

  factory RefundResult.fromJson(Map<String, dynamic> json) => RefundResult(
        id: json['id'] as String,
        saleId: json['sale_id'] as String,
        refundMinor: (json['refund_minor'] as num).toInt(),
        status: json['status'] as String? ?? 'completed',
        reason: json['reason'] as String? ?? '',
        createdBy: json['created_by'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String saleId;
  final int refundMinor;
  final String status;
  final String reason;
  final String createdBy;
  final String createdAt;
}

class ApiException implements Exception {
  const ApiException(this.message, {this.code});
  final String message;
  final String? code;

  bool get isTrialExpired => code == 'trial_expired';

  /// The account is valid but belongs to the other public surface: platform
  /// operators sign in on the console domain, store staff on the POS domain.
  bool get isWrongSurface => code == 'wrong_surface';

  /// 402 because the tenant's plan does not grant the feature. The body names
  /// the missing entitlements, so the UI can say "not in your plan" in the
  /// user's language rather than pasting the server's English sentence.
  bool get isPlanLimit => code == 'plan_limit_exceeded';

  /// 402 from the OCR meter: every active window is spent and the credit
  /// balance is empty. Actionable — the shop can top up and continue.
  bool get isScanLimit => code == 'ocr_window_limit_reached';

  /// Onboarding tour completion was refused because the business goal it
  /// represents is not actually satisfied server-side.
  bool get isGoalNotMet => code == 'goal_not_met';

  bool get isTourCompleted => code == 'tour_completed';
  bool get isTourVersionMismatch => code == 'tour_version_mismatch';
  bool get isTourNotRunning => code == 'tour_not_running';

  @override
  String toString() => message;
}

class Category {
  const Category({
    required this.id,
    required this.name,
    required this.slug,
    this.nameAr = '',
  });

  factory Category.fromJson(Map<String, dynamic> json) => Category(
        id: json['id'] as String,
        name: json['name'] as String,
        slug: json['slug'] as String,
        nameAr: json['name_ar'] as String? ?? '',
      );

  final String id;
  final String name;
  final String slug;
  final String nameAr;

  /// Arabic category name when the device language is Arabic and the store
  /// catalog provides it; otherwise the primary (English) name.
  String displayName(Locale locale) =>
      locale.languageCode == 'ar' && nameAr.isNotEmpty ? nameAr : name;
}

class SaleDetail {
  const SaleDetail({
    required this.id,
    required this.status,
    required this.currency,
    required this.items,
  });

  factory SaleDetail.fromJson(Map<String, dynamic> json) => SaleDetail(
        id: json['id'] as String,
        status: json['status'] as String? ?? 'completed',
        currency: json['currency'] as String? ?? 'EGP',
        items: (json['items'] as List<dynamic>? ?? [])
            .map(
                (item) => SaleDetailItem.fromJson(item as Map<String, dynamic>))
            .toList(),
      );

  final String id;
  final String status;
  final String currency;
  final List<SaleDetailItem> items;
}

class SaleDetailItem {
  const SaleDetailItem({
    required this.id,
    required this.productName,
    required this.quantity,
    required this.unitPriceMinor,
    required this.totalMinor,
  });

  factory SaleDetailItem.fromJson(Map<String, dynamic> json) => SaleDetailItem(
        id: json['id'] as String? ?? '',
        productName: json['product_name'] as String? ?? '',
        quantity: (json['quantity'] as num?)?.toInt() ?? 0,
        unitPriceMinor: (json['unit_price_minor'] as num?)?.toInt() ?? 0,
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
      );

  final String id;
  final String productName;
  final int quantity;
  final int unitPriceMinor;
  final int totalMinor;
}

class SaleSummary {
  const SaleSummary({
    required this.id,
    required this.status,
    required this.subtotalMinor,
    required this.totalMinor,
    required this.currency,
    required this.createdAt,
    required this.paymentMethod,
  });

  factory SaleSummary.fromJson(Map<String, dynamic> json) => SaleSummary(
        id: json['id'] as String,
        status: json['status'] as String? ?? 'completed',
        subtotalMinor: (json['subtotal_minor'] as num).toInt(),
        totalMinor: (json['total_minor'] as num).toInt(),
        currency: json['currency'] as String,
        createdAt: json['created_at'] as String? ?? '',
        paymentMethod:
            PaymentMethod.fromWire(json['payment_method'] as String?),
      );

  final String id;
  final String status;
  final int subtotalMinor;
  final int totalMinor;
  final String currency;
  final String createdAt;
  final PaymentMethod paymentMethod;
}

class SalesPage {
  const SalesPage({
    required this.sales,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<SaleSummary> sales;
  final int total;
  final int page;
  final int limit;
}

class Country {
  const Country({
    required this.code,
    required this.nameEn,
    required this.nameAr,
    required this.currencyCode,
    required this.phoneCode,
  });

  factory Country.fromJson(Map<String, dynamic> json) => Country(
        code: json['code'] as String,
        nameEn: json['name_en'] as String,
        nameAr: json['name_ar'] as String? ?? json['name_en'] as String,
        currencyCode: json['currency_code'] as String,
        phoneCode: json['phone_code'] as String? ?? '',
      );

  final String code;
  final String nameEn;
  final String nameAr;
  final String currencyCode;
  final String phoneCode;
}

class Currency {
  const Currency({
    required this.code,
    required this.nameEn,
    required this.nameAr,
    required this.symbol,
    required this.digitsAfterDecimal,
  });

  factory Currency.fromJson(Map<String, dynamic> json) => Currency(
        code: json['code'] as String,
        nameEn: json['name_en'] as String,
        nameAr: json['name_ar'] as String? ?? json['name_en'] as String,
        symbol: json['symbol'] as String? ?? json['code'] as String,
        digitsAfterDecimal:
            (json['digits_after_decimal'] as num?)?.toInt() ?? 2,
      );

  final String code;
  final String nameEn;
  final String nameAr;
  final String symbol;
  final int digitsAfterDecimal;
}

class CustomersPage {
  const CustomersPage({
    required this.customers,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<Customer> customers;
  final int total;
  final int page;
  final int limit;
}

class SyncPullPage {
  const SyncPullPage({
    required this.items,
    required this.cursor,
    required this.hasMore,
  });

  final List<SyncRow> items;
  final int cursor;
  final bool hasMore;
}

class SyncRow {
  const SyncRow({
    required this.entity,
    required this.id,
    required this.changeSeq,
    required this.changeType,
    required this.data,
  });

  factory SyncRow.fromJson(Map<String, dynamic> json) => SyncRow(
        entity: json['entity'] as String,
        id: json['id'] as String,
        changeSeq: (json['change_seq'] as num).toInt(),
        changeType: json['change_type'] as String? ?? 'upsert',
        data: (json['data'] as Map<String, dynamic>?) ?? const {},
      );

  final String entity;
  final String id;
  final int changeSeq;
  final String changeType;
  final Map<String, dynamic> data;
}

class SyncPushCommand {
  const SyncPushCommand({
    required this.commandId,
    required this.operation,
    required this.payload,
  });

  final String commandId;
  final String operation;
  final Map<String, dynamic> payload;
}

class SyncPushPage {
  const SyncPushPage({required this.results});

  final List<SyncPushResult> results;
}

class SyncPushResult {
  const SyncPushResult({
    required this.commandId,
    required this.status,
    this.replayed = false,
    this.errorCode,
    this.errorDetail,
    this.result = const {},
  });

  factory SyncPushResult.fromJson(Map<String, dynamic> json) => SyncPushResult(
        commandId: json['command_id'] as String? ?? '',
        status: json['status'] as String? ?? '',
        replayed: json['replayed'] as bool? ?? false,
        errorCode: json['error_code'] as String?,
        errorDetail: json['error_detail'] as String?,
        result: (json['result'] as Map<String, dynamic>?) ?? const {},
      );

  final String commandId;
  final String status;
  final bool replayed;
  final String? errorCode;
  final String? errorDetail;
  final Map<String, dynamic> result;

  bool get applied => status == 'applied' || status == 'replayed';
  bool get conflicted => status == 'conflict';
  bool get rejected => status == 'rejected';
}

class TenantSettings {
  const TenantSettings({
    this.defaultPaymentMethod = PaymentMethod.cash,
    this.showStockBadges = true,
    this.receiptFooter = '',
    this.allowNegativeStock = false,
    this.discountMode = DiscountMode.cap,
    this.maxDiscountPct = 0,
    this.managerDiscountOverride = true,
    this.managerClosePin = true,
    this.stockType = 'on_hand',
    this.blockOutOfStock = true,
    this.lowStockThreshold = 5,
    this.lowStockWarning = true,
    this.validateStockPayment = true,
    this.refreshButton = true,
    this.reprintLastReceipt = true,
    this.roundingMode = 'off',
  });

  factory TenantSettings.fromJson(Map<String, dynamic> json) => TenantSettings(
        defaultPaymentMethod: PaymentMethod.fromWire(
            json['pos.default_payment_method'] as String?),
        showStockBadges: (json['pos.show_stock_badges'] as String?) != 'false',
        receiptFooter: json['pos.receipt_footer'] as String? ?? '',
        allowNegativeStock:
            (json['inventory.allow_negative_stock'] as String?) == 'true',
        discountMode:
            DiscountMode.fromSetting(json['pos.discount_mode'] as String?),
        maxDiscountPct: _intSetting(json['pos.max_discount_pct']) ?? 0,
        managerDiscountOverride:
            (json['pos.manager.discount'] as String?) != 'false',
        managerClosePin: (json['pos.manager.close'] as String?) != 'false',
        stockType: json['pos.stock_type'] as String? ?? 'on_hand',
        blockOutOfStock: (json['pos.block_out_of_stock'] as String?) != 'false',
        lowStockThreshold: _intSetting(json['pos.low_stock_threshold']) ?? 5,
        lowStockWarning: (json['pos.low_stock_warning'] as String?) != 'false',
        validateStockPayment:
            (json['pos.validate_stock_payment'] as String?) != 'false',
        refreshButton: (json['pos.refresh_button'] as String?) != 'false',
        reprintLastReceipt:
            (json['pos.reprint_last_receipt'] as String?) != 'false',
        roundingMode: switch (
            roundingDenominator(json['pos.rounding_mode'] as String?)) {
          null => 'off',
          _ => json['pos.rounding_mode'] as String? ?? 'off',
        },
      );

  final PaymentMethod defaultPaymentMethod;
  final bool showStockBadges;
  final String receiptFooter;
  final bool allowNegativeStock;
  final DiscountMode discountMode;
  final int maxDiscountPct;
  final bool managerDiscountOverride;

  /// When true the backend requires the manager PIN to close a register
  /// session opened by a cashier-level user.
  final bool managerClosePin;
  final String stockType;
  final bool blockOutOfStock;
  final int lowStockThreshold;
  final bool lowStockWarning;
  final bool validateStockPayment;
  final bool refreshButton;

  /// `pos.reprint_last_receipt` — show the one-tap reprint action on the POS
  /// app bar. Follows the same `!= 'false'` default-on convention as the
  /// refresh button.
  final bool reprintLastReceipt;

  /// `pos.rounding_mode` — "off", "25", "50" or "100" (minor-unit cash
  /// denomination the payable is rounded to). The pure
  /// [roundingDenominator]/[roundingDelta] helpers expose the payable math.
  final String roundingMode;

  /// Minor-unit cash rounding denomination; 0 means rounding is off.
  int get roundingDenom => roundingDenominator(roundingMode) ?? 0;

  bool get stockBadgeOnHand => stockType != 'available';

  TenantSettings copyWith({
    PaymentMethod? defaultPaymentMethod,
    bool? showStockBadges,
    String? receiptFooter,
    bool? allowNegativeStock,
  }) =>
      TenantSettings(
        defaultPaymentMethod: defaultPaymentMethod ?? this.defaultPaymentMethod,
        showStockBadges: showStockBadges ?? this.showStockBadges,
        receiptFooter: receiptFooter ?? this.receiptFooter,
        allowNegativeStock: allowNegativeStock ?? this.allowNegativeStock,
        discountMode: discountMode,
        maxDiscountPct: maxDiscountPct,
        managerDiscountOverride: managerDiscountOverride,
        managerClosePin: managerClosePin,
        stockType: stockType,
        blockOutOfStock: blockOutOfStock,
        lowStockThreshold: lowStockThreshold,
        lowStockWarning: lowStockWarning,
        validateStockPayment: validateStockPayment,
        refreshButton: refreshButton,
        reprintLastReceipt: reprintLastReceipt,
        roundingMode: roundingMode,
      );

  Map<String, String> toUpdateMap() => {
        'pos.default_payment_method': defaultPaymentMethod.wireName,
        'pos.show_stock_badges': '$showStockBadges',
        'pos.receipt_footer': receiptFooter,
        'inventory.allow_negative_stock': '$allowNegativeStock',
        'pos.rounding_mode': roundingMode,
      };
}

int? _intSetting(dynamic v) {
  if (v is num) return v.toInt();
  return int.tryParse(v as String? ?? '');
}

int _toIntValue(dynamic v) {
  if (v is int) return v;
  if (v is String) return int.tryParse(v) ?? 0;
  return 0;
}
class ProductLot {
  const ProductLot({
    required this.id,
    required this.productId,
    required this.lotNumber,
    this.expiryDate,
    this.serialNumber,
    required this.quantityRemaining,
  });

  factory ProductLot.fromJson(Map<String, dynamic> json) => ProductLot(
        id: json['id'] as String,
        productId: json['product_id'] as String,
        lotNumber: json['lot_number'] as String,
        expiryDate: json['expiry_date'] as String?,
        serialNumber: json['serial_number'] as String?,
        quantityRemaining: (json['quantity_remaining'] as num).toInt(),
      );

  final String id;
  final String productId;
  final String lotNumber;
  final String? expiryDate;
  final String? serialNumber;
  final int quantityRemaining;
}

class ProductVariant {
  const ProductVariant({
    required this.id,
    required this.productId,
    required this.name,
    required this.sku,
    required this.barcode,
    required this.priceMinor,
    required this.stockQuantity,
    required this.isActive,
  });

  factory ProductVariant.fromJson(Map<String, dynamic> json) => ProductVariant(
        id: json['id'] as String,
        productId: json['product_id'] as String,
        name: json['name'] as String,
        sku: json['sku'] as String,
        barcode: json['barcode'] as String? ?? '',
        priceMinor: (json['price_minor'] as num).toInt(),
        stockQuantity: (json['stock_quantity'] as num).toInt(),
        isActive: json['is_active'] as bool? ?? true,
      );

  final String id;
  final String productId;
  final String name;
  final String sku;
  final String barcode;
  final int priceMinor;
  final int stockQuantity;
  final bool isActive;
}

