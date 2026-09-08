import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:uuid/uuid.dart';

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
      throw ApiException(_message(response));
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
        }),
      ),
    );
    if (response.statusCode != 201) {
      throw ApiException(_message(response));
    }
    final data = jsonDecode(response.body)['data'] as Map<String, dynamic>;
    return SaleResult.fromJson(data);
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
  });

  factory SaleResult.fromJson(Map<String, dynamic> json) => SaleResult(
        id: json['id'] as String,
        subtotalMinor: (json['subtotal_minor'] as num).toInt(),
        totalMinor: (json['total_minor'] as num).toInt(),
        currency: json['currency'] as String,
      );

  final String id;
  final int subtotalMinor;
  final int totalMinor;
  final String currency;
}

class ApiException implements Exception {
  const ApiException(this.message);
  final String message;

  @override
  String toString() => message;
}
