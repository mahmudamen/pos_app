import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

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
      final results = await Future.wait([
        widget.apiClient.saasSummary(widget.session),
        widget.apiClient.saasTenants(widget.session, limit: 200),
      ]);
      if (!mounted) return;
      setState(() {
        _summary = results[0] as SaasSummary;
        _tenants = results[1] as SaasTenantsPage;
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
    final tenants = _tenants;
    return Scaffold(
      appBar: AppBar(title: Text(s.controlPanel)),
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
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    Text(s.platformSummary,
                        style: Theme.of(context).textTheme.titleLarge),
                    const SizedBox(height: 12),
                    if (summary != null)
                      Wrap(
                        spacing: 12,
                        runSpacing: 12,
                        children: [
                          _StatCard(
                              label: s.tenants,
                              value: '${summary.totalTenants}'),
                          _StatCard(
                              label: s.users, value: '${summary.totalUsers}'),
                          _StatCard(
                              label: s.totalSales,
                              value: '${summary.totalSales}'),
                          _StatCard(
                              label: s.revenue,
                              value: s.formatMoney(summary.revenueMinor,
                                  summary.supportedCurrency)),
                        ],
                      ),
                    if (summary != null && summary.byBusiness.isNotEmpty) ...[
                      const SizedBox(height: 20),
                      Text(s.businessType,
                          style: Theme.of(context).textTheme.titleLarge),
                      const SizedBox(height: 8),
                      ...summary.byBusiness.map((b) => Card(
                            child: ListTile(
                              dense: true,
                              title: Text(b.businessType),
                              trailing: Text('${b.tenants}'),
                            ),
                          )),
                    ],
                    const SizedBox(height: 20),
                    Text(s.tenants,
                        style: Theme.of(context).textTheme.titleLarge),
                    const SizedBox(height: 8),
                    if (tenants == null || tenants.tenants.isEmpty)
                      const SizedBox()
                    else
                      ...tenants.tenants.map((t) => Card(
                            child: ListTile(
                              leading: const Icon(Icons.storefront_outlined),
                              title: Text(t.name),
                              subtitle: Text(
                                  '${t.slug} · ${t.businessType} · ${t.countryCode}/${t.currencyCode}'),
                              isThreeLine: false,
                              trailing: Text(
                                  '${s.users} ${t.users} · ${s.products} ${t.products}'),
                            ),
                          )),
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