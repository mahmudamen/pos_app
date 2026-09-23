import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/feedback.dart';
import 'package:pos_go_app/core/fonts.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/core/theme.dart';
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
  testWidgets(
      'settings sheet shows the appearance tile and opens the theme picker',
      (tester) async {
    ThemePreference? preferenceChanged;
    ThemeAccent? accentChanged;
    UiFeedback.enabled = false;
    await tester.pumpWidget(_wrap(SettingsScreen(
      session: _managerSession,
      apiClient: _SettingsApi(),
      fontSetting: const FontSetting(),
      themeSetting: const ThemeSetting(),
      onThemePreferenceChanged: (p) => preferenceChanged = p,
      onThemeAccentChanged: (a) => accentChanged = a,
    )));
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(find.text('المظهر'), 200,
        scrollable: find.byType(Scrollable).first);
    expect(find.text('المظهر'), findsOneWidget);

    // The appearance tile opens the theme picker sheet with a preview.
    final appearanceTile = find.ancestor(
        of: find.text('المظهر').first, matching: find.byType(ListTile)).first;
    await tester.ensureVisible(appearanceTile);
    await tester.pumpAndSettle();
    await tester.tap(appearanceTile);
    await tester.pumpAndSettle();
    expect(find.text('معاينة'), findsOneWidget);

    // Choose dark mode + teal accent, then Apply -> callbacks fired.
    await tester.tap(find.byTooltip('dark'));
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.byTooltip('teal'));
    await tester.pump(const Duration(milliseconds: 300));
    await tester.tap(find.text('تطبيق'));
    await tester.pumpAndSettle();

    expect(preferenceChanged, ThemePreference.dark);
    expect(accentChanged, ThemeAccent.teal);
    UiFeedback.enabled = true;
  });

  testWidgets('theme picker preview reflects the chosen brightness',
      (tester) async {
    UiFeedback.enabled = false;
    await tester.pumpWidget(_wrap(SettingsScreen(
      session: _managerSession,
      apiClient: _SettingsApi(),
      fontSetting: const FontSetting(),
      themeSetting: const ThemeSetting(),
    )));
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(find.text('المظهر'), 200,
        scrollable: find.byType(Scrollable).first);
    final appearanceTile = find.ancestor(
        of: find.text('المظهر').first, matching: find.byType(ListTile)).first;
    await tester.ensureVisible(appearanceTile);
    await tester.pumpAndSettle();
    await tester.tap(appearanceTile);
    await tester.pumpAndSettle();

    // Read the preview panel's container color while light is selected.
    final previewFinder = find.byKey(const ValueKey('theme-preview'));
    final lightPreview = tester.widget<AnimatedContainer>(previewFinder);
    final lightSurface = (lightPreview.decoration as BoxDecoration).color;

    // Pick dark and re-read: the surface must change.
    await tester.tap(find.byTooltip('dark'));
    await tester.pump(const Duration(milliseconds: 300));
    final darkPreview = tester.widget<AnimatedContainer>(previewFinder);
    final darkSurface = (darkPreview.decoration as BoxDecoration).color;

    expect(darkSurface, isNot(lightSurface));
    UiFeedback.enabled = true;
  });
}