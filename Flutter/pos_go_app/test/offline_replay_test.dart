import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/app_exception.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/core/storage/local_database.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

class _FakeApiClient extends ApiClient {
  _FakeApiClient() : super();

  bool offline = true;
  int syncCalls = 0;

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
    String? tableId,
    int discountMinor = 0,
    String managerPin = '',
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
  Future<SyncPushPage> syncPush(
    Session session,
    List<SyncPushCommand> commands,
  ) async {
    syncCalls++;
    return SyncPushPage(
      results: [
        SyncPushResult(
          commandId: commands.first.commandId,
          status: 'applied',
        ),
      ],
    );
  }
}

/// Mirrors the LocalDatabase contract in memory. The widget only talks to this
/// subset of methods, so the offline enqueue + replay orchestration can be
/// exercised without opening a real (ffi-hosted) SQLite file inside the fake
/// async test zone.
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
  Future<List<PendingCommand>> pendingCommands() async =>
      commands.where((c) => c.operation == 'create_sale').toList();

  @override
  Future<void> markComplete(String id) async {
    commands.removeWhere((c) => c.id == id);
  }
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
  setUpAll(() {
    sqfliteFfiInit();
  });

  testWidgets('offline checkout enqueues the sale; next load replays it',
      (tester) async {
    final db = _FakeLocalDatabase();
    final fake = _FakeApiClient();
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Restaurant Admin',
      tenantId: 'tenant-1',
      deviceId: 'device-1',
      role: 'manager',
      currencyCode: 'EGP',
    );

    await tester.pumpWidget(
      _wrap(PosScreen(
        session: session,
        apiClient: fake,
        localDatabase: db,
        onSignOut: () {},
      )),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Cappuccino'));
    await tester.pump();
    await tester.tap(find.text('Complete sale'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Confirm payment'));
    await tester.pumpAndSettle();

    expect(db.commands, hasLength(1), reason: 'offline sale must be queued');
    expect(db.commands.single.operation, 'create_sale');

    fake.offline = false;
    await tester.pumpWidget(
      _wrap(KeyedSubtree(
        key: UniqueKey(),
        child: PosScreen(
          session: session,
          apiClient: fake,
          localDatabase: db,
          onSignOut: () {},
        ),
      )),
    );
    await tester.pumpAndSettle();

    expect(fake.syncCalls, greaterThan(0),
        reason: 'replay must hit the sync push endpoint');
    expect(db.commands, isEmpty, reason: 'applied command must be marked complete');
  });
}