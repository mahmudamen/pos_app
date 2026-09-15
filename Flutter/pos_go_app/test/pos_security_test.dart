import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/security.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

Product _product(String id, String name, int stock, {int priceMinor = 350}) {
  return Product(
    id: id,
    name: name,
    sku: id.toUpperCase(),
    barcode: '',
    priceMinor: priceMinor,
    currency: 'EGP',
    stockQuantity: stock,
    categoryId: 'category-1',
  );
}

const _category = Category(id: 'category-1', name: 'Drinks', slug: 'drinks');

const _summary = SessionSummary(
  salesCount: 0,
  subtotalMinor: 0,
  discountMinor: 0,
  taxMinor: 0,
  totalMinor: 0,
  cashMinor: 0,
  cardMinor: 0,
  mobileMinor: 0,
  expectedCashMinor: 0,
);

class _PosSecurityApi extends ApiClient {
  _PosSecurityApi({required this.customSettings});

  final TenantSettings customSettings;
  RegisterSession? openRegister;
  List<Product> productList = const [];

  int verifyPinCalls = 0;
  String lastManagerPin = '';
  int lastDiscountMinor = 0;
  int? lastClosingCashMinor;

  @override
  Future<List<Product>> products(Session session, {String search = ''}) async =>
      productList;

  @override
  Future<List<Category>> categories(Session session) async =>
      const [_category];

  @override
  Future<RegisterSession?> currentSession(Session session) async => openRegister;

  @override
  Future<TenantSettings> settings(Session session) async => customSettings;

  @override
  Future<PinVerifyResult> verifyPin(
    Session session, {
    required String pin,
  }) async {
    verifyPinCalls++;
    return const PinVerifyResult(
      valid: true,
      hasPin: true,
      attemptsLeft: 5,
      locked: false,
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
  }) async {
    lastDiscountMinor = discountMinor;
    lastManagerPin = managerPin;
    return const SaleResult(
      id: 'sale-secure-1',
      subtotalMinor: 350,
      totalMinor: 150,
      currency: 'EGP',
      paymentMethod: PaymentMethod.cash,
    );
  }

  @override
  Future<RegisterSession> closeSession(
    Session session,
    String sessionId, {
    int? closingCashMinor,
    String managerPin = '',
  }) async {
    lastManagerPin = managerPin;
    lastClosingCashMinor = closingCashMinor;
    return const RegisterSession(
      id: 'sess-0001abcd',
      status: 'closed',
      openingCashMinor: 0,
      closingCashMinor: 1000,
      expectedCashMinor: 0,
      cashDifferenceMinor: 1000,
      openedAt: '2026-09-15T09:00:00Z',
      closedAt: '2026-09-15T15:00:00Z',
      openedBy: 'Cashier',
      summary: _summary,
    );
  }
}

const _cashier = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-cashier',
  displayName: 'Cashier',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'cashier',
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

Color? _badgeColor(WidgetTester tester, String text) {
  final containers = tester.widgetList<Container>(
    find.ancestor(of: find.text(text), matching: find.byType(Container)),
  );
  for (final container in containers) {
    final decoration = container.decoration;
    if (decoration is BoxDecoration &&
        decoration.color != null &&
        decoration.borderRadius != null) {
      return decoration.color;
    }
  }
  return null;
}

void main() {
  testWidgets('stock badges show red for out-of-stock, amber for low, '
      'neutral for healthy', (tester) async {
    final api = _PosSecurityApi(
      customSettings: const TenantSettings(lowStockThreshold: 5),
    );
    api.productList = [
      _product('p-out', 'Espresso', 0),
      _product('p-low', 'Toffee Latte', 2),
      _product('p-ok', 'Cappuccino', 10),
    ];

    await tester.pumpWidget(_wrap(PosScreen(
      session: _cashier,
      apiClient: api,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    final theme = Theme.of(tester.element(find.text('Out of stock')));
    expect(_badgeColor(tester, 'Out of stock'), theme.colorScheme.errorContainer);
    expect(_badgeColor(tester, '2'), Colors.amber.shade100);
    expect(
      _badgeColor(tester, '10'),
      Theme.of(tester.element(find.text('10'))).colorScheme.surfaceContainerHighest,
    );
  });

  testWidgets('low-stock filter keeps only items at or under the threshold',
      (tester) async {
    final api = _PosSecurityApi(
      customSettings: const TenantSettings(lowStockThreshold: 5),
    );
    api.productList = [
      _product('p-low', 'Toffee Latte', 2),
      _product('p-ok', 'Cappuccino', 10),
    ];

    await tester.pumpWidget(_wrap(PosScreen(
      session: _cashier,
      apiClient: api,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    expect(find.text('Toffee Latte'), findsOneWidget);
    expect(find.text('Cappuccino'), findsOneWidget);

    await tester.tap(find.text('Low stock only'));
    await tester.pumpAndSettle();

    expect(find.text('Toffee Latte'), findsOneWidget);
    expect(find.text('Cappuccino'), findsNothing);

    await tester.tap(find.text('Low stock only'));
    await tester.pumpAndSettle();

    expect(find.text('Toffee Latte'), findsOneWidget);
    expect(find.text('Cappuccino'), findsOneWidget);
  });

  testWidgets('blockOutOfStock refuses to add an empty product',
      (tester) async {
    final api = _PosSecurityApi(
      customSettings: const TenantSettings(blockOutOfStock: true),
    );
    api.productList = [_product('p-out', 'Espresso', 0)];

    await tester.pumpWidget(_wrap(PosScreen(
      session: _cashier,
      apiClient: api,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Espresso'));
    await tester.pump();

    expect(find.text('Out of stock — cannot add'), findsOneWidget);
  });

  testWidgets('discount above the cap in block mode prompts for the manager '
      'PIN and forwards it with the sale', (tester) async {
    final api = _PosSecurityApi(
      customSettings: const TenantSettings(
        discountMode: DiscountMode.block,
        maxDiscountPct: 5,
        managerDiscountOverride: true,
      ),
    );
    api.productList = [_product('p-ok', 'Cappuccino', 10)];

    await tester.pumpWidget(_wrap(PosScreen(
      session: _cashier,
      apiClient: api,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Cappuccino'));
    await tester.pump();
    await tester.tap(find.text('Complete sale'));
    await tester.pumpAndSettle();

    final sheet = find.byType(PaymentSheet);
    final sheetFields = find.descendant(
      of: sheet,
      matching: find.byType(TextField),
    );
    await tester.enterText(sheetFields.at(4), '2.00');
    await tester.pump();

    expect(
      find.text('Discount exceeds your limit — enter the manager PIN to approve'),
      findsOneWidget,
    );

    await tester.enterText(sheetFields.at(0), '1.50');
    await tester.pump();
    await tester.ensureVisible(find.text('Confirm payment'));
    await tester.tap(find.text('Confirm payment'));
    await tester.pumpAndSettle();

    expect(find.text('Manager PIN'), findsWidgets,
        reason: 'the manager PIN dialog must open');
    await tester.enterText(
        find.widgetWithText(TextField, 'Manager PIN'), '1234');
    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();

    expect(api.verifyPinCalls, 1);
    expect(api.lastManagerPin, '1234');
    expect(api.lastDiscountMinor, 200);
  });

  testWidgets('block mode without a manager override rejects the discount '
      'before any PIN prompt', (tester) async {
    final api = _PosSecurityApi(
      customSettings: const TenantSettings(
        discountMode: DiscountMode.block,
        maxDiscountPct: 5,
        managerDiscountOverride: false,
      ),
    );
    api.productList = [_product('p-ok', 'Cappuccino', 10)];

    await tester.pumpWidget(_wrap(PosScreen(
      session: _cashier,
      apiClient: api,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Cappuccino'));
    await tester.pump();
    await tester.tap(find.text('Complete sale'));
    await tester.pumpAndSettle();

    final sheetFields = find.descendant(
      of: find.byType(PaymentSheet),
      matching: find.byType(TextField),
    );
    await tester.enterText(sheetFields.at(4), '2.00');
    await tester.enterText(sheetFields.at(0), '1.50');
    await tester.pump();
    await tester.ensureVisible(find.text('Confirm payment'));
    await tester.tap(find.text('Confirm payment'));
    await tester.pumpAndSettle();

    expect(find.text('Discount above your limit is not allowed'), findsWidgets);
    expect(find.text('Manager PIN'), findsNothing,
        reason: 'no override means no PIN prompt');
    expect(api.verifyPinCalls, 0);
  });

  testWidgets('closing a cashier session with managerClosePin asks for the '
      'manager PIN and forwards it', (tester) async {
    final api = _PosSecurityApi(customSettings: const TenantSettings());
    api.openRegister = const RegisterSession(
      id: 'sess-0001abcd',
      status: 'open',
      openingCashMinor: 0,
      openedAt: '2026-09-15T09:00:00Z',
      openedBy: 'Cashier',
      summary: _summary,
    );

    await tester.pumpWidget(_wrap(PosScreen(
      session: _cashier,
      apiClient: api,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(OutlinedButton, 'Finish session'));
    await tester.pumpAndSettle();

    await tester.enterText(
        find.widgetWithText(TextField, 'Cash counted'), '10.00');
    await tester.tap(find.widgetWithText(FilledButton, 'Finish session'));
    await tester.pumpAndSettle();

    expect(find.text('Manager PIN'), findsWidgets,
        reason: 'a cashier closing under managerClosePin must verify the PIN');
    await tester.enterText(
        find.widgetWithText(TextField, 'Manager PIN'), '1234');
    await tester.tap(find.text('OK'));
    await tester.pumpAndSettle();

    expect(api.verifyPinCalls, 1);
    expect(api.lastManagerPin, '1234');
    expect(api.lastClosingCashMinor, 1000);
  });
}