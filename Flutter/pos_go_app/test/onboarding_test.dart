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
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/auth/register');
    expect(request.method, 'POST');
    const body =
        '{"data":{"access_token":"access","refresh_token":"refresh","expires_in":3600,"device_id":"dev-1","user":{"id":"u-1","display_name":"Owner","role":"owner","account_type":"standard"},"tenant":{"id":"tenant-new","business_type":"coffee_shop","country_code":"EG","currency_code":"EGP","default_language":"ar","plan":"trial","trial_ends_at":"2026-09-29T00:00:00Z"}}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
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

  testWidgets('first-run flow welcomes, picks a business and registers',
      (tester) async {
    useTallViewport(tester);
    Session? authenticated;
    final client = ApiClient(client: _RegisterClient());
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
        apiClient: client,
        sessionStore: SessionStore(),
        onAuthenticated: (session) => authenticated = session,
      ),
    ));
    await tester.pumpAndSettle();

    expect(find.text('Welcome to POS Go'), findsOneWidget);
    expect(find.text('Create store'), findsOneWidget);

    await tester.tap(find.text('Create store'));
    await tester.pumpAndSettle();

    expect(find.text('Coffee shop'), findsOneWidget);
    expect(find.text('Restaurant'), findsOneWidget);
    expect(find.text('Bookstore'), findsOneWidget);

    await tester.tap(find.text('Coffee shop'));
    await tester.pumpAndSettle();

    expect(find.byType(TextFormField), findsNWidgets(5));
    await tester.enterText(find.byType(TextFormField).at(0), 'Nomad Cafe');
    await tester.enterText(find.byType(TextFormField).at(1), 'Salma');
    await tester.enterText(find.byType(TextFormField).at(2), 'salma@nomad.com');
    await tester.enterText(find.byType(TextFormField).at(3), 'strong-pass-1');
    await tester.enterText(find.byType(TextFormField).at(4), 'Counter 1');
    await tester.ensureVisible(find.text('Sign up & start selling'));
    await tester.tap(find.text('Sign up & start selling'));
    await tester.pumpAndSettle();

    expect(authenticated, isNotNull);
    expect(authenticated!.tenantId, 'tenant-new');
    expect(authenticated!.businessType, 'coffee_shop');
    expect(authenticated!.plan, 'trial');
    expect(authenticated!.isTrial, isTrue);
  });

  testWidgets('wrong password length blocks submit with a helper hint',
      (tester) async {
    useTallViewport(tester);
    final client = ApiClient(client: _RegisterClient());
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
        apiClient: client,
        sessionStore: SessionStore(),
        onAuthenticated: (_) {},
      ),
    ));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Create store'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Coffee shop'));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextFormField).at(0), 'Nomad Cafe');
    await tester.enterText(find.byType(TextFormField).at(1), 'Salma');
    await tester.enterText(find.byType(TextFormField).at(2), 'salma@nomad.com');
    await tester.enterText(find.byType(TextFormField).at(3), 'short');
    await tester.enterText(find.byType(TextFormField).at(4), 'Counter 1');
    await tester.ensureVisible(find.text('Sign up & start selling'));
    await tester.tap(find.text('Sign up & start selling'));
    await tester.pumpAndSettle();

    expect(find.text('Use at least 8 characters'), findsWidgets);
  });
}