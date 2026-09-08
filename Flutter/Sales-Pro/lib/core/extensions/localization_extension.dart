import 'package:flutter/material.dart';

/// Extension methods for easy access to localization
extension LocalizationExtension on BuildContext {
  /// Get localization instance
  // AppLocalizations get loc => AppLocalizations.of(this);

  /// Get current locale
  Locale get locale => Localizations.localeOf(this);

  /// Check if current language is Arabic
  bool get isArabic => locale.languageCode == 'ar';

  /// Check if current language is English
  bool get isEnglish => locale.languageCode == 'en';

  /// Get text direction based on locale
  TextDirection get direction =>
      isArabic ? TextDirection.rtl : TextDirection.ltr;
}
