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

FocusNode _fieldNode(WidgetTester tester, int index) {
  final editable = find.descendant(
    of: find.byType(TextFormField).at(index),
    matching: find.byType(EditableText),
  );
  return tester.widget<EditableText>(editable).focusNode;
}

void main() {
  setUpAll(() {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
    SessionStore().saveDeviceId('test-device');
  });

  Widget wrap() {
    final client = ApiClient(client: _MetaOnlyClient());
    return MaterialApp(
      locale: const Locale('ar'),
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

  testWidgets('field order is store then email then password then device',
      (tester) async {
    useTallViewport(tester);
    await tester.pumpWidget(wrap());
    await tester.pumpAndSettle();

    expect(find.byType(TextFormField), findsNWidgets(4));

    await tester.ensureVisible(find.byType(TextFormField).at(0));
    await tester.tap(find.byType(TextFormField).at(0));
    await tester.pump();
    expect(_fieldNode(tester, 0).hasFocus, isTrue);

    await tester.testTextInput.receiveAction(TextInputAction.next);
    await tester.pump();
    expect(_fieldNode(tester, 1).hasFocus, isTrue,
        reason: 'Next must move tenant -> email, not skip to password');
    expect(_fieldNode(tester, 0).hasFocus, isFalse);

    await tester.testTextInput.receiveAction(TextInputAction.next);
    await tester.pump();
    expect(_fieldNode(tester, 2).hasFocus, isTrue,
        reason: 'Next must move email -> password');

    await tester.testTextInput.receiveAction(TextInputAction.next);
    await tester.pump();
    expect(_fieldNode(tester, 3).hasFocus, isTrue,
        reason: 'Next must move password -> device');
  });

  testWidgets('done on device unfocuses and leaves no focus on any field',
      (tester) async {
    useTallViewport(tester);
    await tester.pumpWidget(wrap());
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.byType(TextFormField).at(3));
    await tester.tap(find.byType(TextFormField).at(3));
    await tester.pump();
    expect(_fieldNode(tester, 3).hasFocus, isTrue);

    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pump();
    expect(_fieldNode(tester, 3).hasFocus, isFalse);
    expect(tester.takeException(), isNull);
  });
}