import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/feedback.dart';
import '../../core/fonts.dart';
import '../../core/payments.dart';
import '../../core/session_store.dart';
import '../../core/theme.dart';
import '../../l10n/strings.dart';
import '../saas/control_panel_screen.dart';
import 'font_picker_sheet.dart';
import 'theme_picker_sheet.dart';

class SettingsScreen extends StatefulWidget {
  const SettingsScreen({
    super.key,
    required this.session,
    required this.apiClient,
    this.sessionStore,
    this.fontSetting = const FontSetting(),
    this.onFontModeChanged,
    this.onFontSizeChanged,
    this.onLanguageChanged,
    this.themeSetting = const ThemeSetting(),
    this.onThemePreferenceChanged,
    this.onThemeAccentChanged,
    this.onSignOut,
  });

  final Session session;
  final ApiClient apiClient;
  final SessionStore? sessionStore;
  final FontSetting fontSetting;
  final ValueChanged<FontMode>? onFontModeChanged;
  final ValueChanged<FontSize>? onFontSizeChanged;
  final ValueChanged<String>? onLanguageChanged;
  final ThemeSetting themeSetting;
  final ValueChanged<ThemePreference>? onThemePreferenceChanged;
  final ValueChanged<ThemeAccent>? onThemeAccentChanged;
  final VoidCallback? onSignOut;

  @override
  State<SettingsScreen> createState() => _SettingsScreenState();
}

class _SettingsScreenState extends State<SettingsScreen> {
  TenantSettings? _settings;
  late FontSetting _font;
  late ThemeSetting _theme;
  bool _soundEnabled = true;
  bool _loading = true;
  bool _saving = false;
  String? _error;

  bool get _canEdit => widget.session.canManageSettings;

  void _openFontPicker() {
    UiFeedback.click();
    showFontPickerSheet(
      context,
      current: _font,
      onFontModeChanged: (mode) {
        setState(() => _font = _font.copyWith(fontMode: mode));
        widget.onFontModeChanged?.call(mode);
      },
      onFontSizeChanged: (size) {
        setState(() => _font = _font.copyWith(size: size));
        widget.onFontSizeChanged?.call(size);
      },
    ).then((_) {
      if (mounted) setState(() {});
    });
  }

  Future<void> _setSoundEnabled(bool enabled) async {
    setState(() => _soundEnabled = enabled);
    UiFeedback.enabled = enabled;
    await widget.sessionStore?.saveSoundEnabled(enabled);
    if (enabled) UiFeedback.confirm();
  }

  void _openThemePicker() {
    UiFeedback.click();
    showThemePickerSheet(
      context,
      current: _theme,
      isArabic: AppStrings.of(context).isArabic,
      onThemePreferenceChanged: (preference) {
        setState(() => _theme = _theme.copyWith(preference: preference));
        widget.onThemePreferenceChanged?.call(preference);
      },
      onThemeAccentChanged: (accent) {
        setState(() => _theme = _theme.copyWith(accent: accent));
        widget.onThemeAccentChanged?.call(accent);
      },
    ).then((_) {
      if (mounted) setState(() {});
    });
  }

  @override
  void initState() {
    super.initState();
    _font = widget.fontSetting;
    _theme = widget.themeSetting;
    _loadSound();
    _load();
  }

  Future<void> _loadSound() async {
    final enabled = await widget.sessionStore?.readSoundEnabled();
    if (enabled == null || !mounted) return;
    setState(() {
      _soundEnabled = enabled;
      UiFeedback.enabled = enabled;
    });
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final settings = await widget.apiClient.settings(widget.session);
      if (!mounted) return;
      setState(() {
        _settings = settings;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  Future<void> _save() async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _saving = true);
    try {
      final updated = await widget.apiClient
          .updateSettings(widget.session, _settings!);
      if (!mounted) return;
      setState(() => _settings = updated);
      messenger.showSnackBar(SnackBar(content: Text(s.saved)));
    } catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  String _roleLabel(AppStrings s) {
    switch (widget.session.role) {
      case 'owner':
        return s.roleOwner;
      case 'manager':
        return s.roleManager;
      case 'saas_admin':
        return s.controlPanel;
      default:
        return s.roleCashier;
    }
  }

  Widget _sectionHeader(AppStrings s, String title) {
    return Padding(
      padding: const EdgeInsets.only(top: 20, bottom: 8),
      child: Text(title, style: Theme.of(context).textTheme.titleMedium),
    );
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final settings = _settings;
    return Scaffold(
      appBar: AppBar(title: Text(s.settings)),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(_error!,
                          style: TextStyle(
                              color: Theme.of(context).colorScheme.error)),
                      const SizedBox(height: 12),
                      FilledButton(onPressed: _load, child: Text(s.retry)),
                    ],
                  ),
                )
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    _sectionHeader(s, s.posSettings),
                    Card(
                      child: settings == null
                          ? const SizedBox()
                          : Column(
                              children: [
                                ListTile(
                                  leading: Icon(Icons.payments_outlined,
                                      color: Theme.of(context)
                                          .colorScheme
                                          .primary),
                                  title: Text(s.defaultPaymentMethod),
                                  subtitle: Text(s.defaultPaymentMethodHint),
                                  trailing: DropdownButton<PaymentMethod>(
                                    value: settings.defaultPaymentMethod,
                                    onChanged: _canEdit
                                        ? (value) => setState(() {
                                              _settings = settings.copyWith(
                                                  defaultPaymentMethod: value);
                                            })
                                        : null,
                                    items: [
                                      for (final method
                                          in PaymentMethod.values)
                                        DropdownMenuItem(
                                          value: method,
                                          child: Text(_paymentLabel(s, method)),
                                        ),
                                    ],
                                  ),
                                ),
                                const Divider(height: 1),
                                ListTile(
                                  leading: Icon(Icons.inventory_2_outlined,
                                      color: Theme.of(context)
                                          .colorScheme
                                          .primary),
                                  title: Text(s.stockBadges),
                                  subtitle: Text(s.stockBadgesHint),
                                  trailing: Switch(
                                    value: settings.showStockBadges,
                                    onChanged: _canEdit
                                        ? (value) => setState(() {
                                              _settings = settings.copyWith(
                                                  showStockBadges: value);
                                            })
                                        : null,
                                  ),
                                ),
                                const Divider(height: 1),
                                Padding(
                                  padding: const EdgeInsets.all(16),
                                  child: TextField(
                                    enabled: _canEdit,
                                    controller: TextEditingController(
                                        text: settings.receiptFooter)
                                      ..selection = TextSelection.collapsed(
                                          offset:
                                              settings.receiptFooter.length),
                                    decoration: InputDecoration(
                                      labelText: s.receiptFooter,
                                      hintText: s.receiptFooterHint,
                                      border: const OutlineInputBorder(),
                                    ),
                                    onChanged: (value) => setState(() {
                                      _settings =
                                          settings.copyWith(receiptFooter: value);
                                    }),
                                    maxLines: 2,
                                  ),
                                ),
                              ],
                            ),
                    ),
                    _sectionHeader(s, s.advancedInventory),
                    Card(
                      child: settings == null
                          ? const SizedBox()
                          : ListTile(
                              leading: Icon(Icons.assignment_late_outlined,
                                  color: Theme.of(context)
                                      .colorScheme
                                      .primary),
                              title: Text(s.allowNegativeStock),
                              subtitle: Text(s.allowNegativeStockHint),
                              trailing: Switch(
                                value: settings.allowNegativeStock,
                                onChanged: _canEdit
                                    ? (value) => setState(() {
                                          _settings = settings.copyWith(
                                              allowNegativeStock: value);
                                        })
                                    : null,
                              ),
                            ),
                    ),
                    if (!_canEdit)
                      Padding(
                        padding: const EdgeInsets.only(top: 8),
                        child: Text(s.settingsReadOnlyHint,
                            style: TextStyle(
                                color: Theme.of(context).colorScheme.onSurfaceVariant)),
                      ),
                    if (_canEdit) ...[
                      const SizedBox(height: 16),
                      FilledButton.icon(
                        onPressed: _saving ? null : _save,
                        icon: const Icon(Icons.save_outlined),
                        label: Text(s.save),
                      ),
                    ],
                    _sectionHeader(s, s.account),
                    Card(
                      child: Column(
                        children: [
                          ListTile(
                            leading: const Icon(Icons.person_outline),
                            title: Text(widget.session.displayName),
                            subtitle: Text(_roleLabel(s)),
                          ),
                          const Divider(height: 1),
                          ListTile(
                            leading: const Icon(Icons.language),
                            title: Text(s.language),
                            trailing: TextButton.icon(
                              onPressed: () {
                                widget.onLanguageChanged
                                    ?.call(s.isArabic ? 'en' : 'ar');
                              },
                              icon: const Icon(Icons.language),
                              label: Text(s.isArabic ? s.english : s.arabic),
                            ),
                          ),
                          const Divider(height: 1),
                          ListTile(
                            leading: const Icon(Icons.font_download_outlined),
                            title: Text(s.font),
                            trailing: Text('${_fontModeLabel(s, _font.fontMode)} · ${_fontSizeLabel(s, _font.size)}'),
                            onTap: _openFontPicker,
                          ),
                          const Divider(height: 1),
                          ListTile(
                            leading: const Icon(Icons.format_size),
                            title: Text(s.fontSize),
                            trailing: Text(_fontSizeLabel(s, _font.size)),
                            onTap: _openFontPicker,
                          ),
                          const Divider(height: 1),
                          ListTile(
                            leading: const Icon(Icons.volume_up_outlined),
                            title: Text(s.soundEffects),
                            subtitle: Text(s.soundEffectsHint),
                            trailing: Switch(
                              value: _soundEnabled,
                              onChanged: _setSoundEnabled,
                            ),
                          ),
                          const Divider(height: 1),
                          ListTile(
                            leading: const Icon(Icons.palette_outlined),
                            title: Text(s.appearance),
                            subtitle: Text(s.themeMode),
                            trailing: Text(
                                '${_themePreferenceLabel(s, _theme.preference)} · ${_themeAccentLabel(_theme.accent)}'),
                            onTap: _openThemePicker,
                          ),
                          if (widget.session.isPlatformAdmin) ...[
                            const Divider(height: 1),
                            ListTile(
                              leading: const Icon(Icons.dashboard_outlined),
                              title: Text(s.controlPanel),
                              onTap: () => Navigator.of(context).push(
                                MaterialPageRoute<void>(
                                  builder: (_) => ControlPanelScreen(
                                    session: widget.session,
                                    apiClient: widget.apiClient,
                                  ),
                                ),
                              ),
                            ),
                          ],
                          if (widget.onSignOut != null) ...[
                            const Divider(height: 1),
                            ListTile(
                              leading: const Icon(Icons.logout),
                              title: Text(s.signOut),
                              onTap: widget.onSignOut,
                            ),
                          ],
                        ],
                      ),
                    ),
                  ],
                ),
    );
  }

  String _fontModeLabel(AppStrings s, FontMode style) {
    switch (style) {
      case FontMode.sans:
        return s.fontSans;
      case FontMode.naskh:
        return s.fontNaskh;
      case FontMode.kufi:
        return s.fontKufi;
    }
  }

  String _fontSizeLabel(AppStrings s, FontSize size) {
    switch (size) {
      case FontSize.small:
        return s.fontSizeSmall;
      case FontSize.medium:
        return s.fontSizeMedium;
      case FontSize.large:
        return s.fontSizeLarge;
    }
  }

  String _themePreferenceLabel(AppStrings s, ThemePreference preference) {
    switch (preference) {
      case ThemePreference.light:
        return s.themeLight;
      case ThemePreference.dark:
        return s.themeDark;
      case ThemePreference.system:
        return s.themeSystem;
    }
  }

  String _themeAccentLabel(ThemeAccent accent) {
    final s = AppStrings.of(context);
    switch (accent) {
      case ThemeAccent.blue:
        return s.themeBlue;
      case ThemeAccent.green:
        return s.themeGreen;
      case ThemeAccent.orange:
        return s.themeOrange;
      case ThemeAccent.purple:
        return s.themePurple;
      case ThemeAccent.teal:
        return s.themeTeal;
      case ThemeAccent.crimson:
        return s.themeCrimson;
    }
  }

  String _paymentLabel(AppStrings s, PaymentMethod method) {
    switch (method) {
      case PaymentMethod.cash:
        return s.cash;
      case PaymentMethod.card:
        return s.card;
      case PaymentMethod.mobile:
        return s.mobilePayment;
    }
  }
}