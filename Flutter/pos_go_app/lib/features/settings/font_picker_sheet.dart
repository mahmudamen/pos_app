import 'package:flutter/material.dart';

import '../../core/feedback.dart';
import '../../core/fonts.dart';
import '../../l10n/strings.dart';

const Duration _optionPopDuration = Duration(milliseconds: 180);

/// Modal bottom sheet letting the user pick a font [FontMode] and [FontSize]
/// from live previews.
///
/// Selection is animated (highlight + scale pop) and accompanied by a click
/// sound; the Apply button plays a confirm sound before popping. Used from
/// both the login screen and the POS settings sheet.
Future<void> showFontPickerSheet(
  BuildContext context, {
  required FontSetting current,
  required ValueChanged<FontMode> onFontModeChanged,
  required ValueChanged<FontSize> onFontSizeChanged,
}) {
  return showModalBottomSheet<void>(
    context: context,
    showDragHandle: true,
    isScrollControlled: true,
    builder: (_) => _FontPickerSheet(
      current: current,
      onFontModeChanged: onFontModeChanged,
      onFontSizeChanged: onFontSizeChanged,
    ),
  );
}

class _FontPickerSheet extends StatefulWidget {
  const _FontPickerSheet({
    required this.current,
    required this.onFontModeChanged,
    required this.onFontSizeChanged,
  });

  final FontSetting current;
  final ValueChanged<FontMode> onFontModeChanged;
  final ValueChanged<FontSize> onFontSizeChanged;

  @override
  State<_FontPickerSheet> createState() => _FontPickerSheetState();
}

class _FontPickerSheetState extends State<_FontPickerSheet> {
  late FontMode _mode;
  late FontSize _size;

  @override
  void initState() {
    super.initState();
    _mode = widget.current.fontMode;
    _size = widget.current.size;
  }

  void _chooseMode(FontMode mode) {
    setState(() => _mode = mode);
    UiFeedback.select();
  }

  void _chooseSize(FontSize size) {
    setState(() => _size = size);
    UiFeedback.select();
  }

  void _apply() {
    UiFeedback.confirm();
    widget.onFontModeChanged(_mode);
    widget.onFontSizeChanged(_size);
    Navigator.of(context).pop();
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(AppStrings.of(context).saved)));
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final isArabic = s.isArabic;
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(s.font, style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 12),
            AnimatedContainer(
              duration: _optionPopDuration,
              child: Column(
                children: [
                  for (final mode in FontMode.values)
                    _FontOption(
                      label: _modeLabel(s, mode),
                      family: resolveFonts(mode, isArabic),
                      selected: mode == _mode,
                      onTap: () => _chooseMode(mode),
                    ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            Text(s.fontSize, style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            Row(
              children: [
                for (final size in FontSize.values) ...[
                  Expanded(
                    child: _FontOption(
                      label: _sizeLabel(s, size),
                      family: resolveFonts(_mode, isArabic),
                      scale: fontScale(size),
                      selected: size == _size,
                      onTap: () => _chooseSize(size),
                    ),
                  ),
                  if (size != FontSize.values.last) const SizedBox(width: 8),
                ],
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

  String _modeLabel(AppStrings s, FontMode mode) {
    switch (mode) {
      case FontMode.sans:
        return s.fontSans;
      case FontMode.naskh:
        return s.fontNaskh;
      case FontMode.kufi:
        return s.fontKufi;
    }
  }

  String _sizeLabel(AppStrings s, FontSize size) {
    switch (size) {
      case FontSize.small:
        return s.fontSizeSmall;
      case FontSize.medium:
        return s.fontSizeMedium;
      case FontSize.large:
        return s.fontSizeLarge;
    }
  }
}

/// A tappable font preview that pops in a scale animation when selected and
/// plays a click sound on tap.
class _FontOption extends StatefulWidget {
  const _FontOption({
    required this.label,
    required this.family,
    required this.selected,
    required this.onTap,
    this.scale = 1.0,
  });

  final String label;
  final String family;
  final bool selected;
  final VoidCallback onTap;
  final double scale;

  @override
  State<_FontOption> createState() => _FontOptionState();
}

class _FontOptionState extends State<_FontOption> {
  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return AnimatedScale(
      scale: widget.selected ? 1.0 : 0.96,
      duration: _optionPopDuration,
      curve: Curves.easeOutBack,
      child: AnimatedContainer(
        duration: _optionPopDuration,
        margin: const EdgeInsets.symmetric(vertical: 4),
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        decoration: BoxDecoration(
          color: widget.selected
              ? scheme.primaryContainer
              : scheme.surfaceContainerHighest,
          borderRadius: BorderRadius.circular(12),
          border: Border.all(
            color: widget.selected ? scheme.primary : Colors.transparent,
            width: 2,
          ),
        ),
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: () {
            UiFeedback.click();
            widget.onTap();
          },
          child: Row(
            children: [
              Expanded(
                child: Text(
                  '${widget.label}  —  ${widget.family}',
                  style: TextStyle(
                    fontFamily: widget.family,
                    fontSize: 15 * widget.scale,
                  ),
                  overflow: TextOverflow.ellipsis,
                ),
              ),
              const SizedBox(width: 8),
              AnimatedSwitcher(
                duration: _optionPopDuration,
                child: widget.selected
                    ? const Icon(Icons.check_circle,
                        key: ValueKey('selected'), size: 22)
                    : const SizedBox(
                        key: ValueKey('unselected'),
                        width: 22,
                        height: 22,
                      ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}