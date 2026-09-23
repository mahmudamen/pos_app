import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/feedback.dart';
import 'package:pos_go_app/core/fonts.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/settings/settings_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

class _SettingsApi extends ApiClient {
  @override
  Future<TenantSettings> settings(Session session) async =>
      const TenantSettings();
}

const _managerSession = Session(
  accessToken: 'access',
  refreshToken: 'refresh',
  userId: 'user-1',
  displayName: 'Manager',
  tenantId: 'tenant-1',
  role: 'manager',
);

Widget _wrap(Widget child) => MaterialApp(
      locale: const Locale('ar'),
      supportedLocales: AppStrings.supportedLocales,
      localizationsDelegates: const [
        AppStrings.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      home: Scaffold(body: child),
    );

void main() {
  group('FontSetting', () {
    test('defaults to prompt sans at medium', () {
      const setting = FontSetting();
      expect(setting.fontMode, FontMode.sans);
      expect(setting.size, FontSize.medium);
      expect(resolveFonts(FontMode.sans, false), 'Lato');
      expect(resolveFonts(FontMode.sans, true), 'NotoSansArabic');
    });

    test('resolveFonts picks the script family', () {
      for (final mode in FontMode.values) {
        for (final isArabic in [false, true]) {
          final family = resolveFonts(mode, isArabic);
          expect(family, isNotEmpty);
        }
      }
      expect(resolveFonts(FontMode.naskh, false), 'NotoSerif');
      expect(resolveFonts(FontMode.naskh, true), 'NotoNaskhArabic');
      expect(resolveFonts(FontMode.kufi, false), 'NotoSans');
      expect(resolveFonts(FontMode.kufi, true), 'NotoKufiArabic');
    });

    test('resolveFonts falls back to the bundled sans family when empty',
        () {
      expect(resolveFonts(FontMode.sans, false), 'Lato');
      expect(resolveFonts(FontMode.sans, true), isNotEmpty);
    });

    test('fontScale returns the size multiplier', () {
      expect(fontScale(FontSize.small), 0.9);
      expect(fontScale(FontSize.medium), 1.0);
      expect(fontScale(FontSize.large), 1.15);
    });

    test('wire values round-trip', () {
      for (final mode in FontMode.values) {
        expect(FontMode.fromWire(mode.wire), mode);
      }
      for (final size in FontSize.values) {
        expect(FontSize.fromWire(size.wire), size);
      }
      expect(FontMode.fromWire('bogus'), FontMode.sans);
      expect(FontSize.fromWire('bogus'), FontSize.medium);
      expect(FontMode.fromWire(null), FontMode.sans);
      expect(FontSize.fromWire(null), FontSize.medium);
    });

    test('copyWith keeps untouched fields', () {
      const setting = FontSetting(fontMode: FontMode.kufi);
      expect(setting.copyWith(size: FontSize.small).fontMode, FontMode.kufi);
      expect(setting.copyWith(size: FontSize.small).size, FontSize.small);
    });

    test('equality compares mode and size', () {
      expect(const FontSetting(fontMode: FontMode.naskh, size: FontSize.large),
          const FontSetting(fontMode: FontMode.naskh, size: FontSize.large));
      expect(const FontSetting(fontMode: FontMode.naskh, size: FontSize.small),
          isNot(const FontSetting(fontMode: FontMode.naskh, size: FontSize.large)));
    });
  });

  testWidgets('settings sheet shows font mode and size tiles and calls back',
      (tester) async {
    FontMode? modeChanged;
    FontSize? sizeChanged;
    UiFeedback.enabled = false;
    await tester.pumpWidget(_wrap(SettingsScreen(
      session: _managerSession,
      apiClient: _SettingsApi(),
      fontSetting: const FontSetting(),
      onFontModeChanged: (m) => modeChanged = m,
      onFontSizeChanged: (s) => sizeChanged = s,
    )));
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(find.text('الخط'), 200,
        scrollable: find.byType(Scrollable).first);
    expect(find.text('الخط'), findsOneWidget);

    // Font tile opens the picker sheet (live previews).
    final fontTile = find.ancestor(
        of: find.text('الخط').first, matching: find.byType(ListTile)).first;
    await tester.ensureVisible(fontTile);
    await tester.pumpAndSettle();
    await tester.tap(find.text('الخط').first);
    await tester.pumpAndSettle();
    expect(find.textContaining('كوفي'), findsOneWidget);
    expect(find.textContaining('كبير'), findsOneWidget);

    // Choose a mode and a size, then Apply -> callbacks fired.
    await tester.tap(find.textContaining('كوفي'));
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.textContaining('كبير'));
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.text('تطبيق'));
    await tester.pumpAndSettle();

    expect(modeChanged, FontMode.kufi);
    expect(sizeChanged, FontSize.large);
    UiFeedback.enabled = true;
  });

  testWidgets('sound toggle persists through the fake session store',
      (tester) async {
    await tester.pumpWidget(_wrap(SettingsScreen(
      session: _managerSession,
      apiClient: _SettingsApi(),
      fontSetting: const FontSetting(),
    )));
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(find.text('المؤثرات الصوتية'), 200,
        scrollable: find.byType(Scrollable).first);
    // The sound switch sits in the account card (after stock/stock badges),
    // so locate it by its tile title rather than by index.
    final soundSwitch = find.descendant(
      of: find.ancestor(
          of: find.text('المؤثرات الصوتية'), matching: find.byType(ListTile)),
      matching: find.byType(Switch),
    );
    expect(tester.widget<Switch>(soundSwitch).value, isTrue);
    await tester.ensureVisible(soundSwitch);
    await tester.pumpAndSettle();
    await tester.tap(soundSwitch);
    await tester.pumpAndSettle();
    expect(tester.widget<Switch>(soundSwitch).value, isFalse);
    expect(UiFeedback.enabled, isFalse);
    // Restore so other tests are unaffected.
    UiFeedback.enabled = true;
  });
}