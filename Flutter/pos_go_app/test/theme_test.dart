import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/theme.dart';

void main() {
  group('ThemePreference', () {
    test('wire values round-trip', () {
      for (final preference in ThemePreference.values) {
        expect(ThemePreference.fromWire(preference.wire), preference);
      }
      expect(ThemePreference.fromWire('bogus'), ThemePreference.light);
      expect(ThemePreference.fromWire(null), ThemePreference.light);
    });

    test('maps to the matching Material ThemeMode', () {
      expect(ThemePreference.light.toThemeMode, ThemeMode.light);
      expect(ThemePreference.dark.toThemeMode, ThemeMode.dark);
      expect(ThemePreference.system.toThemeMode, ThemeMode.system);
    });
  });

  group('ThemeAccent', () {
    test('every seed is opaque and nonzero', () {
      for (final accent in ThemeAccent.values) {
        expect(accent.seed.a, 1.0);
        expect(accent.seed.toARGB32(), isNot(equals(0)));
      }
    });

    test('wire values round-trip', () {
      for (final accent in ThemeAccent.values) {
        expect(ThemeAccent.fromWire(accent.wire), accent);
      }
      expect(ThemeAccent.fromWire('bogus'), ThemeAccent.blue);
      expect(ThemeAccent.fromWire(null), ThemeAccent.blue);
    });
  });

  group('ThemeSetting', () {
    test('defaults to light + blue', () {
      const setting = ThemeSetting();
      expect(setting.preference, ThemePreference.light);
      expect(setting.accent, ThemeAccent.blue);
    });

    test('copyWith keeps untouched fields', () {
      const setting = ThemeSetting(preference: ThemePreference.system);
      expect(setting.copyWith(accent: ThemeAccent.crimson).preference,
          ThemePreference.system);
      expect(setting.copyWith(accent: ThemeAccent.crimson).accent,
          ThemeAccent.crimson);
    });

    test('equality compares preference and accent', () {
      expect(
          const ThemeSetting(preference: ThemePreference.dark, accent: ThemeAccent.purple),
          const ThemeSetting(
              preference: ThemePreference.dark, accent: ThemeAccent.purple));
      expect(
          const ThemeSetting(preference: ThemePreference.dark, accent: ThemeAccent.purple),
          isNot(const ThemeSetting(
              preference: ThemePreference.light, accent: ThemeAccent.purple)));
      expect(
          const ThemeSetting(preference: ThemePreference.dark, accent: ThemeAccent.purple),
          isNot(const ThemeSetting(
              preference: ThemePreference.dark, accent: ThemeAccent.blue)));
    });
  });

  group('buildPosTheme', () {
    test('brightness controls the scaffold background and scheme', () {
      const setting = ThemeSetting(preference: ThemePreference.dark);
      final light = buildPosTheme(setting, Brightness.light,
          fontFamily: 'Lato', fontSizeFactor: 1.0);
      final dark = buildPosTheme(setting, Brightness.dark,
          fontFamily: 'Lato', fontSizeFactor: 1.0);
      expect(light.brightness, Brightness.light);
      expect(dark.brightness, Brightness.dark);
      expect(light.scaffoldBackgroundColor,
          isNot(dark.scaffoldBackgroundColor));
    });

    test('the accent seed drives distinct generated palettes', () {
      final blue = buildPosTheme(const ThemeSetting(accent: ThemeAccent.blue),
          Brightness.light,
          fontFamily: 'Lato', fontSizeFactor: 1.0);
      for (final accent in ThemeAccent.values) {
        if (accent == ThemeAccent.blue) continue;
        final theme = buildPosTheme(ThemeSetting(accent: accent),
            Brightness.light,
            fontFamily: 'Lato', fontSizeFactor: 1.0);
        expect(theme.colorScheme.primary, isNot(blue.colorScheme.primary));
        expect(theme.colorScheme.primaryContainer,
            isNot(blue.colorScheme.primaryContainer));
      }
    });

    test('applies the font family and size factor', () {
      final theme = buildPosTheme(const ThemeSetting(), Brightness.light,
          fontFamily: 'NotoKufiArabic', fontSizeFactor: 1.15);
      expect(theme.textTheme.bodyMedium!.fontFamily, 'NotoKufiArabic');
      expect(theme.textTheme.bodyMedium!.fontSize,
          isNot(ThemeData.light().textTheme.bodyMedium!.fontSize));
    });
  });
}