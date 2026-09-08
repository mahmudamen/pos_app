import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/session_store.dart';

class PosScreen extends StatefulWidget {
  const PosScreen(
      {super.key,
      required this.session,
      required this.apiClient,
      required this.onSignOut});

  final Session session;
  final ApiClient apiClient;
  final VoidCallback onSignOut;

  @override
  State<PosScreen> createState() => _PosScreenState();
}

class _CartLine {
  _CartLine(this.productId, this.name, this.price);
  final String productId;
  final String name;
  final int price;
  int quantity = 1;
}

class _PosScreenState extends State<PosScreen> {
  final _searchController = TextEditingController();
  final List<_CartLine> _cart = [];
  List<Product> _products = [];
  bool _loading = true;
  bool _checkingOut = false;
  String? _catalogError;

  @override
  void initState() {
    super.initState();
    _loadProducts();
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  int get _total =>
      _cart.fold(0, (sum, line) => sum + line.price * line.quantity);

  Future<void> _loadProducts() async {
    try {
      final products = await widget.apiClient.products(widget.session);
      if (!mounted) return;
      setState(() {
        _products = products;
        _loading = false;
      });
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _catalogError = error.toString();
        _loading = false;
      });
    }
  }

  void _add(Product product) {
    setState(() {
      _CartLine? existing;
      for (final line in _cart) {
        if (line.productId == product.id) {
          existing = line;
          break;
        }
      }
      if (existing == null) {
        _cart.add(_CartLine(product.id, product.name, product.priceMinor));
      } else {
        existing.quantity++;
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final query = _searchController.text.toLowerCase();
    final products = _products
        .where((item) =>
            item.name.toLowerCase().contains(query) ||
            item.sku.toLowerCase().contains(query) ||
            item.barcode.toLowerCase().contains(query))
        .toList();
    return Scaffold(
      appBar: AppBar(
        title: const Text('Checkout'),
        actions: [
          Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Center(child: Text(widget.session.displayName))),
          IconButton(
              onPressed: widget.onSignOut,
              tooltip: 'Sign out',
              icon: const Icon(Icons.logout)),
        ],
      ),
      body: LayoutBuilder(builder: (context, constraints) {
        final wide = constraints.maxWidth >= 800;
        final catalog = _Catalog(
            products: products,
            loading: _loading,
            error: _catalogError,
            searchController: _searchController,
            onChanged: (_) => setState(() {}),
            onAdd: _add);
        final cart = _CartPanel(
            cart: _cart,
            total: _total,
            onRemove: (line) => setState(() => _cart.remove(line)),
            onCheckout: _cart.isEmpty || _checkingOut
                ? null
                : () => _checkout(context));
        return wide
            ? Row(children: [
                Expanded(child: catalog),
                SizedBox(width: 360, child: cart)
              ])
            : Column(children: [
                Expanded(child: catalog),
                SizedBox(height: 300, child: cart)
              ]);
      }),
    );
  }

  Future<void> _checkout(BuildContext context) async {
    final messenger = ScaffoldMessenger.of(context);
    setState(() => _checkingOut = true);
    try {
      final result = await widget.apiClient.createSale(
        widget.session,
        _cart
            .map((line) => SaleItemInput(
                  productId: line.productId,
                  quantity: line.quantity,
                ))
            .toList(),
      );
      if (!mounted) return;
      setState(() => _cart.clear());
      messenger.showSnackBar(
        SnackBar(content: Text('Sale completed: ${result.id}')),
      );
    } catch (error) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(error.toString())),
      );
    } finally {
      if (mounted) setState(() => _checkingOut = false);
    }
  }
}

class _Catalog extends StatelessWidget {
  const _Catalog(
      {required this.products,
      required this.loading,
      required this.error,
      required this.searchController,
      required this.onChanged,
      required this.onAdd});
  final List<Product> products;
  final bool loading;
  final String? error;
  final TextEditingController searchController;
  final ValueChanged<String> onChanged;
  final ValueChanged<Product> onAdd;

  @override
  Widget build(BuildContext context) {
    return Padding(
        padding: const EdgeInsets.all(20),
        child: Column(children: [
          TextField(
              controller: searchController,
              onChanged: onChanged,
              decoration: const InputDecoration(
                  prefixIcon: Icon(Icons.search),
                  hintText: 'Search products or scan barcode',
                  border: OutlineInputBorder())),
          const SizedBox(height: 16),
          if (loading)
            const Expanded(child: Center(child: CircularProgressIndicator()))
          else if (error != null)
            Expanded(child: Center(child: Text(error!)))
          else
            Expanded(
                child: GridView.builder(
                    gridDelegate:
                        const SliverGridDelegateWithMaxCrossAxisExtent(
                            maxCrossAxisExtent: 220,
                            childAspectRatio: 1.25,
                            crossAxisSpacing: 12,
                            mainAxisSpacing: 12),
                    itemCount: products.length,
                    itemBuilder: (context, index) {
                      final product = products[index];
                      return Card(
                          child: InkWell(
                              onTap: () => onAdd(product),
                              borderRadius: BorderRadius.circular(12),
                              child: Padding(
                                  padding: const EdgeInsets.all(16),
                                  child: Column(
                                      crossAxisAlignment:
                                          CrossAxisAlignment.start,
                                      mainAxisAlignment:
                                          MainAxisAlignment.spaceBetween,
                                      children: [
                                        Icon(Icons.inventory_2_outlined,
                                            color: Theme.of(context)
                                                .colorScheme
                                                .primary),
                                        Text(product.name,
                                            style: Theme.of(context)
                                                .textTheme
                                                .titleMedium),
                                        Text(
                                            '\$${(product.priceMinor / 100).toStringAsFixed(2)}')
                                      ]))));
                    }))
        ]));
  }
}

class _CartPanel extends StatelessWidget {
  const _CartPanel(
      {required this.cart,
      required this.total,
      required this.onRemove,
      required this.onCheckout});
  final List<_CartLine> cart;
  final int total;
  final ValueChanged<_CartLine> onRemove;
  final VoidCallback? onCheckout;

  @override
  Widget build(BuildContext context) {
    return Container(
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border(left: BorderSide(color: Theme.of(context).dividerColor)),
      ),
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text('Current sale',
              style: Theme.of(context).textTheme.headlineSmall),
          const SizedBox(height: 12),
          Expanded(
            child: cart.isEmpty
                ? const Center(child: Text('Tap a product to add it'))
                : ListView(
                    children: cart
                        .map(
                          (line) => ListTile(
                            title: Text(line.name),
                            subtitle: Text(
                              '${line.quantity} × \$${(line.price / 100).toStringAsFixed(2)}',
                            ),
                            trailing: IconButton(
                              onPressed: () => onRemove(line),
                              icon: const Icon(Icons.delete_outline),
                            ),
                          ),
                        )
                        .toList(),
                  ),
          ),
          const Divider(),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text('Total'),
              Text(
                '\$${(total / 100).toStringAsFixed(2)}',
                style: Theme.of(context).textTheme.headlineSmall,
              ),
            ],
          ),
          const SizedBox(height: 12),
          FilledButton.icon(
            onPressed: onCheckout,
            icon: const Icon(Icons.payments_outlined),
            label: Text(onCheckout == null ? 'Complete sale' : 'Complete sale'),
          ),
        ],
      ),
    );
  }
}
