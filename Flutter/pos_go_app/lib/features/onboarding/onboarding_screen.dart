import 'package:flutter/material.dart';
import 'package:uuid/uuid.dart';

import '../../core/api_client.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

typedef BusinessType = (String, IconData);

class OnboardingScreen extends StatefulWidget {
  const OnboardingScreen({
    super.key,
    required this.apiClient,
    required this.sessionStore,
    required this.onAuthenticated,
    this.onLanguageChanged,
    this.onOpenLogin,
  });

  final ApiClient apiClient;
  final SessionStore sessionStore;
  final ValueChanged<Session> onAuthenticated;
  final ValueChanged<String>? onLanguageChanged;
  final VoidCallback? onOpenLogin;

  @override
  State<OnboardingScreen> createState() => _OnboardingScreenState();
}

class _OnboardingScreenState extends State<OnboardingScreen> {
  final _formKey = GlobalKey<FormState>();
  final _storeNameController = TextEditingController();
  final _displayNameController = TextEditingController();
  final _emailController = TextEditingController();
  final _passwordController = TextEditingController();
  final _deviceController = TextEditingController(text: 'Counter 1');
  String? _deviceId;
  String? _businessType;
  bool _submitting = false;
  String? _error;

  static const List<BusinessType> _businessTypes = [
    ('coffee_shop', Icons.local_cafe_outlined),
    ('restaurant', Icons.restaurant_outlined),
    ('retail', Icons.storefront_outlined),
    ('book_store', Icons.menu_book_outlined),
    ('mobile_shop', Icons.smartphone_outlined),
    ('computer_shop', Icons.laptop_outlined),
    ('grocery', Icons.local_grocery_store_outlined),
  ];

  @override
  void initState() {
    super.initState();
    _loadDeviceId();
  }

  Future<void> _loadDeviceId() async {
    var deviceId = await widget.sessionStore.readDeviceId();
    if (deviceId == null || deviceId.isEmpty) {
      deviceId = const Uuid().v4();
      await widget.sessionStore.saveDeviceId(deviceId);
    }
    if (mounted) setState(() => _deviceId = deviceId);
  }

  String _businessLabel(String wire) {
    final s = AppStrings.of(context);
    switch (wire) {
      case 'coffee_shop':
        return s.coffeeShop;
      case 'restaurant':
        return s.restaurant;
      case 'retail':
        return s.retail;
      case 'book_store':
        return s.bookStore;
      case 'mobile_shop':
        return s.mobileShop;
      case 'computer_shop':
        return s.computerShop;
      case 'grocery':
        return s.grocery;
      default:
        return s.retail;
    }
  }

  void _switchLanguage() {
    final s = AppStrings.of(context);
    widget.onLanguageChanged?.call(s.isArabic ? 'en' : 'ar');
  }

  Future<void> _register() async {
    if (!_formKey.currentState!.validate()) return;
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final session = await widget.apiClient.register(
        storeName: _storeNameController.text.trim(),
        businessType: _businessType!,
        email: _emailController.text.trim(),
        password: _passwordController.text,
        displayName: _displayNameController.text.trim(),
        deviceId: _deviceId!,
        deviceName: _deviceController.text.trim(),
      );
      await widget.sessionStore.saveRememberedLogin(RememberedLogin(
        tenantId: session.tenantId,
        email: _emailController.text.trim(),
        password: _passwordController.text,
        deviceName: _deviceController.text.trim(),
      ));
      if (mounted) widget.onAuthenticated(session);
    } catch (error) {
      if (mounted) setState(() => _error = _describe(error));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  String _describe(Object error) {
    final s = AppStrings.of(context);
    if (error is ApiException && error.isTrialExpired) return s.trialExpired;
    return error.toString();
  }

  @override
  void dispose() {
    _storeNameController.dispose();
    _displayNameController.dispose();
    _emailController.dispose();
    _passwordController.dispose();
    _deviceController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    if (_deviceId == null) {
      return const Scaffold(body: Center(child: CircularProgressIndicator()));
    }
    return Scaffold(
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 440),
          child: Padding(
            padding: const EdgeInsets.all(28),
            child: AnimatedSwitcher(
              duration: const Duration(milliseconds: 250),
              child: _businessType == null
                  ? _buildWelcome(s)
                  : _buildSignup(s),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildWelcome(AppStrings s) {
    return SingleChildScrollView(
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
                label:
                    Text(s.isArabic ? 'English / EN' : 'العربية / AR'),
              ),
            ],
          ),
          ClipRRect(
            borderRadius: BorderRadius.circular(16),
            child: Image.asset(
              'assets/xamltech_logo.png',
              width: 96,
              height: 96,
              fit: BoxFit.contain,
            ),
          ),
          const SizedBox(height: 20),
          Text(s.onboardingWelcomeTitle,
              style: Theme.of(context).textTheme.displaySmall,
              textAlign: TextAlign.center),
          const SizedBox(height: 10),
          Text(s.onboardingWelcomeSubtitle,
              style: Theme.of(context).textTheme.bodyLarge,
              textAlign: TextAlign.center),
          const SizedBox(height: 16),
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.primaryContainer,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(Icons.workspace_premium_outlined,
                    size: 20,
                    color: Theme.of(context).colorScheme.primary),
                const SizedBox(width: 8),
                Expanded(child: Text(s.onboardingFreeTrial)),
              ],
            ),
          ),
          const SizedBox(height: 24),
          FilledButton.icon(
            onPressed: () => setState(() => _businessType = ''),
            icon: const Icon(Icons.add_business_outlined),
            label: Padding(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Text(s.createStore),
            ),
          ),
          const SizedBox(height: 8),
          TextButton(
            onPressed: widget.onOpenLogin,
            child: Text(s.alreadyHaveStore),
          ),
        ],
      ),
    );
  }

  Widget _buildSignup(AppStrings s) {
    if (_businessType == '') {
      return _buildBusinessPicker(s);
    }
    return _buildForm(s);
  }

  Widget _buildBusinessPicker(AppStrings s) {
    return SingleChildScrollView(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(s.pickBusinessType,
              style: Theme.of(context).textTheme.headlineSmall,
              textAlign: TextAlign.center),
          const SizedBox(height: 20),
          GridView.count(
            crossAxisCount: 2,
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            mainAxisSpacing: 12,
            crossAxisSpacing: 12,
            childAspectRatio: 1.5,
            children: [
              for (final (wire, icon) in _businessTypes)
                InkWell(
                  borderRadius: BorderRadius.circular(16),
                  onTap: () => setState(() => _businessType = wire),
                  child: Container(
                    decoration: BoxDecoration(
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(16),
                      border: Border.all(color: Colors.black12),
                    ),
                    child: Column(
                      mainAxisAlignment: MainAxisAlignment.center,
                      children: [
                        Icon(icon, size: 32),
                        const SizedBox(height: 8),
                        Text(_businessLabel(wire),
                            style: Theme.of(context).textTheme.bodyLarge,
                            textAlign: TextAlign.center),
                      ],
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 12),
          TextButton.icon(
            onPressed: widget.onOpenLogin,
            icon: const Icon(Icons.login, size: 18),
            label: Text(s.alreadyHaveStore),
          ),
          const SizedBox(height: 8),
          Text(
            '${s.businessType} · 7',
            style: Theme.of(context).textTheme.bodySmall,
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }

  Widget _buildForm(AppStrings s) {
    return SingleChildScrollView(
      child: Form(
        key: _formKey,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                IconButton(
                  onPressed: _submitting
                      ? null
                      : () => setState(() => _businessType = ''),
                  icon: const Icon(Icons.arrow_back),
                ),
                Expanded(
                  child: Text(
                    _businessLabel(_businessType!),
                    style: Theme.of(context).textTheme.headlineSmall,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                TextButton.icon(
                  onPressed: _switchLanguage,
                  icon: const Icon(Icons.language, size: 18),
                  label: Text(s.isArabic ? 'EN' : 'ع'),
                ),
              ],
            ),
            const SizedBox(height: 12),
            TextFormField(
                controller: _storeNameController,
                textInputAction: TextInputAction.next,
                decoration: InputDecoration(
                    labelText: s.storeName,
                    border: const OutlineInputBorder()),
                validator: (v) =>
                    v == null || v.trim().isEmpty ? s.required : null),
            const SizedBox(height: 12),
            TextFormField(
                controller: _displayNameController,
                textInputAction: TextInputAction.next,
                decoration: InputDecoration(
                    labelText: s.ownerName,
                    border: const OutlineInputBorder()),
                validator: (v) =>
                    v == null || v.trim().isEmpty ? s.required : null),
            const SizedBox(height: 12),
            TextFormField(
                controller: _emailController,
                textInputAction: TextInputAction.next,
                keyboardType: TextInputType.emailAddress,
                decoration: InputDecoration(
                    labelText: s.email, border: const OutlineInputBorder()),
                validator: (v) =>
                    v == null || v.trim().isEmpty ? s.required : null),
            const SizedBox(height: 12),
            TextFormField(
                controller: _passwordController,
                textInputAction: TextInputAction.next,
                obscureText: true,
                decoration: InputDecoration(
                    labelText: s.password,
                    border: const OutlineInputBorder(),
                    helperText: s.passwordMin),
                validator: (v) =>
                    (v == null || v.length < 8) ? s.passwordMin : null),
            const SizedBox(height: 12),
            TextFormField(
                controller: _deviceController,
                textInputAction: TextInputAction.done,
                decoration: InputDecoration(
                    labelText: s.terminalName,
                    border: const OutlineInputBorder()),
                validator: (v) =>
                    v == null || v.trim().isEmpty ? s.required : null),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!,
                  style: TextStyle(
                      color: Theme.of(context).colorScheme.error)),
            ],
            const SizedBox(height: 16),
            FilledButton.icon(
              onPressed: _submitting ? null : _register,
              icon: const Icon(Icons.check_circle_outline),
              label: Padding(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: Text(_submitting ? s.signingUp : s.signUp),
              ),
            ),
          ],
        ),
      ),
    );
  }
}