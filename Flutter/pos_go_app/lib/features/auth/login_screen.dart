import 'package:flutter/material.dart';
import 'package:uuid/uuid.dart';

import '../../core/api_client.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen(
      {super.key,
      required this.apiClient,
      required this.onAuthenticated,
      required this.sessionStore,
      this.onLanguageChanged});

  final ApiClient apiClient;
  final ValueChanged<Session> onAuthenticated;
  final SessionStore sessionStore;
  final ValueChanged<String>? onLanguageChanged;

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _formKey = GlobalKey<FormState>();
  final _tenantNode = FocusNode();
  final _emailNode = FocusNode();
  final _passwordNode = FocusNode();
  final _deviceNode = FocusNode();
  final _tenantController = TextEditingController();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _deviceController = TextEditingController(text: 'Counter 1');
  String? _deviceId;
  bool _loading = true;
  bool _bootingMeta = false;
  bool _rememberMe = false;
  String? _error;
  List<Country> _countries = const [];
  List<Currency> _currencies = const [];
  String _countryCode = 'EG';
  String _currencyCode = 'EGP';

  @override
  void initState() {
    super.initState();
    _loadDeviceId();
    _loadMeta();
  }

  void _nextField(FocusNode node) =>
      FocusScope.of(context).requestFocus(node);

  void _submit() {
    if (_deviceNode.hasFocus) {
      FocusScope.of(context).unfocus();
    }
    _login();
  }

  Future<void> _loadDeviceId() async {
    var deviceId = await widget.sessionStore.readDeviceId();
    if (deviceId == null || deviceId.isEmpty) {
      deviceId = const Uuid().v4();
      await widget.sessionStore.saveDeviceId(deviceId);
    }
    final remembered = await widget.sessionStore.readRememberedLogin();
    if (mounted) {
      setState(() {
        _deviceId = deviceId;
        _loading = false;
        if (remembered != null) {
          _rememberMe = true;
          _tenantController.text = remembered.tenantId;
          _emailController.text = remembered.email;
          _passwordController.text = remembered.password;
          _deviceController.text = remembered.deviceName;
        }
      });
    }
  }

  Future<void> _loadMeta() async {
    setState(() => _bootingMeta = true);
    try {
      final results = await Future.wait([
        widget.apiClient.countries(),
        widget.apiClient.currencies(),
      ]);
      final countries = results[0] as List<Country>;
      final currencies = results[1] as List<Currency>;
      if (!mounted) return;
      setState(() {
        _countries = countries;
        _currencies = currencies;
        if (countries.isNotEmpty) _countryCode = countries.first.code;
        if (currencies.isNotEmpty) _currencyCode = currencies.first.code;
        _bootingMeta = false;
      });
    } catch (_) {
      if (mounted) setState(() => _bootingMeta = false);
    }
  }

  void _switchLanguage() {
    final s = AppStrings.of(context);
    widget.onLanguageChanged?.call(s.isArabic ? 'en' : 'ar');
  }

  @override
  void dispose() {
    _tenantController.dispose();
    _emailController.dispose();
    _passwordController.dispose();
    _deviceController.dispose();
    _tenantNode.dispose();
    _emailNode.dispose();
    _passwordNode.dispose();
    _deviceNode.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final session = await widget.apiClient.login(
        tenantId: _tenantController.text.trim(),
        email: _emailController.text.trim(),
        password: _passwordController.text,
        deviceId: _deviceId!,
        deviceName: _deviceController.text.trim(),
      );
      if (_rememberMe) {
        await widget.sessionStore.saveRememberedLogin(RememberedLogin(
          tenantId: _tenantController.text.trim(),
          email: _emailController.text.trim(),
          password: _passwordController.text,
          deviceName: _deviceController.text.trim(),
        ));
      } else {
        await widget.sessionStore.clearRememberedLogin();
      }
      if (mounted) widget.onAuthenticated(session);
    } catch (error) {
      if (mounted) {
        final s = AppStrings.of(context);
        setState(() => _error = (error is ApiException && error.isTrialExpired)
            ? s.trialExpired
            : error.toString());
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    if (_loading) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: Padding(
            padding: const EdgeInsets.all(28),
            child: Form(
              key: _formKey,
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      TextButton.icon(
                        onPressed: _switchLanguage,
                        icon: const Icon(Icons.language, size: 20),
                        label: Text(s.isArabic ? 'English / EN' : 'العربية / AR'),
                      ),
                    ],
                  ),
                  ClipRRect(
                    borderRadius: BorderRadius.circular(16),
                    child: Image.asset(
                      'assets/xamltech_logo.png',
                      width: 80,
                      height: 80,
                      fit: BoxFit.contain,
                    ),
                  ),
                  const SizedBox(height: 16),
                  Text('POS Go',
                      style: Theme.of(context).textTheme.displaySmall,
                      textAlign: TextAlign.center),
                  const SizedBox(height: 8),
                  Text(s.signInSubtitle,
                      style: Theme.of(context).textTheme.bodyLarge,
                      textAlign: TextAlign.center),
                  const SizedBox(height: 24),
                  TextFormField(
                      controller: _tenantController,
                      focusNode: _tenantNode,
                      textInputAction: TextInputAction.next,
                      decoration: InputDecoration(
                          labelText: s.storeId, border: const OutlineInputBorder()),
                      validator: _required,
                      onFieldSubmitted: (_) => _nextField(_emailNode)),
                  const SizedBox(height: 12),
                  TextFormField(
                      controller: _emailController,
                      focusNode: _emailNode,
                      textInputAction: TextInputAction.next,
                      decoration: InputDecoration(
                          labelText: s.email, border: const OutlineInputBorder()),
                      keyboardType: TextInputType.emailAddress,
                      validator: _required,
                      onFieldSubmitted: (_) => _nextField(_passwordNode)),
                  const SizedBox(height: 12),
                  TextFormField(
                      controller: _passwordController,
                      focusNode: _passwordNode,
                      textInputAction: TextInputAction.next,
                      decoration: InputDecoration(
                          labelText: s.password, border: const OutlineInputBorder()),
                      obscureText: true,
                      validator: _required,
                      onFieldSubmitted: (_) => _nextField(_deviceNode)),
                  const SizedBox(height: 12),
                  TextFormField(
                      controller: _deviceController,
                      focusNode: _deviceNode,
                      textInputAction: TextInputAction.done,
                      decoration: InputDecoration(
                          labelText: s.terminalName, border: const OutlineInputBorder()),
                      validator: _required,
                      onFieldSubmitted: (_) => _submit()),
                  const SizedBox(height: 12),
                  Row(children: [
                    Expanded(
                      child: _bootingMeta
                          ? const SizedBox(
                              height: 56,
                              child: Center(
                                  child: SizedBox(
                                      width: 22,
                                      height: 22,
                                      child: CircularProgressIndicator(strokeWidth: 2))),
                            )
                          : DropdownButtonFormField<String>(
                              initialValue: _countryCode,
                              decoration: InputDecoration(
                                  labelText: s.country,
                                  border: const OutlineInputBorder()),
                              items: _countries
                                  .map((c) => DropdownMenuItem(
                                      value: c.code,
                                      child: Text(s.isArabic
                                          ? c.nameAr
                                          : c.nameEn)))
                                  .toList(),
                              onChanged: (v) {
                                if (v == null) return;
                                setState(() => _countryCode = v);
                              },
                            ),
                    ),
                    const SizedBox(width: 12),
                    Expanded(
                      child: _bootingMeta
                          ? const SizedBox(height: 56)
                          : DropdownButtonFormField<String>(
                              initialValue: _currencyCode,
                              decoration: InputDecoration(
                                  labelText: s.currency,
                                  border: const OutlineInputBorder()),
                              items: _currencies
                                  .map((c) => DropdownMenuItem(
                                      value: c.code,
                                      child: Text(
                                          '${c.symbol} · ${s.isArabic ? c.nameAr : c.nameEn}')))
                                  .toList(),
                              onChanged: (v) {
                                if (v == null) return;
                                setState(() => _currencyCode = v);
                              },
                            ),
                    ),
                  ]),
                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Text(_error!,
                        style: TextStyle(
                            color: Theme.of(context).colorScheme.error)),
                  ],
                  const SizedBox(height: 8),
                  CheckboxListTile(
                    value: _rememberMe,
                    onChanged: (v) => setState(() => _rememberMe = v ?? false),
                    contentPadding: EdgeInsets.zero,
                    controlAffinity: ListTileControlAffinity.leading,
                    dense: true,
                    title: Text(s.rememberLogins),
                  ),
                  const SizedBox(height: 12),
                  FilledButton.icon(
                    onPressed: _loading || _bootingMeta ? null : _login,
                    icon: const Icon(Icons.login),
                    label: Text(_loading ? s.signingIn : s.signIn),
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  String? _required(String? value) =>
      value == null || value.trim().isEmpty ? AppStrings.of(context).required : null;
}