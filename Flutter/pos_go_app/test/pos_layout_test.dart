// The POS screen is the product's main surface, so the affordances a cashier
// depends on are pinned here: it must lay out right-to-left in Arabic at every
// size, keep a search box and category filters reachable, and never overflow.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

const _category = Category(id: 'category-1', name: 'Drinks', slug: 'drinks');

const _manager = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-manager',
  displayName: 'Manager',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'manager',
  currencyCode: 'EGP',
);

class _AuditApi extends ApiClient {
  List<Product> productList = const [];

  @override
  Future<List<Product>> products(Session session, {String search = ''}) async =>
      productList;

  @override
  Future<List<Category>> categories(Session session) async =>
      const [_category];

  @override
  Future<RegisterSession?> currentSession(Session session) async => null;

  @override
  Future<TenantSettings> settings(Session session) async =>
      const TenantSettings();
}

Widget _wrap(Widget child, Locale locale) => MaterialApp(
      locale: locale,
      supportedLocales: AppStrings.supportedLocales,
      localizationsDelegates: const [
        AppStrings.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      home: child,
    );

List<Product> _catalogue(int n) => List.generate(
      n,
      (i) => Product(
        id: 'p$i',
        name: 'منتج رقم $i',
        sku: 'SKU-$i',
        barcode: '622$i',
        priceMinor: 1000 + i,
        currency: 'EGP',
        stockQuantity: 5 + i,
        categoryId: 'category-1',
      ),
    );

Future<void> _pumpPos(
  WidgetTester tester,
  Size size,
  Locale locale,
) async {
  tester.view.physicalSize = size * 3;
  tester.view.devicePixelRatio = 3;
  addTearDown(tester.view.reset);

  final api = _AuditApi()..productList = _catalogue(30);
  await tester.pumpWidget(
    _wrap(PosScreen(session: _manager, apiClient: api, onSignOut: () {}), locale),
  );
  // Completing this without a RenderFlex overflow is itself an assertion.
  await tester.pumpAndSettle();
}

void main() {
  const sizes = [
    Size(1400, 800), // desktop browser
    Size(820, 1180), // tablet
    Size(390, 844), // phone
  ];

  for (final size in sizes) {
    final name = '${size.width.toInt()}x${size.height.toInt()}';

    testWidgets('POS lays out right-to-left in Arabic at $name', (tester) async {
      await _pumpPos(tester, size, AppStrings.defaultLocale);
      expect(
        Directionality.of(tester.element(find.byType(PosScreen))),
        TextDirection.rtl,
        reason: 'Arabic is the default locale, so the POS must mirror',
      );
    });

    testWidgets('POS keeps search and category filters at $name', (tester) async {
      await _pumpPos(tester, size, AppStrings.defaultLocale);
      expect(find.byType(TextField), findsOneWidget,
          reason: 'cashiers need to look a product up by name or barcode');
      final chips = find.byType(FilterChip).evaluate().length +
          find.byType(ChoiceChip).evaluate().length +
          find.byType(ActionChip).evaluate().length;
      expect(chips, greaterThanOrEqualTo(1),
          reason: 'category chips are the only way to narrow the grid');
    });

    testWidgets('POS grid uses the responsive delegate at $name', (tester) async {
      await _pumpPos(tester, size, AppStrings.defaultLocale);
      final grid = find.byType(GridView);
      expect(grid, findsWidgets);
      final delegate = (tester.widget(grid.first) as GridView).gridDelegate;
      expect(delegate, isA<SliverGridDelegateWithMaxCrossAxisExtent>(),
          reason: 'a fixed column count breaks tablet/desktop widths');
    });
  }

  testWidgets('POS switches to left-to-right when English is chosen',
      (tester) async {
    await _pumpPos(tester, const Size(1400, 800), const Locale('en'));
    expect(
      Directionality.of(tester.element(find.byType(PosScreen))),
      TextDirection.ltr,
    );
  });

  testWidgets('POS lays out in English at 390x844', (tester) async {
    // English is the worse case for the session bar: its message shares the row
    // with two buttons, so an uncapped text squeezes and wraps over a dozen
    // lines, pushing the pinned cart (and the checkout button) off-screen.
    // Completing this pump without a RenderFlex overflow is the assertion.
    await _pumpPos(tester, const Size(390, 844), const Locale('en'));

    const screenHeight = 844.0;
    final button = find.widgetWithText(FilledButton, 'Complete sale');
    expect(button, findsOneWidget);
    expect(tester.getRect(button).bottom, lessThanOrEqualTo(screenHeight));
  });
}