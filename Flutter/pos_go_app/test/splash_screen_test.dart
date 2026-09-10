import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/features/splash/splash_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

void main() {
  Widget wrap(Locale locale) {
    return MaterialApp(
      locale: locale,
      supportedLocales: AppStrings.supportedLocales,
      localizationsDelegates: const [
        AppStrings.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      home: const SplashScreen(),
    );
  }

  testWidgets('Arabic splash renders XAMLtech branding', (tester) async {
    await tester.pumpWidget(wrap(const Locale('ar')));
    await tester.pump(const Duration(milliseconds: 300));

    expect(find.text('نظام نقطة البيع'), findsOneWidget);
    expect(find.text('بقوة XAMLtech'), findsOneWidget);
    expect(find.text('xamltech.com'), findsWidgets);
    expect(tester.takeException(), isNull);
  });

  testWidgets('English splash renders brand wordmark', (tester) async {
    await tester.pumpWidget(wrap(const Locale('en')));
    await tester.pump(const Duration(milliseconds: 400));

    expect(find.text('Point of Sale'), findsOneWidget);
    expect(find.text('Powered by XAMLtech'), findsOneWidget);
    expect(find.text('xamltech.com'), findsWidgets);
    expect(tester.takeException(), isNull);
  });

  testWidgets('splash entrance and glow animations settle without errors',
      (tester) async {
    await tester.pumpWidget(wrap(const Locale('ar')));
    await tester.pump(const Duration(seconds: 2));
    await tester.pump(const Duration(seconds: 2));
    expect(tester.takeException(), isNull);
  });
}