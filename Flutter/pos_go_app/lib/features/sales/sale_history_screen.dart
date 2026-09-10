import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/payments.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

class SaleHistoryScreen extends StatefulWidget {
  const SaleHistoryScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<SaleHistoryScreen> createState() => _SaleHistoryScreenState();
}

class _SaleHistoryScreenState extends State<SaleHistoryScreen> {
  SalesPage? _page;
  bool _loading = true;
  String? _error;
  int _currentPage = 1;

  @override
  void initState() {
    super.initState();
    _loadSales();
  }

  Future<void> _loadSales({int page = 1}) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result =
          await widget.apiClient.listSales(widget.session, page: page);
      if (!mounted) return;
      setState(() {
        _page = result;
        _currentPage = page;
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
    return Scaffold(
      appBar: AppBar(title: Text(s.salesHistory)),
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
                          onPressed: () => _loadSales(page: _currentPage),
                          child: Text(s.retry)),
                    ],
                  ),
                )
              : _buildList(context),
    );
  }

  Widget _buildList(BuildContext context) {
    final s = AppStrings.of(context);
    final sales = _page?.sales ?? [];
    final total = _page?.total ?? 0;
    final totalPages = (_page?.limit ?? 50) > 0
        ? (total / (_page?.limit ?? 50)).ceil()
        : 1;

    return Column(
      children: [
        Expanded(
          child: sales.isEmpty
              ? Center(child: Text(s.noSales))
              : ListView.separated(
                  itemCount: sales.length,
                  separatorBuilder: (_, __) => const Divider(height: 1),
                  itemBuilder: (context, index) {
                    final sale = sales[index];
                    final date = sale.createdAt.isNotEmpty
                        ? s.formatDate(
                            DateTime.parse(sale.createdAt).toLocal())
                        : '';
                    return ListTile(
                      leading: CircleAvatar(
                        child: Text(sale.status[0].toUpperCase()),
                      ),
                      title: Text(s.formatMoney(
                          sale.totalMinor, widget.session.currencyCode)),
                      subtitle: Text(date),
                      trailing: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Chip(
                            label: Text(
                              switch (sale.paymentMethod) {
                                PaymentMethod.card => s.card,
                                PaymentMethod.mobile => s.mobilePayment,
                                PaymentMethod.cash => s.cash,
                              },
                            ),
                            visualDensity: VisualDensity.compact,
                          ),
                          const SizedBox(width: 4),
                          Chip(
                            label: Text(sale.status),
                            visualDensity: VisualDensity.compact,
                          ),
                        ],
                      ),
                    );
                  },
                ),
        ),
        if (totalPages > 1)
          Padding(
            padding: const EdgeInsets.all(8.0),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                IconButton(
                  onPressed: _currentPage > 1
                      ? () => _loadSales(page: _currentPage - 1)
                      : null,
                  icon: const Icon(Icons.chevron_left),
                ),
                Text('${s.pageOf} $_currentPage / $totalPages'),
                IconButton(
                  onPressed: _currentPage < totalPages
                      ? () => _loadSales(page: _currentPage + 1)
                      : null,
                  icon: const Icon(Icons.chevron_right),
                ),
              ],
            ),
          ),
      ],
    );
  }
}