import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/restaurants.dart';
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

  testWidgets('restaurant flow: assign table, tip, split the bill',
      (tester) async {
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

    await tester.tap(find.text('Pick a table'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('T01'));
    await tester.pumpAndSettle();
    expect(find.text('T01'), findsWidgets);

    await tester.tap(find.text('Cappuccino').first);
    await tester.pump();
    await tester.tap(find.text('Cappuccino').first);
    await tester.pump();
    await tester.tap(find.text('Complete sale'));
    await tester.pumpAndSettle();

    expect(find.text('Payment method'), findsOneWidget);
    expect(find.text('T01'), findsWidgets);
    // Cash(0), card(1), mobile(2), tip(3), discount(4).
    final sheetFields = find.descendant(
      of: find.byType(PaymentSheet),
      matching: find.byType(TextField),
    );
    expect(sheetFields, findsNWidgets(5));
    await tester.enterText(sheetFields.at(3), '5.00');
    await tester.pump();
    expect(find.text('E£12.00'), findsWidgets);

    await tester.tap(find.text('Confirm payment'));
    await tester.pumpAndSettle();

    expect(api.createdSaleIds, hasLength(1));
    expect(api.lastTableId, 't1');
    expect(api.lastPayments!.single.amountMinor, 700);
    expect(api.lastPayments!.single.tipMinor, 500);

    await tester.tap(find.text('Split bill'));
    await tester.pumpAndSettle();
    expect(find.text('Cappuccino'), findsWidgets);
    expect(find.text('Split bill'), findsWidgets);

    await tester.tap(find.byKey(const Key('cover-0-item-0-minus')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('cover-1-item-0-plus')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('split-action')));
    await tester.pumpAndSettle();
    expect(api.splitCalls, hasLength(1));
    final groups = api.splitCalls.single;
    expect(groups, hasLength(2));
    expect(groups[0].single.quantity, 1);
    expect(groups[1].single.quantity, 1);
    expect(find.text('OK'), findsOneWidget);
  });
}

class _StubApiClient extends ApiClient {
  _StubApiClient() : super();

  final List<String> createdSaleIds = [];
  List<PaymentInput>? lastPayments;
  String? lastTableId;
  final List<List<List<SplitBillLine>>> splitCalls = [];

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

  static const _tables = [
    RestaurantTable(
      id: 't1',
      floorId: 'f1',
      name: 'T01',
      seats: 4,
      status: 'free',
      floorName: 'Ground',
    ),
  ];

  @override
  Future<List<Product>> products(Session session, {String search = ''}) async =>
      _products;

  @override
  Future<List<Category>> categories(Session session) async => _categories;

  @override
  Future<List<Floor>> floors(Session session) async => const [
        Floor(id: 'f1', name: 'Ground', sortOrder: 0),
      ];

  @override
  Future<List<RestaurantTable>> tables(Session session) async => _tables;

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
    String? tableId,
    int discountMinor = 0,
    String managerPin = '',
    int roundingMinor = 0,
  }) async {
    final id = 'sale-${createdSaleIds.length + 1}';
    createdSaleIds.add(id);
    lastPayments = payments;
    lastTableId = tableId;
    return SaleResult(
      id: id,
      subtotalMinor: 350,
      totalMinor: 350,
      currency: 'EGP',
      paymentMethod: PaymentMethod.cash,
    );
  }

  @override
  Future<SaleDetail> saleDetail(Session session, String saleId) async =>
      SaleDetail(
        id: saleId,
        status: 'completed',
        currency: 'EGP',
        items: const [
          SaleDetailItem(
              id: 'line-1',
              productName: 'Cappuccino',
              quantity: 2,
              unitPriceMinor: 350,
              totalMinor: 700),
        ],
      );

  @override
  Future<SplitBillResult> splitSale(
    Session session,
    String saleId,
    List<List<SplitBillLine>> children,
  ) async {
    splitCalls.add(children);
    return SplitBillResult(
      parentSaleId: saleId,
      children: const ['child-a', 'child-b'],
      currency: 'EGP',
    );
  }
}