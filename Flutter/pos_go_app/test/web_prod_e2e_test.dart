@TestOn('browser')
library;

import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/restaurants.dart';
import 'package:pos_go_app/core/session_store.dart';

// Credentials from .secrets/testing/DEMO_CREDENTIALS.txt:
//   --dart-define=DEMO_PASSWORD='<password>'
const demoEmail = 'demo-restaurant-manager@xamltech.com';
const demoPassword = String.fromEnvironment('DEMO_PASSWORD');

/// The `integration_test/network_e2e_test.dart` flows, runnable in a real
/// browser with `flutter test --platform chrome`.
///
/// Why a copy: `flutter test --platform chrome` compiles this file to
/// JavaScript and runs it in headless Chrome, which exercises the same
/// `ApiClient` / `http` / `SessionStore` stack that `dart:io` used to break
/// on web. `flutter drive -d web-server` would additionally drive the UI, but
/// it needs chromedriver and a full app compile.
///
/// Against production it needs a CORS-permitting origin, because the browser
/// enforces CORS on every `https://api.xamltech.com` call:
///
///   flutter test --platform chrome test/web_prod_e2e_test.dart \
///     --dart-define=API_BASE_URL=https://api.xamltech.com
///
/// Run it through a Chrome wrapper that adds `--disable-web-security` to test
/// the client itself, independently of the server's allow-list.
void main() {
  // flutter_secure_storage goes through a platform channel, so the binding must
  // exist even though these tests never pump a widget.
  TestWidgetsFlutterBinding.ensureInitialized();

  // Opt-in: these calls create real sales, floors and tables on the target
  // backend, and a browser enforces CORS, so they must never run by accident in
  // `flutter test` / CI.
  //
  //   CHROME_EXECUTABLE=/path/to/chrome \
  //   flutter test --platform chrome test/web_prod_e2e_test.dart \
  //     --dart-define=WEB_E2E=1 \
  //     --dart-define=API_BASE_URL=https://api.xamltech.com
  //
  // Production's CORS allow-list is an exact match on `Origin`
  // (internal/transport/http/middleware.go), so only a same-origin deployment
  // or exactly `https://localhost` may call it unaided. To exercise the client
  // against production regardless, point CHROME_EXECUTABLE at a wrapper that
  // appends `--disable-web-security`.
  const runWebE2E = bool.fromEnvironment('WEB_E2E');
  if (!runWebE2E) {
    test('web E2E is opt-in', () {}, skip: 'set --dart-define=WEB_E2E=1 to run against a live backend');
    return;
  }

  const apiBaseUrl = String.fromEnvironment(
    'API_BASE_URL',
    defaultValue: 'http://127.0.0.1:8080',
  );

  Future<Map<String, dynamic>> postJson(
    Session session,
    String path,
    Map<String, dynamic> body,
  ) async {
    final response = await http.post(
      Uri.parse('$apiBaseUrl$path'),
      headers: {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ${session.accessToken}',
      },
      body: jsonEncode(body),
    );
    expect(response.statusCode, 201, reason: '$path returned ${response.statusCode}');
    return jsonDecode(response.body)['data'] as Map<String, dynamic>;
  }

  test('web E2E: login, browse, sell, receipt, logout', () async {
    final api = ApiClient();
    final session = await api.login(
      tenantId: 'demo-restaurant',
      email: demoEmail,
      password: demoPassword,
      deviceId: 'web-e2e-device',
      deviceName: 'Web E2E',
    );
    expect(session.accessToken, isNotEmpty);
    expect(session.currencyCode, anyOf('EGP', ''));

    expect(await api.countries(), isNotEmpty);
    expect(await api.currencies(), isNotEmpty);
    expect(await api.categories(session), isNotEmpty);

    final products = await api.products(session);
    expect(products, isNotEmpty);
    final product = products.firstWhere((p) => p.stockQuantity > 0);

    RegisterSession? current;
    try {
      current = await api.currentSession(session);
    } on ApiException {
      current = null;
    }
    final register = current ?? await api.openSession(session, openingCashMinor: 0);
    expect(register.id, isNotEmpty);

    final sale = await api.createSale(
      session,
      [SaleItemInput(productId: product.id, quantity: 1)],
      idempotencyKey: 'web-e2e-${DateTime.now().millisecondsSinceEpoch}',
      payments: [
        PaymentInput(method: PaymentMethod.cash, amountMinor: product.priceMinor),
      ],
      sessionId: register.id,
    );
    expect(sale.id, isNotEmpty);
    expect(sale.totalMinor, greaterThan(0));

    final receipt = await api.fetchReceipt(session, sale.id);
    expect(receipt.saleId, sale.id);

    await api.logout(session);
  });

  test('web E2E: secure storage round-trips the session in a browser',
      () async {
    final api = ApiClient();
    final store = SessionStore();
    final session = await api.login(
      tenantId: 'demo-restaurant',
      email: demoEmail,
      password: demoPassword,
      deviceId: 'web-e2e-device',
      deviceName: 'Web E2E',
    );

    // flutter_secure_storage on web is WebCrypto + localStorage, so this only
    // works on a secure origin. A plain-http non-localhost page throws
    // "Cannot read properties of undefined (reading 'generateKey')".
    await store.save(session);
    final restored = await store.read();
    expect(restored, isNotNull);
    expect(restored!.accessToken, session.accessToken);
    expect(restored.tenantId, session.tenantId);
    await store.clear();
  });

  test('web E2E: restaurant table, tip and split bill', () async {
    final api = ApiClient();
    final session = await api.login(
      tenantId: 'demo-restaurant',
      email: demoEmail,
      password: demoPassword,
      deviceId: 'web-e2e-device',
      deviceName: 'Web E2E',
    );
    expect(session.accessToken, isNotEmpty);

    final ts = DateTime.now().millisecondsSinceEpoch;
    final floor =
        await postJson(session, '/v1/floors', {'name': 'Web E2E Floor $ts', 'sort_order': 99});
    final table = await postJson(session, '/v1/tables', {
      'floor_id': floor['id'],
      'name': 'WEB-$ts',
      'seats': 4,
    });
    final tableId = table['id'] as String;

    RegisterSession? current;
    try {
      current = await api.currentSession(session);
    } on ApiException {
      current = null;
    }
    final register = current ?? await api.openSession(session, openingCashMinor: 0);

    final products = await api.products(session);
    final withStock = products.where((p) => p.stockQuantity > 0).take(2).toList();
    expect(withStock.length, 2);

    final sale = await api.createSale(
      session,
      [for (final p in withStock) SaleItemInput(productId: p.id, quantity: 1)],
      idempotencyKey: 'web-e2e-restaurant-$ts',
      payments: [
        PaymentInput(
          method: PaymentMethod.cash,
          amountMinor: withStock.fold<int>(0, (sum, p) => sum + p.priceMinor),
          tipMinor: 500,
        ),
      ],
      sessionId: register.id,
      tableId: tableId,
    );
    expect(sale.id, isNotEmpty);

    final detail = await api.saleDetail(session, sale.id);
    expect(detail.items, hasLength(2));
    final split = await api.splitSale(session, sale.id, [
      [SplitBillLine(saleItemId: detail.items[0].id, quantity: 1)],
      [SplitBillLine(saleItemId: detail.items[1].id, quantity: 1)],
    ]);
    expect(split.parentSaleId, sale.id);
    expect(split.children, hasLength(2));
    expect(split.currency, 'EGP');

    final tablesAfter = await api.tables(session);
    expect(tablesAfter.firstWhere((t) => t.id == tableId).status, 'occupied');

    await api.logout(session);
  });
}