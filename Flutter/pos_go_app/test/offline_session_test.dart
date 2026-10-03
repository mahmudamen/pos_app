import 'dart:convert';

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

const _session = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-1',
  displayName: 'Restaurant Admin',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'manager',
  currencyCode: 'EGP',
);

/// Offline for as long as [offline] is true, then answers normally.
class _FlakyApiClient extends ApiClient {
  bool offline = true;
  int openSessionCalls = 0;
  final List<Map<String, dynamic>> pushedPayloads = [];

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

  void _guard() {
    if (offline) throw const NetworkException('offline');
  }

  @override
  Future<List<Product>> products(Session session, {String search = ''}) async {
    _guard();
    return _products;
  }

  @override
  Future<List<Category>> categories(Session session) async {
    _guard();
    return _categories;
  }

  @override
  Future<RegisterSession?> currentSession(Session session) async {
    _guard();
    return null;
  }

  @override
  Future<TenantSettings> settings(Session session) async {
    _guard();
    return const TenantSettings();
  }

  @override
  Future<RegisterSession> openSession(
    Session session, {
    required int openingCashMinor,
  }) async {
    openSessionCalls++;
    _guard();
    return RegisterSession(
      id: 'server-session-1',
      status: 'open',
      openingCashMinor: openingCashMinor,
      openedAt: '2026-03-01T09:00:00Z',
      openedBy: 'Restaurant Admin',
      summary: const SessionSummary(
        salesCount: 0,
        subtotalMinor: 0,
        discountMinor: 0,
        taxMinor: 0,
        totalMinor: 0,
        cashMinor: 0,
        cardMinor: 0,
        mobileMinor: 0,
        expectedCashMinor: 0,
      ),
    );
  }

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
    _guard();
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
    _guard();
    pushedPayloads.add(commands.first.payload);
    return SyncPushPage(
      results: [
        SyncPushResult(commandId: commands.first.commandId, status: 'applied'),
      ],
    );
  }
}

class _FakeLocalDatabase extends LocalDatabase {
  _FakeLocalDatabase() : super(factory: databaseFactoryFfi);

  final List<PendingCommand> commands = [];
  final List<LocalRegisterSession> sessions = [];

  /// With the link down the till has to trade from the cached catalog, which is
  /// what the screen falls back to; serving it here is what makes the product
  /// grid reachable while offline.
  @override
  Future<List<Product>> cachedProducts() async => _FlakyApiClient._products;

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

  @override
  Future<void> saveLocalSession(LocalRegisterSession session) async {
    sessions.removeWhere((s) => s.id == session.id);
    sessions.add(session);
  }

  @override
  Future<LocalRegisterSession?> openLocalSession() async {
    for (final session in sessions.reversed) {
      if (session.isOpen) return session;
    }
    return null;
  }

  @override
  Future<List<LocalRegisterSession>> localSessions() async =>
      List<LocalRegisterSession>.of(sessions);
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

/// Lets the confirmation snackbar time out: it sits over the checkout button
/// and would swallow the tap.
Future<void> _clearSnackbars(WidgetTester tester) =>
    tester.pumpAndSettle(const Duration(seconds: 5));

/// Remounts the screen so bootstrap and the replay both run again, as they do
/// when the app comes back to the foreground on a restored link.
Future<void> _reopen(
  WidgetTester tester,
  ApiClient api,
  LocalDatabase db,
) async {
  await tester.pumpWidget(_wrap(KeyedSubtree(
    key: UniqueKey(),
    child: PosScreen(
      session: _session,
      apiClient: api,
      localDatabase: db,
      onSignOut: () {},
    ),
  )));
  await tester.pumpAndSettle();
}

void main() {
  setUpAll(sqfliteFfiInit);

  testWidgets('offline shift: the cashier can open a session without a server',
      (tester) async {
    final api = _FlakyApiClient();
    final db = _FakeLocalDatabase();

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      localDatabase: db,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    // The bar offers to open a shift even though every call is failing.
    expect(find.text('Open session'), findsOneWidget);
    await tester.tap(find.text('Open session'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Open session').last);
    await tester.pumpAndSettle();

    expect(api.openSessionCalls, greaterThan(0),
        reason: 'the attempt is made before it can fail');
    expect(find.textContaining('opened offline'), findsOneWidget);
    expect(find.textContaining('Offline'), findsOneWidget,
        reason: 'the bar must show the shift is device-local');
  });

  testWidgets('offline sale carries the local shift, never a server session id',
      (tester) async {
    final api = _FlakyApiClient();
    final db = _FakeLocalDatabase();

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      localDatabase: db,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Open session'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Open session').last);
    await tester.pumpAndSettle();
    await _clearSnackbars(tester);

    await tester.tap(find.text('Cappuccino'));
    await tester.pump();
    await tester.tap(find.text('Complete sale'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Confirm payment'));
    await tester.pumpAndSettle();

    expect(db.commands, hasLength(1));
    final payload =
        jsonDecode(db.commands.single.payload) as Map<String, dynamic>;
    expect(payload['local_session_id'], isNotNull,
        reason: 'the sale must be attributed to the device-local shift');
    expect(payload.containsKey('session_id'), isFalse,
        reason: 'a client-generated id would be rejected as an unknown session');
  });

  testWidgets('reconnecting opens the server session and replays into it',
      (tester) async {
    final api = _FlakyApiClient();
    final db = _FakeLocalDatabase();

    // Take a sale while offline, under a local shift.
    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      localDatabase: db,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Open session'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Open session').last);
    await tester.pumpAndSettle();
    await _clearSnackbars(tester);
    await tester.tap(find.text('Cappuccino'));
    await tester.pump();
    await tester.tap(find.text('Complete sale'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Confirm payment'));
    await tester.pumpAndSettle();
    expect(db.commands, hasLength(1));

    // The link returns: replay must open the real session first.
    api.offline = false;
    await _reopen(tester, api, db);

    expect(api.pushedPayloads, hasLength(1));
    final pushed = api.pushedPayloads.single;
    expect(pushed['session_id'], 'server-session-1',
        reason: 'the sale lands inside the server session the Z-report reads');
    expect(pushed.containsKey('local_session_id'), isFalse,
        reason: 'the client id must not reach the server');
    expect(db.commands, isEmpty, reason: 'the applied sale leaves the queue');
  });

  testWidgets('a queued sale with no shift still replays, unattributed',
      (tester) async {
    final api = _FlakyApiClient();
    final db = _FakeLocalDatabase();

    // No shift opened: sell offline, then reconnect.
    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      localDatabase: db,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Cappuccino'));
    await tester.pump();
    await tester.tap(find.text('Complete sale'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Confirm payment'));
    await tester.pumpAndSettle();
    expect(db.commands, hasLength(1));

    api.offline = false;
    await _reopen(tester, api, db);

    expect(api.pushedPayloads, hasLength(1));
    expect(api.pushedPayloads.single.containsKey('session_id'), isFalse);
    expect(db.commands, isEmpty);
  });
}