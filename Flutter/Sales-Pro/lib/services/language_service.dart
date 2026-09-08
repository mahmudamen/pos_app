import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// Service for managing app language settings
class LanguageService extends ChangeNotifier {
  static const String _languageKey = 'language_code';
  Locale _currentLocale = const Locale('ar'); // Default to Arabic

  Locale get currentLocale => _currentLocale;
  bool get isArabic => _currentLocale.languageCode == 'ar';
  bool get isEnglish => _currentLocale.languageCode == 'en';

  /// Initialize language from saved preferences
  Future<void> init() async {
    final prefs = await SharedPreferences.getInstance();
    final savedLanguage = prefs.getString(_languageKey);

    if (savedLanguage != null) {
      _currentLocale = Locale(savedLanguage);
    } else {
      // Default to Arabic
      _currentLocale = const Locale('ar');
      await _saveLanguage('ar');
    }
    notifyListeners();
  }

  /// Change app language
  Future<void> changeLanguage(String languageCode) async {
    if (_currentLocale.languageCode == languageCode) return;

    _currentLocale = Locale(languageCode);
    await _saveLanguage(languageCode);
    notifyListeners();
  }

  /// Toggle between Arabic and English
  Future<void> toggleLanguage() async {
    final newLanguage = isArabic ? 'en' : 'ar';
    await changeLanguage(newLanguage);
  }

  /// Save language preference
  Future<void> _saveLanguage(String languageCode) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_languageKey, languageCode);
  }

  /// Get text direction based on current locale
  TextDirection get textDirection =>
      isArabic ? TextDirection.rtl : TextDirection.ltr;
}
