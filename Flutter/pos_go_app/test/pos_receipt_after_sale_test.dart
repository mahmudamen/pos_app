import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/app_exception.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/receipts.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/restaurants.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/core/storage/local_database.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/features/sales/receipt_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

class _FakeApiClient extends ApiClient {
  bool offline = true;
  bool failReceipt = false;
  int fetchReceiptCalls = 0;

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
  Future<List<RestaurantTable>> tables(Session session) async => const [
        RestaurantTable(id: 't1', floorId: 'f1', name: 'T1'),
      ];

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
    if (offline) {
      throw const NetworkException('offline');
    }
    return const SaleResult(
      id: 'sale-99',
      subtotalMinor: 350,
      totalMinor: 350,
      currency: 'EGP',
      paymentMethod: PaymentMethod.cash,
    );
  }

  @override
  Future<SaleReceipt> fetchReceipt(Session session, String saleId) async {
    fetchReceiptCalls++;
    if (failReceipt) {
      throw const ApiException('receipt unavailable');
    }
    return SaleReceipt(
      tenantName: 'Demo Store',
      saleId: saleId,
      status: 'completed',
      createdAt: '2026-10-06 12:00:00',
      cashier: 'Cashier',
      currency: 'EGP',
      subtotalMinor: 350,
      totalMinor: 350,
    );
  }
}

/// Mirrors the LocalDatabase contract in memory so the offline branch can be
/// exercised without opening a real SQLite file inside the fake async zone.
class _FakeLocalDatabase extends LocalDatabase {
  _FakeLocalDatabase() : super(factory: databaseFactoryFfi);

  final List<PendingCommand> commands = [];

  @override
  Future<List<Product>> cachedProducts() async => const [];

  @override
  Future<void> cacheProducts(List<Product> products) async {}

  @override
  Future<void> enqueue(PendingCommand command) async {
    commands.add(command);
  }

  @override
  Future<List<PendingCommand>> pendingCommands() async => commands;


  @override
  Future<List<PendingCommand>> failedCommands() async => const [];

  @override
  Future<void> markComplete(String id) async {
    commands.removeWhere((c) => c.id == id);
  }

  @override
  Future<void> saveLocalSession(LocalRegisterSession session) async {}

  @override
  Future<LocalRegisterSession?> openLocalSession() async => null;

  @override
  Future<List<LocalRegisterSession>> localSessions() async => const [];
}

const _session = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-1',
  displayName: 'Store Manager',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'manager',
  currencyCode: 'EGP',
);

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

Future<void> _checkout(WidgetTester tester) async {
  await tester.tap(find.text('Cappuccino'));
  await tester.pump();
  await tester.tap(find.text('Complete sale'));
  await tester.pumpAndSettle();
  await tester.tap(find.text('Confirm payment'));
  await tester.pumpAndSettle();
}

void main() {
  setUpAll(sqfliteFfiInit);

  testWidgets('a completed sale opens the on-screen receipt', (tester) async {
    final api = _FakeApiClient()..offline = false;

    await tester.pumpWidget(
      _wrap(PosScreen(session: _session, apiClient: api, onSignOut: () {})),
    );
    await tester.pumpAndSettle();

    await _checkout(tester);

    expect(api.fetchReceiptCalls, 1,
        reason: 'the sale id must be read back as a receipt');
    expect(find.byType(ReceiptScreen), findsOneWidget);
    expect(find.textContaining('sale-99'), findsWidgets,
        reason: 'the receipt must identify the sale it renders');
    expect(tester.takeException(), isNull);
  });

  testWidgets('closing the receipt returns to the POS with the completion bar',
      (tester) async {
    final api = _FakeApiClient()..offline = false;

    await tester.pumpWidget(
      _wrap(PosScreen(session: _session, apiClient: api, onSignOut: () {})),
    );
    await tester.pumpAndSettle();

    await _checkout(tester);
    expect(find.byType(ReceiptScreen), findsOneWidget);

    await tester.pageBack();
    await tester.pumpAndSettle();

    expect(find.byType(ReceiptScreen), findsNothing);
    expect(find.byType(PosScreen), findsOneWidget);
    expect(find.textContaining('Sale completed'), findsOneWidget,
        reason: 'the confirmation bar waits for the receipt to close');
    expect(find.text('Cappuccino'), findsOneWidget,
        reason: 'the POS grid must be usable again after the receipt');
  });

  testWidgets('a failed receipt read still confirms the sale', (tester) async {
    final api = _FakeApiClient()
      ..offline = false
      ..failReceipt = true;

    await tester.pumpWidget(
      _wrap(PosScreen(session: _session, apiClient: api, onSignOut: () {})),
    );
    await tester.pumpAndSettle();

    await _checkout(tester);

    expect(api.fetchReceiptCalls, 1);
    expect(find.byType(ReceiptScreen), findsNothing,
        reason: 'a broken receipt must never block or stack a route');
    expect(find.textContaining('Sale completed'), findsOneWidget,
        reason: 'the sale itself still succeeded and must be confirmed');
    expect(find.text('Cappuccino'), findsOneWidget);
  });

  testWidgets('a queued offline sale never reaches for a receipt',
      (tester) async {
    final api = _FakeApiClient(); // offline by default
    final db = _FakeLocalDatabase();

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      localDatabase: db,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    await _checkout(tester);

    expect(db.commands, hasLength(1),
        reason: 'the offline sale must be queued instead of posted');
    expect(api.fetchReceiptCalls, 0,
        reason: 'a queued sale has no id yet, so there is no receipt to show');
    expect(find.byType(ReceiptScreen), findsNothing);
  });

  testWidgets('a sale taken on a table offers split bill once the receipt '
      'is closed', (tester) async {
    final api = _FakeApiClient()..offline = false;

    await tester.pumpWidget(
      _wrap(PosScreen(session: _session, apiClient: api, onSignOut: () {})),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(ActionChip, 'Pick a table'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('T1'));
    await tester.pumpAndSettle();

    await _checkout(tester);
    expect(find.byType(ReceiptScreen), findsOneWidget,
        reason: 'the receipt opens for a table sale too');

    await tester.pageBack();
    await tester.pumpAndSettle();

    expect(
      find.widgetWithText(SnackBarAction, 'Split bill'),
      findsOneWidget,
      reason: 'split bill stays one tap away, back on the POS grid',
    );
  });
}
