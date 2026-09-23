import 'package:flutter/material.dart';

/// Light/dark/system preference persisted across sign-outs.
///
/// Mirrors Material's [ThemeMode] but keeps a stable wire value in
/// `SessionStore` (`app_theme_mode` key).
enum ThemePreference {
  light('light'),
  dark('dark'),
  system('system');

  const ThemePreference(this.wire);

  /// Wire value persisted in `SessionStore`.
  final String wire;

  static ThemePreference fromWire(String? value) {
    switch (value) {
      case 'dark':
        return ThemePreference.dark;
      case 'system':
        return ThemePreference.system;
      default:
        return ThemePreference.light;
    }
  }

  ThemeMode get toThemeMode {
    switch (this) {
      case ThemePreference.light:
        return ThemeMode.light;
      case ThemePreference.dark:
        return ThemeMode.dark;
      case ThemePreference.system:
        return ThemeMode.system;
    }
  }
}

/// Brand accent color chosen by the user. Acts as the `ColorScheme.fromSeed`
/// seed so the whole Material palette (primary/secondary/surfaces) follows.
enum ThemeAccent {
  blue('blue', Color(0xff00adee)),
  green('green', Color(0xff2e7d32)),
  orange('orange', Color(0xffff6f00)),
  purple('purple', Color(0xff7b1fa2)),
  teal('teal', Color(0xff00897b)),
  crimson('crimson', Color(0xffd32f2f));

  const ThemeAccent(this.wire, this.seed);

  /// Wire value persisted in `SessionStore` (`app_theme_accent` key).
  final String wire;

  /// Seed color driving the generated Material palette.
  final Color seed;

  static ThemeAccent fromWire(String? value) {
    for (final accent in ThemeAccent.values) {
      if (accent.wire == value) return accent;
    }
    return ThemeAccent.blue;
  }
}

/// The user's theme preference: a light/dark/system mode plus an accent seed.
@immutable
class ThemeSetting {
  const ThemeSetting({
    this.preference = ThemePreference.light,
    this.accent = ThemeAccent.blue,
  });

  final ThemePreference preference;
  final ThemeAccent accent;

  ThemeSetting copyWith({
    ThemePreference? preference,
    ThemeAccent? accent,
  }) =>
      ThemeSetting(
        preference: preference ?? this.preference,
        accent: accent ?? this.accent,
      );

  @override
  bool operator ==(Object other) =>
      other is ThemeSetting &&
      other.preference == preference &&
      other.accent == accent;

  @override
  int get hashCode => Object.hash(preference, accent);
}

/// Builds the app theme for one brightness from a [ThemeSetting].
///
/// Pure and test-friendly: everything is derived from the accent seed and the
/// brightness, with the existing font family/size scale applied on top.
ThemeData buildPosTheme(
  ThemeSetting setting,
  Brightness brightness, {
  required String fontFamily,
  required double fontSizeFactor,
}) {
  final scheme = ColorScheme.fromSeed(
    seedColor: setting.accent.seed,
    brightness: brightness,
  );
  final typography = Typography.material2021(colorScheme: scheme);
  final baseColors =
      brightness == Brightness.dark ? typography.white : typography.black;
  // The M3 color/geometry split means size-bearing geometry lives in
  // englishLike/tall/dense; merge it with the scheme-colored base so the
  // theme carries real, scalable font sizes (the runtime `Theme.of` would
  // otherwise merge no-op geometry over our textTheme).
  final textTheme = typography.englishLike
      .merge(baseColors)
      .apply(fontFamily: fontFamily, fontSizeFactor: fontSizeFactor);
  return ThemeData(
    colorScheme: scheme,
    scaffoldBackgroundColor: brightness == Brightness.dark
        ? const Color(0xff11141a)
        : const Color(0xfff4f7f6),
    useMaterial3: true,
    fontFamily: fontFamily,
    textTheme: textTheme,
  );
}