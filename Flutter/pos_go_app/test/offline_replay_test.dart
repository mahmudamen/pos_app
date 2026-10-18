import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/app_exception.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/receipts.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/core/storage/local_database.dart';
import 'package:pos_go_app/features/pos/pending_sync_screen.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/features/sales/receipt_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

class _FakeApiClient extends ApiClient {
  _FakeApiClient() : super();

  bool offline = true;
  bool reject = false;
  bool conflict = false;
  String? errorDetail;
  int syncCalls = 0;

  /// Counts receipt reads so a queued sale can be shown never to reach for one.
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
          status: reject
              ? 'rejected'
              : conflict
                  ? 'conflict'
                  : 'applied',
          errorCode: (reject || conflict) ? 'insufficient_stock' : null,
          errorDetail:
              (reject || conflict) ? (errorDetail ?? 'insufficient stock') : null,
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
  Future<void> cacheCategories(List<Category> categories) async {}

  @override
  Future<List<Category>> cachedCategories() async => const [];

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

  final List<PendingCommand> failed = [];

  @override
  Future<List<PendingCommand>> failedCommands() async =>
      List.unmodifiable(failed);

  @override
  Future<void> failCommand(
    String id, {
    String? errorCode,
    String? errorDetail,
    bool conflict = false,
  }) async {
    final index = commands.indexWhere((c) => c.id == id);
    if (index == -1) return;
    final command = commands.removeAt(index);
    failed.add(PendingCommand(
      id: command.id,
      idempotencyKey: command.idempotencyKey,
      operation: command.operation,
      payload: command.payload,
      attempts: command.attempts + 1,
      lastError: errorCode,
      errorDetail: errorDetail,
      isConflict: conflict,
    ));
  }

  @override
  Future<void> retryCommand(String id) async {
    final index = failed.indexWhere((c) => c.id == id);
    if (index == -1) return;
    final command = failed.removeAt(index);
    commands.add(PendingCommand(
      id: command.id,
      idempotencyKey: command.idempotencyKey,
      operation: command.operation,
      payload: command.payload,
      attempts: command.attempts,
    ));
  }

  @override
  Future<void> dropCommand(String id) async {
    failed.removeWhere((c) => c.id == id);
  }

  @override
  Future<void> saveLocalSession(LocalRegisterSession session) async {}

  @override
  Future<LocalRegisterSession?> openLocalSession() async => null;

  @override
  Future<List<LocalRegisterSession>> localSessions() async => const [];
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
    expect(fake.fetchReceiptCalls, 0,
        reason: 'a queued sale has no id yet, so there is no receipt to show');
    expect(find.byType(ReceiptScreen), findsNothing,
        reason: 'the offline path must stay on the POS grid');

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

  testWidgets('a rejected offline sale lands on the failed-sync screen',
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
    expect(db.commands, hasLength(1));

    // Let the offline-save snackbar expire so the replay notice is the one
    // on screen (both share one messenger across pumpWidget calls).
    await tester.pump(const Duration(seconds: 5));
    await tester.pumpAndSettle();

    fake.reject = true;
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

    expect(db.commands, isEmpty,
        reason: 'a refused command must leave the replay queue');
    expect(db.failed, hasLength(1),
        reason: '...but stay visible as a failure instead of vanishing');
    expect(db.failed.single.lastError, 'insufficient_stock');

    await tester.tap(find.text('View'));
    await tester.pumpAndSettle();
    expect(find.byType(PendingSyncScreen), findsOneWidget);
    expect(find.text('Queued sale · 1 items'), findsOneWidget);
    expect(find.text('insufficient_stock · 1 attempts'), findsOneWidget);

    await tester.tap(find.byIcon(Icons.delete_outline));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilledButton, 'Discard'));
    await tester.pumpAndSettle();

    expect(db.failed, isEmpty, reason: 'discard drops the command');
    expect(find.text('Offline sales synced'), findsOneWidget,
        reason: 'the screen falls back to its empty state');
  });

  testWidgets('a conflict is reviewed apart from a failure, and retry '
      'refreshes first', (tester) async {
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
    db.failed.addAll(const [
      PendingCommand(
        id: 'conflict-1',
        idempotencyKey: 'key-1',
        operation: 'create_sale',
        payload: '{"items":[{},{}]}',
        attempts: 1,
        lastError: 'insufficient_stock',
        errorDetail: 'insufficient stock',
        isConflict: true,
      ),
      PendingCommand(
        id: 'rejected-1',
        idempotencyKey: 'key-2',
        operation: 'create_sale',
        payload: '{"items":[{}]}',
        attempts: 2,
        lastError: 'validation_error',
      ),
    ]);
    var refreshed = 0;
    var replayed = 0;

    await tester.pumpWidget(
      _wrap(PendingSyncScreen(
        apiClient: fake,
        session: session,
        localDatabase: db,
        onReplay: () async => replayed++,
        onRefresh: () async => refreshed++,
      )),
    );
    await tester.pumpAndSettle();

    expect(find.text('Needs review (1)'), findsOneWidget);
    expect(find.text('Failed sync (1)'), findsOneWidget);
    expect(
      find.textContaining('Stock ran out before this sale synced'),
      findsOneWidget,
      reason: 'a conflict is explained in plain language',
    );
    expect(find.text('insufficient stock'), findsOneWidget,
        reason: "the server's own detail is kept underneath");
    expect(find.text('validation_error · 2 attempts'), findsOneWidget,
        reason: 'a rejection still shows its raw code');

    // The conflict's refresh retries against fresh state.
    await tester.tap(find.byIcon(Icons.refresh).first);
    await tester.pumpAndSettle();
    expect(refreshed, 1);
    expect(replayed, 0);
    expect(db.failed.map((c) => c.id), isNot(contains('conflict-1')),
        reason: 'retry moves the conflict back into the replay queue');
    expect(find.text('Needs review (1)'), findsNothing);

    // The rejection's retry goes straight back through the plain replay.
    await tester.tap(find.byIcon(Icons.refresh).first);
    await tester.pumpAndSettle();
    expect(replayed, 1);
    expect(refreshed, 1, reason: 'a rejection must not trigger a refresh');
  });

  testWidgets('the replay loop flags a conflicted sale as recoverable',
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
    expect(db.commands, hasLength(1));

    await tester.pump(const Duration(seconds: 5));
    await tester.pumpAndSettle();

    fake.conflict = true;
    fake.errorDetail = 'insufficient stock';
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

    expect(db.failed, hasLength(1));
    expect(db.failed.single.isConflict, isTrue,
        reason: 'a 409 push result must be parked as a conflict, not a failure');
    expect(db.failed.single.errorDetail, 'insufficient stock');
  });

  testWidgets('a stalled checkout queues the sale instead of hanging forever',
      (tester) async {
    final db = _FakeLocalDatabase();
    final fake = _HangApiClient();
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

    final previous = PosScreen.saleTimeout;
    PosScreen.saleTimeout = const Duration(milliseconds: 100);
    addTearDown(() => PosScreen.saleTimeout = previous);

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
    await tester.pump(); // sheet dismisses, the call starts
    await tester.pump(const Duration(milliseconds: 300)); // the timeout fires
    await tester.pumpAndSettle();

    expect(db.commands, hasLength(1),
        reason: 'a timeout must land in the offline queue, not hang forever');
  });

}


/// createSale never answers, like a request that stalls on a dead link.
class _HangApiClient extends _FakeApiClient {
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
  }) =>
      Completer<SaleResult>().future;
}
