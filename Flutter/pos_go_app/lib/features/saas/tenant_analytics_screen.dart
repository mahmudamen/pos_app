import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/saas.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

class TenantAnalyticsScreen extends StatefulWidget {
  const TenantAnalyticsScreen({
    super.key,
    required this.session,
    required this.apiClient,
    required this.tenantId,
    required this.tenantName,
  });

  final Session session;
  final ApiClient apiClient;
  final String tenantId;
  final String tenantName;

  @override
  State<TenantAnalyticsScreen> createState() => _TenantAnalyticsScreenState();
}

class _TenantAnalyticsScreenState extends State<TenantAnalyticsScreen> {
  TenantAnalytics? _analytics;
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
      final analytics = await widget.apiClient
          .saasTenantAnalytics(widget.session, widget.tenantId);
      if (!mounted) return;
      setState(() {
        _analytics = analytics;
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

  Future<void> _suspend() async {
    final s = AppStrings.of(context);
    final controller = TextEditingController();
    await showDialog<void>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(s.suspendTenant),
        content: TextField(
          controller: controller,
          decoration: InputDecoration(
            hintText: s.suspendTenantHint,
          ),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx), child: Text(s.cancel)),
          FilledButton(
            onPressed: () async {
              Navigator.pop(ctx);
              try {
                await widget.apiClient.platformSuspendTenant(
                  widget.session,
                  widget.tenantId,
                  reason: controller.text,
                );
                if (!mounted) return;
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(
                    content: Text(s.tenantSuspended),
                    behavior: SnackBarBehavior.floating,
                  ),
                );
              } catch (_) {
                if (!mounted) return;
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(
                    content: Text(s.actionFailed),
                    behavior: SnackBarBehavior.floating,
                  ),
                );
              }
            },
            child: Text(s.suspendTenant),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.tenantName),
        actions: [
          IconButton(
            tooltip: s.suspendTenant,
            icon: const Icon(Icons.block),
            onPressed: _suspend,
          ),
        ],
      ),
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
              : _AnalyticsBody(analytics: _analytics!),
    );
  }
}

class _AnalyticsBody extends StatelessWidget {
  const _AnalyticsBody({required this.analytics});

  final TenantAnalytics analytics;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final a = analytics;
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Text('${a.tenant.slug} · ${a.tenant.businessType}',
            style: Theme.of(context).textTheme.bodyMedium),
        const SizedBox(height: 8),
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: [
            Chip(
              label: Text('${s.planLabel} ${a.tenant.plan}'),
              visualDensity: VisualDensity.compact,
            ),
            if (a.tenant.status == 'suspended')
              Chip(
                label: Text(s.suspendedLabel),
                visualDensity: VisualDensity.compact,
                backgroundColor: Theme.of(context).colorScheme.error,
              ),
            Chip(
              label: Text('${s.users} ${a.users}'),
              visualDensity: VisualDensity.compact,
            ),
            Chip(
              label: Text('${s.products} ${a.products}'),
              visualDensity: VisualDensity.compact,
            ),
          ],
        ),
        const SizedBox(height: 20),
        Text(s.todayStats, style: Theme.of(context).textTheme.titleLarge),
        const SizedBox(height: 12),
        Wrap(
          spacing: 12,
          runSpacing: 12,
          children: [
            _StatCard(
                label: s.revenueToday,
                value: s.formatMoney(a.today.revenueMinor, 'EGP')),
            _StatCard(label: s.salesCount, value: '${a.today.salesCount}'),
            _StatCard(
                label: s.avgSale,
                value: s.formatMoney(a.today.avgSaleMinor, 'EGP')),
            _StatCard(label: s.itemsSold, value: '${a.today.itemsSold}'),
          ],
        ),
        if (a.revenueTrend.isNotEmpty) ...[
          const SizedBox(height: 20),
          Text(s.revenueTrend, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 8),
          Card(
            child: Padding(
              padding: const EdgeInsets.all(12),
              child: _RevenueBars(trend: a.revenueTrend),
            ),
          ),
        ],
        if (a.topProducts.isNotEmpty) ...[
          const SizedBox(height: 20),
          Text(s.topProducts, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 8),
          ...a.topProducts.map(
            (p) => Card(
              child: ListTile(
                dense: true,
                leading: const Icon(Icons.inventory_2_outlined),
                title: Text(p.productName),
                subtitle: Text('${p.sku} · ×${p.quantity}'),
                trailing: Text(s.formatMoney(p.revenueMinor, 'EGP')),
              ),
            ),
          ),
        ],
        if (a.recentSales.isNotEmpty) ...[
          const SizedBox(height: 20),
          Text(s.recentSales, style: Theme.of(context).textTheme.titleLarge),
          const SizedBox(height: 8),
          ...a.recentSales.map(
            (r) => Card(
              child: ListTile(
                dense: true,
                leading: const Icon(Icons.receipt_long_outlined),
                title: Text(r.cashier.isEmpty
                    ? r.paymentMethod
                    : '${r.cashier} · ${r.paymentMethod}'),
                subtitle: Text(r.createdAt),
                trailing: Text(s.formatMoney(r.totalMinor, r.currency)),
              ),
            ),
          ),
        ],
      ],
    );
  }
}

class _RevenueBars extends StatelessWidget {
  const _RevenueBars({required this.trend});

  final List<RevenueTrendPoint> trend;

  @override
  Widget build(BuildContext context) {
    final max =
        trend.fold<int>(0, (m, p) => p.revenueMinor > m ? p.revenueMinor : m);
    const barWidth = 34.0;
    return SizedBox(
      height: 140,
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.end,
        children: [
          for (final p in trend)
            SizedBox(
              width: barWidth,
              child: Column(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  if (max > 0)
                    Container(
                      height: 110 * (p.revenueMinor / max),
                      margin: const EdgeInsets.symmetric(horizontal: 2),
                      decoration: BoxDecoration(
                        color: Theme.of(context).colorScheme.primary,
                        borderRadius: BorderRadius.circular(4),
                      ),
                    ),
                  const SizedBox(height: 4),
                  Text(
                    p.day.length >= 5 ? p.day.substring(5) : p.day,
                    style: Theme.of(context).textTheme.labelSmall,
                  ),
                ],
              ),
            ),
        ],
      ),
    );
  }
}

class _StatCard extends StatelessWidget {
  const _StatCard({required this.label, required this.value});

  final String label;
  final String value;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 120,
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(label, style: Theme.of(context).textTheme.bodySmall),
              const SizedBox(height: 6),
              Text(value, style: Theme.of(context).textTheme.titleMedium),
            ],
          ),
        ),
      ),
    );
  }
}