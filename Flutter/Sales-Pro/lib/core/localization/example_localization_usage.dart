import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:sales_pro/core/localization/app_localizations.dart';
import 'package:sales_pro/services/language_service.dart';

/// Example screen showing how to use the localization system
class ExampleLocalizationScreen extends StatelessWidget {
  const ExampleLocalizationScreen({super.key});

  @override
  Widget build(BuildContext context) {
    // Get localization instance
    final loc = AppLocalizations.of(context);
    final languageService = Provider.of<LanguageService>(context);

    return Scaffold(
      appBar: AppBar(
        title: Text(loc.appFullName),
        actions: [
          // Language toggle button
          IconButton(
            icon: Icon(
              languageService.isArabic
                  ? Icons.language
                  : Icons.language_outlined,
            ),
            onPressed: () => languageService.toggleLanguage(),
            tooltip: languageService.isArabic ? 'English' : 'العربية',
          ),
        ],
      ),
      body: Directionality(
        textDirection: languageService.textDirection,
        child: Padding(
          padding: const EdgeInsets.all(16.0),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Example 1: Simple translations
              Text(
                loc.welcome,
                style: Theme.of(context).textTheme.headlineMedium,
              ),
              const SizedBox(height: 8),
              Text(loc.welcomeBack),
              const SizedBox(height: 24),

              // Example 2: Translations with parameters
              Text(
                loc.customerCredit('1000'),
                style: const TextStyle(fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 8),
              Text(loc.itemStock('50')),
              const SizedBox(height: 24),

              // Example 3: Using translate method directly
              Text(loc.translate('login')),
              const SizedBox(height: 24),

              // Example 4: Language switcher
              Card(
                child: ListTile(
                  leading: const Icon(Icons.language),
                  title: Text(
                    loc.translate(
                      'language',
                      params: {
                        'current': languageService.isArabic
                            ? 'العربية'
                            : 'English',
                      },
                    ),
                  ),
                  subtitle: Text(
                    languageService.isArabic
                        ? 'اضغط لتغيير اللغة إلى English'
                        : 'Press to change language to العربية',
                  ),
                  trailing: Switch(
                    value: languageService.isEnglish,
                    onChanged: (_) => languageService.toggleLanguage(),
                  ),
                ),
              ),
              const SizedBox(height: 24),

              // Example 5: Form with localized labels
              TextField(
                decoration: InputDecoration(
                  labelText: loc.enterUsername,
                  border: const OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 16),
              TextField(
                decoration: InputDecoration(
                  labelText: loc.enterPassword,
                  border: const OutlineInputBorder(),
                ),
                obscureText: true,
              ),
              const SizedBox(height: 16),

              // Example 6: Localized buttons
              Row(
                children: [
                  Expanded(
                    child: ElevatedButton(
                      onPressed: () {},
                      child: Text(loc.login),
                    ),
                  ),
                  const SizedBox(width: 16),
                  Expanded(
                    child: OutlinedButton(
                      onPressed: () {},
                      child: Text(loc.signup),
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

/// Example of using localization in a stateless widget function
String getLocalizedTitle(BuildContext context) {
  final loc = AppLocalizations.of(context);
  return loc.appFullName;
}

/// Example of using localization with parameters
String getInvoiceTotal(BuildContext context, double total) {
  final loc = AppLocalizations.of(context);
  return loc.grandTotalValue(total.toStringAsFixed(2), 'SAR');
}

/// Example of conditional text based on language
Widget buildLanguageSpecificWidget(BuildContext context) {
  final languageService = Provider.of<LanguageService>(context, listen: false);

  if (languageService.isArabic) {
    return const Text('محتوى عربي خاص');
  } else {
    return const Text('Special English content');
  }
}
