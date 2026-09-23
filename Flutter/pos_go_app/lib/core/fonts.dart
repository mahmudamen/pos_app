import 'package:flutter/material.dart';

/// Font mode pairing a Latin OFL family with an Arabic OFL family.
///
/// Families are registered statically in `pubspec.yaml` under `flutter.fonts`
/// (each Regular + Bold). [resolveFonts] is the pure selector used by the
/// theme; it never returns an unresolved family because every mode has both a
/// Latin and an Arabic face bundled.
enum FontMode {
  sans('sans', 'Lato', 'NotoSansArabic'),
  naskh('naskh', 'NotoSerif', 'NotoNaskhArabic'),
  kufi('kufi', 'NotoSans', 'NotoKufiArabic');

  const FontMode(this.wire, this.latinFamily, this.arabicFamily);

  /// Wire value persisted in `SessionStore` (`app_font_family` key).
  final String wire;

  /// Registered OFL family used for Latin script.
  final String latinFamily;

  /// Registered OFL family used for Arabic script.
  final String arabicFamily;

  static FontMode fromWire(String? value) {
    switch (value) {
      case 'naskh':
        return FontMode.naskh;
      case 'kufi':
        return FontMode.kufi;
      default:
        return FontMode.sans;
    }
  }
}

/// Size scale tiers applied to the whole text theme.
enum FontSize {
  small('small', 0.9),
  medium('medium', 1.0),
  large('large', 1.15);

  const FontSize(this.wire, this.scale);

  /// Wire value persisted in `SessionStore` (`app_font_size` key).
  final String wire;

  /// Multiplier applied to the base [TextTheme] sizes.
  final double scale;

  static FontSize fromWire(String? value) {
    switch (value) {
      case 'small':
        return FontSize.small;
      case 'large':
        return FontSize.large;
      default:
        return FontSize.medium;
    }
  }
}

/// Pure selector: the concrete registered family to bind for the current
/// script. Arabic always gets an Arabic face (its variable font also covers
/// Latin glyphs); Latin gets the mode's Latin family with a fallback to the
/// bundled sans family if a face is ever missing.
String resolveFonts(FontMode mode, bool isArabic) {
  final family = isArabic ? mode.arabicFamily : mode.latinFamily;
  if (family.isEmpty) return FontMode.sans.latinFamily;
  return family;
}

/// Pure selector: numeric multiplier for a size tier.
double fontScale(FontSize size) => size.scale;

/// The user's font preference: a mode (family pair) plus a size scale.
@immutable
class FontSetting {
  const FontSetting({
    this.fontMode = FontMode.sans,
    this.size = FontSize.medium,
  });

  final FontMode fontMode;
  final FontSize size;

  FontSetting copyWith({FontMode? fontMode, FontSize? size}) {
    return FontSetting(
      fontMode: fontMode ?? this.fontMode,
      size: size ?? this.size,
    );
  }

  @override
  bool operator ==(Object other) =>
      other is FontSetting &&
      other.fontMode == fontMode &&
      other.size == size;

  @override
  int get hashCode => Object.hash(fontMode, size);
}