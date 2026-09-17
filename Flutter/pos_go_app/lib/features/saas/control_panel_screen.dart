import 'dart:async';

import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/layout.dart';
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
                        onRefresh: _load,
                      ),
                      _TenantsTab(
                        session: widget.session,
                        apiClient: widget.apiClient,
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
  const _OverviewTab({
    required this.summary,
    required this.tenants,
    required this.onRefresh,
  });

  final SaasSummary? summary;
  final SaasTenantsPage tenants;
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final summary = this.summary;
    return RefreshIndicator(
      onRefresh: onRefresh,
      child: MaxWidthBox(
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
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
                      subtitle: Text(
                          '${b.tenants} ${s.tenants} · ${b.users} ${s.users} · '
                          '${s.formatMoney(b.revenueMinor, summary.supportedCurrency)}'),
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
        ),
      ),
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
  const _TenantsTab({required this.session, required this.apiClient});

  final Session session;
  final ApiClient apiClient;

  @override
  State<_TenantsTab> createState() => _TenantsTabState();
}

class _TenantsTabState extends State<_TenantsTab> {
  SaasTenantsPage? _page;
  bool _loading = true;
  String? _error;
  String _query = '';
  String _typeFilter = '';
  String _statusFilter = '';
  String _planFilter = '';
  int _pageIndex = 1;
  Timer? _debounce;

  static const int _limit = 50;

  @override
  void initState() {
    super.initState();
    _fetch();
  }

  @override
  void dispose() {
    _debounce?.cancel();
    super.dispose();
  }

  void _onQueryChanged(String value) {
    setState(() => _query = value);
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 350), () {
      _pageIndex = 1;
      _fetch();
    });
  }

  void _applyFilter(void Function() update) {
    setState(() {
      _pageIndex = 1;
      update();
    });
    _fetch();
  }

  Future<void> _fetch() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final q = _query.trim();
      final page = await widget.apiClient.saasTenants(
        widget.session,
        page: _pageIndex,
        limit: _limit,
        q: q.isEmpty ? null : q,
        businessType: _typeFilter.isEmpty ? null : _typeFilter,
        status: _statusFilter.isEmpty ? null : _statusFilter,
        plan: _planFilter.isEmpty ? null : _planFilter,
      );
      if (!mounted) return;
      setState(() {
        _page = page;
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

  void _goTo(int page) {
    if (page < 1 || _page == null) return;
    _pageIndex = page;
    _fetch();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final tenants = _page?.tenants ?? const <SaasTenant>[];
    final types = <String>{};
    final plans = <String>{};
    for (final t in tenants) {
      if (t.businessType.isNotEmpty) types.add(t.businessType);
      final plan = t.plan.isEmpty ? 'standard' : t.plan;
      plans.add(plan);
    }
    final typeList = types.toList()..sort();
    final planList = plans.toList()..sort();

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
          child: TextField(
            onChanged: _onQueryChanged,
            decoration: InputDecoration(
              prefixIcon: const Icon(Icons.search),
              hintText: s.searchTenants,
              border: const OutlineInputBorder(),
              isDense: true,
            ),
          ),
        ),
        _filterRow([
          FilterChip(
            label: Text(s.all),
            selected: _typeFilter.isEmpty,
            onSelected: (_) => _applyFilter(() => _typeFilter = ''),
          ),
          ...typeList.map((type) => FilterChip(
                label: Text(type),
                selected: _typeFilter == type,
                onSelected: (_) =>
                    _applyFilter(() => _typeFilter = type),
              )),
        ]),
        _filterRow([
          FilterChip(
            label: Text(s.all),
            selected: _statusFilter.isEmpty,
            onSelected: (_) => _applyFilter(() => _statusFilter = ''),
          ),
          FilterChip(
            label: Text(s.activeLabel),
            selected: _statusFilter == 'active',
            onSelected: (_) => _applyFilter(() => _statusFilter = 'active'),
          ),
          FilterChip(
            label: Text(s.suspendedLabel),
            selected: _statusFilter == 'suspended',
            onSelected: (_) =>
                _applyFilter(() => _statusFilter = 'suspended'),
          ),
        ]),
        if (planList.isNotEmpty)
          _filterRow([
            FilterChip(
              label: Text(s.plans),
              selected: _planFilter.isEmpty,
              onSelected: (_) => _applyFilter(() => _planFilter = ''),
            ),
            ...planList.map((plan) => FilterChip(
                  label: Text(plan),
                  selected: _planFilter == plan,
                  onSelected: (_) => _applyFilter(() => _planFilter = plan),
                )),
          ]),
        Expanded(
          child: _loading
              ? const Center(child: CircularProgressIndicator())
              : _error != null
                  ? _ErrorView(error: _error!, onRetry: _fetch)
                  : RefreshIndicator(
                      onRefresh: _fetch,
                      child: tenants.isEmpty
                          ? ListView(
                              physics: const AlwaysScrollableScrollPhysics(),
                              children: [
                                const SizedBox(height: 120),
                                Center(child: Text(s.noTenantsFound)),
                              ],
                            )
                          : Column(
                              children: [
                                Expanded(
                                  child: ListView.builder(
                                    physics:
                                        const AlwaysScrollableScrollPhysics(),
                                    padding: const EdgeInsets.all(12),
                                    itemCount: tenants.length,
                                    itemBuilder: (context, index) {
                                      final t = tenants[index];
                                      return _TenantCard(
                                        tenant: t,
                                        onTap: () =>
                                            Navigator.of(context).push(
                                          MaterialPageRoute<void>(
                                            builder: (_) =>
                                                TenantAnalyticsScreen(
                                              session: widget.session,
                                              apiClient: widget.apiClient,
                                              tenantId: t.id,
                                              tenantName: t.name,
                                            ),
                                          ),
                                        ),
                                      );
                                    },
                                  ),
                                ),
                                if ((_page?.total ?? 0) > _limit)
                                  _Pager(
                                    page: _page?.page ?? 1,
                                    total: _page?.total ?? 0,
                                    limit: _limit,
                                    onPrevious: () => _goTo(
                                        (_page?.page ?? 1) - 1),
                                    onNext: () =>
                                        _goTo((_page?.page ?? 1) + 1),
                                  ),
                              ],
                            ),
                    ),
        ),
      ],
    );
  }

  Widget _filterRow(List<Widget> chips) {
    return SizedBox(
      height: 44,
      child: ListView(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12),
        children: [
          for (var i = 0; i < chips.length; i++)
            if (i > 0)
              Padding(
                padding: const EdgeInsets.only(left: 8),
                child: chips[i],
              )
            else
              chips[i],
        ],
      ),
    );
  }
}

class _TenantCard extends StatelessWidget {
  const _TenantCard({required this.tenant, required this.onTap});

  final SaasTenant tenant;
  final VoidCallback onTap;

  String _headerLine(SaasTenant t, String currency) {
    final parts = [t.slug, t.businessType];
    if (t.countryCode.isNotEmpty) parts.add('$t.countryCode/$currency');
    return parts.join(' · ');
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final t = tenant;
    final currency = t.currencyCode.isEmpty ? 'EGP' : t.currencyCode;
    final theme = Theme.of(context);
    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.center,
            children: [
              const Icon(Icons.storefront_outlined),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(t.name, style: theme.textTheme.titleMedium),
                    const SizedBox(height: 2),
                    Text(_headerLine(t, currency),
                        style: theme.textTheme.bodySmall),
                    if (t.createdAt.isNotEmpty)
                      Text('${s.joinedOn} ${_fmtDate(s, t.createdAt)}',
                          style: theme.textTheme.bodySmall),
                    const SizedBox(height: 4),
                    Text(
                      '${s.revenue} ${s.formatMoney(t.revenueMinor, currency)} · '
                      '${s.totalSales} ${t.totalSales}',
                      style: theme.textTheme.bodyMedium,
                    ),
                    const SizedBox(height: 6),
                    Wrap(
                      spacing: 6,
                      runSpacing: 4,
                      children: [
                        _PlanChip(plan: t.plan),
                        if (t.status == 'suspended')
                          Chip(
                            label: Text(s.suspendedLabel),
                            visualDensity: VisualDensity.compact,
                            backgroundColor: theme.colorScheme.error,
                          ),
                      ],
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 12),
              Text('${s.users} ${t.users} · ${s.products} ${t.products}',
                  style: theme.textTheme.bodySmall),
            ],
          ),
        ),
      ),
    );
  }
}

class _Pager extends StatelessWidget {
  const _Pager({
    required this.page,
    required this.total,
    required this.limit,
    required this.onPrevious,
    required this.onNext,
  });

  final int page;
  final int total;
  final int limit;
  final VoidCallback onPrevious;
  final VoidCallback onNext;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final totalPages = (total / limit).ceil();
    return Padding(
      padding: const EdgeInsets.all(8),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          IconButton(
            onPressed: page > 1 ? onPrevious : null,
            icon: const Icon(Icons.chevron_left),
          ),
          Text('${s.pageOf} $page / $totalPages'),
          IconButton(
            onPressed: page < totalPages ? onNext : null,
            icon: const Icon(Icons.chevron_right),
          ),
        ],
      ),
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
                Text(
                    '${s.trialType}: ${t.trialType} · ${s.trialSource}: ${t.source}'),
                Text('${s.trialStartedAt}: ${_fmtDate(s, t.startedAt)}'),
                Text('${s.trialExpiresAt}: ${_fmtDate(s, t.expiresAt)}'),
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
    final label = switch (status) {
      'active' => s.statusActive,
      'pending' => s.statusPending,
      'expired' => s.statusExpired,
      'consumed' => s.statusConsumed,
      'revoked' => s.statusRevoked,
      'converted' => s.statusConverted,
      'canceled' => s.statusRevoked,
      _ => status,
    };
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
                Text(_fmtDate(s, e.createdAt)),
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

String _fmtDate(AppStrings s, String iso) {
  if (iso.isEmpty) return '';
  final parsed = DateTime.tryParse(iso);
  if (parsed == null) return iso;
  return s.formatDate(parsed.toLocal());
}