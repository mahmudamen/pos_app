import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/saas.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';
import 'tenant_analytics_screen.dart';
import 'trial_settings_screen.dart';

class ControlPanelScreen extends StatefulWidget {
  const ControlPanelScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<ControlPanelScreen> createState() => _ControlPanelScreenState();
}

class _ControlPanelScreenState extends State<ControlPanelScreen> {
  SaasSummary? _summary;
  SaasTenantsPage? _tenants;
  List<TrialEntitlement> _trials = const [];
  List<AuditEntry> _audit = const [];
  bool _loading = true;
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
      final results = await Future.wait<Object>([
        widget.apiClient.saasSummary(widget.session),
        widget.apiClient.saasTenants(widget.session, limit: 200),
        widget.apiClient.platformTrialEntitlements(widget.session),
        widget.apiClient.platformAudit(widget.session, limit: 200),
      ]);
      if (!mounted) return;
      setState(() {
        _summary = results[0] as SaasSummary;
        _tenants = results[1] as SaasTenantsPage;
        _trials = results[2] as List<TrialEntitlement>;
        _audit = results[3] as List<AuditEntry>;
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

  Future<void> _runTrialAction(
      TrialEntitlement trial, String action, {int? extraDays}) async {
    try {
      await widget.apiClient.trialChange(
        widget.session,
        trial.id,
        action,
        extraDays: extraDays,
      );
      if (!mounted) return;
      final s = AppStrings.of(context);
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(action == 'extend'
            ? s.trialExtended
            : action == 'revoke'
                ? s.trialRevoked
                : s.trialConverted),
        behavior: SnackBarBehavior.floating,
      ));
      _load();
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(AppStrings.of(context).actionFailed),
        behavior: SnackBarBehavior.floating,
      ));
    }
  }

  Future<void> _promptExtend(TrialEntitlement trial) async {
    final controller = TextEditingController(text: '7');
    final days = await showDialog<int>(
      context: context,
      builder: (ctx) {
        final s = AppStrings.of(ctx);
        return AlertDialog(
          title: Text(s.extendTrial),
          content: TextField(
            controller: controller,
            keyboardType: TextInputType.number,
            autofocus: true,
            decoration: InputDecoration(
              labelText: s.extendDays,
              hintText: s.extendDaysHint,
            ),
          ),
          actions: [
            TextButton(
                onPressed: () => Navigator.pop(ctx), child: Text(s.cancel)),
            FilledButton(
              onPressed: () =>
                  Navigator.pop(ctx, int.tryParse(controller.text)),
              child: Text(s.extendTrial),
            ),
          ],
        );
      },
    );
    if (days == null || days < 1) return;
    await _runTrialAction(trial, 'extend', extraDays: days);
  }

  Future<void> _confirmTrialAction(
      TrialEntitlement trial, String action) async {
    final s = AppStrings.of(context);
    final message = action == 'revoke' ? s.revokeTrial : s.convertTrial;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(message),
        content: Text('${s.trialStatus}: ${trial.status}'),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: Text(s.cancel)),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(s.ok),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    await _runTrialAction(trial, action);
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return DefaultTabController(
      length: 4,
      child: Scaffold(
        appBar: AppBar(
          title: Text(s.controlPanel),
          bottom: TabBar(
            isScrollable: true,
            tabs: [
              Tab(text: s.platformOverview),
              Tab(text: s.platformTenantsTab),
              Tab(text: s.platformTrialsTab),
              Tab(text: s.platformAuditTab),
            ],
          ),
          actions: [
            IconButton(
              tooltip: s.trialSettings,
              icon: const Icon(Icons.tune),
              onPressed: () => Navigator.of(context).push(MaterialPageRoute(
                builder: (_) => TrialSettingsScreen(
                  session: widget.session,
                  apiClient: widget.apiClient,
                ),
              )),
            ),
            IconButton(
              tooltip: s.refresh,
              icon: const Icon(Icons.refresh),
              onPressed: _load,
            ),
          ],
        ),
        body: _loading
            ? const Center(child: CircularProgressIndicator())
            : _error != null
                ? _ErrorView(error: _error!, onRetry: _load)
                : TabBarView(
                    children: [
                      _OverviewTab(
                        summary: _summary,
                        tenants: _tenants ?? const SaasTenantsPage(
                            tenants: [], total: 0, page: 1, limit: 200),
                      ),
                      _TenantsTab(
                        session: widget.session,
                        apiClient: widget.apiClient,
                        tenants: _tenants?.tenants ?? const [],
                      ),
                      _TrialsTab(
                        trials: _trials,
                        onExtend: _promptExtend,
                        onRevoke: (t) => _confirmTrialAction(t, 'revoke'),
                        onConvert: (t) => _confirmTrialAction(t, 'convert'),
                      ),
                      _AuditTab(audit: _audit),
                    ],
                  ),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});

  final String error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Text(error,
              style: TextStyle(color: Theme.of(context).colorScheme.error)),
          const SizedBox(height: 12),
          FilledButton(onPressed: onRetry, child: Text(s.retry)),
        ],
      ),
    );
  }
}

class _OverviewTab extends StatelessWidget {
  const _OverviewTab({required this.summary, required this.tenants});

  final SaasSummary? summary;
  final SaasTenantsPage tenants;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final summary = this.summary;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text(s.platformSummary, style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 12),
        if (summary != null)
          Wrap(
            spacing: 12,
            runSpacing: 12,
            children: [
              _StatCard(label: s.tenants, value: '${summary.totalTenants}'),
              _StatCard(label: s.users, value: '${summary.totalUsers}'),
              _StatCard(label: s.totalSales, value: '${summary.totalSales}'),
              _StatCard(
                  label: s.revenue,
                  value: s.formatMoney(
                      summary.revenueMinor, summary.supportedCurrency)),
            ],
          ),
        if (summary != null && summary.byBusiness.isNotEmpty) ...[
          const SizedBox(height: 20),
          Text(s.businessType, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 8),
          ...summary.byBusiness.map((b) => Card(
                child: ListTile(
                  dense: true,
                  title: Text(b.businessType),
                  subtitle: Text('${b.tenants} ${s.tenants}'),
                  leading: const Icon(Icons.category_outlined),
                ),
              )),
        ],
        const SizedBox(height: 20),
        Text(s.plans, style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 8),
        ..._planCounts(tenants).entries.map((e) => Card(
              child: ListTile(
                dense: true,
                title: Text(e.key),
                subtitle: Text('${e.value} ${s.tenants}'),
                leading: const Icon(Icons.card_membership_outlined),
              ),
            )),
        if (tenants.tenants.isEmpty) ...[
          const SizedBox(height: 8),
          Text(s.noTenantsFound,
              style: Theme.of(context).textTheme.bodyMedium),
        ],
      ],
    );
  }

  Map<String, int> _planCounts(SaasTenantsPage page) {
    final counts = <String, int>{};
    for (final t in page.tenants) {
      final plan = t.plan.isEmpty ? 'standard' : t.plan;
      counts[plan] = (counts[plan] ?? 0) + 1;
    }
    return counts;
  }
}

class _TenantsTab extends StatefulWidget {
  const _TenantsTab({
    required this.session,
    required this.apiClient,
    required this.tenants,
  });

  final Session session;
  final ApiClient apiClient;
  final List<SaasTenant> tenants;

  @override
  State<_TenantsTab> createState() => _TenantsTabState();
}

class _TenantsTabState extends State<_TenantsTab> {
  String _query = '';
  String _typeFilter = '';

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final all = widget.tenants;
    final types = <String>{};
    for (final t in all) {
      if (t.businessType.isNotEmpty) types.add(t.businessType);
    }
    final typeList = types.toList()..sort();

    final filtered = all.where((t) {
      final q = _query.trim().toLowerCase();
      final matchesQuery = q.isEmpty ||
          t.name.toLowerCase().contains(q) ||
          t.slug.toLowerCase().contains(q);
      final matchesType = _typeFilter.isEmpty || t.businessType == _typeFilter;
      return matchesQuery && matchesType;
    }).toList();

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.all(12),
          child: TextField(
            onChanged: (v) => setState(() => _query = v),
            decoration: InputDecoration(
              prefixIcon: const Icon(Icons.search),
              hintText: s.searchTenants,
              border: const OutlineInputBorder(),
              isDense: true,
            ),
          ),
        ),
        SizedBox(
          height: 44,
          child: ListView(
            scrollDirection: Axis.horizontal,
            padding: const EdgeInsets.symmetric(horizontal: 12),
            children: [
              FilterChip(
                label: Text(s.all),
                selected: _typeFilter.isEmpty,
                onSelected: (_) => setState(() => _typeFilter = ''),
              ),
              ...typeList.map((type) => Padding(
                    padding: const EdgeInsets.only(left: 8),
                    child: FilterChip(
                      label: Text(type),
                      selected: _typeFilter == type,
                      onSelected: (_) =>
                          setState(() => _typeFilter = type),
                    ),
                  )),
            ],
          ),
        ),
        Expanded(
          child: filtered.isEmpty
              ? Center(child: Text(s.noTenantsFound))
              : ListView.builder(
                  itemCount: filtered.length,
                  itemBuilder: (context, index) {
                    final t = filtered[index];
                    return Card(
                      margin: const EdgeInsets.symmetric(
                          horizontal: 12, vertical: 4),
                      child: ListTile(
                        leading: const Icon(Icons.storefront_outlined),
                        title: Text(t.name),
                        subtitle: Text(
                            '${t.slug} · ${t.businessType} · ${t.countryCode}/${t.currencyCode}'),
                        isThreeLine: false,
                        trailing: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          crossAxisAlignment: CrossAxisAlignment.end,
                          children: [
                            Text(
                                '${s.users} ${t.users} · ${s.products} ${t.products}'),
                            const SizedBox(height: 4),
                            Wrap(
                              spacing: 6,
                              children: [
                                _PlanChip(plan: t.plan),
                                if (t.status == 'suspended')
                                  Chip(
                                    label: Text(s.suspendedLabel),
                                    visualDensity: VisualDensity.compact,
                                    backgroundColor:
                                        Theme.of(context).colorScheme.error,
                                  ),
                              ],
                            ),
                          ],
                        ),
                        onTap: () => Navigator.of(context).push(
                          MaterialPageRoute<void>(
                            builder: (_) => TenantAnalyticsScreen(
                              session: widget.session,
                              apiClient: widget.apiClient,
                              tenantId: t.id,
                              tenantName: t.name,
                            ),
                          ),
                        ),
                      ),
                    );
                  },
                ),
        ),
      ],
    );
  }
}

class _PlanChip extends StatelessWidget {
  const _PlanChip({required this.plan});

  final String plan;

  @override
  Widget build(BuildContext context) {
    final palette = Theme.of(context).colorScheme;
    return Chip(
      label: Text(plan),
      visualDensity: VisualDensity.compact,
      backgroundColor: plan == 'trial'
          ? palette.tertiaryContainer
          : plan == 'premium'
              ? palette.primaryContainer
              : palette.secondaryContainer,
    );
  }
}

class _TrialsTab extends StatelessWidget {
  const _TrialsTab({
    required this.trials,
    required this.onExtend,
    required this.onRevoke,
    required this.onConvert,
  });

  final List<TrialEntitlement> trials;
  final void Function(TrialEntitlement) onExtend;
  final void Function(TrialEntitlement) onRevoke;
  final void Function(TrialEntitlement) onConvert;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    if (trials.isEmpty) {
      return Center(child: Text(s.noTrialsYet));
    }
    return ListView.builder(
      padding: const EdgeInsets.all(12),
      itemCount: trials.length,
      itemBuilder: (context, index) {
        final t = trials[index];
        return Card(
          margin: const EdgeInsets.symmetric(vertical: 4),
          child: ListTile(
            leading: Icon(
              t.status == 'active'
                  ? Icons.check_circle_outline
                  : t.status == 'pending'
                      ? Icons.hourglass_empty
                      : Icons.cancel_outlined,
              color: _statusColor(context, t.status),
            ),
            title: Text(t.organizationId),
            subtitle: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text('${s.trialType}: ${t.trialType} · ${s.trialSource}: ${t.source}'),
                Text('${s.trialStartedAt}: ${t.startedAt}'),
                Text('${s.trialExpiresAt}: ${t.expiresAt}'),
                if (t.reason.isNotEmpty) Text('${s.reasonLabel}: ${t.reason}'),
              ],
            ),
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (t.status == 'active') ...[
                  IconButton(
                    tooltip: s.extendTrial,
                    icon: const Icon(Icons.add_circle_outline),
                    onPressed: () => onExtend(t),
                  ),
                  IconButton(
                    tooltip: s.revokeTrial,
                    icon: const Icon(Icons.remove_circle_outline),
                    onPressed: () => onRevoke(t),
                  ),
                  IconButton(
                    tooltip: s.convertTrial,
                    icon: const Icon(Icons.card_membership_outlined),
                    onPressed: () => onConvert(t),
                  ),
                ],
                _TrialStatusChip(status: t.status),
              ],
            ),
          ),
        );
      },
    );
  }

  Color _statusColor(BuildContext context, String status) {
    final scheme = Theme.of(context).colorScheme;
    switch (status) {
      case 'active':
        return scheme.primary;
      case 'pending':
        return scheme.tertiary;
      case 'expired':
        return scheme.error;
      default:
        return scheme.outline;
    }
  }
}

class _TrialStatusChip extends StatelessWidget {
  const _TrialStatusChip({required this.status});

  final String status;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final label = status == 'active'
        ? s.statusActive
        : status == 'pending'
            ? s.statusPending
            : status == 'expired'
                ? s.statusExpired
                : status;
    return Chip(label: Text(label), visualDensity: VisualDensity.compact);
  }
}

class _AuditTab extends StatelessWidget {
  const _AuditTab({required this.audit});

  final List<AuditEntry> audit;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    if (audit.isEmpty) {
      return Center(child: Text(s.noAuditEntries));
    }
    return ListView.builder(
      padding: const EdgeInsets.all(12),
      itemCount: audit.length,
      itemBuilder: (context, index) {
        final e = audit[index];
        return Card(
          margin: const EdgeInsets.symmetric(vertical: 4),
          child: ListTile(
            leading: const Icon(Icons.history),
            title: Text(e.action),
            subtitle: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(e.createdAt),
                if (e.reason.isNotEmpty) Text('${s.reasonLabel}: ${e.reason}'),
                Text('${s.actorLabel}: ${_shortId(e.actorUserId)}'),
              ],
            ),
            trailing: Text(_shortId(e.tenantId)),
          ),
        );
      },
    );
  }

  String _shortId(String id) =>
      id.length > 8 ? id.substring(0, 8) : id;
}

class _StatCard extends StatelessWidget {
  const _StatCard({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 150,
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(label, style: Theme.of(context).textTheme.bodyMedium),
              const SizedBox(height: 8),
              Text(value, style: Theme.of(context).textTheme.headlineSmall),
            ],
          ),
        ),
      ),
    );
  }
}