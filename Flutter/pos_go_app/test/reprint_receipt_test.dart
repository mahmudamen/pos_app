import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/receipts.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/core/storage/local_database.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/features/sales/receipt_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

class _FakeApiClient extends ApiClient {
  bool failReceipt = false;
  int fetchReceiptCalls = 0;
  TenantSettings configured = const TenantSettings();

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
  Future<TenantSettings> settings(Session session) async => configured;

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
  }) async =>
      const SaleResult(
        id: 'sale-99',
        subtotalMinor: 350,
        totalMinor: 350,
        currency: 'EGP',
        paymentMethod: PaymentMethod.cash,
      );

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

/// Mirrors the LocalDatabase contract in memory so the widget tests never open
/// a real SQLite file inside the fake async zone (the same rule the offline
/// tests follow); the real schema is covered by local_database_test.
class _FakeLocalDatabase extends LocalDatabase {
  _FakeLocalDatabase() : super(factory: databaseFactoryFfi);

  final Map<String, String> lastSales = {};
  final List<PendingCommand> commands = [];

  @override
  Future<void> saveLastSale(String tenantId, String saleId) async {
    lastSales[tenantId] = saleId;
  }

  @override
  Future<String?> readLastSale(String tenantId) async => lastSales[tenantId];

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

Future<void> _flushSnackbar(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 350));
  await tester.pumpAndSettle();
}

void main() {
  setUpAll(sqfliteFfiInit);

  late _FakeLocalDatabase db;

  setUp(() {
    db = _FakeLocalDatabase();
  });

  testWidgets('reprint opens the receipt of the remembered sale',
      (tester) async {
    await db.saveLastSale('tenant-1', 'sale-42');
    final api = _FakeApiClient();

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      localDatabase: db,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Reprint last receipt'));
    await tester.pumpAndSettle();

    expect(api.fetchReceiptCalls, 1);
    expect(find.byType(ReceiptScreen), findsOneWidget);
    expect(find.textContaining('sale-42'), findsWidgets);
  });

  testWidgets('a completed sale is remembered and reprintable from the grid',
      (tester) async {
    final api = _FakeApiClient();

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      localDatabase: db,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    await _checkout(tester);
    expect(find.byType(ReceiptScreen), findsOneWidget,
        reason: 'the sale opens its receipt as usual');

    await tester.pageBack();
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Reprint last receipt'));
    await tester.pumpAndSettle();

    expect(api.fetchReceiptCalls, 2,
        reason: 'the second fetch must be for the sale just completed');
    expect(find.byType(ReceiptScreen), findsOneWidget);
    expect(find.textContaining('sale-99'), findsWidgets);
  });

  testWidgets('reprint with no sale on this device says so', (tester) async {
    final api = _FakeApiClient();

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      localDatabase: db,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Reprint last receipt'));
    await _flushSnackbar(tester);

    expect(api.fetchReceiptCalls, 0);
    expect(find.byType(ReceiptScreen), findsNothing);
    expect(find.text('No receipt to reprint yet'), findsOneWidget);
  });

  testWidgets('an unreadable last receipt is reported, not swallowed',
      (tester) async {
    await db.saveLastSale('tenant-1', 'sale-42');
    final api = _FakeApiClient()..failReceipt = true;

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      localDatabase: db,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Reprint last receipt'));
    await _flushSnackbar(tester);

    expect(api.fetchReceiptCalls, 1);
    expect(find.byType(ReceiptScreen), findsNothing);
    expect(find.text('Last receipt unavailable'), findsOneWidget);
  });

  testWidgets('the reprint action can be switched off per tenant',
      (tester) async {
    await db.saveLastSale('tenant-1', 'sale-42');
    final api = _FakeApiClient()
      ..configured = const TenantSettings(reprintLastReceipt: false);

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      localDatabase: db,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    expect(find.byTooltip('Reprint last receipt'), findsNothing);
  });
}
