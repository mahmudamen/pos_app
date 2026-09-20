import 'dart:async';

import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/billing.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

/// SaaS billing control plane: subscriptions (assign/change-plan/status),
/// invoices + payment flow (create/pay/void/refund), and platform-wide user
/// administration for merchant stores. Reached from the control panel.
class BillingScreen extends StatefulWidget {
  const BillingScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<BillingScreen> createState() => _BillingScreenState();
}

class _BillingScreenState extends State<BillingScreen> {
  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return DefaultTabController(
      length: 3,
      child: Scaffold(
        appBar: AppBar(
          title: Text(s.billing),
          bottom: TabBar(
            isScrollable: true,
            tabs: [
              Tab(text: s.subscriptions),
              Tab(text: s.invoices),
              Tab(text: s.platformUsers),
            ],
          ),
        ),
        body: TabBarView(
          children: [
            _SubscriptionsTab(
              session: widget.session,
              apiClient: widget.apiClient,
            ),
            _InvoicesTab(
              session: widget.session,
              apiClient: widget.apiClient,
            ),
            _UsersTab(
              session: widget.session,
              apiClient: widget.apiClient,
            ),
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

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.label, required this.color});

  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Chip(
      label: Text(label),
      visualDensity: VisualDensity.compact,
      backgroundColor: color,
    );
  }
}

Color _statusColor(BuildContext context, String status) {
  final scheme = Theme.of(context).colorScheme;
  switch (status) {
    case 'active':
    case 'paid':
      return scheme.primaryContainer;
    case 'trial':
    case 'grace_period':
      return scheme.tertiaryContainer;
    case 'past_due':
    case 'suspended':
      return scheme.errorContainer;
    case 'cancelled':
    case 'void':
    case 'refunded':
      return scheme.surfaceContainerHighest;
    default:
      return scheme.surfaceContainerHighest;
  }
}

String _subscriptionLabel(AppStrings s, String status) => switch (status) {
      'trial' => s.subTrial,
      'active' => s.subActive,
      'grace_period' => s.subGracePeriod,
      'past_due' => s.subPastDue,
      'suspended' => s.subSuspended,
      'cancelled' => s.subCancelled,
      _ => status,
    };

String _invoiceLabel(AppStrings s, String status) => switch (status) {
      'open' => s.invOpen,
      'paid' => s.invPaid,
      'void' => s.invVoid,
      'refunded' => s.invRefunded,
      _ => status,
    };

String _fmtDate(AppStrings s, String iso) {
  if (iso.isEmpty) return '';
  final parsed = DateTime.tryParse(iso);
  if (parsed == null) return iso;
  return s.formatDate(parsed.toLocal());
}

/// Shared horizontal filter chip row.
class _FilterRow extends StatelessWidget {
  const _FilterRow({required this.chips});

  final List<Widget> chips;

  @override
  Widget build(BuildContext context) {
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

// ---------------------------------------------------------------------------
// Subscriptions
// ---------------------------------------------------------------------------

class _SubscriptionsTab extends StatefulWidget {
  const _SubscriptionsTab({required this.session, required this.apiClient});

  final Session session;
  final ApiClient apiClient;

  @override
  State<_SubscriptionsTab> createState() => _SubscriptionsTabState();
}

class _SubscriptionsTabState extends State<_SubscriptionsTab> {
  BillingSubscriptionsPage? _page;
  List<BillingPlan> _plans = const [];
  List<SaasTenant> _tenants = const [];
  BillingSummaryData? _summary;
  bool _loading = true;
  String? _error;
  String _statusFilter = '';
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _fetch();
  }

  Future<void> _fetch() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final results = await Future.wait<Object>([
        widget.apiClient.billingSubscriptions(
          widget.session,
          status: _statusFilter.isEmpty ? null : _statusFilter,
          limit: 100,
        ),
        widget.apiClient.billingPlans(widget.session),
        widget.apiClient.saasTenants(widget.session, limit: 200),
        widget.apiClient.billingSummary(widget.session),
      ]);
      if (!mounted) return;
      setState(() {
        _page = results[0] as BillingSubscriptionsPage;
        _plans = results[1] as List<BillingPlan>;
        _tenants = (results[2] as SaasTenantsPage).tenants;
        _summary = results[3] as BillingSummaryData;
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

  void _snack(ScaffoldMessengerState messenger, String message) {
    messenger.showSnackBar(SnackBar(
      content: Text(message),
      behavior: SnackBarBehavior.floating,
    ));
  }

  Future<void> _assign() async {
    final result = await _assignSubscriptionDialog(
      context,
      tenants: _tenants,
      plans: _plans,
    );
    if (result == null || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    final s = AppStrings.of(context);
    setState(() => _busy = true);
    try {
      await widget.apiClient.assignSubscription(
        widget.session,
        result.tenantId,
        planCode: result.planCode,
        status: result.status,
        trialDays: result.trialDays,
      );
      _snack(messenger, s.subscriptionAssigned);
      _fetch();
    } catch (e) {
      _snack(messenger, e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _changePlan(BillingSubscription sub) async {
    final code = await _changePlanDialog(context, plans: _plans);
    if (code == null || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    final s = AppStrings.of(context);
    setState(() => _busy = true);
    try {
      await widget.apiClient.changeSubscriptionPlan(
          widget.session, sub.id, code);
      _snack(messenger, s.planChanged);
      _fetch();
    } catch (e) {
      _snack(messenger, e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _setStatus(BillingSubscription sub, String status) async {
    if (!mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    final s = AppStrings.of(context);
    setState(() => _busy = true);
    try {
      await widget.apiClient.setSubscriptionStatus(widget.session, sub.id, status);
      _snack(messenger, s.subscriptionStatusChanged);
      _fetch();
    } catch (e) {
      _snack(messenger, e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null) {
      return _ErrorView(error: _error!, onRetry: _fetch);
    }
    final subs = _page?.subscriptions ?? const <BillingSubscription>[];
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
          child: Row(
            children: [
              Expanded(
                child: FilledButton.icon(
                  onPressed: _busy ? null : _assign,
                  icon: const Icon(Icons.add_card_outlined),
                  label: Text(s.assignSubscription),
                ),
              ),
            ],
          ),
        ),
        if (_summary != null)
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
            child: MaxWidthBox(
              child: Wrap(
                spacing: 12,
                runSpacing: 12,
                children: [
                  _MiniStat(
                    label: s.mrr,
                    value:
                        s.formatMoney(_summary!.mrrMinor, _summary!.currency),
                  ),
                  _MiniStat(
                    label: s.outstanding,
                    value: s.formatMoney(
                        _summary!.outstandingMinor, _summary!.currency),
                  ),
                  _MiniStat(
                      label: s.activePlans, value: '${_summary!.activePlans}'),
                ],
              ),
            ),
          ),
        _FilterRow(chips: [
          FilterChip(
            label: Text(s.all),
            selected: _statusFilter.isEmpty,
            onSelected: (_) {
              setState(() => _statusFilter = '');
              _fetch();
            },
          ),
          ...subscriptionStatuses.map((status) => FilterChip(
                label: Text(_subscriptionLabel(s, status)),
                selected: _statusFilter == status,
                onSelected: (_) {
                  setState(() => _statusFilter = status);
                  _fetch();
                },
              )),
        ]),
        Expanded(
          child: subs.isEmpty
              ? ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  children: [
                    const SizedBox(height: 120),
                    Center(child: Text(s.noSubscriptions)),
                  ],
                )
              : RefreshIndicator(
                  onRefresh: _fetch,
                  child: ListView.builder(
                    physics: const AlwaysScrollableScrollPhysics(),
                    padding: const EdgeInsets.all(12),
                    itemCount: subs.length,
                    itemBuilder: (context, index) {
                      final sub = subs[index];
                      return Card(
                        margin: const EdgeInsets.symmetric(vertical: 4),
                        child: ListTile(
                          leading: const Icon(Icons.card_membership_outlined),
                          title: Text(sub.tenantName),
                          subtitle: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                  '${sub.planName} · ${s.formatMoney(sub.priceMinor, sub.currency)} '
                                  '/ ${sub.billingPeriod} · ${_fmtDate(s, sub.currentPeriodEnd)}'),
                              Wrap(
                                spacing: 6,
                                runSpacing: 4,
                                children: [
                                  _StatusChip(
                                    label:
                                        _subscriptionLabel(s, sub.status),
                                    color: _statusColor(context, sub.status),
                                  ),
                                  if (sub.cancelAtPeriodEnd)
                                    _StatusChip(
                                      label: s.cancelSubscription,
                                      color: Theme.of(context)
                                          .colorScheme
                                          .surfaceContainerHighest,
                                    ),
                                ],
                              ),
                            ],
                          ),
                          trailing: PopupMenuButton<String>(
                            tooltip: s.subscriptionStatus,
                            onSelected: (action) {
                              if (action == 'change_plan') {
                                _changePlan(sub);
                              } else {
                                _setStatus(sub, action);
                              }
                            },
                            itemBuilder: (context) => [
                              PopupMenuItem(
                                value: 'change_plan',
                                child: Text(s.changePlan),
                              ),
                              if (sub.status != 'active' &&
                                  sub.status != 'cancelled')
                                PopupMenuItem(
                                  value: 'active',
                                  child: Text(s.activateLabel),
                                ),
                              if (sub.status != 'suspended' &&
                                  sub.status != 'cancelled')
                                PopupMenuItem(
                                  value: 'suspended',
                                  child: Text(s.subSuspended),
                                ),
                              if (sub.status != 'cancelled')
                                PopupMenuItem(
                                  value: 'cancelled',
                                  child: Text(s.cancelSubscription),
                                ),
                            ],
                          ),
                        ),
                      );
                    },
                  ),
                ),
        ),
      ],
    );
  }
}

class _MiniStat extends StatelessWidget {
  const _MiniStat({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 130,
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(label, style: Theme.of(context).textTheme.bodySmall),
              const SizedBox(height: 4),
              Text(value, style: Theme.of(context).textTheme.titleMedium),
            ],
          ),
        ),
      ),
    );
  }
}

class _AssignResult {
  const _AssignResult({
    required this.tenantId,
    required this.planCode,
    this.status = 'trial',
    this.trialDays,
  });

  final String tenantId;
  final String planCode;
  final String status;
  final int? trialDays;
}

Future<_AssignResult?> _assignSubscriptionDialog(
  BuildContext context, {
  required List<SaasTenant> tenants,
  required List<BillingPlan> plans,
}) {
  final s = AppStrings.of(context);
  String? tenantId;
  String? planCode;
  String status = 'trial';
  final trialDays = TextEditingController();
  if (tenants.isNotEmpty) tenantId = tenants.first.id;
  final activePlans = plans.where((p) => p.isActive).toList();
  if (activePlans.isNotEmpty) planCode = activePlans.first.code;

  return showDialog<_AssignResult>(
    context: context,
    builder: (ctx) => StatefulBuilder(
      builder: (ctx, setState) => AlertDialog(
        title: Text(s.assignSubscription),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              DropdownButtonFormField<String>(
                initialValue: tenantId,
                decoration: InputDecoration(
                    labelText: s.selectTenant, border: const OutlineInputBorder()),
                items: tenants
                    .map((t) =>
                        DropdownMenuItem(value: t.id, child: Text(t.name)))
                    .toList(),
                onChanged: (v) => setState(() => tenantId = v),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: planCode,
                decoration: InputDecoration(
                    labelText: s.selectPlan, border: const OutlineInputBorder()),
                items: activePlans
                    .map((p) => DropdownMenuItem(
                        value: p.code, child: Text('${p.name} (${p.code})')))
                    .toList(),
                onChanged: (v) => setState(() => planCode = v),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: status,
                decoration: InputDecoration(
                    labelText: s.subscriptionStatus,
                    border: const OutlineInputBorder()),
                items: subscriptionStatuses
                    .map((st) => DropdownMenuItem(
                        value: st, child: Text(_subscriptionLabel(s, st))))
                    .toList(),
                onChanged: (v) => setState(() => status = v ?? 'trial'),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: trialDays,
                keyboardType: TextInputType.number,
                decoration: InputDecoration(
                  labelText: s.trialDays,
                  hintText: '0',
                  border: const OutlineInputBorder(),
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx), child: Text(s.cancel)),
          FilledButton(
            onPressed: tenantId == null || planCode == null
                ? null
                : () => Navigator.pop(
                    ctx,
                    _AssignResult(
                      tenantId: tenantId!,
                      planCode: planCode!,
                      status: status,
                      trialDays: int.tryParse(trialDays.text),
                    ),
                  ),
            child: Text(s.ok),
          ),
        ],
      ),
    ),
  );
}

Future<String?> _changePlanDialog(
  BuildContext context, {
  required List<BillingPlan> plans,
}) {
  final s = AppStrings.of(context);
  final active = plans.where((p) => p.isActive).toList();
  String? code = active.isNotEmpty ? active.first.code : null;
  return showDialog<String>(
    context: context,
    builder: (ctx) => StatefulBuilder(
      builder: (ctx, setState) => AlertDialog(
        title: Text(s.changePlan),
        content: DropdownButtonFormField<String>(
          initialValue: code,
          decoration: InputDecoration(
              labelText: s.selectPlan, border: const OutlineInputBorder()),
          items: active
              .map((p) => DropdownMenuItem(
                  value: p.code,
                  child: Text('${p.name} — ${s.formatMoney(p.priceMinor, p.currency)}')))
              .toList(),
          onChanged: (v) => setState(() => code = v),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx), child: Text(s.cancel)),
          FilledButton(
            onPressed: code == null
                ? null
                : () => Navigator.pop(ctx, code),
            child: Text(s.ok),
          ),
        ],
      ),
    ),
  );
}

// ---------------------------------------------------------------------------
// Invoices + payments
// ---------------------------------------------------------------------------

class _InvoicesTab extends StatefulWidget {
  const _InvoicesTab({required this.session, required this.apiClient});

  final Session session;
  final ApiClient apiClient;

  @override
  State<_InvoicesTab> createState() => _InvoicesTabState();
}

class _InvoicesTabState extends State<_InvoicesTab> {
  BillingInvoicesPage? _page;
  List<SaasTenant> _tenants = const [];
  List<PaymentProvider> _providers = const [];
  bool _loading = true;
  String? _error;
  String _statusFilter = '';
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _fetch();
  }

  Future<void> _fetch() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final results = await Future.wait<Object>([
        widget.apiClient.billingInvoices(
          widget.session,
          status: _statusFilter.isEmpty ? null : _statusFilter,
          limit: 100,
        ),
        widget.apiClient.saasTenants(widget.session, limit: 200),
        widget.apiClient.paymentProviders(widget.session),
      ]);
      if (!mounted) return;
      setState(() {
        _page = results[0] as BillingInvoicesPage;
        _tenants = (results[1] as SaasTenantsPage).tenants;
        _providers = results[2] as List<PaymentProvider>;
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

  void _snack(ScaffoldMessengerState messenger, String message) {
    messenger.showSnackBar(SnackBar(
      content: Text(message),
      behavior: SnackBarBehavior.floating,
    ));
  }

  Future<void> _create() async {
    final result = await _createInvoiceDialog(
      context,
      tenants: _tenants,
      providers: _providers,
    );
    if (result == null || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    final s = AppStrings.of(context);
    setState(() => _busy = true);
    try {
      await widget.apiClient.createInvoice(
        widget.session,
        result.tenantId,
        amountMinor: result.amountMinor,
        description: result.description,
        provider: result.provider,
        dueAt: result.dueAt,
      );
      _snack(messenger, s.invoiceCreated);
      _fetch();
    } catch (e) {
      _snack(messenger, e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _pay(BillingInvoice inv) async {
    String provider = inv.provider == 'manual' || inv.provider.isEmpty
        ? 'manual'
        : inv.provider;
    final s = AppStrings.of(context);
    provider = await showDialog<String>(
          context: context,
          builder: (ctx) => StatefulBuilder(
            builder: (ctx, setState) => AlertDialog(
              title: Text(s.pay),
              content: DropdownButtonFormField<String>(
                initialValue: provider,
                decoration: InputDecoration(
                    labelText: s.provider, border: const OutlineInputBorder()),
                items: _providers
                    .map((p) => DropdownMenuItem(
                        value: p.name, child: Text(p.name)))
                    .toList(),
                onChanged: (v) => setState(() => provider = v ?? 'manual'),
              ),
              actions: [
                TextButton(
                    onPressed: () => Navigator.pop(ctx), child: Text(s.cancel)),
                FilledButton(
                    onPressed: () => Navigator.pop(ctx, provider),
                    child: Text(s.pay)),
              ],
            ),
          ),
        ) ??
        '';
    if (provider.isEmpty || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _busy = true);
    try {
      await widget.apiClient.payInvoice(widget.session, inv.id,
          provider: provider);
      _snack(messenger, s.invoicePaid);
      _fetch();
    } catch (e) {
      _snack(messenger, e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _void(BillingInvoice inv) async {
    if (!mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    final s = AppStrings.of(context);
    setState(() => _busy = true);
    try {
      await widget.apiClient.voidInvoice(widget.session, inv.id);
      _snack(messenger, s.invoiceVoided);
      _fetch();
    } catch (e) {
      _snack(messenger, e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _refund(BillingInvoice inv) async {
    if (!mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    final s = AppStrings.of(context);
    setState(() => _busy = true);
    try {
      await widget.apiClient.refundInvoice(widget.session, inv.id);
      _snack(messenger, s.invoiceRefunded);
      _fetch();
    } catch (e) {
      _snack(messenger, e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null) {
      return _ErrorView(error: _error!, onRetry: _fetch);
    }
    final invoices = _page?.invoices ?? const <BillingInvoice>[];
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
          child: Row(
            children: [
              Expanded(
                child: FilledButton.icon(
                  onPressed: _busy ? null : _create,
                  icon: const Icon(Icons.note_add_outlined),
                  label: Text(s.createInvoice),
                ),
              ),
            ],
          ),
        ),
        _FilterRow(chips: [
          FilterChip(
            label: Text(s.all),
            selected: _statusFilter.isEmpty,
            onSelected: (_) {
              setState(() => _statusFilter = '');
              _fetch();
            },
          ),
          ...invoiceStatuses.map((status) => FilterChip(
                label: Text(_invoiceLabel(s, status)),
                selected: _statusFilter == status,
                onSelected: (_) {
                  setState(() => _statusFilter = status);
                  _fetch();
                },
              )),
        ]),
        Expanded(
          child: invoices.isEmpty
              ? ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  children: [
                    const SizedBox(height: 120),
                    Center(child: Text(s.noInvoices)),
                  ],
                )
              : RefreshIndicator(
                  onRefresh: _fetch,
                  child: ListView.builder(
                    physics: const AlwaysScrollableScrollPhysics(),
                    padding: const EdgeInsets.all(12),
                    itemCount: invoices.length,
                    itemBuilder: (context, index) {
                      final inv = invoices[index];
                      return Card(
                        margin: const EdgeInsets.symmetric(vertical: 4),
                        child: ListTile(
                          leading: const Icon(Icons.receipt_long_outlined),
                          title: Text(inv.tenantName),
                          subtitle: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                  '${s.formatMoney(inv.amountMinor, inv.currency)} · ${inv.description}'),
                              Text(
                                  '${s.provider}: ${inv.provider} · ${_fmtDate(s, inv.createdAt)}'),
                            ],
                          ),
                          trailing: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              _StatusChip(
                                label: _invoiceLabel(s, inv.status),
                                color: _statusColor(context, inv.status),
                              ),
                              if (inv.isOpen) ...[
                                IconButton(
                                  tooltip: s.pay,
                                  icon: const Icon(Icons.payments_outlined),
                                  onPressed: _busy ? null : () => _pay(inv),
                                ),
                                IconButton(
                                  tooltip: s.voidInvoice,
                                  icon: const Icon(Icons.delete_outline),
                                  onPressed: _busy ? null : () => _void(inv),
                                ),
                              ] else if (inv.isPaid)
                                PopupMenuButton<String>(
                                  onSelected: (_) => _refund(inv),
                                  itemBuilder: (context) => [
                                    PopupMenuItem(
                                      value: 'refund',
                                      child: Text(s.refund),
                                    ),
                                  ],
                                ),
                            ],
                          ),
                        ),
                      );
                    },
                  ),
                ),
        ),
      ],
    );
  }
}

class _InvoiceResult {
  const _InvoiceResult({
    required this.tenantId,
    required this.amountMinor,
    required this.description,
    this.provider,
    this.dueAt,
  });

  final String tenantId;
  final int amountMinor;
  final String description;
  final String? provider;
  final String? dueAt;
}

Future<_InvoiceResult?> _createInvoiceDialog(
  BuildContext context, {
  required List<SaasTenant> tenants,
  required List<PaymentProvider> providers,
}) {
  final s = AppStrings.of(context);
  String? tenantId;
  if (tenants.isNotEmpty) tenantId = tenants.first.id;
  String provider = providers.isNotEmpty ? providers.first.name : 'manual';
  final amount = TextEditingController();
  final description = TextEditingController();
  final dueAt = TextEditingController();

  return showDialog<_InvoiceResult>(
    context: context,
    builder: (ctx) => StatefulBuilder(
      builder: (ctx, setState) => AlertDialog(
        title: Text(s.createInvoice),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              DropdownButtonFormField<String>(
                initialValue: tenantId,
                decoration: InputDecoration(
                    labelText: s.selectTenant, border: const OutlineInputBorder()),
                items: tenants
                    .map((t) =>
                        DropdownMenuItem(value: t.id, child: Text(t.name)))
                    .toList(),
                onChanged: (v) => setState(() => tenantId = v),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: amount,
                keyboardType: TextInputType.number,
                decoration: InputDecoration(
                  labelText: s.amountLabel,
                  hintText: s.formatMoney(29900, 'EGP'),
                  border: const OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: description,
                decoration: InputDecoration(
                  labelText: s.descriptionLabel,
                  hintText: 'Business monthly',
                  border: const OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: provider,
                decoration: InputDecoration(
                    labelText: s.provider, border: const OutlineInputBorder()),
                items: providers
                    .map((p) => DropdownMenuItem(
                        value: p.name, child: Text(p.name)))
                    .toList(),
                onChanged: (v) => setState(() => provider = v ?? 'manual'),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: dueAt,
                keyboardType: TextInputType.datetime,
                decoration: InputDecoration(
                  labelText: s.dueAt,
                  hintText: 'YYYY-MM-DD',
                  border: const OutlineInputBorder(),
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx), child: Text(s.cancel)),
          FilledButton(
            onPressed: tenantId == null
                ? null
                : () => Navigator.pop(
                    ctx,
                    _InvoiceResult(
                      tenantId: tenantId!,
                      amountMinor: int.tryParse(amount.text) ?? 0,
                      description: description.text.trim(),
                      provider: provider,
                      dueAt: dueAt.text.trim().isEmpty
                          ? null
                          : dueAt.text.trim(),
                    ),
                  ),
            child: Text(s.createInvoice),
          ),
        ],
      ),
    ),
  );
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

class _UsersTab extends StatefulWidget {
  const _UsersTab({required this.session, required this.apiClient});

  final Session session;
  final ApiClient apiClient;

  @override
  State<_UsersTab> createState() => _UsersTabState();
}

class _UsersTabState extends State<_UsersTab> {
  SaasUsersPage? _page;
  List<SaasTenant> _tenants = const [];
  bool _loading = true;
  String? _error;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _fetch();
  }

  Future<void> _fetch() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final results = await Future.wait<Object>([
        widget.apiClient.saasUsers(widget.session, limit: 200),
        widget.apiClient.saasTenants(widget.session, limit: 200),
      ]);
      if (!mounted) return;
      setState(() {
        _page = results[0] as SaasUsersPage;
        _tenants = (results[1] as SaasTenantsPage).tenants;
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

  void _snack(ScaffoldMessengerState messenger, String message) {
    messenger.showSnackBar(SnackBar(
      content: Text(message),
      behavior: SnackBarBehavior.floating,
    ));
  }

  Future<void> _add() async {
    final result = await _createUserDialog(context, tenants: _tenants);
    if (result == null || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    final s = AppStrings.of(context);
    setState(() => _busy = true);
    try {
      await widget.apiClient.createSaasUser(
        widget.session,
        result.tenantId,
        email: result.email,
        displayName: result.displayName,
        password: result.password,
        role: result.role,
        accountType: result.accountType,
      );
      _snack(messenger, s.userCreated);
      _fetch();
    } catch (e) {
      _snack(messenger, e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null) {
      return _ErrorView(error: _error!, onRetry: _fetch);
    }
    final users = _page?.users ?? const <SaasUser>[];
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
          child: Row(
            children: [
              Expanded(
                child: FilledButton.icon(
                  onPressed: _busy ? null : _add,
                  icon: const Icon(Icons.person_add_alt_outlined),
                  label: Text(s.addUser),
                ),
              ),
            ],
          ),
        ),
        Expanded(
          child: users.isEmpty
              ? ListView(
                  physics: const AlwaysScrollableScrollPhysics(),
                  children: [
                    const SizedBox(height: 120),
                    Center(child: Text(s.noUsers)),
                  ],
                )
              : RefreshIndicator(
                  onRefresh: _fetch,
                  child: ListView.builder(
                    physics: const AlwaysScrollableScrollPhysics(),
                    padding: const EdgeInsets.all(12),
                    itemCount: users.length,
                    itemBuilder: (context, index) {
                      final u = users[index];
                      return Card(
                        margin: const EdgeInsets.symmetric(vertical: 4),
                        child: ListTile(
                          leading: Icon(
                            u.isActive
                                ? Icons.person_outline
                                : Icons.person_off_outlined,
                            color: u.isActive
                                ? null
                                : Theme.of(context).colorScheme.error,
                          ),
                          title: Text(u.displayName),
                          subtitle: Text(
                              '${u.email} · ${u.tenantName} · ${u.role}'),
                          trailing: Wrap(
                            spacing: 6,
                            children: [
                              if (u.accessLevel.isNotEmpty)
                                _StatusChip(
                                  label: u.accessLevel,
                                  color: Theme.of(context)
                                      .colorScheme
                                      .tertiaryContainer,
                                ),
                              _StatusChip(
                                label: u.isActive
                                    ? s.subActive
                                    : s.subSuspended,
                                color: u.isActive
                                    ? Theme.of(context)
                                        .colorScheme
                                        .primaryContainer
                                    : Theme.of(context).colorScheme.errorContainer,
                              ),
                            ],
                          ),
                        ),
                      );
                    },
                  ),
                ),
        ),
      ],
    );
  }
}

class _UserResult {
  const _UserResult({
    required this.tenantId,
    required this.email,
    required this.displayName,
    required this.password,
    required this.role,
    this.accountType,
  });

  final String tenantId;
  final String email;
  final String displayName;
  final String password;
  final String role;
  final String? accountType;
}

Future<_UserResult?> _createUserDialog(
  BuildContext context, {
  required List<SaasTenant> tenants,
}) {
  final s = AppStrings.of(context);
  String? tenantId;
  if (tenants.isNotEmpty) tenantId = tenants.first.id;
  String role = 'cashier';
  String accountType = 'standard';
  final email = TextEditingController();
  final displayName = TextEditingController();
  final password = TextEditingController();

  return showDialog<_UserResult>(
    context: context,
    builder: (ctx) => StatefulBuilder(
      builder: (ctx, setState) => AlertDialog(
        title: Text(s.addUser),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              DropdownButtonFormField<String>(
                initialValue: tenantId,
                decoration: InputDecoration(
                    labelText: s.selectTenant, border: const OutlineInputBorder()),
                items: tenants
                    .map((t) =>
                        DropdownMenuItem(value: t.id, child: Text(t.name)))
                    .toList(),
                onChanged: (v) => setState(() => tenantId = v),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: email,
                keyboardType: TextInputType.emailAddress,
                onChanged: (_) => setState(() {}),
                decoration: InputDecoration(
                  labelText: s.email,
                  border: const OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: displayName,
                onChanged: (_) => setState(() {}),
                decoration: InputDecoration(
                  labelText: s.displayName,
                  border: const OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: password,
                obscureText: true,
                onChanged: (_) => setState(() {}),
                decoration: InputDecoration(
                  labelText: s.password,
                  border: const OutlineInputBorder(),
                ),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: role,
                decoration: InputDecoration(
                    labelText: s.userRole, border: const OutlineInputBorder()),
                items: platformUserRoles
                    .map((r) =>
                        DropdownMenuItem(value: r, child: Text(r)))
                    .toList(),
                onChanged: (v) => setState(() => role = v ?? 'cashier'),
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                initialValue: accountType,
                decoration: InputDecoration(
                    labelText: s.accountType,
                    border: const OutlineInputBorder()),
                items: platformAccountTypes
                    .map((t) =>
                        DropdownMenuItem(value: t, child: Text(t)))
                    .toList(),
                onChanged: (v) =>
                    setState(() => accountType = v ?? 'standard'),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx), child: Text(s.cancel)),
          FilledButton(
            onPressed: tenantId == null ||
                    email.text.trim().length < 3 ||
                    displayName.text.trim().isEmpty ||
                    password.text.length < 8
                ? null
                : () => Navigator.pop(
                    ctx,
                    _UserResult(
                      tenantId: tenantId!,
                      email: email.text.trim(),
                      displayName: displayName.text.trim(),
                      password: password.text,
                      role: role,
                      accountType: accountType,
                    ),
                  ),
            child: Text(s.addUser),
          ),
        ],
      ),
    ),
  );
}