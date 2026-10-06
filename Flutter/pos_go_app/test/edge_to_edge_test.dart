import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

/// A gesture navigation handle / 3-button navigation bar, the smallest inset
/// Android 15 hands a foreground app (48dp).
const double _gestureBar = 48;

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

class _FakeApiClient extends ApiClient {
  _FakeApiClient() : super();

  @override
  Future<List<Product>> products(Session session, {String search = ''}) async =>
      List.generate(
        30,
        (i) => Product(
          id: 'p$i',
          name: 'Cappuccino $i',
          sku: 'SKU-$i',
          barcode: '622$i',
          priceMinor: 1000 + i,
          currency: 'EGP',
          stockQuantity: 5 + i,
          categoryId: 'category-1',
        ),
      );

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
  }) async =>
      throw UnimplementedError();
}

/// Wraps [child] in an app whose media query reports a system navigation
/// inset, the way Android 15 does once edge-to-edge is enforced.
Widget _withInsets(Widget child, {required double bottom}) {
  return MaterialApp(
    locale: const Locale('en'),
    supportedLocales: AppStrings.supportedLocales,
    localizationsDelegates: const [
      AppStrings.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    builder: (context, inner) => MediaQuery(
      data: MediaQuery.of(context).copyWith(
        padding: MediaQuery.of(context).padding.copyWith(bottom: bottom),
        viewPadding:
            MediaQuery.of(context).viewPadding.copyWith(bottom: bottom),
      ),
      child: inner!,
    ),
    home: child,
  );
}

void main() {
  Future<void> pumpPos(WidgetTester tester, {required double bottom}) async {
    // A phone-sized surface in English. The narrow layout pins the cart under
    // the catalog, and English is the worse case for the session bar: its
    // message sits next to two buttons, so a squeezed text wraps over many
    // lines and pushes the cart off the bottom of the screen entirely.
    tester.view.physicalSize = const Size(390, 844) * 3;
    tester.view.devicePixelRatio = 3.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(
      _withInsets(
        PosScreen(
          session: _session,
          apiClient: _FakeApiClient(),
          onSignOut: () {},
        ),
        bottom: bottom,
      ),
    );
    await tester.pumpAndSettle();
  }

  testWidgets('checkout button clears the system navigation inset',
      (tester) async {
    await pumpPos(tester, bottom: _gestureBar);

    final button = find.widgetWithText(FilledButton, 'Complete sale');
    expect(button, findsOneWidget, reason: 'the checkout button must exist');

    final buttonBottom = tester.getRect(button).bottom;
    final screenHeight = tester.view.physicalSize.height /
        tester.view.devicePixelRatio;

    // The whole button, not just its centre, has to sit above the gesture bar.
    expect(
      buttonBottom,
      lessThanOrEqualTo(screenHeight - _gestureBar),
      reason: 'checkout button overlaps the system navigation inset',
    );
  });

  testWidgets('checkout button is still fully on screen with no inset',
      (tester) async {
    await pumpPos(tester, bottom: 0);

    final buttonBottom =
        tester.getRect(find.widgetWithText(FilledButton, 'Complete sale')).bottom;
    final screenHeight = tester.view.physicalSize.height /
        tester.view.devicePixelRatio;

    expect(buttonBottom, lessThanOrEqualTo(screenHeight));
  });

  testWidgets('a taller inset pushes the button further up', (tester) async {
    // Guards against the pad being applied to the wrong edge or clipped away by
    // the panel's own layout.
    await pumpPos(tester, bottom: 0);
    final flush = tester
        .getRect(find.widgetWithText(FilledButton, 'Complete sale'))
        .bottom;

    await pumpPos(tester, bottom: _gestureBar);
    final inset = tester
        .getRect(find.widgetWithText(FilledButton, 'Complete sale'))
        .bottom;

    expect(inset, lessThan(flush));
  });
}
