import 'package:flutter_test/flutter_test.dart';
import 'package:flutter/material.dart';
import 'package:sales_pro/core/localization/app_localizations.dart';

void main() {
  group('AppLocalizations Tests', () {
    test('English locale should be supported', () {
      const delegate = AppLocalizations.delegate;
      expect(delegate.isSupported(const Locale('en')), true);
    });

    test('Arabic locale should be supported', () {
      const delegate = AppLocalizations.delegate;
      expect(delegate.isSupported(const Locale('ar')), true);
    });

    test('Unsupported locale should return false', () {
      const delegate = AppLocalizations.delegate;
      expect(delegate.isSupported(const Locale('fr')), false);
    });

    test('Load English localizations', () async {
      final localizations = AppLocalizations(const Locale('en'));
      final loaded = await localizations.load();

      expect(loaded, true);
      expect(localizations.locale.languageCode, 'en');
    });

    test('Load Arabic localizations', () async {
      final localizations = AppLocalizations(const Locale('ar'));
      final loaded = await localizations.load();

      expect(loaded, true);
      expect(localizations.locale.languageCode, 'ar');
    });

    test('Translate method should return correct text', () async {
      final localizations = AppLocalizations(const Locale('en'));
      await localizations.load();

      expect(localizations.translate('login'), isNotEmpty);
      expect(localizations.login, isNotEmpty);
    });

    test('Parameter replacement should work', () async {
      final localizations = AppLocalizations(const Locale('en'));
      await localizations.load();

      final result = localizations.translate(
        'customerCredit',
        params: {'credit': '1000'},
      );

      expect(result, contains('1000'));
    });

    test('Multiple parameters should work', () async {
      final localizations = AppLocalizations(const Locale('en'));
      await localizations.load();

      final result = localizations.grandTotalValue('500', 'SAR');

      expect(result, contains('500'));
      expect(result, contains('SAR'));
    });

    test('Missing key should return the key itself', () async {
      final localizations = AppLocalizations(const Locale('en'));
      await localizations.load();

      const nonExistentKey = 'nonExistentKey123';
      expect(localizations.translate(nonExistentKey), nonExistentKey);
    });
  });
}
