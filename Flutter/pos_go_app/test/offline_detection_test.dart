import 'dart:async';
import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
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

/// Checkout throws whatever [checkoutError] holds, so a test can choose between
/// the network being down and the client itself being broken.
class _FailingApiClient extends ApiClient {
  _FailingApiClient(this.checkoutError);

  final Object checkoutError;

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

  @override
  Future<List<Product>> products(Session session, {String search = ''}) async =>
      _products;

  @override
  Future<List<Category>> categories(Session session) async =>
      const [Category(id: 'category-1', name: 'Drinks', slug: 'drinks')];

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
    throw checkoutError;
  }
}

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

/// Adds a product and walks the payment sheet to the point where checkout
/// calls the API.
Future<void> _checkoutCappuccino(WidgetTester tester) async {
  await tester.tap(find.text('Cappuccino'));
  await tester.pump();
  await tester.tap(find.text('Complete sale'));
  await tester.pumpAndSettle();
  await tester.tap(find.text('Confirm payment'));
  await tester.pumpAndSettle();
}

void main() {
  setUpAll(sqfliteFfiInit);

  group('isTransportFailure', () {
    test('treats a dead link as offline', () {
      expect(isTransportFailure(const NetworkException('down')), isTrue);
      expect(isTransportFailure(const SocketException('no route')), isTrue);
      expect(isTransportFailure(http.ClientException('closed')), isTrue);
      expect(isTransportFailure(TimeoutException('slow')), isTrue);
    });

    test('does not treat a rejection or a bug as offline', () {
      expect(isTransportFailure(const ApiException('server said no')), isFalse);
      expect(isTransportFailure(StateError('bad state')), isFalse);
      expect(isTransportFailure(const FormatException('bad json')), isFalse);
      expect(isTransportFailure(const StockException('out of stock')), isFalse);
    });
  });

  group('ApiClient transport failures', () {
    test('a dead link surfaces as NetworkException', () async {
      final api = ApiClient(
        client: MockClient((_) async => throw const SocketException('down')),
      );

      await expectLater(
        api.products(_session),
        throwsA(isA<NetworkException>()),
      );
    });

    test('a server response is still mapped, not treated as offline', () async {
      final api = ApiClient(
        client: MockClient((_) async => http.Response(
              jsonEncode({
                'error': {'code': 'not_found', 'message': 'nope'},
              }),
              404,
            )),
      );

      await expectLater(api.products(_session), throwsA(isA<ApiException>()));
    });
  });

  group('checkout when the network cuts', () {
    testWidgets('a transport failure queues the sale locally',
        (tester) async {
      final db = _FakeLocalDatabase();

      await tester.pumpWidget(_wrap(PosScreen(
        session: _session,
        apiClient: _FailingApiClient(const NetworkException('offline')),
        localDatabase: db,
        onSignOut: () {},
      )));
      await tester.pumpAndSettle();
      await _checkoutCappuccino(tester);

      expect(db.commands, hasLength(1));
      expect(db.commands.single.operation, 'create_sale');
    });

    testWidgets('a raw socket failure also queues the sale locally',
        (tester) async {
      final db = _FakeLocalDatabase();

      await tester.pumpWidget(_wrap(PosScreen(
        session: _session,
        apiClient: _FailingApiClient(const SocketException('no route')),
        localDatabase: db,
        onSignOut: () {},
      )));
      await tester.pumpAndSettle();
      await _checkoutCappuccino(tester);

      expect(db.commands, hasLength(1));
    });

    testWidgets('a client bug is surfaced, never queued as a sale',
        (tester) async {
      final db = _FakeLocalDatabase();

      await tester.pumpWidget(_wrap(PosScreen(
        session: _session,
        apiClient: _FailingApiClient(StateError('cart is corrupt')),
        localDatabase: db,
        onSignOut: () {},
      )));
      await tester.pumpAndSettle();
      await _checkoutCappuccino(tester);

      expect(db.commands, isEmpty,
          reason: 'only a transport failure may hand a sale to the queue');
      expect(find.textContaining('cart is corrupt'), findsOneWidget);
    });

    testWidgets('a server rejection is surfaced, never queued as a sale',
        (tester) async {
      final db = _FakeLocalDatabase();

      await tester.pumpWidget(_wrap(PosScreen(
        session: _session,
        apiClient: _FailingApiClient(const ApiException('out of stock')),
        localDatabase: db,
        onSignOut: () {},
      )));
      await tester.pumpAndSettle();
      await _checkoutCappuccino(tester);

      expect(db.commands, isEmpty);
      expect(find.textContaining('out of stock'), findsOneWidget);
    });
  });
}