import 'package:flutter/material.dart';
import 'package:uuid/uuid.dart';

import '../../core/api_client.dart';
import '../../core/fonts.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

typedef BusinessType = (String, IconData);
typedef OnboardingInterest = (String, IconData);

/// The onboarding wizard steps after the language-toggle welcome.
enum _OnboardingStep { businessType, interests, plans, account }

class OnboardingScreen extends StatefulWidget {
  const OnboardingScreen({
    super.key,
    required this.apiClient,
    required this.sessionStore,
    required this.onAuthenticated,
    this.onLanguageChanged,
    this.fontSetting = const FontSetting(),
    this.onFontModeChanged,
    this.onFontSizeChanged,
    this.onOpenLogin,
  });

  final ApiClient apiClient;
  final SessionStore sessionStore;
  final ValueChanged<Session> onAuthenticated;
  final ValueChanged<String>? onLanguageChanged;
  final FontSetting fontSetting;
  final ValueChanged<FontMode>? onFontModeChanged;
  final ValueChanged<FontSize>? onFontSizeChanged;
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
  final Set<String> _interests = {};
  String _plan = 'trial';
  _OnboardingStep? _step;
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
    ('bakery', Icons.bakery_dining_outlined),
    ('shawerma', Icons.kebab_dining_outlined),
    ('falafel', Icons.lunch_dining_outlined),
    ('pharmacy', Icons.local_pharmacy_outlined),
    ('butcher', Icons.soup_kitchen_outlined),
    ('fruits_veg', Icons.eco_outlined),
    ('clothing', Icons.checkroom_outlined),
    ('sweets', Icons.cake_outlined),
    ('jewelry', Icons.diamond_outlined),
    ('hardware', Icons.handyman_outlined),
  ];

  /// Feature interests the merchant can signal on first use. All features are
  /// included in the trial — selection is informational later.
  static const List<OnboardingInterest> _interestChoices = [
    ('inventory', Icons.inventory_2_outlined),
    ('loyalty', Icons.card_membership_outlined),
    ('tables', Icons.table_restaurant_outlined),
    ('analytics', Icons.insights_outlined),
    ('receipts', Icons.print_outlined),
    ('discounts', Icons.local_offer_outlined),
    ('sync', Icons.sync_outlined),
    ('refunds', Icons.replay_outlined),
  ];

  static const List<({String code, IconData icon, int priceMinor, bool popular})> _plans = [
    (code: 'standard', icon: Icons.store_outlined, priceMinor: 49900, popular: false),
    (code: 'premium', icon: Icons.workspace_premium_outlined, priceMinor: 99900, popular: true),
    (code: 'enterprise', icon: Icons.business_outlined, priceMinor: 249900, popular: false),
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
      case 'bakery':
        return s.bakery;
      case 'shawerma':
        return s.shawerma;
      case 'falafel':
        return s.falafel;
      case 'pharmacy':
        return s.pharmacy;
      case 'butcher':
        return s.butcher;
      case 'fruits_veg':
        return s.fruitsVeg;
      case 'clothing':
        return s.clothing;
      case 'sweets':
        return s.sweets;
      case 'jewelry':
        return s.jewelry;
      case 'hardware':
        return s.hardware;
      default:
        return s.retail;
    }
  }

  String _interestLabel(String wire) {
    final s = AppStrings.of(context);
    switch (wire) {
      case 'inventory':
        return s.featureInventory;
      case 'loyalty':
        return s.featureLoyalty;
      case 'tables':
        return s.featureTables;
      case 'analytics':
        return s.featureAnalytics;
      case 'receipts':
        return s.featureReceipts;
      case 'discounts':
        return s.featureDiscounts;
      case 'sync':
        return s.featureSync;
      case 'refunds':
        return s.featureRefunds;
      default:
        return wire;
    }
  }

  String _planName(String code) {
    final s = AppStrings.of(context);
    switch (code) {
      case 'premium':
        return s.planPremium;
      case 'enterprise':
        return s.planEnterprise;
      default:
        return s.planStandard;
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
        interests: _interests.toList(),
        plan: _plan,
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

  void _goTo(_OnboardingStep step) {
    if (_submitting) return;
    setState(() => _step = step);
  }

  void _back() {
    if (_step == null) return;
    switch (_step!) {
      case _OnboardingStep.businessType:
        setState(() => _step = null);
      case _OnboardingStep.interests:
        _goTo(_OnboardingStep.businessType);
      case _OnboardingStep.plans:
        _goTo(_OnboardingStep.interests);
      case _OnboardingStep.account:
        _goTo(_OnboardingStep.plans);
    }
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
          constraints: const BoxConstraints(maxWidth: 520),
          child: Padding(
            padding: const EdgeInsets.all(28),
            child: AnimatedSwitcher(
              duration: const Duration(milliseconds: 250),
              child: _step == null ? _buildWelcome(s) : _buildStep(s),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildWelcome(AppStrings s) {
    return SingleChildScrollView(
      primary: false,
      key: const PageStorageKey('onboarding-welcome'),
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
          _TrialBanner(text: s.onboardingFreeTrial),
          const SizedBox(height: 12),
          _LimitsRow(next: s),
          const SizedBox(height: 24),
          FilledButton.icon(
            onPressed: () => _goTo(_OnboardingStep.businessType),
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

  Widget _buildStep(AppStrings s) {
    final step = _step!;
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        _StepHeader(
          stepIndex: step.index,
          onBack: _back,
          onLanguage: _switchLanguage,
          title: switch (step) {
            _OnboardingStep.businessType =>
              _businessType == null ? s.pickBusinessType : _businessLabel(_businessType!),
            _OnboardingStep.interests => s.interestTitle,
            _OnboardingStep.plans => s.plansTitle,
            _OnboardingStep.account => '',
          },
        ),
        const SizedBox(height: 12),
        Expanded(
          child: switch (step) {
            _OnboardingStep.businessType => _buildBusinessPicker(s),
            _OnboardingStep.interests => _buildInterests(s),
            _OnboardingStep.plans => _buildPlans(s),
            _OnboardingStep.account => _buildForm(s),
          },
        ),
      ],
    );
  }

  Widget _buildBusinessPicker(AppStrings s) {
    return SingleChildScrollView(
      primary: false,
      key: const PageStorageKey('onboarding-business'),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const SizedBox(height: 8),
          GridView.count(
            crossAxisCount: 2,
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            mainAxisSpacing: 12,
            crossAxisSpacing: 12,
            childAspectRatio: 1.4,
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
                        Icon(icon, size: 28),
                        const SizedBox(height: 6),
                        Padding(
                          padding:
                              const EdgeInsets.symmetric(horizontal: 6),
                          child: Text(_businessLabel(wire),
                              style: Theme.of(context)
                                  .textTheme
                                  .bodyMedium,
                              textAlign: TextAlign.center,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis),
                        ),
                      ],
                    ),
                  ),
                ),
            ],
          ),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: _businessType == null
                ? null
                : () => _goTo(_OnboardingStep.interests),
            icon: const Icon(Icons.arrow_forward),
            label: Text(s.continueLabel),
          ),
          const SizedBox(height: 8),
          Text(
            '${s.businessType} · ${_businessTypes.length}',
            style: Theme.of(context).textTheme.bodySmall,
            textAlign: TextAlign.center,
          ),
        ],
      ),
    );
  }

  Widget _buildInterests(AppStrings s) {
    return SingleChildScrollView(
      primary: false,
      key: const PageStorageKey('onboarding-interests'),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const SizedBox(height: 8),
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: Text(s.interestSubtitle,
                style: Theme.of(context).textTheme.bodyMedium,
                textAlign: TextAlign.center),
          ),
          for (final (wire, icon) in _interestChoices)
            Card(
              margin: const EdgeInsets.only(bottom: 10),
              child: CheckboxListTile(
                value: _interests.contains(wire),
                onChanged: (v) => setState(() {
                  if (v ?? false) {
                    _interests.add(wire);
                  } else {
                    _interests.remove(wire);
                  }
                }),
                secondary: Icon(icon),
                title: Text(_interestLabel(wire)),
                controlAffinity: ListTileControlAffinity.trailing,
              ),
            ),
          Text(s.interestsHint,
              style: Theme.of(context).textTheme.bodySmall,
              textAlign: TextAlign.center),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: () => _goTo(_OnboardingStep.plans),
            icon: const Icon(Icons.arrow_forward),
            label: Text(s.continueLabel),
          ),
        ],
      ),
    );
  }

  Widget _buildPlans(AppStrings s) {
    return SingleChildScrollView(
      primary: false,
      key: const PageStorageKey('onboarding-plans'),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const SizedBox(height: 8),
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: Text(s.plansSubtitle,
                style: Theme.of(context).textTheme.bodyMedium,
                textAlign: TextAlign.center),
          ),
          _PlanCard(
            selected: _plan == 'trial',
            onTap: () => setState(() => _plan = 'trial'),
            title: s.planTrial,
            subtitle: s.trialBadge,
            price: '',
            features: [
              s.planLimitsLabel(2, 150),
              s.trialLimitDays,
              s.trialLimitNoCard,
            ],
            isTrial: true,
          ),
          const SizedBox(height: 12),
          for (final plan in _plans)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: _PlanCard(
                selected: _plan == plan.code,
                onTap: () => setState(() => _plan = plan.code),
                title: _planName(plan.code),
                subtitle: _planDescription(s, plan.code),
                price: '${s.formatMoney(plan.priceMinor, 'EGP')}${s.planPerMonth}',
                icon: plan.icon,
                features: _planFeatures(s, plan.code),
                popular: plan.popular,
              ),
            ),
          if (_plan != 'trial')
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Text(
                s.planAfterTrialLabel(_planName(_plan)),
                style: Theme.of(context).textTheme.bodySmall,
                textAlign: TextAlign.center,
              ),
            ),
          const SizedBox(height: 8),
          Container(
            padding: const EdgeInsets.all(14),
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(12),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(s.trialLimitsTitle,
                    style: Theme.of(context).textTheme.titleSmall),
                const SizedBox(height: 8),
                _LimitsRow(next: s),
              ],
            ),
          ),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: () => _goTo(_OnboardingStep.account),
            icon: const Icon(Icons.rocket_launch_outlined),
            label: Padding(
              padding: const EdgeInsets.symmetric(vertical: 12),
              child: Text(s.startTrial),
            ),
          ),
        ],
      ),
    );
  }

  String _planDescription(AppStrings s, String code) {
    switch (code) {
      case 'premium':
        return s.planPremiumDesc;
      case 'enterprise':
        return s.planEnterpriseDesc;
      default:
        return s.planStandardDesc;
    }
  }

  List<String> _planFeatures(AppStrings s, String code) {
    final labels = <String, String>{
      'featureInventory': s.featureInventory,
      'featureAnalytics': s.featureAnalytics,
      'featureReceipts': s.featureReceipts,
      'featureLoyalty': s.featureLoyalty,
      'featureRefunds': s.featureRefunds,
      'featureSync': s.featureSync,
      'featureTables': s.featureTables,
    };
    const base = ['featureInventory', 'featureAnalytics', 'featureReceipts'];
    final features = <String>[for (final k in base) labels[k]!];
    switch (code) {
      case 'premium':
        features.addAll([
          s.planLimitsLabel(15, 2000),
          s.featureLoyalty,
          s.featureRefunds,
        ]);
      case 'enterprise':
        features.addAll([
          s.planUnlimitedUsers,
          s.planUnlimitedProducts,
          s.featureSync,
          s.featureTables,
          s.planExtraSupport,
        ]);
      default:
        features.add(s.planLimitsLabel(5, 500));
    }
    return features;
  }

  Widget _buildForm(AppStrings s) {
    return SingleChildScrollView(
      primary: false,
      key: const PageStorageKey('onboarding-account'),
      child: Form(
        key: _formKey,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
              decoration: BoxDecoration(
                color: Theme.of(context).colorScheme.primaryContainer,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Row(
                children: [
                  Icon(Icons.rocket_launch_outlined,
                      size: 20, color: Theme.of(context).colorScheme.primary),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      _interests.isEmpty
                          ? s.trialPlanNotice
                          : '${s.trialPlanNotice} · ${_interests.length} ${s.interestsShort}',
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ),
                ],
              ),
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

/// Trial banner pill used on the welcome and limit panels.
class _TrialBanner extends StatelessWidget {
  const _TrialBanner({required this.text});

  final String text;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(12),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.workspace_premium_outlined,
              size: 20, color: Theme.of(context).colorScheme.primary),
          const SizedBox(width: 8),
          Expanded(child: Text(text)),
        ],
      ),
    );
  }
}

/// The four concrete "free trial limits" bullet rows.
class _LimitsRow extends StatelessWidget {
  const _LimitsRow({required this.next});

  final AppStrings next;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _LimitBullet(icon: Icons.event_outlined, text: next.trialLimitDays),
        _LimitBullet(icon: Icons.group_outlined, text: next.trialLimitUsers),
        _LimitBullet(icon: Icons.inventory_2_outlined, text: next.trialLimitProducts),
        _LimitBullet(icon: Icons.credit_card_off_outlined, text: next.trialLimitNoCard),
      ],
    );
  }
}

class _LimitBullet extends StatelessWidget {
  const _LimitBullet({required this.icon, required this.text});

  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Row(
        children: [
          Icon(icon, size: 18, color: theme.colorScheme.primary),
          const SizedBox(width: 10),
          Expanded(child: Text(text, style: theme.textTheme.bodySmall)),
        ],
      ),
    );
  }
}

/// Selectable plan card used in the plans step.
class _PlanCard extends StatelessWidget {
  const _PlanCard({
    required this.selected,
    required this.onTap,
    required this.title,
    required this.subtitle,
    required this.price,
    required this.features,
    this.icon,
    this.popular = false,
    this.isTrial = false,
  });

  final bool selected;
  final VoidCallback onTap;
  final String title;
  final String subtitle;
  final String price;
  final List<String> features;
  final IconData? icon;
  final bool popular;
  final bool isTrial;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final accent =
        selected ? theme.colorScheme.primary : theme.colorScheme.outline;
    return InkWell(
      borderRadius: BorderRadius.circular(16),
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          color: isTrial
              ? theme.colorScheme.primaryContainer
              : selected
                  ? theme.colorScheme.secondaryContainer
                  : theme.colorScheme.surface,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
              width: selected ? 2 : 1, color: selected ? accent : Colors.black12),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(icon ?? Icons.card_giftcard_outlined,
                    size: 22, color: theme.colorScheme.primary),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(title, style: theme.textTheme.titleMedium),
                      const SizedBox(height: 2),
                      Text(subtitle,
                          style: theme.textTheme.bodySmall,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis),
                    ],
                  ),
                ),
                if (popular)
                  Container(
                    padding:
                        const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                    decoration: BoxDecoration(
                        color: theme.colorScheme.tertiaryContainer,
                        borderRadius: BorderRadius.circular(999)),
                    child: Text(AppStrings.of(context).planPopular,
                        style: theme.textTheme.labelSmall),
                  ),
                const SizedBox(width: 8),
                if (selected)
                  Icon(Icons.check_circle, color: theme.colorScheme.primary),
              ],
            ),
            const SizedBox(height: 12),
            Text(price,
                style: theme.textTheme.headlineSmall?.copyWith(
                    color: theme.colorScheme.primary,
                    fontWeight: FontWeight.bold)),
            const SizedBox(height: 8),
            for (final feature in features)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 2),
                child: Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Icon(Icons.check, size: 16, color: Colors.green),
                    const SizedBox(width: 8),
                    Expanded(
                        child:
                            Text(feature, style: theme.textTheme.bodySmall)),
                  ],
                ),
              ),
          ],
        ),
      ),
    );
  }
}

/// Header with back arrow, progress dots, title and language toggle.
class _StepHeader extends StatelessWidget {
  const _StepHeader({
    required this.stepIndex,
    required this.onBack,
    required this.onLanguage,
    required this.title,
  });

  final int stepIndex;
  final VoidCallback onBack;
  final VoidCallback onLanguage;
  final String title;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        Row(
          children: [
            IconButton(onPressed: onBack, icon: const Icon(Icons.arrow_back)),
            Expanded(
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: [
                  for (var i = 0; i < 4; i++)
                    AnimatedContainer(
                      duration: const Duration(milliseconds: 150),
                      width: i == stepIndex ? 20 : 8,
                      height: 8,
                      margin: const EdgeInsets.symmetric(horizontal: 3),
                      decoration: BoxDecoration(
                        color: i <= stepIndex
                            ? theme.colorScheme.primary
                            : theme.colorScheme.outlineVariant,
                        borderRadius: BorderRadius.circular(4),
                      ),
                    ),
                ],
              ),
            ),
            TextButton.icon(
              onPressed: onLanguage,
              icon: const Icon(Icons.language, size: 18),
              label: Text(
                AppStrings.of(context).isArabic ? 'EN' : 'ع',
                style: const TextStyle(fontSize: 14),
              ),
            ),
          ],
        ),
        const SizedBox(height: 4),
        if (title.isNotEmpty)
          Text(title,
              style: theme.textTheme.headlineSmall,
              textAlign: TextAlign.center,
              maxLines: 2,
              overflow: TextOverflow.ellipsis),
      ],
    );
  }
}