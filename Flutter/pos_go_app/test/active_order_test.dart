import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

const _session = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-1',
  displayName: 'Restaurant Admin',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'cashier',
  currencyCode: 'EGP',
);

/// A full catalog so the grid and the cart both render the product name.
class _CatalogApi extends ApiClient {
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
}

Widget _wrap(Widget child) {
  return MaterialApp(
    locale: const Locale('en'),
    supportedLocales: AppStrings.supportedLocales,
    localizationsDelegates: const [
      AppStrings.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    home: child,
  );
}

void main() {
  setUp(() {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
  });

  testWidgets('an in-progress order survives a remount of the POS screen',
      (tester) async {
    final api = _CatalogApi();
    final store = SessionStore();

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      sessionStore: store,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Cappuccino'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));
    expect(find.text('Cappuccino'), findsNWidgets(2),
        reason: 'grid tile + cart line');

    final saved = await store.readActiveOrders('tenant-1');
    expect(saved, isNotNull, reason: 'the cart must be persisted as it changes');
    expect(saved, contains('Cappuccino'));

    // A remount stands in for the process being killed and restarted mid-sale.
    await tester.pumpWidget(_wrap(KeyedSubtree(
      key: UniqueKey(),
      child: PosScreen(
        session: _session,
        apiClient: api,
        sessionStore: store,
        onSignOut: () {},
      ),
    )));
    await tester.pumpAndSettle();

    expect(find.text('Cappuccino'), findsNWidgets(2),
        reason: 'the restored cart line joins the grid again');
  });
}
