import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/session_store.dart';

void main() {
  test('session parses the Go API envelope', () {
    final session = Session.fromJson({
      'access_token': 'access',
      'refresh_token': 'refresh',
      'user': {'id': 'user-1', 'display_name': 'Cashier'},
      'tenant': {'id': 'tenant-1'},
    });

    expect(session.accessToken, 'access');
    expect(session.displayName, 'Cashier');
    expect(session.tenantId, 'tenant-1');
  });

  test('products parses the Go catalog response with authorization', () async {
    final client = ApiClient(client: _FakeClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final products = await client.products(session, search: 'coffee');

    expect(products, hasLength(1));
    expect(products.single.name, 'Espresso');
    expect(products.single.priceMinor, 280);
  });

  test('createSale sends product quantities and idempotency key', () async {
    final client = ApiClient(client: _SaleClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final sale = await client.createSale(
      session,
      const [SaleItemInput(productId: 'product-1', quantity: 2)],
      idempotencyKey: 'checkout-1',
    );

    expect(sale.id, 'sale-1');
    expect(sale.totalMinor, 560);
  });

  test('refresh rotates both tokens', () async {
    final client = ApiClient(client: _DirectRefreshClient());
    const session = Session(
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final refreshed = await client.refresh(session);

    expect(refreshed.accessToken, 'new-access');
    expect(refreshed.refreshToken, 'new-refresh');
    expect(refreshed.userId, 'user-1');
  });

  test('refreshes once on unauthorized responses and retries with new token',
      () async {
    Session? refreshedSession;
    final client = ApiClient(
      client: _RefreshClient(),
      onSessionRefreshed: (session) async => refreshedSession = session,
    );
    const session = Session(
      accessToken: 'expired-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final products = await client.products(session);

    expect(products.single.id, 'product-1');
    expect(refreshedSession?.accessToken, 'new-access-token');
    expect(refreshedSession?.refreshToken, 'new-refresh-token');
  });

  test('createProduct sends catalog fields', () async {
    final client = ApiClient(client: _CreateProductClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Manager',
      tenantId: 'tenant-1',
    );

    final product = await client.createProduct(
      session,
      name: 'Espresso',
      sku: 'ESP-1',
      priceMinor: 280,
      currency: 'USD',
      stockQuantity: 4,
    );

    expect(product.name, 'Espresso');
    expect(product.stockQuantity, 4);
  });
}

class _FakeClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.url.queryParameters['search'], 'coffee');
    const body =
        '{"data":[{"id":"product-1","name":"Espresso","sku":"ESP-1","barcode":"0123","price_minor":280,"currency":"USD","stock_quantity":4}],"meta":{"request_id":"test"}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _SaleClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.headers['Idempotency-Key'], 'checkout-1');
    expect(request.method, 'POST');
    const body =
        '{"data":{"id":"sale-1","subtotal_minor":560,"total_minor":560,"currency":"USD"},"meta":{"request_id":"test"}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _DirectRefreshClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.headers['content-type'], 'application/json');
    const body =
        '{"data":{"access_token":"new-access","refresh_token":"new-refresh","expires_in":900}}';
    return http.StreamedResponse(Stream.value(body.codeUnits), 200);
  }
}

class _RefreshClient extends http.BaseClient {
  var productAttempts = 0;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    if (request.url.path.endsWith('/auth/refresh')) {
      const body =
          '{"data":{"access_token":"new-access-token","refresh_token":"new-refresh-token"}}';
      return http.StreamedResponse(
        Stream.value(body.codeUnits),
        200,
        headers: {'content-type': 'application/json'},
      );
    }
    productAttempts++;
    if (productAttempts == 1) {
      return http.StreamedResponse(Stream.value('{}'.codeUnits), 401);
    }
    expect(request.headers['Authorization'], 'Bearer new-access-token');
    const body =
        '{"data":[{"id":"product-1","name":"Espresso","sku":"ESP-1","barcode":"0123","price_minor":280,"currency":"USD","stock_quantity":4}]}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _CreateProductClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/products');
    expect(request.headers['Authorization'], 'Bearer access-token');
    const body =
        '{"data":{"id":"product-1","name":"Espresso","sku":"ESP-1","barcode":"","price_minor":280,"cost_minor":0,"currency":"USD","stock_quantity":4,"is_active":true}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}
