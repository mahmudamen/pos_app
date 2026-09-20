import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/inventory.dart';
import '../../core/layout.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

enum _AdjustType { refill, remove, setOnHand }

class InventoryScreen extends StatefulWidget {
  const InventoryScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<InventoryScreen> createState() => _InventoryScreenState();
}

class _InventoryScreenState extends State<InventoryScreen> {
  bool _loadingProducts = true;
  String? _productsError;
  List<Product> _products = [];

  bool _loadingHistory = true;
  String? _historyError;
  InventoryAdjustmentPage? _history;
  int _historyPage = 1;
  bool _saving = false;
  bool _allowNegativeStock = false;

  @override
  void initState() {
    super.initState();
    _loadSettings();
    _loadProducts();
    _loadHistory();
  }

  Future<void> _loadSettings() async {
    try {
      final settings = await widget.apiClient.settings(widget.session);
      if (!mounted) return;
      setState(() => _allowNegativeStock = settings.allowNegativeStock);
    } catch (_) {
      // Offline or unavailable: keep the default (strict stock).
    }
  }

  Future<void> _loadProducts() async {
    setState(() {
      _loadingProducts = true;
      _productsError = null;
    });
    try {
      final products = await widget.apiClient.products(widget.session);
      if (!mounted) return;
      setState(() {
        _products = products;
        _loadingProducts = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _productsError = e.toString();
        _loadingProducts = false;
      });
    }
  }

  Future<void> _loadHistory({int page = 1}) async {
    setState(() {
      _loadingHistory = true;
      _historyError = null;
    });
    try {
      final result = await widget.apiClient
          .listInventoryAdjustments(widget.session, page: page);
      if (!mounted) return;
      setState(() {
        _history = result;
        _historyPage = page;
        _loadingHistory = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _historyError = e.toString();
        _loadingHistory = false;
      });
    }
  }

  void _openAdjustSheet(Product product) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (_) => Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
        ),
        child: SafeArea(
          child: _AdjustSheet(
            product: product,
            strings: AppStrings.of(context),
            currency: widget.session.currencyCode,
            allowNegativeStock: _allowNegativeStock,
            saving: _saving,
            onSubmit: _adjust,
          ),
        ),
      ),
    );
  }

  Future<void> _adjust(
      Product product, _AdjustType type, int qty, String note) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    if (qty <= 0) {
      messenger.showSnackBar(SnackBar(content: Text(s.invalidQuantity)));
      return;
    }
    int delta;
    String reason;
    switch (type) {
      case _AdjustType.refill:
        delta = qty;
        reason = 'restock';
      case _AdjustType.remove:
        if (qty > product.stockQuantity && !_allowNegativeStock) {
          messenger.showSnackBar(
              SnackBar(content: Text(s.cannotGoNegative)));
          return;
        }
        delta = -qty;
        reason = 'damaged';
      case _AdjustType.setOnHand:
        delta = qty - product.stockQuantity;
        if (delta == 0) {
          messenger.showSnackBar(SnackBar(content: Text(s.invalidQuantity)));
          return;
        }
        reason = 'count';
    }
    setState(() => _saving = true);
    try {
      final result = await widget.apiClient.createInventoryAdjustment(
        widget.session,
        productId: product.id,
        reason: reason,
        quantityDelta: delta,
        note: note,
      );
      if (!mounted) return;
      setState(() {
        _saving = false;
        final index = _products.indexWhere((p) => p.id == product.id);
        if (index >= 0) {
          _products[index] = _copyWithStock(_products[index], result.newStock);
        }
      });
      messenger.showSnackBar(SnackBar(
        content: Text('${s.stockUpdated}: ${s.stockNow} ${result.newStock}'),
      ));
      await _loadHistory(page: 1);
    } catch (e) {
      if (!mounted) return;
      setState(() => _saving = false);
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  Product _copyWithStock(Product p, int stock) => Product(
        id: p.id,
        name: p.name,
        sku: p.sku,
        barcode: p.barcode,
        priceMinor: p.priceMinor,
        currency: p.currency,
        stockQuantity: stock,
        categoryId: p.categoryId,
        costMinor: p.costMinor,
        imageUrl: p.imageUrl,
        isActive: p.isActive,
        selforderEnabled: p.selforderEnabled,
      );

  Future<void> _toggleSelfOrderPublish(Product product, bool value) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    try {
      final updated = await widget.apiClient.updateProduct(
        widget.session,
        product.id,
        selforderEnabled: value,
      );
      if (!mounted) return;
      setState(() {
        final index = _products.indexWhere((p) => p.id == product.id);
        if (index >= 0) _products[index] = updated;
      });
      if (value && updated.stockQuantity <= 0) {
        messenger.showSnackBar(
          SnackBar(content: Text('${s.onQRMenu}: ${s.hiddenOutOfStock}')),
        );
      } else {
        messenger.showSnackBar(SnackBar(
          content: Text(value ? s.onQRMenu : s.notOnQRMenu),
        ));
      }
    } catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return DefaultTabController(
      length: 2,
      child: Scaffold(
        appBar: AppBar(
          title: Text(s.inventory),
          bottom: TabBar(
            tabs: [
              Tab(text: s.adjustStock),
              Tab(text: s.adjustmentsHistory),
            ],
          ),
        ),
        body: TabBarView(
          children: [
            _buildProductsTab(context),
            _buildHistoryTab(context),
          ],
        ),
      ),
    );
  }

  Widget _buildProductsTab(BuildContext context) {
    final s = AppStrings.of(context);
    if (_loadingProducts) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_productsError != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(_productsError!,
                style: TextStyle(color: Theme.of(context).colorScheme.error)),
            const SizedBox(height: 12),
            FilledButton(onPressed: _loadProducts, child: Text(s.retry)),
          ],
        ),
      );
    }
    if (_products.isEmpty) {
      return Center(child: Text(s.products));
    }
    return MaxWidthBox(
      child: ListView.separated(
        itemCount: _products.length,
        separatorBuilder: (_, __) => const Divider(height: 1),
        itemBuilder: (context, index) {
          final product = _products[index];
          final negative = product.stockQuantity < 0;
          final out = product.stockQuantity <= 0;
          return ListTile(
            leading: _ProductThumb(product: product),
            title: Text(product.name),
            subtitle: Text(product.sku),
            trailing: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text('${s.onHand}: ',
                            style: Theme.of(context).textTheme.bodySmall),
                        Text('${product.stockQuantity}',
                            style:
                                Theme.of(context).textTheme.titleMedium?.copyWith(
                                    color: out
                                        ? Theme.of(context).colorScheme.error
                                        : null,
                                    fontWeight: FontWeight.bold)),
                      ],
                    ),
                    const SizedBox(height: 2),
                    Text(
                      negative
                          ? s.backorder
                          : out
                              ? s.outOfStock
                              : s.adjustStock,
                      style: Theme.of(context).textTheme.bodySmall?.copyWith(
                          color: out
                              ? Theme.of(context).colorScheme.error
                              : Theme.of(context).colorScheme.primary),
                    ),
                  ],
                ),
                const SizedBox(width: 4),
                Tooltip(
                  message: product.selforderEnabled
                      ? (out ? '${s.publishForQR}: ${s.hiddenOutOfStock}' : s.onQRMenu)
                      : s.publishForQR,
                  child: Column(
                    mainAxisAlignment: MainAxisAlignment.center,
                    children: [
                      Icon(
                        product.publishedForSelfOrder
                            ? Icons.public
                            : Icons.public_off,
                        size: 18,
                        color: product.publishedForSelfOrder
                            ? Theme.of(context).colorScheme.primary
                            : Theme.of(context).colorScheme.outline,
                      ),
                      Transform.scale(
                        scale: 0.75,
                        child: Switch(
                          value: product.selforderEnabled,
                          onChanged: _saving
                              ? null
                              : (v) => _toggleSelfOrderPublish(product, v),
                        ),
                      ),
                    ],
                  ),
                ),
              ],
            ),
            onTap: () => _openAdjustSheet(product),
          );
        },
      ),
    );
  }

  Widget _buildHistoryTab(BuildContext context) {
    final s = AppStrings.of(context);
    if (_loadingHistory) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_historyError != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(_historyError!,
                style: TextStyle(color: Theme.of(context).colorScheme.error)),
            const SizedBox(height: 12),
            FilledButton(
                onPressed: () => _loadHistory(page: _historyPage),
                child: Text(s.retry)),
          ],
        ),
      );
    }
    final items = _history?.items ?? [];
    if (items.isEmpty) {
      return Center(child: Text(s.noAdjustments));
    }
    final total = _history?.total ?? 0;
    final limit = _history?.limit ?? 50;
    final totalPages = limit > 0 ? (total / limit).ceil() : 1;
    return MaxWidthBox(
      child: Column(
        children: [
          Expanded(
            child: ListView.separated(
              itemCount: items.length,
              separatorBuilder: (_, __) => const Divider(height: 1),
            itemBuilder: (context, index) {
              final item = items[index];
              return ListTile(
                leading: Icon(
                  item.isCredit ? Icons.add_box : Icons.remove_circle_outline,
                  color: item.isCredit
                      ? Theme.of(context).colorScheme.primary
                      : Theme.of(context).colorScheme.error,
                ),
                title: Text(item.productName),
                subtitle: Text(
                    '${_reasonLabel(s, item.reason)} · ${item.createdAt}'),
                trailing: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    Text(
                      item.isCredit
                          ? '+${item.quantityDelta}'
                          : '${item.quantityDelta}',
                      style: Theme.of(context).textTheme.titleMedium?.copyWith(
                            fontWeight: FontWeight.bold,
                            color: item.isCredit
                                ? Theme.of(context).colorScheme.primary
                                : Theme.of(context).colorScheme.error,
                          ),
                    ),
                    if (item.createdBy.isNotEmpty)
                      Text(item.createdBy,
                          style: Theme.of(context).textTheme.bodySmall),
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
                  onPressed: _historyPage > 1
                      ? () => _loadHistory(page: _historyPage - 1)
                      : null,
                  icon: const Icon(Icons.chevron_left),
                ),
                Text('${s.pageOf} $_historyPage / $totalPages'),
                IconButton(
                  onPressed: _historyPage < totalPages
                      ? () => _loadHistory(page: _historyPage + 1)
                      : null,
                  icon: const Icon(Icons.chevron_right),
                ),
              ],
            ),
          ),
      ],
      ),
    );
  }

  String _reasonLabel(AppStrings s, String reason) {
    switch (reason) {
      case 'restock':
        return s.reasonRestock;
      case 'damaged':
        return s.reasonDamaged;
      case 'count':
        return s.reasonCount;
      default:
        return reason;
    }
  }
}

class _ProductThumb extends StatelessWidget {
  const _ProductThumb({required this.product});

  final Product product;

  @override
  Widget build(BuildContext context) {
    if (product.imageUrl.isEmpty) {
      return CircleAvatar(
          radius: 20,
          child: Icon(Icons.inventory_2_outlined,
              color: Theme.of(context).colorScheme.primary));
    }
    return ClipOval(
      child: Image.network(
        product.imageUrl,
        width: 40,
        height: 40,
        fit: BoxFit.cover,
        errorBuilder: (_, __, ___) => CircleAvatar(
          radius: 20,
          child: Icon(Icons.inventory_2_outlined,
              color: Theme.of(context).colorScheme.primary),
        ),
      ),
    );
  }
}

class _AdjustSheet extends StatefulWidget {
  const _AdjustSheet({
    required this.product,
    required this.strings,
    required this.currency,
    required this.allowNegativeStock,
    required this.saving,
    required this.onSubmit,
  });

  final Product product;
  final AppStrings strings;
  final String currency;
  final bool allowNegativeStock;
  final bool saving;
  final Future<void> Function(Product product, _AdjustType type, int qty,
      String note) onSubmit;

  @override
  State<_AdjustSheet> createState() => _AdjustSheetState();
}

class _AdjustSheetState extends State<_AdjustSheet> {
  final TextEditingController _quantity = TextEditingController();
  final TextEditingController _note = TextEditingController();
  _AdjustType _type = _AdjustType.refill;

  @override
  void dispose() {
    _quantity.dispose();
    _note.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = widget.strings;
    final product = widget.product;
    final isRemove = _type == _AdjustType.remove;
    final isSet = _type == _AdjustType.setOnHand;

    return SingleChildScrollView(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 16, 16, 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(s.adjustStock, style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 8),
            Text(
              '${product.name} · ${s.currentStock}: ${product.stockQuantity}',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
            const SizedBox(height: 12),
            SegmentedButton<_AdjustType>(
              segments: [
                ButtonSegment(
                  value: _AdjustType.refill,
                  label: Text(s.refillPurchase),
                  icon: const Icon(Icons.add),
                ),
                ButtonSegment(
                  value: _AdjustType.remove,
                  label: Text(s.removeStock),
                  icon: const Icon(Icons.remove),
                ),
                ButtonSegment(
                  value: _AdjustType.setOnHand,
                  label: Text(s.setOnHandNow),
                  icon: const Icon(Icons.track_changes),
                ),
              ],
              selected: {_type},
              onSelectionChanged: (selection) {
                setState(() => _type = selection.first);
              },
            ),
            const SizedBox(height: 12),
            if (isRemove && widget.allowNegativeStock)
              Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: Text(
                  s.negativeStockAllowed,
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: Theme.of(context).colorScheme.error),
                ),
              ),
            TextField(
              controller: _quantity,
              keyboardType: TextInputType.number,
              decoration: InputDecoration(
                labelText: isRemove
                    ? s.quantityToRemove
                    : isSet
                        ? s.newOnHand
                        : s.quantityToAdd,
                hintText: '${product.stockQuantity}',
                border: const OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _note,
              decoration: InputDecoration(
                labelText: s.adjustmentNote,
                hintText: s.requestNoteHint,
                border: const OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton(
                    onPressed: widget.saving
                        ? null
                        : () => Navigator.of(context).pop(),
                    child: Text(s.cancel)),
                const SizedBox(width: 8),
                FilledButton(
                  onPressed: widget.saving
                      ? null
                      : () {
                          final qty = int.tryParse(_quantity.text.trim());
                          if (qty == null || qty <= 0) {
                            ScaffoldMessenger.of(context).showSnackBar(
                                SnackBar(content: Text(s.invalidQuantity)));
                            return;
                          }
                          final note = _note.text.trim();
                          Navigator.of(context).pop();
                          widget.onSubmit(product, _type, qty, note);
                        },
                  child: Text(widget.saving ? s.saving : s.save),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}