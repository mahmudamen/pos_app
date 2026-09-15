import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/onboarding/onboarding_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

class _RegisterClient extends http.BaseClient {
  Map<String, dynamic>? lastBody;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/auth/register');
    expect(request.method, 'POST');
    final req = request as http.Request;
    lastBody = jsonDecode(req.body) as Map<String, dynamic>;
    const body =
        '{"data":{"access_token":"access","refresh_token":"refresh","expires_in":3600,"device_id":"dev-1","user":{"id":"u-1","display_name":"Owner","role":"owner","account_type":"standard"},"tenant":{"id":"tenant-new","business_type":"coffee_shop","country_code":"EG","currency_code":"EGP","default_language":"ar","plan":"trial","trial_ends_at":"2026-09-29T00:00:00Z"}}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

Future<void> pumpOnboarding(
  WidgetTester tester,
  http.BaseClient client, {
  ValueChanged<Session>? onAuthenticated,
}) async {
  await tester.pumpWidget(MaterialApp(
    locale: const Locale('en'),
    supportedLocales: AppStrings.supportedLocales,
    localizationsDelegates: const [
      AppStrings.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    home: OnboardingScreen(
      apiClient: ApiClient(client: client),
      sessionStore: SessionStore(),
      onAuthenticated: onAuthenticated ?? (_) {},
    ),
  ));
  await tester.pumpAndSettle();
}

void main() {
  setUpAll(() {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
    SessionStore().saveDeviceId('test-device');
  });

  void useTallViewport(WidgetTester tester) {
    tester.view.physicalSize = const Size(1080, 2760);
    tester.view.devicePixelRatio = 3.0;
    addTearDown(tester.view.reset);
  }

  testWidgets('first-run flow welcomes, picks business + interests, choose a '
      'plan and registers', (tester) async {
    useTallViewport(tester);
    Session? authenticated;
    final client = _RegisterClient();
    await pumpOnboarding(tester, client,
        onAuthenticated: (session) => authenticated = session);

    expect(find.text('Welcome to POS Go'), findsOneWidget);
    expect(find.text('Create store'), findsOneWidget);
    expect(find.text('Try every feature free for 15 days'), findsOneWidget);
    expect(find.text('Up to 2 users'), findsOneWidget);

    await tester.tap(find.text('Create store'));
    await tester.pumpAndSettle();

    expect(find.text('Coffee shop'), findsOneWidget);
    expect(find.text('Restaurant'), findsOneWidget);
    expect(find.text('Bookstore'), findsOneWidget);

    await tester.tap(find.text('Coffee shop'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Continue'));
    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();

    // Interests step: multi-select feature chips.
    expect(find.text('What matters most for your store?'), findsOneWidget);
    expect(find.text('Inventory control'), findsOneWidget);
    expect(find.text('Tables & split bills'), findsOneWidget);
    await tester.tap(
        find.widgetWithText(CheckboxListTile, 'Inventory control'));
    await tester.tap(
        find.widgetWithText(CheckboxListTile, 'Tables & split bills'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Continue'));
    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();

    // Plans step: free trial highlighted, paid plans offered with prices.
    expect(find.text('Choose your plan'), findsOneWidget);
    expect(find.text('Free trial'), findsWidgets);
    expect(find.text('Standard'), findsOneWidget);
    expect(find.text('Premium'), findsOneWidget);
    expect(find.text('Enterprise'), findsOneWidget);
    expect(find.text('E£499.00/month'), findsOneWidget);
    expect(find.text('E£999.00/month'), findsOneWidget);
    expect(find.text('E£2499.00/month'), findsOneWidget);
    expect(find.text('Free trial limits'), findsOneWidget);
    expect(find.text('Up to 2 users'), findsOneWidget);

    await tester.ensureVisible(find.text('Start free 15-day trial'));
    await tester.tap(find.text('Start free 15-day trial'));
    await tester.pumpAndSettle();

    expect(find.byType(TextFormField), findsNWidgets(5));
    await tester.enterText(find.byType(TextFormField).at(0), 'Nomad Cafe');
    await tester.enterText(find.byType(TextFormField).at(1), 'Salma');
    await tester.enterText(find.byType(TextFormField).at(2), 'salma@nomad.com');
    await tester.enterText(find.byType(TextFormField).at(3), 'strong-pass-1');
    await tester.enterText(find.byType(TextFormField).at(4), 'Counter 1');
    await tester.ensureVisible(find.text('Sign up & start selling'));
    await tester.ensureVisible(find.text('Sign up & start selling'));
    await tester.tap(find.text('Sign up & start selling'));
    await tester.pumpAndSettle();

    expect(authenticated, isNotNull);
    expect(authenticated!.tenantId, 'tenant-new');
    expect(authenticated!.businessType, 'coffee_shop');
    expect(authenticated!.plan, 'trial');
    expect(authenticated!.isTrial, isTrue);

    // The chosen interests and plan ride along in the register payload.
    expect(client.lastBody!['interests'],
        containsAllInOrder(['inventory', 'tables']));
    expect(client.lastBody!['plan'], 'trial');
  });

  testWidgets('selecting a paid plan still registers on a free trial and the '
      'payload carries the plan', (tester) async {
    useTallViewport(tester);
    Session? authenticated;
    final client = _RegisterClient();
    await pumpOnboarding(tester, client,
        onAuthenticated: (session) => authenticated = session);

    await tester.tap(find.text('Create store'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Grocery'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Continue'));
    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Continue'));
    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Premium'));
    await tester.tap(find.text('Premium'));
    await tester.pumpAndSettle();
    expect(find.text('After the trial you can upgrade to the Premium plan'),
        findsOneWidget);

    await tester.ensureVisible(find.text('Start free 15-day trial'));
    await tester.tap(find.text('Start free 15-day trial'));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextFormField).at(0), 'Grocery Mate');
    await tester.enterText(find.byType(TextFormField).at(1), 'Ali');
    await tester.enterText(find.byType(TextFormField).at(2), 'ali@grocery.com');
    await tester.enterText(find.byType(TextFormField).at(3), 'strong-pass-1');
    await tester.enterText(find.byType(TextFormField).at(4), 'Counter 1');
    await tester.ensureVisible(find.text('Sign up & start selling'));
    await tester.tap(find.text('Sign up & start selling'));
    await tester.pumpAndSettle();

    expect(authenticated, isNotNull);
    expect(client.lastBody!['plan'], 'premium');
    expect(client.lastBody!['interests'], isEmpty);
  });

  testWidgets('wrong password length blocks submit with a helper hint',
      (tester) async {
    useTallViewport(tester);
    await pumpOnboarding(tester, _RegisterClient());

    await tester.tap(find.text('Create store'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Coffee shop'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Continue'));
    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Continue'));
    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Start free 15-day trial'));
    await tester.tap(find.text('Start free 15-day trial'));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextFormField).at(0), 'Nomad Cafe');
    await tester.enterText(find.byType(TextFormField).at(1), 'Salma');
    await tester.enterText(find.byType(TextFormField).at(2), 'salma@nomad.com');
    await tester.enterText(find.byType(TextFormField).at(3), 'short');
    await tester.enterText(find.byType(TextFormField).at(4), 'Counter 1');
    await tester.ensureVisible(find.text('Sign up & start selling'));
    await tester.ensureVisible(find.text('Sign up & start selling'));
    await tester.tap(find.text('Sign up & start selling'));
    await tester.pumpAndSettle();

    expect(find.text('Use at least 8 characters'), findsWidgets);
  });

  testWidgets('back arrow returns from the plans step to interests',
      (tester) async {
    useTallViewport(tester);
    await pumpOnboarding(tester, _RegisterClient());

    await tester.tap(find.text('Create store'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Coffee shop'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Continue'));
    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Continue'));
    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();

    expect(find.text('Choose your plan'), findsOneWidget);
    await tester.tap(find.byIcon(Icons.arrow_back));
    await tester.pumpAndSettle();
    expect(find.text('What matters most for your store?'), findsOneWidget);
  });
}