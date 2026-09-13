import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/registers.dart';

/// Real-backend E2E on the device (E4). Logs in against the live backend and
/// drives the checkout loop over the network:
///
///   login -> categories -> products -> register open -> create sale ->
///   receipt -> logout
///
/// Point it at a live backend with:
///
///   flutter test integration_test/network_e2e_test.dart -d <device> \
///     --dart-define=API_BASE_URL=http://<host>:<port>
///
/// Uses the demo restaurant tenant (admin@demo-restaurant.com / admin). Fails
/// fast with a thrown ApiException if the backend is unreachable or the demo
/// tenant is not seeded on the target instance.
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  test('network E2E: login, browse, sell, receipt, logout', () async {
    final api = ApiClient();
    final session = await api.login(
      tenantId: 'demo-restaurant',
      email: 'admin@demo-restaurant.com',
      password: 'admin',
      deviceId: 'integration-test-device',
      deviceName: 'Network E2E',
    );
    expect(session.accessToken, isNotEmpty);
    expect(session.tenantId, isNotEmpty);
    expect(session.currencyCode, anyOf('EGP', ''));

    final countries = await api.countries();
    expect(countries, isNotEmpty);
    final currencies = await api.currencies();
    expect(currencies, isNotEmpty);

    final categories = await api.categories(session);
    expect(categories, isNotEmpty);

    final products = await api.products(session);
    expect(products, isNotEmpty);
    final product = products.firstWhere((p) => p.stockQuantity > 0);

    RegisterSession? current;
    try {
      current = await api.currentSession(session);
    } on ApiException {
      current = null;
    }
    final register =
        current ?? await api.openSession(session, openingCashMinor: 0);
    expect(register.id, isNotEmpty);
    expect(register.status, anyOf('open', 'completed'));

    final sale = await api.createSale(
      session,
      [SaleItemInput(productId: product.id, quantity: 1)],
      idempotencyKey: 'e2e-${DateTime.now().millisecondsSinceEpoch}',
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
}