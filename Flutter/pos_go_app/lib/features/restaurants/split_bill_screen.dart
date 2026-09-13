import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/restaurants.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

class SplitBillScreen extends StatefulWidget {
  const SplitBillScreen({
    super.key,
    required this.session,
    required this.apiClient,
    required this.saleId,
    required this.currencyCode,
  });

  final Session session;
  final ApiClient apiClient;
  final String saleId;
  final String currencyCode;

  @override
  State<SplitBillScreen> createState() => _SplitBillScreenState();
}

class _SplitBillScreenState extends State<SplitBillScreen> {
  SaleDetail? _sale;
  String? _error;
  bool _splitting = false;
  int _covers = 2;
  late List<List<int>> _alloc;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _error = null;
      _sale = null;
    });
    try {
      final sale = await widget.apiClient.saleDetail(widget.session, widget.saleId);
      if (!mounted) return;
      setState(() {
        _sale = sale;
        _alloc = List.generate(
          _covers,
          (_) => List<int>.filled(sale.items.length, 0),
        );
        if (sale.items.isNotEmpty) {
          _alloc[0] = sale.items.map((item) => item.quantity).toList();
        }
      });
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = error.toString());
    }
  }

  int _coverTotal(List<int> lineQuantities) {
    final items = _sale?.items ?? const <SaleDetailItem>[];
    var total = 0;
    for (var i = 0; i < items.length; i++) {
      total += items[i].unitPriceMinor * lineQuantities[i];
    }
    return total;
  }

  Future<void> _split() async {
    final s = AppStrings.of(context);
    final sale = _sale;
    if (sale == null) return;
    final messenger = ScaffoldMessenger.of(context);

    for (var i = 0; i < sale.items.length; i++) {
      var sum = 0;
      for (var cover = 0; cover < _covers; cover++) {
        sum += _alloc[cover][i];
      }
      if (sum != sale.items[i].quantity) {
        messenger.showSnackBar(SnackBar(content: Text(s.splitQuantityCheck)));
        return;
      }
    }
    for (var cover = 0; cover < _covers; cover++) {
      if (_alloc[cover].every((qty) => qty == 0)) {
        messenger.showSnackBar(SnackBar(content: Text(s.splitEmptyChild)));
        return;
      }
    }

    final children = <List<SplitBillLine>>[];
    for (var cover = 0; cover < _covers; cover++) {
      final lines = <SplitBillLine>[];
      for (var i = 0; i < sale.items.length; i++) {
        if (_alloc[cover][i] > 0) {
          lines.add(SplitBillLine(
            saleItemId: sale.items[i].id,
            quantity: _alloc[cover][i],
          ));
        }
      }
      children.add(lines);
    }

    setState(() => _splitting = true);
    try {
      final result = await widget.apiClient.splitSale(
        widget.session,
        widget.saleId,
        children,
      );
      if (!mounted) return;
      setState(() => _splitting = false);
      await showDialog<void>(
        context: context,
        builder: (_) => AlertDialog(
          title: Text(s.splitSuccess),
          content: Text(
            '${s.splitSavedAs}: ${result.children.length}\n'
            '${result.children.map((id) => id.length <= 8 ? id : id.substring(0, 8)).join('  •  ')}',
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(),
              child: Text(s.ok),
            ),
          ],
        ),
      );
      if (mounted) Navigator.of(context).pop();
    } catch (error) {
      if (!mounted) return;
      setState(() => _splitting = false);
      messenger.showSnackBar(
        SnackBar(content: Text('${s.splitFailed}: ${error.toString()}')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final sale = _sale;
    return Scaffold(
      appBar: AppBar(title: Text(s.splitBill)),
      body: _error != null
          ? Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(_error!,
                      style:
                          TextStyle(color: Theme.of(context).colorScheme.error)),
                  const SizedBox(height: 12),
                  FilledButton(onPressed: _load, child: Text(s.retry)),
                ],
              ),
            )
          : sale == null
              ? const Center(child: CircularProgressIndicator())
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Text(s.covers,
                            style: Theme.of(context).textTheme.titleMedium),
                        Row(
                          children: [
                            IconButton(
                              key: const Key('cover-minus'),
                              onPressed: _covers > 2
                                  ? () => setState(() {
                                        _covers--;
                                        _alloc.removeLast();
                                      })
                                  : null,
                              icon: const Icon(Icons.remove_circle_outline),
                            ),
                            Text('$_covers',
                                style:
                                    Theme.of(context).textTheme.titleMedium),
                            IconButton(
                              key: const Key('cover-plus'),
                              onPressed: _covers < sale.items.length
                                  ? () => setState(() {
                                        _covers++;
                                        _alloc.add(
                                          List<int>.filled(
                                              sale.items.length, 0),
                                        );
                                      })
                                  : null,
                              icon: const Icon(Icons.add_circle_outline),
                            ),
                          ],
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    for (var cover = 0; cover < _covers; cover++)
                      Card(
                        margin: const EdgeInsets.only(bottom: 12),
                        child: Padding(
                          padding: const EdgeInsets.all(12),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.stretch,
                            children: [
                              Row(
                                mainAxisAlignment:
                                    MainAxisAlignment.spaceBetween,
                                children: [
                                  Text(
                                    '${s.coverLabel} ${cover + 1}',
                                    style: Theme.of(context)
                                        .textTheme
                                        .titleSmall,
                                  ),
                                  Text(
                                    s.formatMoney(_coverTotal(_alloc[cover]),
                                        widget.currencyCode),
                                    style: Theme.of(context)
                                        .textTheme
                                        .titleSmall,
                                  ),
                                ],
                              ),
                              const Divider(height: 16),
                              for (var i = 0; i < sale.items.length; i++)
                                if (sale.items[i].quantity > 0)
                                  Row(
                                    children: [
                                      Expanded(
                                        child: Text(
                                          sale.items[i].productName,
                                          overflow: TextOverflow.ellipsis,
                                        ),
                                      ),
                                      IconButton(
                                        key: Key('cover-$cover-item-$i-minus'),
                                        visualDensity:
                                            VisualDensity.compact,
                                        onPressed: _alloc[cover][i] > 0
                                            ? () => setState(() =>
                                                _alloc[cover][i]--)
                                            : null,
                                        icon: const Icon(
                                            Icons.remove_circle_outline,
                                            size: 22),
                                      ),
                                      SizedBox(
                                        width: 28,
                                        child: Text(
                                          '${_alloc[cover][i]} / ${sale.items[i].quantity}',
                                          textAlign: TextAlign.center,
                                          style: Theme.of(context)
                                              .textTheme
                                              .titleSmall,
                                        ),
                                      ),
                                      IconButton(
                                        key: Key('cover-$cover-item-$i-plus'),
                                        visualDensity:
                                            VisualDensity.compact,
                                        onPressed: _alloc[cover][i] <
                                                sale.items[i].quantity
                                            ? () => setState(() =>
                                                _alloc[cover][i]++)
                                            : null,
                                        icon: const Icon(
                                            Icons.add_circle_outline,
                                            size: 22),
                                      ),
                                    ],
                                  ),
                            ],
                          ),
                        ),
                      ),
                    Text(s.splitHint,
                        style: Theme.of(context).textTheme.bodySmall),
                    const SizedBox(height: 16),
                    FilledButton.icon(
                      key: const Key('split-action'),
                      onPressed: _splitting ? null : _split,
                      icon: _splitting
                          ? const SizedBox(
                              width: 18,
                              height: 18,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.call_split),
                      label: Text(s.splitSaleAction),
                    ),
                  ],
                ),
    );
  }
}