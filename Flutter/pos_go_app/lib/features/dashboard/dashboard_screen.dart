import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/dashboard.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

class DashboardScreen extends StatefulWidget {
  const DashboardScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<DashboardScreen> createState() => _DashboardScreenState();
}

class _DashboardScreenState extends State<DashboardScreen> {
  DashboardSummary? _summary;
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
      final summary = await widget.apiClient.dashboardSummary(widget.session);
      if (!mounted) return;
      setState(() {
        _summary = summary;
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

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final summary = _summary;
    return Scaffold(
      appBar: AppBar(title: Text(s.dashboard)),
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
              : RefreshIndicator(
                  onRefresh: _load,
                  child: ListView(
                    physics: const AlwaysScrollableScrollPhysics(),
                    padding: const EdgeInsets.all(16),
                    children: [
                      Text(summary?.date ?? '',
                          style: Theme.of(context).textTheme.titleLarge),
                      const SizedBox(height: 12),
                      if (summary != null) ...[
                        Wrap(
                          spacing: 12,
                          runSpacing: 12,
                          children: [
                            _StatCard(
                                label: s.revenueToday,
                                value: s.formatMoney(summary.today.revenueMinor,
                                    widget.session.currencyCode)),
                            _StatCard(
                              label: s.salesCount,
                              value: '${summary.today.salesCount}',
                            ),
                            _StatCard(
                              label: s.avgSale,
                              value: s.formatMoney(summary.today.avgSaleMinor,
                                  widget.session.currencyCode),
                            ),
                            _StatCard(
                                label: s.itemsSold,
                                value: '${summary.today.itemsSold}'),
                          ],
                        ),
                        if (summary.paymentMix.isNotEmpty) ...[
                          const SizedBox(height: 20),
                          Text(s.paymentMix,
                              style: Theme.of(context).textTheme.titleLarge),
                          const SizedBox(height: 8),
                          ...summary.paymentMix.map((mix) => Card(
                                child: ListTile(
                                  dense: true,
                                  leading: Icon(_methodIcon(mix.method)),
                                  title: Text(s.methodLabel(mix.method)),
                                  trailing: Text(s.formatMoney(mix.amountMinor,
                                      widget.session.currencyCode)),
                                ),
                              )),
                        ],
                        if (summary.topProducts.isNotEmpty) ...[
                          const SizedBox(height: 20),
                          Text(s.topProducts,
                              style: Theme.of(context).textTheme.titleLarge),
                          const SizedBox(height: 8),
                          ...summary.topProducts.map((p) => Card(
                                child: ListTile(
                                  dense: true,
                                  title: Text(p.productName),
                                  subtitle: Text(p.sku),
                                  trailing: Text(
                                      '${p.quantity} · ${s.formatMoney(p.revenueMinor, widget.session.currencyCode)}'),
                                ),
                              )),
                        ],
                        if (summary.perCashier.isNotEmpty) ...[
                          const SizedBox(height: 20),
                          Text(s.perCashier,
                              style: Theme.of(context).textTheme.titleLarge),
                          const SizedBox(height: 8),
                          ...summary.perCashier.map((cs) => Card(
                                child: ListTile(
                                  dense: true,
                                  leading: const Icon(Icons.person_outline),
                                  title: Text(cs.cashier),
                                  subtitle: Text('${cs.salesCount} ${s.salesCount}'),
                                  trailing: Text(s.formatMoney(cs.revenueMinor,
                                      widget.session.currencyCode)),
                                ),
                              )),
                        ],
                        if (summary.recentSales.isNotEmpty) ...[
                          const SizedBox(height: 20),
                          Text(s.recentSales,
                              style: Theme.of(context).textTheme.titleLarge),
                          const SizedBox(height: 8),
                          ...summary.recentSales.map((sale) => Card(
                                child: ListTile(
                                  dense: true,
                                  leading: Icon(_methodIcon(sale.paymentMethod)),
                                  title: Text(s.methodLabel(sale.paymentMethod)),
                                  subtitle: Text(sale.createdAt),
                                  trailing: Text(s.formatMoney(sale.totalMinor,
                                      widget.session.currencyCode)),
                                ),
                              )),
                        ],
                      ],
                    ],
                  ),
                ),
    );
  }
}

IconData _methodIcon(String method) {
  switch (method) {
    case 'card':
      return Icons.credit_card;
    case 'mobile':
      return Icons.phone_android;
    default:
      return Icons.payments_outlined;
  }
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