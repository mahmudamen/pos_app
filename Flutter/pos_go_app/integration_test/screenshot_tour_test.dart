import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/auth/login_screen.dart';
import 'package:pos_go_app/features/community/community_hub_screen.dart';
import 'package:pos_go_app/features/community/national_hub_screen.dart';
import 'package:pos_go_app/features/customers/customers_screen.dart';
import 'package:pos_go_app/features/dashboard/dashboard_screen.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/features/pos/session_screen.dart';
import 'package:pos_go_app/features/sales/sale_history_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

/// Screenshot tour of the real app against a live backend.
///
/// Drives the real LoginScreen, then renders each main screen (POS grid,
/// dashboard, sales history, customers, sessions, community hub, national
/// hub) and records a screenshot per screen via `binding.takeScreenshot`.
/// The companion driver `test_driver/screenshot_driver.dart` persists the
/// PNGs to `screenshots/`.
///
///   flutter drive --driver=test_driver/screenshot_driver.dart \
///     --target=integration_test/screenshot_tour_test.dart -d <device> \
///     --dart-define=API_BASE_URL=https://api.xamltech.com
void main() {
  final binding = IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  const tenant = 'demo-restaurant';
  // Passwords come from .secrets/testing/DEMO_CREDENTIALS.txt:
  //   --dart-define=DEMO_PASSWORD='<password>'
  const email = 'demo-restaurant-manager@xamltech.com';
  const password = String.fromEnvironment('DEMO_PASSWORD');

  Future<void> pumpUntil(
    WidgetTester tester,
    Finder finder, {
    Duration timeout = const Duration(seconds: 45),
  }) async {
    final end = DateTime.now().add(timeout);
    while (DateTime.now().isBefore(end)) {
      await tester.pump(const Duration(milliseconds: 200));
      if (finder.evaluate().isNotEmpty) return;
    }
    fail('Timed out waiting for $finder');
  }

  /// Signs in by tapping the submit button. The soft keyboard is dismissed
  /// first: while it is up it owns the taps aimed at the button, and the tap
  /// is silently swallowed.
  Future<void> submit(
    WidgetTester tester,
    Session? Function() authed, {
    String buttonLabel = 'Sign in',
  }) async {
    FocusManager.instance.primaryFocus?.unfocus();
    await tester.pump(const Duration(milliseconds: 600));
    final button = find.text(buttonLabel);
    expect(button, findsOneWidget, reason: 'sign-in button $buttonLabel');
    await tester.ensureVisible(button);
    await tester.pump(const Duration(milliseconds: 400));
    await tester.tap(button);
    await tester.pump(const Duration(milliseconds: 400));
  }

  Widget harness(Widget home) => MaterialApp(
        debugShowCheckedModeBanner: false,
        locale: const Locale('en'),
        supportedLocales: AppStrings.supportedLocales,
        localizationsDelegates: const [
          AppStrings.delegate,
          GlobalMaterialLocalizations.delegate,
          GlobalWidgetsLocalizations.delegate,
          GlobalCupertinoLocalizations.delegate,
        ],
        theme: ThemeData(useMaterial3: true, colorSchemeSeed: Colors.green),
        home: home,
      );

  testWidgets('screenshot tour: login -> POS -> dashboard -> sales -> customers'
      ' -> sessions -> community -> national', (tester) async {
    final api = ApiClient();
    final store = SessionStore();
    await store.saveLanguage('en');

    // ---------- 01 login ----------
    Session? authed;
    await tester.pumpWidget(harness(LoginScreen(
      apiClient: api,
      sessionStore: store,
      onAuthenticated: (s) => authed = s,
    )));
    await pumpUntil(tester, find.text('Sign in'));
    await binding.convertFlutterSurfaceToImage();
    await tester.pump(const Duration(milliseconds: 400));
    await binding.takeScreenshot('01-login');

    // ---------- 01b login, credentials filled in ----------
    // tester.enterText is a no-op here: integration_test binds with
    // registerTestTextInput = false, so on a device there is no test text
    // input to deliver the text through. The form is filled the way the app
    // itself fills it when "remember logins" is on -- through SessionStore --
    // and the submit button is then tapped, so the login is still a real one
    // made by the real screen.
    await store.saveRememberedLogin(const RememberedLogin(
      tenantId: tenant,
      email: email,
      password: password,
      deviceName: 'posTab',
    ));
    // A placeholder in between, so the second LoginScreen is a new State and
    // its initState actually reads the store back.
    await tester.pumpWidget(harness(const SizedBox.shrink()));
    await tester.pump(const Duration(milliseconds: 300));
    await tester.pumpWidget(harness(LoginScreen(
      apiClient: api,
      sessionStore: store,
      onAuthenticated: (s) => authed = s,
    )));
    await pumpUntil(tester, find.text('Sign in'));
    await tester.pump(const Duration(milliseconds: 1500));
    expect(find.text(tenant), findsOneWidget, reason: 'prefilled tenant');
    await binding.takeScreenshot('01b-login-filled');
    await submit(tester, () => authed);

    // Wait for the real async login to complete and report the session.
    final loginEnd = DateTime.now().add(const Duration(seconds: 40));
    while (authed == null && DateTime.now().isBefore(loginEnd)) {
      await tester.pump(const Duration(milliseconds: 200));
    }
    final session = authed;
    if (session == null) {
      final texts = tester
          .widgetList<Text>(find.byType(Text))
          .map((w) => w.data ?? '')
          .where((d) => d.isNotEmpty)
          .join(' | ');
      // ignore: avoid_print
      print('LOGIN-DIAG $texts');
    }
    expect(session, isNotNull);
    tester.takeException();

    // ---------- 02 POS grid + cart ----------
    await tester.pumpWidget(harness(PosScreen(
      session: session!,
      apiClient: api,
      onSignOut: () {},
    )));
    await pumpUntil(tester, find.byType(GridView));
    await tester.pump(const Duration(milliseconds: 1500));
    tester.takeException();
    await binding.takeScreenshot('02-pos-grid');

    // tap the first product tile so the cart holds an item
    await tester.tap(find
        .descendant(of: find.byType(GridView), matching: find.byType(InkWell))
        .first);
    await tester.pump(const Duration(milliseconds: 800));
    tester.takeException();
    await binding.takeScreenshot('03-pos-cart');

    // ---------- 04 dashboard ----------
    await tester.pumpWidget(harness(
        DashboardScreen(session: session, apiClient: api)));
    await pumpUntil(tester, find.text('Dashboard'));
    await tester.pump(const Duration(milliseconds: 1200));
    await binding.takeScreenshot('04-dashboard');

    // ---------- 05 sales history ----------
    await tester.pumpWidget(harness(
        SaleHistoryScreen(session: session, apiClient: api)));
    await pumpUntil(tester, find.text('Sales History'));
    await tester.pump(const Duration(milliseconds: 1200));
    await binding.takeScreenshot('05-sales-history');

    // ---------- 06 customers ----------
    await tester.pumpWidget(
        harness(CustomersScreen(session: session, apiClient: api)));
    await pumpUntil(tester, find.text('Customers'));
    await tester.pump(const Duration(milliseconds: 1200));
    await binding.takeScreenshot('06-customers');

    // ---------- 07 sessions history ----------
    await tester.pumpWidget(harness(
        SessionHistoryScreen(session: session, apiClient: api)));
    await pumpUntil(tester, find.text('Session reports'));
    await tester.pump(const Duration(milliseconds: 1200));
    await binding.takeScreenshot('07-session-reports');

    // ---------- 08 community hub ----------
    await tester.pumpWidget(harness(
        CommunityHubScreen(session: session, apiClient: api)));
    await pumpUntil(tester, find.text('Community'));
    await tester.pump(const Duration(milliseconds: 800));
    await binding.takeScreenshot('08-community-hub');

    // ---------- 09 national community ----------
    await tester.pumpWidget(harness(
        NationalHubScreen(session: session, apiClient: api)));
    await pumpUntil(tester, find.text('Egypt national community'));
    await tester.pump(const Duration(milliseconds: 1200));
    await binding.takeScreenshot('09-national-community');
  });
}