import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../core/localization/app_localizations.dart';
import '../services/language_service.dart';
import '../core/widgets/localization_widgets.dart';

/// Test screen to verify localization system is working
class LocalizationTestScreen extends StatelessWidget {
  const LocalizationTestScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final loc = AppLocalizations.of(context);
    final languageService = Provider.of<LanguageService>(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(loc.appFullName),
        actions: const [LanguageSwitcherButton()],
      ),
      body: LocalizedBuilder(
        builder: (context, direction) {
          return SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                // Language Info Card
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Row(
                          children: [
                            const Icon(Icons.info, color: Colors.blue),
                            const SizedBox(width: 8),
                            Text(
                              'Current Language Info',
                              style: Theme.of(context).textTheme.titleLarge,
                            ),
                          ],
                        ),
                        const Divider(),
                        _InfoRow(
                          'Language Code:',
                          languageService.currentLocale.languageCode
                              .toUpperCase(),
                        ),
                        _InfoRow('Text Direction:', direction.name),
                        _InfoRow(
                          'Is Arabic:',
                          languageService.isArabic.toString(),
                        ),
                        _InfoRow(
                          'Is English:',
                          languageService.isEnglish.toString(),
                        ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 16),

                // Simple Translations
                _SectionTitle('1. Simple Translations'),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _TranslationRow('welcome', loc.welcome),
                        _TranslationRow('login', loc.login),
                        _TranslationRow('signup', loc.signup),
                        _TranslationRow('logout', loc.logout),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 16),

                // Translations with Parameters
                _SectionTitle('2. Translations with Parameters'),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _TranslationRow(
                          'customerCredit',
                          loc.customerCredit('1000'),
                        ),
                        _TranslationRow('itemStock', loc.itemStock('50')),
                        _TranslationRow(
                          'grandTotalValue',
                          loc.grandTotalValue('500', 'SAR'),
                        ),
                        _TranslationRow(
                          'invoiceTotal',
                          loc.invoiceTotal('750'),
                        ),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 16),

                // Language Switchers
                _SectionTitle('3. Language Switcher Widgets'),
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(16),
                    child: Column(
                      children: [
                        const LanguageSettingsTile(),
                        const Divider(),
                        Row(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: const [LanguageSwitcher()],
                        ),
                        const Divider(),
                        const LanguageDropdown(),
                      ],
                    ),
                  ),
                ),
                const SizedBox(height: 16),

                // App Info
                _SectionTitle('4. App Information'),
                Card(
                  child: ListTile(
                    leading: const Icon(Icons.info_outline),
                    title: Text(loc.appFullName),
                    subtitle: Text(loc.salesProVersion),
                  ),
                ),
                const SizedBox(height: 16),

                // Test Dialog Button
                Center(
                  child: ElevatedButton.icon(
                    onPressed: () => LanguageSelectionDialog.show(context),
                    icon: const Icon(Icons.language),
                    label: const Text('Show Language Selection Dialog'),
                  ),
                ),
                const SizedBox(height: 32),
              ],
            ),
          );
        },
      ),
    );
  }
}

class _SectionTitle extends StatelessWidget {
  final String title;

  const _SectionTitle(this.title);

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8),
      child: Text(
        title,
        style: Theme.of(context).textTheme.titleMedium?.copyWith(
          fontWeight: FontWeight.bold,
          color: Colors.blue,
        ),
      ),
    );
  }
}

class _InfoRow extends StatelessWidget {
  final String label;
  final String value;

  const _InfoRow(this.label, this.value);

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        children: [
          SizedBox(
            width: 120,
            child: Text(
              label,
              style: const TextStyle(fontWeight: FontWeight.w500),
            ),
          ),
          Expanded(
            child: Text(value, style: TextStyle(color: Colors.grey[700])),
          ),
        ],
      ),
    );
  }
}

class _TranslationRow extends StatelessWidget {
  final String translationKey;
  final String translation;

  const _TranslationRow(this.translationKey, this.translation);

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 150,
            child: Text(
              translationKey,
              style: TextStyle(
                fontFamily: 'monospace',
                fontSize: 12,
                color: Colors.grey[600],
              ),
            ),
          ),
          const Icon(Icons.arrow_forward, size: 16),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              translation,
              style: const TextStyle(fontWeight: FontWeight.w500),
            ),
          ),
        ],
      ),
    );
  }
}
