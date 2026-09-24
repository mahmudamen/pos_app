import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/auth/login_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

const _secStorage = MethodChannel('plugins.it_nomads.com/flutter_secure_storage');

void main() {
  setUpAll(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(_secStorage, (call) async => null);
  });

  testWidgets('login screen fits narrow phone viewports without overflow',
      (tester) async {
    tester.view.physicalSize = const Size(360 * 3, 700 * 3);
    tester.view.devicePixelRatio = 3.0;
    addTearDown(tester.view.reset);

    await tester.pumpWidget(MaterialApp(
      locale: const Locale('en'),
      supportedLocales: AppStrings.supportedLocales,
      localizationsDelegates: const [AppStrings.delegate],
      home: LoginScreen(
        apiClient: ApiClient(),
        sessionStore: SessionStore(),
        onAuthenticated: (_) {},
      ),
    ));
    // Flush the fake-async init work (device id + meta bootstrap) so the form
    // settles, then ensure no layout overflow and the form stays tappable.
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 400));
    expect(tester.takeException(), isNull);
    expect(find.text('Sign in'), findsOneWidget);
    await tester.tap(find.text('Sign in'));
    await tester.pump();
    expect(tester.takeException(), isNull);
  });
}