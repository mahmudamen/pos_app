import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../core/api_client.dart';
import '../../core/saas.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

class TrialSettingsScreen extends StatefulWidget {
  const TrialSettingsScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<TrialSettingsScreen> createState() => _TrialSettingsScreenState();
}

class _TrialSettingsScreenState extends State<TrialSettingsScreen> {
  TrialPolicy? _policy;
  bool _loading = true;
  bool _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final policy = await widget.apiClient.platformTrialSettings(widget.session);
      if (!mounted) return;
      setState(() {
        _policy = policy;
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

  Future<void> _save(TrialPolicy policy) async {
    setState(() => _saving = true);
    try {
      final updated =
          await widget.apiClient.platformUpdateTrialSettings(widget.session, policy);
      if (!mounted) return;
      setState(() {
        _policy = updated;
        _saving = false;
      });
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(AppStrings.of(context).saved),
        behavior: SnackBarBehavior.floating,
      ));
    } catch (_) {
      if (!mounted) return;
      setState(() => _saving = false);
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(AppStrings.of(context).policySaveFailed),
        behavior: SnackBarBehavior.floating,
      ));
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final policy = _policy;
    return Scaffold(
      appBar: AppBar(title: Text(s.trialSettings)),
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
                      FilledButton(
                          onPressed: _load, child: Text(s.retry)),
                    ],
                  ),
                )
              : _PolicyForm(
                  policy: policy!,
                  saving: _saving,
                  onSave: _save,
                ),
    );
  }
}

class _PolicyForm extends StatefulWidget {
  const _PolicyForm({
    required this.policy,
    required this.saving,
    required this.onSave,
  });

  final TrialPolicy policy;
  final bool saving;
  final void Function(TrialPolicy) onSave;

  @override
  State<_PolicyForm> createState() => _PolicyFormState();
}

class _PolicyFormState extends State<_PolicyForm> {
  late final TextEditingController _duration;
  late final TextEditingController _maxOrgs;
  late final TextEditingController _maxInstallations;
  late final TextEditingController _rate;
  late bool _email;
  late bool _phone;
  late bool _device;
  late bool _promo;

  @override
  void initState() {
    super.initState();
    final p = widget.policy;
    _duration = TextEditingController(text: '${p.trialDurationDays}');
    _maxOrgs = TextEditingController(text: '${p.maxOrganizationsPerAccount}');
    _maxInstallations =
        TextEditingController(text: '${p.maxActiveInstallations}');
    _rate = TextEditingController(text: '${p.registerRatePerIpPerHour}');
    _email = p.requireEmailVerification;
    _phone = p.requirePhoneVerification;
    _device = p.requireDeviceIntegrity;
    _promo = p.promoTrialsEnabled;
  }

  @override
  void dispose() {
    _duration.dispose();
    _maxOrgs.dispose();
    _maxInstallations.dispose();
    _rate.dispose();
    super.dispose();
  }

  void _submit() {
    final updated = TrialPolicy(
      trialDurationDays: int.tryParse(_duration.text) ?? 14,
      trialScope: widget.policy.trialScope,
      requireEmailVerification: _email,
      requirePhoneVerification: _phone,
      requireDeviceIntegrity: _device,
      maxOrganizationsPerAccount: int.tryParse(_maxOrgs.text) ?? 0,
      maxActiveInstallations: int.tryParse(_maxInstallations.text) ?? 0,
      suspiciousRegistrationPolicy: widget.policy.suspiciousRegistrationPolicy,
      registerRatePerIpPerHour: int.tryParse(_rate.text) ?? 0,
      promoTrialsEnabled: _promo,
      offlineTrialPolicy: widget.policy.offlineTrialPolicy,
    );
    widget.onSave(updated);
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        TextField(
          controller: _duration,
          keyboardType: TextInputType.number,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          decoration: InputDecoration(
            labelText: s.durationDaysLabel,
            border: const OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 16),
        TextField(
          controller: _maxOrgs,
          keyboardType: TextInputType.number,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          decoration: InputDecoration(
            labelText: s.maxOrgsLabel,
            border: const OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 16),
        TextField(
          controller: _maxInstallations,
          keyboardType: TextInputType.number,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          decoration: InputDecoration(
            labelText: s.maxInstallationsLabel,
            border: const OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 16),
        TextField(
          controller: _rate,
          keyboardType: TextInputType.number,
          inputFormatters: [FilteringTextInputFormatter.digitsOnly],
          decoration: InputDecoration(
            labelText: s.rateLimitLabel,
            border: const OutlineInputBorder(),
          ),
        ),
        const SizedBox(height: 16),
        SwitchListTile(
          title: Text(s.requireEmailVerificationLabel),
          value: _email,
          onChanged: (v) => setState(() => _email = v),
        ),
        SwitchListTile(
          title: Text(s.requirePhoneVerificationLabel),
          value: _phone,
          onChanged: (v) => setState(() => _phone = v),
        ),
        SwitchListTile(
          title: Text(s.requireDeviceIntegrityLabel),
          value: _device,
          onChanged: (v) => setState(() => _device = v),
        ),
        SwitchListTile(
          title: Text(s.promoEnabledLabel),
          value: _promo,
          onChanged: (v) => setState(() => _promo = v),
        ),
        const SizedBox(height: 24),
        FilledButton.icon(
          onPressed: widget.saving ? null : _submit,
          icon: widget.saving
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.save_outlined),
          label: Text(widget.saving ? s.saving : s.save),
        ),
      ],
    );
  }
}