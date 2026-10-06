import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

/// On-device proof that the Android 15 edge-to-edge work behaves as the widget
/// test assumes.
///
/// `test/edge_to_edge_test.dart` feeds `MediaQuery` a *simulated* inset, so it
/// cannot catch the two things only a real device can answer: whether Android
/// 15 actually hands Flutter a non-zero bottom padding here, and whether the
/// pinned checkout button clears it in the app's real geometry.
///
/// The fake API keeps this offline: only the insets and the layout are under
/// test, not the network.
class _FakeApiClient extends ApiClient {
  @override
  Future<List<Product>> products(Session session, {String search = ''}) async =>
      List.generate(
        12,
        (i) => Product(
          id: 'p$i',
          name: 'Item $i',
          sku: 'SKU-$i',
          barcode: '622${i}0000',
          priceMinor: 1000 + i,
          currency: 'EGP',
          stockQuantity: 5 + i,
          categoryId: 'c1',
        ),
      );

  @override
  Future<List<Category>> categories(Session session) async =>
      const [Category(id: 'c1', name: 'Drinks', slug: 'drinks')];

  @override
  Future<RegisterSession?> currentSession(Session session) async => null;

  @override
  Future<TenantSettings> settings(Session session) async =>
      const TenantSettings();
}

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('checkout button clears the real system navigation inset',
      (tester) async {
    // English is the locale that regressed: the session-bar message shares its
    // row with two buttons, so an uncapped text wraps and shoves the cart
    // off-screen.
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
          session: const Session(
            accessToken: 'token',
            refreshToken: 'refresh',
            userId: 'user-1',
            displayName: 'Device Probe',
            tenantId: 'tenant-1',
            deviceId: 'device-1',
            role: 'manager',
            currencyCode: 'EGP',
          ),
          apiClient: _FakeApiClient(),
          onSignOut: () {},
        ),
      ),
    );
    await tester.pumpAndSettle();

    final size = tester.view.physicalSize / tester.view.devicePixelRatio;
    final media = MediaQuery.of(tester.element(find.byType(PosScreen)));
    final button = find.widgetWithText(FilledButton, 'Complete sale');
    expect(button, findsOneWidget, reason: 'the checkout button must exist');
    final rect = tester.getRect(button);

    debugPrint('EDGE-PROBE screen=$size dpr=${tester.view.devicePixelRatio} '
        'padding=${media.padding} viewPadding=${media.viewPadding} '
        'button=$rect');

    // The claim under test: Android 15 does hand us a real inset...
    expect(
      media.padding.bottom,
      greaterThan(0),
      reason: 'expected a non-zero bottom inset on a real Android 15 device',
    );
    // ...and the whole button sits above it, not merely its centre.
    expect(
      rect.bottom,
      lessThanOrEqualTo(size.height - media.padding.bottom),
      reason: 'checkout button overlaps the system navigation inset',
    );
  });
}