// Arabic must be the default language and must drive RTL on both web and
// Android. These assertions pin the behaviour that a Flutter widget test can
// prove deterministically (no device/browser needed): the app's own delegate
// set resolves `ar` to RTL, Arabic is listed first so it is the fallback, and
// the Arabic font family is what gets bound.
import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/fonts.dart';
import 'package:pos_go_app/l10n/strings.dart';

Widget _app({Locale? locale}) => MaterialApp(
      locale: locale,
      supportedLocales: AppStrings.supportedLocales,
      localizationsDelegates: const [
        AppStrings.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      home: Builder(
        builder: (context) => Directionality.of(context) == TextDirection.rtl
            ? const Text('rtl')
            : const Text('ltr'),
      ),
    );

Future<void> _pumpWithLocale(WidgetTester tester, Locale? locale) async {
  await tester.pumpWidget(_app(locale: locale));
  // Localizations load asynchronously; a couple of pumps is enough and avoids
  // pumpAndSettle spinning on any long-lived animation.
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 100));
}

void main() {
  testWidgets('the app default locale is Arabic and renders right-to-left',
      (tester) async {
    // The app passes an explicit locale (app.dart `_locale`), initialised from
    // this constant, so the default is Arabic regardless of the device locale.
    await _pumpWithLocale(tester, AppStrings.defaultLocale);
    expect(find.text('rtl'), findsOneWidget);
    final strings = AppStrings.of(tester.element(find.text('rtl')));
    expect(strings.isArabic, isTrue);
  });

  testWidgets('locale ar renders right-to-left', (tester) async {
    await _pumpWithLocale(tester, const Locale('ar'));
    expect(find.text('rtl'), findsOneWidget);
    expect(AppStrings.of(tester.element(find.text('rtl'))).isArabic, isTrue);
  });

  testWidgets('locale en renders left-to-right', (tester) async {
    await _pumpWithLocale(tester, const Locale('en'));
    expect(find.text('ltr'), findsOneWidget);
    expect(AppStrings.of(tester.element(find.text('ltr'))).isArabic, isFalse);
  });

  test('supported locales put Arabic first, so it is the fallback', () {
    expect(AppStrings.supportedLocales.first.languageCode, 'ar');
    expect(
      AppStrings.supportedLocales.map((l) => l.languageCode),
      containsAll(<String>['ar', 'en']),
    );
  });

  test('Arabic locale binds an Arabic font family, Latin does not', () {
    for (final mode in FontMode.values) {
      final arabic = resolveFonts(mode, true);
      final latin = resolveFonts(mode, false);
      expect(arabic, isNotEmpty, reason: 'Arabic needs its own face');
      expect(latin, isNotEmpty);
      // Latin must never fall back to an Arabic face, or English UI in an
      // Arabic session would render with Arabic metrics.
      expect(
        resolveFonts(mode, true) == resolveFonts(mode, false),
        isFalse,
        reason: '$mode should differ between scripts',
      );
    }
  });

  test('Arabic strings are actually Arabic, not English placeholders', () {
    final ar = AppStrings(const Locale('ar'));
    final en = AppStrings(const Locale('en'));
    expect(ar.signIn, 'تسجيل الدخول');
    expect(en.signIn, 'Sign in');
    // Every key must differ between the two locales, otherwise some screen
    // silently falls back to English inside an Arabic session.
    expect(ar.appTitle, isNot(en.appTitle));
    expect(ar.email, isNot(en.email));
    expect(ar.password, isNot(en.password));
    expect(ar.isArabic, isTrue);
    expect(en.isArabic, isFalse);
  });
}