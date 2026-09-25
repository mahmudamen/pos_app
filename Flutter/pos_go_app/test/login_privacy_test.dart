import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/auth/login_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

class _WrongSurfaceClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    if (request.url.path.endsWith('/meta/countries')) {
      const body =
          '{"data":[{"code":"EG","name_en":"Egypt","name_ar":"مصر","currency_code":"EGP","phone_code":"+20"}],"meta":{"request_id":"t"}}';
      return http.StreamedResponse(
        Stream.value(body.codeUnits),
        200,
        headers: {'content-type': 'application/json'},
      );
    }
    if (request.url.path.endsWith('/meta/currencies')) {
      const body =
          '{"data":[{"code":"EGP","name_en":"Egyptian Pound","name_ar":"جنيه مصري","symbol":"E£","digits_after_decimal":2}],"meta":{"request_id":"t"}}';
      return http.StreamedResponse(
        Stream.value(body.codeUnits),
        200,
        headers: {'content-type': 'application/json'},
      );
    }
    const body =
        '{"error":{"code":"wrong_surface","message":"sign in with a platform administrator account"},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      403,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _MetaOnlyClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    if (request.url.path.endsWith('/meta/countries')) {
      const body =
          '{"data":[{"code":"EG","name_en":"Egypt","name_ar":"مصر","currency_code":"EGP","phone_code":"+20"}],"meta":{"request_id":"t"}}';
      return http.StreamedResponse(
        Stream.value(body.codeUnits),
        200,
        headers: {'content-type': 'application/json'},
      );
    }
    const body =
        '{"data":[{"code":"EGP","name_en":"Egyptian Pound","name_ar":"جنيه مصري","symbol":"E£","digits_after_decimal":2}],"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

void main() {
  setUpAll(() {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
    SessionStore().saveDeviceId('test-device');
  });

  Widget wrap(Locale locale, {http.Client? client}) {
    return MaterialApp(
      locale: locale,
      supportedLocales: AppStrings.supportedLocales,
      localizationsDelegates: const [
        AppStrings.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      home: LoginScreen(
        apiClient: ApiClient(client: client ?? _MetaOnlyClient()),
        onAuthenticated: (_) {},
        sessionStore: SessionStore(),
      ),
    );
  }

  void useTallViewport(WidgetTester tester) {
    tester.view.physicalSize = const Size(1080, 2760);
    tester.view.devicePixelRatio = 3.0;
    addTearDown(tester.view.reset);
  }

  testWidgets('login shows a privacy policy link', (tester) async {
    useTallViewport(tester);
    await tester.pumpWidget(wrap(const Locale('en')));
    await tester.pumpAndSettle();

    final link = find.widgetWithText(TextButton, 'Privacy Policy');
    expect(link, findsOneWidget);
    await tester.ensureVisible(link);
    expect(tester.takeException(), isNull);
  });

  testWidgets('privacy link is localized in Arabic', (tester) async {
    useTallViewport(tester);
    await tester.pumpWidget(wrap(const Locale('ar')));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(TextButton, 'سياسة الخصوصية'), findsOneWidget);
  });

  testWidgets('a console account explains the other domain in English',
      (tester) async {
    useTallViewport(tester);
    await tester.pumpWidget(
        wrap(const Locale('en'), client: _WrongSurfaceClient()));
    await tester.pumpAndSettle();

    await tester.enterText(
        find.widgetWithText(TextFormField, 'Store ID'), 'acme');
    await tester.enterText(
        find.widgetWithText(TextFormField, 'Email'), 'admin@posgo.saas');
    await tester.enterText(
        find.widgetWithText(TextFormField, 'Password'), 'admin');
    await tester.tap(find.widgetWithText(FilledButton, 'Sign in'));
    await tester.pumpAndSettle();

    expect(
      find.textContaining('platform console account'),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('a console account explains the other domain in Arabic',
      (tester) async {
    useTallViewport(tester);
    await tester.pumpWidget(
        wrap(const Locale('ar'), client: _WrongSurfaceClient()));
    await tester.pumpAndSettle();

    await tester.enterText(
        find.widgetWithText(TextFormField, 'معرف المتجر'), 'acme');
    await tester.enterText(
        find.widgetWithText(TextFormField, 'البريد الإلكتروني'),
        'admin@posgo.saas');
    await tester.enterText(
        find.widgetWithText(TextFormField, 'كلمة المرور'), 'admin');
    await tester.tap(find.widgetWithText(FilledButton, 'تسجيل الدخول'));
    await tester.pumpAndSettle();

    expect(find.textContaining('حساب إدارة المنصة'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}