import 'package:flutter/material.dart';

import '../../core/feedback.dart';
import '../../core/fonts.dart';
import '../../core/theme.dart';
import '../../l10n/strings.dart';

const Duration _optionPopDuration = Duration(milliseconds: 180);

/// Modal bottom sheet letting the user pick a theme [ThemePreference]
/// (light/dark/system) and accent [ThemeAccent] seed from live previews.
///
/// Selection is animated (highlight + scale pop) and accompanied by a click
/// sound; the Apply button plays a confirm sound before popping. Used from the
/// POS settings sheet.
Future<void> showThemePickerSheet(
  BuildContext context, {
  required ThemeSetting current,
  required bool isArabic,
  required ValueChanged<ThemePreference> onThemePreferenceChanged,
  required ValueChanged<ThemeAccent> onThemeAccentChanged,
}) {
  return showModalBottomSheet<void>(
    context: context,
    showDragHandle: true,
    isScrollControlled: true,
    builder: (_) => _ThemePickerSheet(
      current: current,
      isArabic: isArabic,
      onThemePreferenceChanged: onThemePreferenceChanged,
      onThemeAccentChanged: onThemeAccentChanged,
    ),
  );
}

class _ThemePickerSheet extends StatefulWidget {
  const _ThemePickerSheet({
    required this.current,
    required this.isArabic,
    required this.onThemePreferenceChanged,
    required this.onThemeAccentChanged,
  });

  final ThemeSetting current;
  final bool isArabic;
  final ValueChanged<ThemePreference> onThemePreferenceChanged;
  final ValueChanged<ThemeAccent> onThemeAccentChanged;

  @override
  State<_ThemePickerSheet> createState() => _ThemePickerSheetState();
}

class _ThemePickerSheetState extends State<_ThemePickerSheet> {
  late ThemePreference _preference;
  late ThemeAccent _accent;

  @override
  void initState() {
    super.initState();
    _preference = widget.current.preference;
    _accent = widget.current.accent;
  }

  void _choosePreference(ThemePreference preference) {
    setState(() => _preference = preference);
    UiFeedback.select();
  }

  void _chooseAccent(ThemeAccent accent) {
    setState(() => _accent = accent);
    UiFeedback.select();
  }

  void _apply() {
    UiFeedback.confirm();
    widget.onThemePreferenceChanged(_preference);
    widget.onThemeAccentChanged(_accent);
    Navigator.of(context).pop();
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(AppStrings.of(context).saved)));
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(s.appearance, style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 12),
            _PreviewPanel(preference: _preference, accent: _accent),
            const SizedBox(height: 16),
            Text(s.themeMode, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            Row(
              children: [
                for (final preference in ThemePreference.values) ...[
                  Expanded(
                    child: _ModeOption(
                      label: _modeLabel(s, preference),
                      icon: _modeIcon(preference),
                      tooltip: preference.wire,
                      selected: preference == _preference,
                      onTap: () => _choosePreference(preference),
                    ),
                  ),
                  if (preference != ThemePreference.values.last)
                    const SizedBox(width: 8),
                ],
              ],
            ),
            const SizedBox(height: 16),
            Text(s.accentColor, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            Wrap(
              spacing: 10,
              runSpacing: 10,
              children: [
                for (final accent in ThemeAccent.values)
                  _AccentSwatch(
                    accent: accent,
                    selected: accent == _accent,
                    onTap: () => _chooseAccent(accent),
                  ),
              ],
            ),
            const SizedBox(height: 20),
            FilledButton.icon(
              onPressed: _apply,
              icon: const Icon(Icons.check),
              label: Text(s.apply),
            ),
          ],
        ),
      ),
    );
  }

  String _modeLabel(AppStrings s, ThemePreference preference) {
    switch (preference) {
      case ThemePreference.light:
        return s.themeLight;
      case ThemePreference.dark:
        return s.themeDark;
      case ThemePreference.system:
        return s.themeSystem;
    }
  }

  IconData _modeIcon(ThemePreference preference) {
    switch (preference) {
      case ThemePreference.light:
        return Icons.light_mode_outlined;
      case ThemePreference.dark:
        return Icons.dark_mode_outlined;
      case ThemePreference.system:
        return Icons.brightness_auto_outlined;
    }
  }
}

/// A live preview of how the chosen accent + brightness will look.
class _PreviewPanel extends StatelessWidget {
  const _PreviewPanel({required this.preference, required this.accent});

  final ThemePreference preference;
  final ThemeAccent accent;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final brightness = preference == ThemePreference.light
        ? Brightness.light
        : preference == ThemePreference.dark
            ? Brightness.dark
            : Theme.of(context).brightness;
    final preview = buildPosTheme(
      ThemeSetting(preference: preference, accent: accent),
      brightness,
      fontFamily: resolveFonts(FontMode.sans, s.isArabic),
      fontSizeFactor: 1.0,
    );
    return AnimatedContainer(
      key: const ValueKey('theme-preview'),
      duration: _optionPopDuration,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: preview.colorScheme.surfaceContainerHighest,
        borderRadius: BorderRadius.circular(16),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Container(
                width: 16,
                height: 16,
                decoration: BoxDecoration(
                  color: preview.colorScheme.primary,
                  shape: BoxShape.circle,
                ),
              ),
              const SizedBox(width: 8),
              Text(s.preview,
                  style: preview.textTheme.titleMedium?.copyWith(
                    color: preview.colorScheme.onSurface,
                    fontWeight: FontWeight.bold,
                  )),
            ],
          ),
          const SizedBox(height: 10),
          Text(s.themeMode,
              style: preview.textTheme.bodyMedium?.copyWith(
                color: preview.colorScheme.onSurfaceVariant,
              )),
          const SizedBox(height: 8),
          Row(
            children: [
              _MiniChip(
                color: preview.colorScheme.primaryContainer,
                onColor: preview.colorScheme.onPrimaryContainer,
                label: s.themeLight,
              ),
              const SizedBox(width: 8),
              _MiniChip(
                color: preview.colorScheme.secondaryContainer,
                onColor: preview.colorScheme.onSecondaryContainer,
                label: s.themeDark,
              ),
            ],
          ),
        ],
      ),
    );
  }
}

class _MiniChip extends StatelessWidget {
  const _MiniChip({required this.color, required this.onColor, required this.label});

  final Color color;
  final Color onColor;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
      decoration: BoxDecoration(
        color: color,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(label,
          style: Theme.of(context)
              .textTheme
              .labelSmall
              ?.copyWith(color: onColor)),
    );
  }
}

/// A tappable light/dark/system mode tile that pops in a scale animation when
/// selected and plays a click sound on tap.
class _ModeOption extends StatelessWidget {
  const _ModeOption({
    required this.label,
    required this.icon,
    required this.tooltip,
    required this.selected,
    required this.onTap,
  });

  final String label;
  final IconData icon;
  final String tooltip;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return AnimatedScale(
      scale: selected ? 1.0 : 0.96,
      duration: _optionPopDuration,
      curve: Curves.easeOutBack,
      child: AnimatedContainer(
        duration: _optionPopDuration,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: selected
              ? scheme.primaryContainer
              : scheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: selected ? scheme.primary : Colors.transparent,
            width: 2,
          ),
        ),
        child: Tooltip(
          message: tooltip,
          child: InkWell(
            borderRadius: BorderRadius.circular(12),
            onTap: () {
              UiFeedback.click();
              onTap();
            },
            child: Column(
              children: [
                Icon(icon,
                    size: 20,
                    color: selected ? scheme.primary : scheme.onSurfaceVariant),
                const SizedBox(height: 4),
                Text(label),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _AccentSwatch extends StatelessWidget {
  const _AccentSwatch({
    required this.accent,
    required this.selected,
    required this.onTap,
  });

  final ThemeAccent accent;
  final bool selected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Tooltip(
      message: accent.wire,
      child: InkResponse(
        onTap: () {
          UiFeedback.click();
          onTap();
        },
        child: AnimatedContainer(
          duration: _optionPopDuration,
          width: 40,
          height: 40,
          decoration: BoxDecoration(
            color: accent.seed,
            shape: BoxShape.circle,
            border: Border.all(
              color: selected ? scheme.onSurface : Colors.transparent,
              width: selected ? 3 : 0,
            ),
          ),
          child: selected
              ? const Icon(Icons.check, color: Colors.white, size: 20)
              : null,
        ),
      ),
    );
  }
}