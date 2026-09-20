import 'dart:convert';
import 'dart:typed_data';
import 'dart:ui' show Locale;

import 'package:http/http.dart' as http;
import 'package:uuid/uuid.dart';

import 'customers.dart';
import 'billing.dart';
import 'dashboard.dart';
import 'inventory.dart';
import 'payments.dart';
import 'purchases.dart';
import 'receipts.dart';
import 'restaurants.dart';
import 'registers.dart';
import 'saas.dart';
import 'security.dart';
import 'selforder.dart';
import 'session_store.dart';

/// SaaS control-plane models live in saas.dart (exported for convenience).
export 'saas.dart' show SaasSummary, SaasTenant, SaasTenantsPage;

class ApiClient {
  ApiClient({http.Client? client, this.onSessionRefreshed})
      : _client = client ?? http.Client();

  final http.Client _client;
  final Future<void> Function(Session session)? onSessionRefreshed;
  final String baseUrl = const String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://127.0.0.1:8080',
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
        Uri.parse('$baseUrl/v1/saas/summary'),
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
    final uri = Uri.parse('$baseUrl/v1/saas/tenants').replace(
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
        Uri.parse('$baseUrl/v1/saas/tenants/$tenantId/analytics'),
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
        Uri.parse('$baseUrl/v1/saas/billing/summary'),
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
        Uri.parse('$baseUrl/v1/saas/plans'),
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
        Uri.parse('$baseUrl/v1/saas/payment-providers'),
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
    final uri = Uri.parse('$baseUrl/v1/saas/subscriptions').replace(
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
        Uri.parse('$baseUrl/v1/saas/tenants/$tenantId/subscription'),
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
        Uri.parse('$baseUrl/v1/saas/tenants/$tenantId/subscription'),
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
        Uri.parse('$baseUrl/v1/saas/subscriptions/$subscriptionId/status'),
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
        Uri.parse('$baseUrl/v1/saas/subscriptions/$subscriptionId/change-plan'),
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
    final uri = Uri.parse('$baseUrl/v1/saas/invoices').replace(
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
        Uri.parse('$baseUrl/v1/saas/tenants/$tenantId/invoices'),
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
        Uri.parse('$baseUrl/v1/saas/invoices/$invoiceId/pay'),
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
        Uri.parse('$baseUrl/v1/saas/invoices/$invoiceId/void'),
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
        Uri.parse('$baseUrl/v1/saas/invoices/$invoiceId/refund'),
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
    final uri = Uri.parse('$baseUrl/v1/saas/users').replace(
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
        Uri.parse('$baseUrl/v1/saas/tenants/$tenantId/users'),
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
            '$baseUrl/v1/platform/trial/entitlements?limit=$limit&offset=$offset'),
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
            '$baseUrl/v1/platform/trial/entitlements/$entitlementId/$action'),
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
        Uri.parse('$baseUrl/v1/platform/audit?limit=$limit&offset=$offset'),
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
        Uri.parse('$baseUrl/v1/platform/trial/settings'),
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
        Uri.parse('$baseUrl/v1/platform/trial/settings'),
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
        Uri.parse('$baseUrl/v1/platform/tenants/$tenantId/suspend'),
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
        Uri.parse('$baseUrl/v1/platform/tenants/$tenantId/activate'),
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
      limit: _toIntValue(meta['limit']) == 0 ? limit : _toIntValue(meta['limit']),
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
    return PurchasesPage.fromJson(jsonDecode(respond.body) as Map<String, dynamic>);
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

  Future<void> fulfillProductRequest(
      Session session, String requestId) async {
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
          if (closingCashMinor != null)
            'closing_cash_minor': closingCashMinor,
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
  }) async {
    final uri = Uri.parse('$baseUrl/v1/registers').replace(
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

  Future<RegisterSession> sessionDetail(Session session, String sessionId) async {
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
    final response = await request(session.accessToken);
    if (response.statusCode != 401) return response;

    final refreshed = await _refresh(session);
    final retry = await request(refreshed.accessToken);
    return retry;
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
  bool isRecentlyAdded(DateTime now, {Duration within = const Duration(days: 7)}) {
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
        paymentMethod: PaymentMethod.fromWire(json['payment_method'] as String?),
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
            .map((item) => SaleDetailItem.fromJson(item as Map<String, dynamic>))
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
        digitsAfterDecimal: (json['digits_after_decimal'] as num?)?.toInt() ?? 2,
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

  factory SyncPushResult.fromJson(Map<String, dynamic> json) =>
      SyncPushResult(
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
  });

  factory TenantSettings.fromJson(Map<String, dynamic> json) => TenantSettings(
        defaultPaymentMethod: PaymentMethod.fromWire(
            json['pos.default_payment_method'] as String?),
        showStockBadges:
            (json['pos.show_stock_badges'] as String?) != 'false',
        receiptFooter: json['pos.receipt_footer'] as String? ?? '',
        allowNegativeStock:
            (json['inventory.allow_negative_stock'] as String?) == 'true',
        discountMode: DiscountMode.fromSetting(
            json['pos.discount_mode'] as String?),
        maxDiscountPct: _intSetting(json['pos.max_discount_pct']) ?? 0,
        managerDiscountOverride:
            (json['pos.manager.discount'] as String?) != 'false',
        managerClosePin: (json['pos.manager.close'] as String?) != 'false',
        stockType: json['pos.stock_type'] as String? ?? 'on_hand',
        blockOutOfStock:
            (json['pos.block_out_of_stock'] as String?) != 'false',
        lowStockThreshold:
            _intSetting(json['pos.low_stock_threshold']) ?? 5,
        lowStockWarning:
            (json['pos.low_stock_warning'] as String?) != 'false',
        validateStockPayment:
            (json['pos.validate_stock_payment'] as String?) != 'false',
        refreshButton: (json['pos.refresh_button'] as String?) != 'false',
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

  bool get stockBadgeOnHand => stockType != 'available';

  TenantSettings copyWith({
    PaymentMethod? defaultPaymentMethod,
    bool? showStockBadges,
    String? receiptFooter,
    bool? allowNegativeStock,
  }) =>
      TenantSettings(
        defaultPaymentMethod:
            defaultPaymentMethod ?? this.defaultPaymentMethod,
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
      );

  Map<String, String> toUpdateMap() => {
        'pos.default_payment_method': defaultPaymentMethod.wireName,
        'pos.show_stock_badges': '$showStockBadges',
        'pos.receipt_footer': receiptFooter,
        'inventory.allow_negative_stock': '$allowNegativeStock',
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
