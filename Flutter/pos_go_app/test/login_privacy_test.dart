import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/auth/login_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

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

  Widget wrap(Locale locale) {
    final client = ApiClient(client: _MetaOnlyClient());
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
        apiClient: client,
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
}