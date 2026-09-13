import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:uuid/uuid.dart';

import 'customers.dart';
import 'dashboard.dart';
import 'inventory.dart';
import 'payments.dart';
import 'receipts.dart';
import 'restaurants.dart';
import 'registers.dart';
import 'session_store.dart';

class ApiClient {
  ApiClient({http.Client? client, this.onSessionRefreshed})
      : _client = client ?? http.Client();

  final http.Client _client;
  final Future<void> Function(Session session)? onSessionRefreshed;
  final String baseUrl = const String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://127.0.0.1:8080',
  );

  Future<Session> login({
    required String tenantId,
    required String email,
    required String password,
    required String deviceId,
    required String deviceName,
  }) async {
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
    return Session.fromJson(data);
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
      }),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response), code: _errorCode(response));
    }
    final data = jsonDecode(response.body)['data'] as Map<String, dynamic>;
    return Session.fromJson(data);
  }

  Future<void> logout(Session session) async {
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

  Future<Session> refresh(Session session) async {
    final response = await _client.post(
      Uri.parse('$baseUrl/v1/auth/refresh'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'refresh_token': session.refreshToken}),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as Map<String, dynamic>;
    return session.copyWith(
      accessToken: data['access_token'] as String,
      refreshToken: data['refresh_token'] as String,
    );
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
  }) async {
    final uri = Uri.parse('$baseUrl/v1/saas/tenants').replace(
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
    return SaasTenantsPage(
      tenants: (body['data'] as List<dynamic>)
          .map((item) => SaasTenant.fromJson(item as Map<String, dynamic>))
          .toList(),
      total: (meta['total'] as num?)?.toInt() ?? 0,
      page: (meta['page'] as num?)?.toInt() ?? page,
      limit: (meta['limit'] as num?)?.toInt() ?? limit,
    );
  }

  Future<DashboardSummary> dashboardSummary(Session session) async {
    final response = await _authenticatedRequest(
      session,
      (accessToken) => _client.get(
        Uri.parse('$baseUrl/v1/dashboard/summary'),
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
    final response = await _client.post(
      Uri.parse('$baseUrl/v1/auth/refresh'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'refresh_token': session.refreshToken}),
    );
    if (response.statusCode != 200) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as Map<String, dynamic>;
    final refreshed = session.copyWith(
      accessToken: data['access_token'] as String,
      refreshToken: data['refresh_token'] as String,
    );
    await onSessionRefreshed?.call(refreshed);
    return refreshed;
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
    this.isActive = true,
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
        isActive: json['is_active'] as bool? ?? true,
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
  final bool isActive;
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
  });

  factory SaleResult.fromJson(Map<String, dynamic> json) => SaleResult(
        id: json['id'] as String,
        subtotalMinor: (json['subtotal_minor'] as num).toInt(),
        totalMinor: (json['total_minor'] as num).toInt(),
        currency: json['currency'] as String,
        paymentMethod: PaymentMethod.fromWire(json['payment_method'] as String?),
      );

  final String id;
  final int subtotalMinor;
  final int totalMinor;
  final String currency;
  final PaymentMethod paymentMethod;
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
  });

  factory Category.fromJson(Map<String, dynamic> json) => Category(
        id: json['id'] as String,
        name: json['name'] as String,
        slug: json['slug'] as String,
      );

  final String id;
  final String name;
  final String slug;
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

class SaasBusinessCount {
  const SaasBusinessCount({required this.businessType, required this.tenants});

  factory SaasBusinessCount.fromJson(Map<String, dynamic> json) =>
      SaasBusinessCount(
        businessType: json['business_type'] as String,
        tenants: (json['tenants'] as num).toInt(),
      );

  final String businessType;
  final int tenants;
}

class SaasSummary {
  const SaasSummary({
    required this.totalTenants,
    required this.totalUsers,
    required this.totalSales,
    required this.revenueMinor,
    required this.byBusiness,
    this.supportedCountry = 'EG',
    this.supportedCurrency = 'EGP',
  });

  factory SaasSummary.fromJson(Map<String, dynamic> json) => SaasSummary(
        totalTenants: (json['total_tenants'] as num).toInt(),
        totalUsers: (json['total_users'] as num).toInt(),
        totalSales: (json['total_sales'] as num).toInt(),
        revenueMinor: (json['revenue_minor'] as num).toInt(),
        byBusiness: (json['by_business'] as List<dynamic>? ?? const [])
            .map((item) =>
                SaasBusinessCount.fromJson(item as Map<String, dynamic>))
            .toList(),
        supportedCountry: json['supported_country'] as String? ?? 'EG',
        supportedCurrency: json['supported_currency'] as String? ?? 'EGP',
      );

  final int totalTenants;
  final int totalUsers;
  final int totalSales;
  final int revenueMinor;
  final List<SaasBusinessCount> byBusiness;
  final String supportedCountry;
  final String supportedCurrency;
}

class SaasTenant {
  const SaasTenant({
    required this.id,
    required this.name,
    required this.slug,
    required this.businessType,
    required this.countryCode,
    required this.currencyCode,
    required this.defaultLanguage,
    required this.users,
    required this.products,
  });

  factory SaasTenant.fromJson(Map<String, dynamic> json) => SaasTenant(
        id: json['id'] as String,
        name: json['name'] as String,
        slug: json['slug'] as String,
        businessType: json['business_type'] as String? ?? '',
        countryCode: json['country_code'] as String? ?? '',
        currencyCode: json['currency_code'] as String? ?? '',
        defaultLanguage: json['default_language'] as String? ?? '',
        users: (json['users'] as num?)?.toInt() ?? 0,
        products: (json['products'] as num?)?.toInt() ?? 0,
      );

  final String id;
  final String name;
  final String slug;
  final String businessType;
  final String countryCode;
  final String currencyCode;
  final String defaultLanguage;
  final int users;
  final int products;
}

class SaasTenantsPage {
  const SaasTenantsPage({
    required this.tenants,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<SaasTenant> tenants;
  final int total;
  final int page;
  final int limit;
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
  });

  factory TenantSettings.fromJson(Map<String, dynamic> json) => TenantSettings(
        defaultPaymentMethod: PaymentMethod.fromWire(
            json['pos.default_payment_method'] as String?),
        showStockBadges:
            (json['pos.show_stock_badges'] as String?) != 'false',
        receiptFooter: json['pos.receipt_footer'] as String? ?? '',
        allowNegativeStock:
            (json['inventory.allow_negative_stock'] as String?) == 'true',
      );

  final PaymentMethod defaultPaymentMethod;
  final bool showStockBadges;
  final String receiptFooter;
  final bool allowNegativeStock;

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
      );

  Map<String, String> toUpdateMap() => {
        'pos.default_payment_method': defaultPaymentMethod.wireName,
        'pos.show_stock_badges': '$showStockBadges',
        'pos.receipt_footer': receiptFooter,
        'inventory.allow_negative_stock': '$allowNegativeStock',
      };
}

int _toIntValue(dynamic v) {
  if (v is int) return v;
  if (v is String) return int.tryParse(v) ?? 0;
  return 0;
}
