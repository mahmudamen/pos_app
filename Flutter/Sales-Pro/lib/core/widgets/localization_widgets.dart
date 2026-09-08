import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import 'package:sales_pro/services/language_service.dart';

/// Widget builder that automatically handles text direction
class LocalizedBuilder extends StatelessWidget {
  final Widget Function(BuildContext context, TextDirection direction) builder;

  const LocalizedBuilder({super.key, required this.builder});

  @override
  Widget build(BuildContext context) {
    final languageService = Provider.of<LanguageService>(context);
    return Directionality(
      textDirection: languageService.textDirection,
      child: builder(context, languageService.textDirection),
    );
  }
}

/// Language switcher button widget
class LanguageSwitcherButton extends StatelessWidget {
  final IconData? icon;
  final String? tooltip;
  final ButtonStyle? style;

  const LanguageSwitcherButton({
    super.key,
    this.icon,
    this.tooltip,
    this.style,
  });

  @override
  Widget build(BuildContext context) {
    final languageService = Provider.of<LanguageService>(context);

    return IconButton(
      icon: Icon(icon ?? Icons.language),
      tooltip: tooltip ?? (languageService.isArabic ? 'English' : 'العربية'),
      onPressed: () => languageService.toggleLanguage(),
      style: style,
    );
  }
}

/// Language switcher as a switch widget
class LanguageSwitcher extends StatelessWidget {
  final String? arabicLabel;
  final String? englishLabel;

  const LanguageSwitcher({super.key, this.arabicLabel, this.englishLabel});

  @override
  Widget build(BuildContext context) {
    final languageService = Provider.of<LanguageService>(context);

    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(arabicLabel ?? 'عربي'),
        Switch(
          value: languageService.isEnglish,
          onChanged: (_) => languageService.toggleLanguage(),
        ),
        Text(englishLabel ?? 'English'),
      ],
    );
  }
}

/// Language selector dropdown
class LanguageDropdown extends StatelessWidget {
  final String? arabicLabel;
  final String? englishLabel;
  final EdgeInsets? padding;

  const LanguageDropdown({
    super.key,
    this.arabicLabel,
    this.englishLabel,
    this.padding,
  });

  @override
  Widget build(BuildContext context) {
    final languageService = Provider.of<LanguageService>(context);

    return Padding(
      padding: padding ?? EdgeInsets.zero,
      child: DropdownButton<String>(
        value: languageService.currentLocale.languageCode,
        items: [
          DropdownMenuItem(
            value: 'ar',
            child: Row(
              children: [
                const Text('🇸🇦', style: TextStyle(fontSize: 20)),
                const SizedBox(width: 8),
                Text(arabicLabel ?? 'العربية'),
              ],
            ),
          ),
          DropdownMenuItem(
            value: 'en',
            child: Row(
              children: [
                const Text('🇬🇧', style: TextStyle(fontSize: 20)),
                const SizedBox(width: 8),
                Text(englishLabel ?? 'English'),
              ],
            ),
          ),
        ],
        onChanged: (value) {
          if (value != null) {
            languageService.changeLanguage(value);
          }
        },
      ),
    );
  }
}

/// List tile for language selection in settings
class LanguageSettingsTile extends StatelessWidget {
  final String? title;
  final IconData? icon;

  const LanguageSettingsTile({super.key, this.title, this.icon});

  @override
  Widget build(BuildContext context) {
    final languageService = Provider.of<LanguageService>(context);

    return ListTile(
      leading: Icon(icon ?? Icons.language),
      title: Text(title ?? 'اللغة / Language'),
      subtitle: Text(languageService.isArabic ? 'العربية' : 'English'),
      trailing: LanguageSwitcher(arabicLabel: 'ع', englishLabel: 'EN'),
    );
  }
}

/// Dialog for language selection
class LanguageSelectionDialog extends StatelessWidget {
  final String? title;

  const LanguageSelectionDialog({super.key, this.title});

  @override
  Widget build(BuildContext context) {
    final languageService = Provider.of<LanguageService>(context);

    return AlertDialog(
      title: Text(title ?? 'اختر اللغة / Select Language'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          RadioListTile<String>(
            title: const Row(
              children: [
                Text('🇸🇦', style: TextStyle(fontSize: 24)),
                SizedBox(width: 12),
                Text('العربية'),
              ],
            ),
            value: 'ar',
            groupValue: languageService.currentLocale.languageCode,
            onChanged: (value) {
              if (value != null) {
                languageService.changeLanguage(value);
                Navigator.pop(context);
              }
            },
          ),
          RadioListTile<String>(
            title: const Row(
              children: [
                Text('🇬🇧', style: TextStyle(fontSize: 24)),
                SizedBox(width: 12),
                Text('English'),
              ],
            ),
            value: 'en',
            groupValue: languageService.currentLocale.languageCode,
            onChanged: (value) {
              if (value != null) {
                languageService.changeLanguage(value);
                Navigator.pop(context);
              }
            },
          ),
        ],
      ),
    );
  }

  /// Show the language selection dialog
  static Future<void> show(BuildContext context, {String? title}) {
    return showDialog(
      context: context,
      builder: (context) => LanguageSelectionDialog(title: title),
    );
  }
}
