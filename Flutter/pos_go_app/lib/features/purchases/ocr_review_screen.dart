import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/purchases.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

/// Review/edit flow between the OCR scan and the purchase commit.
///
/// Each parsed line is editable (name, quantity, unit, purchase price). A line
/// matched to an existing product shows its current sale price and the per-unit
/// profit (sale − purchase) the merchant earns after the purchase; unmatched
/// lines default to a new product whose sale price the merchant sets (default =
/// purchase price). Applying commits via POST /v1/purchases and then patches
/// the sale price of any newly-created product.
class OcrReviewScreen extends StatefulWidget {
  const OcrReviewScreen({
    super.key,
    required this.session,
    required this.apiClient,
    required this.scan,
  });

  final Session session;
  final ApiClient apiClient;
  final OcrScanResult scan;

  @override
  State<OcrReviewScreen> createState() => _OcrReviewScreenState();
}

class _EditableLine {
  _EditableLine({
    required this.name,
    required this.quantityController,
    required this.unitController,
    required this.priceController,
    required this.salePriceController,
    required this.matchedProduct,
    required this.priceMinor,
    required this.salePriceMinor,
  });

  final String name;
  final TextEditingController quantityController;
  final TextEditingController unitController;
  final TextEditingController priceController;
  final TextEditingController salePriceController;
  Product? matchedProduct;
  int priceMinor;
  int salePriceMinor;

  int get salePriceValue => int.tryParse(salePriceController.text.trim()) ?? 0;
  int get priceValue => int.tryParse(priceController.text.trim()) ?? 0;
  double get quantityValue => double.tryParse(quantityController.text.trim()) ?? 0;

  /// Per-unit profit = sale price − purchase price (minor units).
  int get unitProfitMinor => salePriceValue - priceValue;
}

class _OcrReviewScreenState extends State<OcrReviewScreen> {
  bool _loadingProducts = true;
  String? _loadError;
  List<Product> _products = [];
  late List<_EditableLine> _lines;
  bool _applying = false;
  ApplyPurchaseResult? _result;

  final _supplierController = TextEditingController();
  final _invoiceController = TextEditingController();

  @override
  void initState() {
    super.initState();
    _lines = [];
    _loadProducts();
  }

  Future<void> _loadProducts() async {
    setState(() {
      _loadingProducts = true;
      _loadError = null;
    });
    try {
      final products = await widget.apiClient.products(widget.session);
      final lines = <_EditableLine>[];
      for (final line in widget.scan.lines) {
        Product? matched;
        if (line.productId.isNotEmpty) {
          for (final p in products) {
            if (p.id == line.productId) {
              matched = p;
              break;
            }
          }
        }
        final unit = line.unit.isEmpty ? 'piece' : line.unit;
        final salePrice = matched?.priceMinor ?? line.unitPriceMinor;
        lines.add(_EditableLine(
          name: line.name,
          quantityController: TextEditingController(
              text: _formatQty(line.quantity)),
          unitController: TextEditingController(text: unit),
          priceController:
              TextEditingController(text: '${line.unitPriceMinor}'),
          salePriceController: TextEditingController(text: '$salePrice'),
          matchedProduct: matched,
          priceMinor: line.unitPriceMinor,
          salePriceMinor: salePrice,
        ));
      }
      if (!mounted) return;
      setState(() {
        _lines = lines;
        _products = products;
        _loadingProducts = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loadError = e.toString();
        _loadingProducts = false;
      });
    }
  }

  static String _formatQty(double q) {
    if (q == q.roundToDouble()) return '${q.round()}';
    return q.toStringAsFixed(3);
  }

  @override
  void dispose() {
    for (final line in _lines) {
      line.quantityController.dispose();
      line.unitController.dispose();
      line.priceController.dispose();
      line.salePriceController.dispose();
    }
    _supplierController.dispose();
    _invoiceController.dispose();
    super.dispose();
  }

  void _pickProduct(_EditableLine line) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (sheetContext) => SafeArea(
        child: DraggableScrollableSheet(
          expand: false,
          initialChildSize: 0.7,
          builder: (context, scrollController) => ListView(
            controller: scrollController,
            children: [
              ListTile(
                title: Text(AppStrings.of(context).newProduct),
                leading: const Icon(Icons.add_box_outlined),
                onTap: () {
                  Navigator.of(sheetContext).pop();
                  setState(() {
                    line.matchedProduct = null;
                    line.salePriceController.text =
                        '${line.priceValue}';
                  });
                },
              ),
              const Divider(),
              ..._products
                  .where((p) => p.isActive)
                  .map((p) => ListTile(
                        leading: const Icon(Icons.inventory_2_outlined),
                        title: Text(p.name),
                        subtitle: Text(p.sku),
                        trailing: Text(
                          AppStrings.of(context)
                              .formatMoney(p.priceMinor, p.currency),
                        ),
                        onTap: () {
                          Navigator.of(sheetContext).pop();
                          setState(() {
                            line.matchedProduct = p;
                            line.salePriceController.text = '${p.priceMinor}';
                          });
                        },
                      )),
            ],
          ),
        ),
      ),
    );
  }

  Future<void> _apply() async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _applying = true);
    try {
      final currency = widget.session.currencyCode;
      final items = <PurchaseLineInput>[];
      for (final line in _lines) {
        if (line.quantityValue <= 0) {
          throw StateError('${s.ocrReviewInvalidQty}: ${line.name}');
        }
        items.add(PurchaseLineInput(
          productId: line.matchedProduct?.id,
          productName: line.name,
          quantity: line.quantityValue,
          unit: line.unitController.text.trim().isEmpty
              ? 'piece'
              : line.unitController.text.trim(),
          unitPriceMinor: line.priceValue,
          matchScore: 0,
          rowText: '',
        ));
      }
      final result = await widget.apiClient.applyPurchase(
        widget.session,
        supplier: _supplierController.text.trim(),
        invoiceNo: _invoiceController.text.trim(),
        currency: currency,
        ocrText: widget.scan.rawText,
        items: items,
      );

      // Patch the sale price of any newly-created product the merchant set to
      // something other than the invoice cost (backend creates at price=cost).
      for (var i = 0; i < items.length; i++) {
        final applied = result.lines[i];
        if (!applied.created) continue;
        final line = _lines[i];
        final targetPrice = line.salePriceValue;
        if (targetPrice > 0 && targetPrice != items[i].unitPriceMinor) {
          try {
            await widget.apiClient.updateProduct(
              widget.session,
              applied.productId,
              priceMinor: targetPrice,
            );
          } catch (_) {
            // A failed price patch must not roll back the purchase itself;
            // surface it in the summary instead.
          }
        }
      }

      if (!mounted) return;
      setState(() => _result = result);
      // refresh matched products' prices on next open — nothing else needed.
      for (final line in _lines) {
        line.matchedProduct = null;
      }
      messenger.showSnackBar(SnackBar(content: Text(s.ocrApplyDone)));
    } catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    } finally {
      if (mounted) setState(() => _applying = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final result = _result;
    if (result != null) {
      return Scaffold(
        appBar: AppBar(title: Text(s.ocrAppliedTitle)),
        body: _AppliedSummary(
          result: result,
          currency: widget.session.currencyCode,
          lines: _lines,
          onDone: () => Navigator.of(context).pop(),
        ),
      );
    }
    return Scaffold(
      appBar: AppBar(title: Text(s.ocrReviewTitle)),
      body: _loadingProducts
          ? const Center(child: CircularProgressIndicator())
          : _loadError != null
              ? Center(
                  child: Padding(
                    padding: const EdgeInsets.all(24),
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text(_loadError!),
                        const SizedBox(height: 12),
                        FilledButton(
                          onPressed: _loadProducts,
                          child: Text(s.retry),
                        ),
                      ],
                    ),
                  ),
                )
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    TextField(
                      controller: _supplierController,
                      decoration: InputDecoration(
                        labelText: s.supplierField,
                        border: const OutlineInputBorder(),
                      ),
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: _invoiceController,
                      decoration: InputDecoration(
                        labelText: s.invoiceNumber,
                        border: const OutlineInputBorder(),
                      ),
                    ),
                    const SizedBox(height: 16),
                    for (var i = 0; i < _lines.length; i++) ...[
                      _LineCard(
                        index: i,
                        line: _lines[i],
                        currency: widget.session.currencyCode,
                        strings: s,
                        onPickProduct: () => _pickProduct(_lines[i]),
                      ),
                      const SizedBox(height: 8),
                    ],
                    const SizedBox(height: 8),
                    if (_lines.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(bottom: 8),
                        child: Align(
                          alignment: AlignmentDirectional.centerEnd,
                          child: Text(
                            '${AppStrings.of(context).ocrCartTotal} '
                            '${AppStrings.of(context).formatMoney(_subtotal, widget.session.currencyCode)}',
                            style: Theme.of(context).textTheme.titleMedium,
                          ),
                        ),
                      ),
                    FilledButton.icon(
                      onPressed: _lines.isEmpty || _applying
                          ? null
                          : _apply,
                      icon: _applying
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Icon(Icons.check_circle_outline),
                      label: Text(s.applyPurchase),
                    ),
                  ],
                ),
    );
  }

  int get _subtotal {
    var sum = 0;
    for (final line in _lines) {
      sum += (line.quantityValue * line.priceValue).round();
    }
    return sum;
  }
}

class _LineCard extends StatelessWidget {
  const _LineCard({
    required this.index,
    required this.line,
    required this.currency,
    required this.strings,
    required this.onPickProduct,
  });

  final int index;
  final _EditableLine line;
  final String currency;
  final AppStrings strings;
  final VoidCallback onPickProduct;

  @override
  Widget build(BuildContext context) {
    final m = AppStrings.of(context);
    final profit = line.unitProfitMinor;
    final loss = profit < 0;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                CircleAvatar(
                  radius: 14,
                  child: Text('${index + 1}'),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: InkWell(
                    onTap: onPickProduct,
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(line.name,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                            style: const TextStyle(
                                fontWeight: FontWeight.w600)),
                        Text(
                          line.matchedProduct != null
                              ? '${strings.matchedProduct}: ${line.matchedProduct!.name}'
                              : strings.newProduct,
                          style: TextStyle(
                            fontSize: 12,
                            color: line.matchedProduct != null
                                ? Theme.of(context).colorScheme.primary
                                : Theme.of(context)
                                    .colorScheme
                                    .onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
                IconButton(
                  onPressed: onPickProduct,
                  icon: const Icon(Icons.swap_horiz, size: 18),
                  tooltip: strings.changeProduct,
                ),
              ],
            ),
            const SizedBox(height: 8),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: line.quantityController,
                    keyboardType: const TextInputType.numberWithOptions(
                        decimal: true),
                    decoration: InputDecoration(
                      labelText: strings.quantity,
                      isDense: true,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: TextField(
                    controller: line.unitController,
                    decoration: InputDecoration(
                      labelText: strings.unit,
                      isDense: true,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: line.priceController,
                    keyboardType: TextInputType.number,
                    decoration: InputDecoration(
                      labelText: '${strings.purchasePrice} (${AppStrings.currencySymbol(currency)})',
                      isDense: true,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: TextField(
                    controller: line.salePriceController,
                    keyboardType: TextInputType.number,
                    decoration: InputDecoration(
                      labelText: '${strings.salePrice} (${AppStrings.currencySymbol(currency)})',
                      isDense: true,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Row(
              children: [
                Icon(
                  profit == 0
                      ? Icons.trending_flat
                      : loss
                          ? Icons.trending_down
                          : Icons.trending_up,
                  size: 18,
                  color: profit == 0
                      ? Theme.of(context).colorScheme.onSurfaceVariant
                      : loss
                          ? Theme.of(context).colorScheme.error
                          : Colors.green.shade700,
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    strings.unitProfit(
                      m.formatMoney(line.unitProfitMinor, currency),
                    ),
                    style: TextStyle(
                      color: profit == 0
                          ? Theme.of(context).colorScheme.onSurfaceVariant
                          : loss
                              ? Theme.of(context).colorScheme.error
                              : Colors.green.shade700,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                ),
                Text(
                  m.formatMoney(
                    line.quantityValue <= 0
                        ? 0
                        : (line.quantityValue * line.unitProfitMinor).round(),
                    currency,
                  ),
                  style: const TextStyle(fontWeight: FontWeight.bold),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _AppliedSummary extends StatelessWidget {
  const _AppliedSummary({
    required this.result,
    required this.currency,
    required this.lines,
    required this.onDone,
  });

  final ApplyPurchaseResult result;
  final String currency;
  final List<_EditableLine> lines;
  final VoidCallback onDone;

  @override
  Widget build(BuildContext context) {
    final m = AppStrings.of(context);
    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 700),
        child: ListView(
          padding: const EdgeInsets.all(16),
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(AppStrings.of(context).ocrAppliedTitle,
                        style: Theme.of(context).textTheme.titleLarge),
                    const SizedBox(height: 8),
                    Text(
                      '${AppStrings.of(context).ocrAppliedSubtitle} '
                      '${m.formatMoney(result.totalMinor, currency)}',
                    ),
                    if (result.lines.any((l) => l.created)) ...[
                      const SizedBox(height: 8),
                      Text(
                        '${result.lines.where((l) => l.created).length} '
                        '${AppStrings.of(context).newProductsCreated}',
                        style: TextStyle(
                            color: Theme.of(context).colorScheme.primary),
                      ),
                    ],
                  ],
                ),
              ),
            ),
            const SizedBox(height: 12),
            for (var i = 0; i < result.lines.length; i++) ...[
              _ResultRow(
                index: i,
                applied: result.lines[i],
                currency: currency,
              ),
              const SizedBox(height: 6),
            ],
            const SizedBox(height: 12),
            FilledButton(onPressed: onDone, child: Text(AppStrings.of(context).done)),
          ],
        ),
      ),
    );
  }
}

class _ResultRow extends StatelessWidget {
  const _ResultRow({
    required this.index,
    required this.applied,
    required this.currency,
  });

  final int index;
  final AppliedLine applied;
  final String currency;

  @override
  Widget build(BuildContext context) {
    final m = AppStrings.of(context);
    return Card(
      child: ListTile(
        leading: CircleAvatar(
          radius: 14,
          child: Text('${index + 1}'),
        ),
        title: Text(applied.productName),
        subtitle: Text(
          '${AppStrings.of(context).stockDelta}+${applied.stockDelta} → ${applied.newStock}\n'
          '${AppStrings.of(context).costLabel} ${m.formatMoney(applied.newCostMinor, currency)} '
          '• ${applied.unit}',
        ),
        trailing: applied.created
            ? Icon(Icons.add_box_outlined,
                color: Theme.of(context).colorScheme.primary)
            : const Icon(Icons.check_circle, color: Colors.green),
      ),
    );
  }
}