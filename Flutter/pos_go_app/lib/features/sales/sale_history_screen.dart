import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import '../../core/api_client.dart';
import '../../core/layout.dart';
import '../../core/payments.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';
import '../restaurants/split_bill_screen.dart';
import 'receipt_screen.dart';

class SaleHistoryScreen extends StatefulWidget {
  const SaleHistoryScreen({
    super.key,
    required this.session,
    required this.apiClient,
    this.sessionStore,
  });

  final Session session;
  final ApiClient apiClient;
  final SessionStore? sessionStore;

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

  Future<void> _confirmRefund(SaleSummary sale) async {
    final s = AppStrings.of(context);
    final result = await showDialog<_RefundDialogResult>(
      context: context,
      builder: (_) => _RefundDialog(
        total: sale.totalMinor,
        currencyCode: widget.session.currencyCode,
      ),
    );
    if (result == null) return;
    try {
      await widget.apiClient.refundSale(
        widget.session,
        sale.id,
        reason: result.reason,
        managerPin: result.managerPin,
      );
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(s.refundSuccess)),
      );
      await _loadSales(page: _currentPage);
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('${s.refundFailed}: ${e.message}')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(s.salesHistory)),
      body: MaxWidthBox(
        child: _loading
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
      ),
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
                          if (widget.session.isManager &&
                              sale.status == 'completed')
                            IconButton(
                              tooltip: s.refundSale,
                              icon: const Icon(Icons.undo),
                              onPressed: () => _confirmRefund(sale),
                            ),
                          if (sale.status == 'completed')
                            IconButton(
                              tooltip: s.splitBill,
                              icon: const Icon(Icons.call_split),
                              onPressed: () => Navigator.of(context).push(
                                MaterialPageRoute<void>(
                                  builder: (_) => SplitBillScreen(
                                    session: widget.session,
                                    apiClient: widget.apiClient,
                                    saleId: sale.id,
                                    currencyCode: widget.session.currencyCode,
                                  ),
                                ),
                              ),
                            ),
                          IconButton(
                            tooltip: s.receipt,
                            icon: const Icon(Icons.receipt_long_outlined),
                            onPressed: () async {
                              try {
                                final receipt = await widget.apiClient
                                    .fetchReceipt(widget.session, sale.id);
                                if (!context.mounted) return;
                                await Navigator.of(context).push(
                                  MaterialPageRoute<void>(
                                    builder: (_) => ReceiptScreen(
                                      receipt: receipt,
                                      apiClient: widget.apiClient,
                                      session: widget.session,
                                      sessionStore: widget.sessionStore,
                                    ),
                                  ),
                                );
                              } catch (e) {
                                if (!context.mounted) return;
                                ScaffoldMessenger.of(context).showSnackBar(
                                  SnackBar(content: Text(e.toString())),
                                );
                              }
                            },
                          ),
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

class _RefundDialogResult {
  const _RefundDialogResult({required this.reason, required this.managerPin});

  final String reason;
  final String managerPin;
}

class _RefundDialog extends StatefulWidget {
  const _RefundDialog({required this.total, required this.currencyCode});

  final int total;
  final String currencyCode;

  @override
  State<_RefundDialog> createState() => _RefundDialogState();
}

class _RefundDialogState extends State<_RefundDialog> {
  final _reason = TextEditingController();
  final _pin = TextEditingController();

  @override
  void dispose() {
    _reason.dispose();
    _pin.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final strings = AppStrings.of(context);
    return AlertDialog(
      title: Text(strings.refundSale),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(strings.formatMoney(widget.total, widget.currencyCode)),
            const SizedBox(height: 12),
            TextField(
              controller: _reason,
              maxLength: 255,
              maxLines: 2,
              decoration: InputDecoration(
                labelText: strings.refundReason,
                hintText: strings.refundReasonHint,
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _pin,
              obscureText: true,
              keyboardType: TextInputType.number,
              inputFormatters: [
                FilteringTextInputFormatter.digitsOnly,
                LengthLimitingTextInputFormatter(8),
              ],
              decoration: InputDecoration(
                labelText: strings.refundPin,
                hintText: strings.refundPinHint,
              ),
            ),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(strings.cancel),
        ),
        FilledButton(
          onPressed: () => Navigator.of(context).pop(_RefundDialogResult(
            reason: _reason.text.trim(),
            managerPin: _pin.text.trim(),
          )),
          child: Text(strings.refund),
        ),
      ],
    );
  }
}