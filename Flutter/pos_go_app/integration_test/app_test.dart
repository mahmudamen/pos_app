import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

/// Device-level smoke test for the checkout loop (E4). It drives the real
/// widget tree against an in-memory fake ApiClient, so it needs no backend and
/// no network while still exercising navigation, cart, and the register
/// session bar on a real device/emulator:
///
///   flutter test integration_test/app_test.dart
///   flutter drive --driver=test_driver/integration_test.dart \
///     --target=integration_test/app_test.dart
///
/// The widget-level equivalent (runs on the host with `flutter test`) lives in
/// test/offline_replay_test.dart.
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('login-less checkout flow: cart to completion', (tester) async {
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Device Test',
      tenantId: 'tenant-1',
      deviceId: 'device-1',
      role: 'manager',
      currencyCode: 'EGP',
    );
    final api = _StubApiClient();

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: AppStrings.supportedLocales,
        localizationsDelegates: const [
          AppStrings.delegate,
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        home: PosScreen(
          session: session,
          apiClient: api,
          onSignOut: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Complete sale'), findsOneWidget);
    expect(find.text('No open session on this terminal'), findsOneWidget);

    await tester.tap(find.text('Cappuccino'));
    await tester.pump();
    await tester.tap(find.text('Complete sale'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Confirm payment'));
    await tester.pumpAndSettle();

    expect(api.createdSaleIds, hasLength(1));
    expect(find.textContaining(api.createdSaleIds.single), findsWidgets);
  });
}

class _StubApiClient extends ApiClient {
  _StubApiClient() : super();

  final List<String> createdSaleIds = [];

  static const _products = [
    Product(
      id: 'product-1',
      name: 'Cappuccino',
      sku: 'CAP',
      barcode: '',
      priceMinor: 350,
      currency: 'EGP',
      stockQuantity: 10,
      categoryId: 'category-1',
    ),
  ];

  static const _categories = [
    Category(id: 'category-1', name: 'Drinks', slug: 'drinks'),
  ];

  @override
  Future<List<Product>> products(Session session, {String search = ''}) async =>
      _products;

  @override
  Future<List<Category>> categories(Session session) async => _categories;

  @override
  Future<RegisterSession?> currentSession(Session session) async => null;

  @override
  Future<TenantSettings> settings(Session session) async =>
      const TenantSettings();

  @override
  Future<SaleResult> createSale(
    Session session,
    List<SaleItemInput> items, {
    String? idempotencyKey,
    List<PaymentInput>? payments,
    String? sessionId,
  }) async {
    final id = 'sale-${createdSaleIds.length + 1}';
    createdSaleIds.add(id);
    return SaleResult(
      id: id,
      subtotalMinor: 350,
      totalMinor: 350,
      currency: 'EGP',
      paymentMethod: PaymentMethod.cash,
    );
  }
}