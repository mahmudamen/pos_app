import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:uuid/uuid.dart';

import '../../core/api_client.dart';
import '../../core/payments.dart';
import '../../core/registers.dart';
import '../../core/session_store.dart';
import '../../core/storage/local_database.dart';
import '../../l10n/strings.dart';
import '../customers/customers_screen.dart';
import '../dashboard/dashboard_screen.dart';
import '../sales/sale_history_screen.dart';
import '../settings/settings_screen.dart';
import 'session_screen.dart';

class PosScreen extends StatefulWidget {
  const PosScreen(
      {super.key,
      required this.session,
      required this.apiClient,
      required this.onSignOut,
      this.localDatabase,
      this.onLanguageChanged});

  final Session session;
  final ApiClient apiClient;
  final VoidCallback onSignOut;
  final LocalDatabase? localDatabase;
  final ValueChanged<String>? onLanguageChanged;

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
  List<Category> _categories = [];
  String? _selectedCategoryId;
  bool _loading = true;
  bool _checkingOut = false;
  String? _catalogError;
  TenantSettings _settings = const TenantSettings();
  RegisterSession? _registerSession;
  bool _sessionLoading = true;

  @override
  void initState() {
    super.initState();
    _bootstrap();
  }

  Future<void> _bootstrap() async {
    await _loadData();
    await _loadSettings();
    await _loadRegisterSession();
  }

  Future<void> _loadRegisterSession() async {
    try {
      final current = await widget.apiClient.currentSession(widget.session);
      if (!mounted) return;
      setState(() {
        _registerSession = current;
        _sessionLoading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() => _sessionLoading = false);
    }
  }

  Future<void> _loadSettings() async {
    try {
      final settings = await widget.apiClient.settings(widget.session);
      if (!mounted) return;
      setState(() => _settings = settings);
    } catch (_) {
      // Offline or unavailable: keep the last known / default settings.
    }
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
  }

  int get _total =>
      _cart.fold(0, (sum, line) => sum + line.price * line.quantity);

  Future<void> _loadData() async {
    try {
      final results = await Future.wait([
        widget.apiClient.products(widget.session),
        widget.apiClient.categories(widget.session),
      ]);
      final products = results[0] as List<Product>;
      final categories = results[1] as List<Category>;
      if (widget.localDatabase != null) {
        await widget.localDatabase!.cacheProducts(products);
      }
      if (!mounted) return;
      setState(() {
        _products = products;
        _categories = categories;
        _loading = false;
      });
      unawaited(_replayPending());
    } catch (error) {
      if (widget.localDatabase != null) {
        final cached = await widget.localDatabase!.cachedProducts();
        if (cached.isNotEmpty) {
          if (!mounted) return;
          setState(() {
            _products = cached;
            _loading = false;
            _catalogError = null;
          });
          return;
        }
      }
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

  void _openSettings(BuildContext context) {
    Navigator.of(context)
        .push(
          MaterialPageRoute<void>(
            builder: (_) => SettingsScreen(
              session: widget.session,
              apiClient: widget.apiClient,
              onLanguageChanged: widget.onLanguageChanged,
              onSignOut: widget.onSignOut,
            ),
          ),
        )
        .then((_) {
          if (mounted) _loadSettings();
        });
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final query = _searchController.text.toLowerCase();
    final products = _products
        .where((item) =>
            (_selectedCategoryId == null ||
                item.categoryId == _selectedCategoryId) &&
            (item.name.toLowerCase().contains(query) ||
                item.sku.toLowerCase().contains(query) ||
                item.barcode.toLowerCase().contains(query)))
        .toList();
    return Scaffold(
      appBar: AppBar(
        title: Text(s.checkout),
        actions: [
          IconButton(
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(
                builder: (_) => DashboardScreen(
                  session: widget.session,
                  apiClient: widget.apiClient,
                ),
              ),
            ),
            tooltip: s.dashboard,
            icon: const Icon(Icons.insights),
          ),
          IconButton(
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(
                builder: (_) => SaleHistoryScreen(
                  session: widget.session,
                  apiClient: widget.apiClient,
                ),
              ),
            ),
            tooltip: s.salesHistory,
            icon: const Icon(Icons.receipt_long),
          ),
          IconButton(
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute(
                builder: (_) => CustomersScreen(
                  session: widget.session,
                  apiClient: widget.apiClient,
                ),
              ),
            ),
            tooltip: s.customers,
            icon: const Icon(Icons.groups_outlined),
          ),
          Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Center(child: Text(widget.session.displayName))),
          IconButton(
            onPressed: () => _openSettings(context),
            tooltip: s.settings,
            icon: const Icon(Icons.settings_outlined),
          ),
        ],
      ),
      body: Column(
        children: [
          _SessionBar(
            session: _registerSession,
            loading: _sessionLoading,
            strings: s,
            currency: widget.session.currencyCode,
            onOpen: () => _openSessionFlow(context),
            onFinish: () => _finishSessionFlow(context),
            onHistory: () => _openSessionHistory(context),
          ),
          Expanded(
            child: LayoutBuilder(builder: (context, constraints) {
              final wide = constraints.maxWidth >= 800;
              final catalog = _Catalog(
                  products: products,
                  categories: _categories,
                  selectedCategoryId: _selectedCategoryId,
                  loading: _loading,
                  error: _catalogError,
                  searchController: _searchController,
                  strings: s,
                  currency: widget.session.currencyCode,
                  showStockBadges: _settings.showStockBadges,
                  onChanged: (_) => setState(() {}),
                  onCategorySelected: (id) =>
                      setState(() => _selectedCategoryId = id),
                  onAdd: _add);
              final cart = _CartPanel(
                  cart: _cart,
                  total: _total,
                  strings: s,
                  currency: widget.session.currencyCode,
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
          ),
        ],
      ),
    );
  }

  void _openSessionHistory(BuildContext context) {
    Navigator.of(context)
        .push(
          MaterialPageRoute<void>(
            builder: (_) => SessionHistoryScreen(
              session: widget.session,
              apiClient: widget.apiClient,
            ),
          ),
        )
        .then((_) {
          if (mounted) _loadRegisterSession();
        });
  }

  Future<void> _openSessionFlow(BuildContext context) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final controller = TextEditingController(text: '0');
    final openingMinor = await showDialog<int>(
      context: context,
      builder: (_) => AlertDialog(
        title: Text(s.openSessionTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(s.openSessionHint),
            const SizedBox(height: 16),
            TextField(
              controller: controller,
              autofocus: true,
              keyboardType:
                  const TextInputType.numberWithOptions(decimal: true),
              decoration: InputDecoration(
                labelText: s.startingCash,
                border: const OutlineInputBorder(),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            child: Text(s.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(
              minorFromInput(controller.text) ?? 0,
            ),
            child: Text(s.openSession),
          ),
        ],
      ),
    );
    if (openingMinor == null || !mounted) return;
    try {
      final opened = await widget.apiClient.openSession(
        widget.session,
        openingCashMinor: openingMinor,
      );
      if (!mounted) return;
      setState(() => _registerSession = opened);
      messenger.showSnackBar(
          SnackBar(content: Text('${s.posSession} ${s.statusOpen}')));
    } catch (error) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  Future<void> _finishSessionFlow(BuildContext context) async {
    final s = AppStrings.of(context);
    final session = _registerSession;
    if (session == null) return;
    final messenger = ScaffoldMessenger.of(context);
    final controller = TextEditingController();
    final countedMinor = await showDialog<int>(
      context: context,
      builder: (_) => AlertDialog(
        title: Text(s.finishSession),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(s.finishSessionConfirm),
            const SizedBox(height: 16),
            TextField(
              controller: controller,
              autofocus: true,
              keyboardType:
                  const TextInputType.numberWithOptions(decimal: true),
              decoration: InputDecoration(
                labelText: s.countedCash,
                hintText: s.countedCashHint,
                border: const OutlineInputBorder(),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            child: Text(s.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(
              minorFromInput(controller.text),
            ),
            child: Text(s.finishSession),
          ),
        ],
      ),
    );
    if (countedMinor == null || !mounted) return;
    try {
      final closed = await widget.apiClient.closeSession(
        widget.session,
        session.id,
        closingCashMinor: countedMinor,
      );
      if (!context.mounted) return;
      setState(() => _registerSession = null);
      messenger.showSnackBar(
          SnackBar(content: Text('${s.zReport} ${s.statusClosed}')));
      await Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) => SessionReportScreen(
            sessionData: closed,
            apiClient: widget.apiClient,
          ),
        ),
      );
    } catch (error) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(error.toString())));
    }
  }

  Future<void> _checkout(BuildContext context) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final payments = await showModalBottomSheet<List<PaymentInput>>(
      context: context,
      isScrollControlled: true,
      builder: (_) => PaymentSheet(
        strings: s,
        currency: widget.session.currencyCode,
        totalMinor: _total,
        defaultMethod: _settings.defaultPaymentMethod,
        receiptFooter: _settings.receiptFooter,
      ),
    );
    if (payments == null || !mounted) return;
    final idempotencyKey = const Uuid().v4();
    final items = _cart
        .map((line) => SaleItemInput(
              productId: line.productId,
              quantity: line.quantity,
            ))
        .toList();
    setState(() => _checkingOut = true);
    try {
      final result = await widget.apiClient.createSale(
        widget.session,
        items,
        idempotencyKey: idempotencyKey,
        payments: payments,
        sessionId: _registerSession?.id,
      );
      if (!mounted) return;
      setState(() => _cart.clear());
      if (_registerSession != null) {
        await _loadRegisterSession();
      }
      messenger.showSnackBar(
        SnackBar(content: Text('${s.saleCompleted}: ${result.id}')),
      );
    } catch (error) {
      if (error is ApiException) {
        if (!mounted) return;
        messenger.showSnackBar(SnackBar(content: Text(error.toString())));
      } else {
        final db = widget.localDatabase;
        if (db != null) {
          await db.enqueue(PendingCommand(
            id: const Uuid().v4(),
            idempotencyKey: idempotencyKey,
            operation: 'create_sale',
            payload: jsonEncode({
              'items': items
                  .map((item) => {
                        'product_id': item.productId,
                        'quantity': item.quantity,
                      })
                  .toList(),
              if (payments.isNotEmpty)
                'payments': payments.map((p) => p.toJson()).toList(),
              if (_registerSession != null)
                'session_id': _registerSession!.id,
            }),
          ));
          if (!mounted) return;
          setState(() => _cart.clear());
          messenger.showSnackBar(
            SnackBar(content: Text(s.offlineSaved)),
          );
        } else {
          if (!mounted) return;
          messenger.showSnackBar(SnackBar(content: Text(error.toString())));
        }
      }
    } finally {
      if (mounted) setState(() => _checkingOut = false);
    }
  }

  Future<void> _replayPending() async {
    final db = widget.localDatabase;
    if (db == null) return;
    final pending = await db.pendingCommands();
    final queued = pending.where((c) => c.operation == 'create_sale').toList();
    if (queued.isEmpty) return;
    var synced = 0;
    for (final command in queued) {
      try {
        final payload = jsonDecode(command.payload) as Map<String, dynamic>;
        final items = (payload['items'] as List)
            .map((item) => SaleItemInput(
                  productId: (item as Map)['product_id'] as String,
                  quantity: item['quantity'] as int,
                ))
            .toList();
        final payments = (payload['payments'] as List?)
            ?.map((p) => PaymentInput.fromJson(p as Map<String, dynamic>))
            .toList();
        await widget.apiClient.createSale(
          widget.session,
          items,
          idempotencyKey: command.idempotencyKey,
          payments: payments,
          sessionId: payload['session_id'] as String?,
        );
        await db.markComplete(command.id);
        synced++;
      } catch (_) {
        break;
      }
    }
    if (synced > 0 && mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(AppStrings.of(context).offlineSynced)),
      );
    }
  }
}

class _Catalog extends StatelessWidget {
  const _Catalog(
      {required this.products,
      required this.categories,
      required this.selectedCategoryId,
      required this.loading,
      required this.error,
      required this.searchController,
      required this.strings,
      required this.currency,
      required this.showStockBadges,
      required this.onChanged,
      required this.onCategorySelected,
      required this.onAdd});
  final List<Product> products;
  final List<Category> categories;
  final String? selectedCategoryId;
  final bool loading;
  final String? error;
  final TextEditingController searchController;
  final AppStrings strings;
  final String currency;
  final bool showStockBadges;
  final ValueChanged<String> onChanged;
  final ValueChanged<String?> onCategorySelected;
  final ValueChanged<Product> onAdd;

  @override
  Widget build(BuildContext context) {
    return Padding(
        padding: const EdgeInsets.all(20),
        child: Column(children: [
          TextField(
              controller: searchController,
              onChanged: onChanged,
              decoration: InputDecoration(
                  prefixIcon: const Icon(Icons.search),
                  hintText: strings.searchProducts,
                  border: const OutlineInputBorder())),
          if (categories.isNotEmpty) ...[
            const SizedBox(height: 12),
            SizedBox(
              height: 36,
              child: ListView(
                scrollDirection: Axis.horizontal,
                children: [
                  Padding(
                    padding: const EdgeInsets.only(right: 8),
                    child: FilterChip(
                      label: Text(strings.all),
                      selected: selectedCategoryId == null,
                      onSelected: (_) => onCategorySelected(null),
                    ),
                  ),
                  ...categories.map((cat) => Padding(
                        padding: const EdgeInsets.only(right: 8),
                        child: FilterChip(
                          label: Text(cat.name),
                          selected: selectedCategoryId == cat.id,
                          onSelected: (_) => onCategorySelected(
                              selectedCategoryId == cat.id ? null : cat.id),
                        ),
                      )),
                ],
              ),
            ),
          ],
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
                                  child: Stack(children: [
                                    if (showStockBadges)
                                      Positioned(
                                        top: 0,
                                        right: 0,
                                        child: _StockBadge(
                                            quantity: product.stockQuantity,
                                            strings: strings),
                                      ),
                                    Column(
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
                                          Text(strings.formatMoney(
                                              product.priceMinor, currency))
                                        ]),
                                  ]))));
                    }))
        ]));
  }
}

class _CartPanel extends StatelessWidget {
  const _CartPanel(
      {required this.cart,
      required this.total,
      required this.strings,
      required this.currency,
      required this.onRemove,
      required this.onCheckout});
  final List<_CartLine> cart;
  final int total;
  final AppStrings strings;
  final String currency;
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
          Text(strings.currentSale,
              style: Theme.of(context).textTheme.headlineSmall),
          const SizedBox(height: 12),
          Expanded(
            child: cart.isEmpty
                ? Center(child: Text(strings.tapToAdd))
                : ListView(
                    children: cart
                        .map(
                          (line) => ListTile(
                            title: Text(line.name),
                            subtitle: Text(
                              '${line.quantity} × ${strings.formatMoney(line.price, currency)}',
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
              Text(strings.total),
              Text(
                strings.formatMoney(total, currency),
                style: Theme.of(context).textTheme.headlineSmall,
              ),
            ],
          ),
          const SizedBox(height: 12),
          FilledButton.icon(
            onPressed: onCheckout,
            icon: const Icon(Icons.payments_outlined),
            label: Text(strings.completeSale),
          ),
        ],
      ),
    );
  }
}

class PaymentSheet extends StatefulWidget {
  const PaymentSheet({
    super.key,
    required this.strings,
    required this.currency,
    required this.totalMinor,
    this.defaultMethod = PaymentMethod.cash,
    this.receiptFooter = '',
  });

  final AppStrings strings;
  final String currency;
  final int totalMinor;
  final PaymentMethod defaultMethod;
  final String receiptFooter;

  @override
  State<PaymentSheet> createState() => _PaymentSheetState();
}

class _PaymentSheetState extends State<PaymentSheet> {
  late final TextEditingController _cashController;
  late final TextEditingController _cardController;
  late final TextEditingController _mobileController;
  String? _error;

  @override
  void initState() {
    super.initState();
    final full = (widget.totalMinor / 100).toStringAsFixed(2);
    _cashController =
        TextEditingController(text: widget.defaultMethod == PaymentMethod.cash ? full : '');
    _cardController =
        TextEditingController(text: widget.defaultMethod == PaymentMethod.card ? full : '');
    _mobileController =
        TextEditingController(text: widget.defaultMethod == PaymentMethod.mobile ? full : '');
  }

  @override
  void dispose() {
    _cashController.dispose();
    _cardController.dispose();
    _mobileController.dispose();
    super.dispose();
  }

  void _confirm() {
    final cashMinor = minorFromInput(_cashController.text);
    final cardMinor = minorFromInput(_cardController.text);
    final mobileMinor = minorFromInput(_mobileController.text);
    if (cashMinor == null || cardMinor == null || mobileMinor == null) {
      setState(() => _error = widget.strings.paymentRequired);
      return;
    }
    try {
      final payments = PaymentSplit.allocate(
        totalMinor: widget.totalMinor,
        parts: {
          PaymentMethod.cash: cashMinor,
          PaymentMethod.card: cardMinor,
          PaymentMethod.mobile: mobileMinor,
        },
      );
      Navigator.of(context).pop(payments);
    } on ArgumentError {
      setState(() => _error = widget.strings.paymentRequired);
    }
  }

  Widget _methodRow(AppStrings s, String label, TextEditingController controller) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Row(
        children: [
          SizedBox(width: 120, child: Text(label)),
          Expanded(
            child: TextField(
              controller: controller,
              keyboardType:
                  const TextInputType.numberWithOptions(decimal: true),
              decoration: const InputDecoration(
                  isDense: true, border: OutlineInputBorder()),
            ),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final s = widget.strings;
    return Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(s.payment,
                  style: Theme.of(context).textTheme.headlineSmall),
              const SizedBox(height: 4),
              Text(s.formatMoney(widget.totalMinor, widget.currency),
                  style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 16),
              _methodRow(s, s.cash, _cashController),
              _methodRow(s, s.card, _cardController),
              _methodRow(s, s.mobilePayment, _mobileController),
              Text(s.remainingLabel,
                  style: Theme.of(context).textTheme.bodySmall),
              if (widget.receiptFooter.isNotEmpty) ...[
                const SizedBox(height: 8),
                Text(widget.receiptFooter,
                    style: Theme.of(context)
                        .textTheme
                        .bodySmall
                        ?.copyWith(fontStyle: FontStyle.italic)),
              ],
              if (_error != null) ...[
                const SizedBox(height: 8),
                Text(_error!,
                    style: TextStyle(
                        color: Theme.of(context).colorScheme.error)),
              ],
              const SizedBox(height: 16),
              FilledButton.icon(
                onPressed: _confirm,
                icon: const Icon(Icons.payments_outlined),
                label: Text(s.confirmPayment),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _SessionBar extends StatelessWidget {
  const _SessionBar({
    required this.session,
    required this.loading,
    required this.strings,
    required this.currency,
    required this.onOpen,
    required this.onFinish,
    required this.onHistory,
  });

  final RegisterSession? session;
  final bool loading;
  final AppStrings strings;
  final String currency;
  final VoidCallback onOpen;
  final VoidCallback onFinish;
  final VoidCallback onHistory;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final current = session;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
      child: loading
          ? const SizedBox(height: 40)
          : Card(
              margin: EdgeInsets.zero,
              child: Padding(
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                child: current == null
                    ? Row(
                        children: [
                          Icon(Icons.play_circle_outline,
                              color: theme.colorScheme.primary),
                          const SizedBox(width: 12),
                          Expanded(
                              child: Text(strings.noOpenSession,
                                  style: theme.textTheme.bodyMedium)),
                          const SizedBox(width: 4),
                          IconButton(
                            onPressed: onHistory,
                            tooltip: strings.sessionsHistory,
                            icon: const Icon(Icons.history),
                          ),
                          FilledButton.icon(
                            onPressed: onOpen,
                            icon: const Icon(Icons.add),
                            label: Text(strings.openSession),
                          ),
                        ],
                      )
                    : Row(
                        children: [
                          Icon(Icons.bolt, color: Colors.green.shade700),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  '${strings.posSession} ${_shortId(current.id)}',
                                  style: theme.textTheme.titleSmall,
                                ),
                                Text(
                                  '${current.openedBy} · ${strings.formatMoney(current.summary.expectedCashMinor, currency)} ${strings.expectedCash}',
                                  style: theme.textTheme.bodySmall,
                                ),
                              ],
                            ),
                          ),
                          const SizedBox(width: 4),
                          IconButton(
                            onPressed: onHistory,
                            tooltip: strings.sessionsHistory,
                            icon: const Icon(Icons.history),
                          ),
                          OutlinedButton.icon(
                            onPressed: onFinish,
                            icon: const Icon(Icons.stop_circle_outlined),
                            label: Text(strings.finishSession),
                          ),
                        ],
                      ),
              ),
            ),
    );
  }

  String _shortId(String id) => id.isEmpty
      ? '—'
      : id.split('-').first.substring(0, 4).toUpperCase();
}

class _StockBadge extends StatelessWidget {
  const _StockBadge({required this.quantity, required this.strings});

  final int quantity;
  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final out = quantity <= 0;
    final background =
        out ? colorScheme.errorContainer : colorScheme.surfaceContainerHighest;
    final foreground =
        out ? colorScheme.onErrorContainer : colorScheme.onSurfaceVariant;
    final label = out ? strings.outOfStock : '$quantity';
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: background,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(out ? Icons.remove_shopping_cart : Icons.inventory_2,
              size: 14, color: foreground),
          const SizedBox(width: 4),
          Text(label,
              style: Theme.of(context)
                  .textTheme
                  .labelSmall
                  ?.copyWith(color: foreground)),
        ],
      ),
    );
  }
}