import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:uuid/uuid.dart';

import '../../core/api_client.dart';
import '../../core/focus_mode.dart';
import '../../core/payments.dart';
import '../../core/registers.dart';
import '../../core/security.dart';
import '../../core/session_store.dart';
import '../../core/storage/local_database.dart';
import '../../l10n/strings.dart';
import '../customers/customers_screen.dart';
import '../dashboard/dashboard_screen.dart';
import '../inventory/inventory_screen.dart';
import '../restaurants/split_bill_screen.dart';
import '../restaurants/table_picker_sheet.dart';
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
      this.sessionStore,
      this.onLanguageChanged});

  final Session session;
  final ApiClient apiClient;
  final VoidCallback onSignOut;
  final LocalDatabase? localDatabase;
  final SessionStore? sessionStore;
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

class _OpenOrder {
  _OpenOrder(this.number);
  final int number;
  final List<_CartLine> items = [];
  String? tableId;
  String? tableName;

  int get total =>
      items.fold(0, (sum, line) => sum + line.price * line.quantity);
  bool get isEmpty => items.isEmpty;
}

class _PosScreenState extends State<PosScreen> {
  static const _pendingBatchLimit = 10;
  final _searchController = TextEditingController();
  final List<_OpenOrder> _orders = [];
  int _selectedOrder = 0;
  int _orderCounter = 0;
  List<Product> _products = [];
  List<Category> _categories = [];
  String? _selectedCategoryId;
  bool _loading = true;
  bool _checkingOut = false;
  bool _lowStockOnly = false;
  String? _catalogError;
  TenantSettings _settings = const TenantSettings();
  RegisterSession? _registerSession;
  bool _sessionLoading = true;
  bool _focusMode = false;

  _OpenOrder get _activeOrder => _orders[_selectedOrder];

  @override
  void initState() {
    super.initState();
    _orders.add(_OpenOrder(++_orderCounter));
    _bootstrap();
    _restoreFocusMode();
  }

  Future<void> _restoreFocusMode() async {
    final enabled = await widget.sessionStore?.readFocusMode() ?? false;
    if (!mounted || !enabled) return;
    await FocusMode.apply(enabled);
    if (mounted) setState(() => _focusMode = enabled);
  }

  Future<void> _toggleFocusMode(BuildContext context) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final enabled = !_focusMode;
    setState(() => _focusMode = enabled);
    await widget.sessionStore?.saveFocusMode(enabled);
    final dndActive = await FocusMode.apply(enabled);
    if (!mounted) return;
    if (enabled) {
      messenger.showSnackBar(SnackBar(
        content: Text(dndActive ? s.focusModeOn : s.focusModeDndHint),
        action: dndActive
            ? null
            : SnackBarAction(
                label: s.focusModeGrant,
                onPressed: FocusMode.openDndSettings,
              ),
        duration: const Duration(seconds: 4),
      ));
    } else {
      messenger.showSnackBar(SnackBar(content: Text(s.focusModeOff)));
    }
  }

  @override
  void dispose() {
    _searchController.dispose();
    super.dispose();
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

  int get _total => _activeOrder.total;

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

  bool _isLowStock(Product product) =>
      _settings.lowStockWarning &&
      product.stockQuantity >= 0 &&
      product.stockQuantity <= _settings.lowStockThreshold;

  void _add(Product product) {
    final s = AppStrings.of(context);
    if (_settings.blockOutOfStock && product.stockQuantity <= 0) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.blockedOutOfStock)));
      return;
    }
    setState(() {
      _CartLine? existing;
      for (final line in _activeOrder.items) {
        if (line.productId == product.id) {
          existing = line;
          break;
        }
      }
      if (existing == null) {
        _activeOrder.items
            .add(_CartLine(product.id, product.name, product.priceMinor));
      } else {
        _bumpQuantity(existing, product, s);
      }
    });
  }

  void _bumpQuantity(_CartLine line, Product product, AppStrings s) {
    if (_settings.blockOutOfStock &&
        product.stockQuantity >= 0 &&
        line.quantity >= product.stockQuantity) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.maxStockReached)));
      return;
    }
    line.quantity++;
  }

  void _increase(_CartLine line) {
    Product? product;
    for (final candidate in _products) {
      if (candidate.id == line.productId) {
        product = candidate;
        break;
      }
    }
    if (product != null) {
      setState(() => _bumpQuantity(line, product!, AppStrings.of(context)));
    } else {
      setState(() => line.quantity++);
    }
  }

  void _decrease(_CartLine line) {
    setState(() {
      if (line.quantity > 1) {
        line.quantity--;
      } else {
        _activeOrder.items.remove(line);
      }
    });
  }

  void _newOrder() {
    setState(() {
      _orders.add(_OpenOrder(++_orderCounter));
      _selectedOrder = _orders.length - 1;
    });
  }

  void _selectOrder(int index) => setState(() => _selectedOrder = index);

  void _closeOrder(int index) {
    setState(() {
      if (_orders.length == 1) {
        final order = _orders[0];
        order.items.clear();
        order.tableId = null;
        order.tableName = null;
        _selectedOrder = 0;
        return;
      }
      _orders.removeAt(index);
      if (_selectedOrder >= _orders.length) {
        _selectedOrder = _orders.length - 1;
      } else if (_selectedOrder >= index && _selectedOrder > 0) {
        _selectedOrder--;
      }
    });
  }

  Future<void> _pickTable(BuildContext context) async {
    final active = _activeOrder;
    final result = await TablePickerSheet.pick(
      context: context,
      apiClient: widget.apiClient,
      session: widget.session,
      currentTableId: active.tableId,
    );
    if (result == null || !mounted) return;
    setState(() {
      if (result.cleared) {
        active.tableId = null;
        active.tableName = null;
      } else if (result.table != null) {
        active.tableId = result.table!.id;
        active.tableName = result.table!.name;
      }
    });
  }

  void _openSplitBill(BuildContext context, String saleId) {
    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => SplitBillScreen(
          session: widget.session,
          apiClient: widget.apiClient,
          saleId: saleId,
          currencyCode: widget.session.currencyCode,
        ),
      ),
    );
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
            (!_lowStockOnly || _isLowStock(item)) &&
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
          if (widget.session.isManager)
            IconButton(
              onPressed: () => Navigator.of(context).push(
                MaterialPageRoute(
                  builder: (_) => InventoryScreen(
                    session: widget.session,
                    apiClient: widget.apiClient,
                  ),
                ),
              ),
              tooltip: s.inventory,
              icon: const Icon(Icons.inventory_2),
            ),
          IconButton(
            onPressed: () => _toggleFocusMode(context),
            tooltip: s.focusMode,
            icon: Icon(_focusMode
                ? Icons.center_focus_weak
                : Icons.center_focus_strong),
          ),
          if (_settings.refreshButton)
            IconButton(
              onPressed: () {
                setState(() => _loading = true);
                _loadData();
              },
              tooltip: s.refresh,
              icon: const Icon(Icons.refresh),
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
                  lowStockOnly: _lowStockOnly,
                  lowStockEnabled: _settings.lowStockWarning,
                  lowStockThreshold: _settings.lowStockThreshold,
                  onToggleLowStock: () =>
                      setState(() => _lowStockOnly = !_lowStockOnly),
                  onChanged: (_) => setState(() {}),
                  onCategorySelected: (id) =>
                      setState(() => _selectedCategoryId = id),
                  onAdd: _add);
              final cart = _CartPanel(
                  orders: _orders,
                  selectedOrder: _selectedOrder,
                  strings: s,
                  currency: widget.session.currencyCode,
                  tableName: _activeOrder.tableName,
                  onPickTable: () => _pickTable(context),
                  onSelect: _selectOrder,
                  onNewOrder: _newOrder,
                  onIncrease: _increase,
                  onDecrease: _decrease,
                  onRemove: (line) => setState(() => _activeOrder.items.remove(line)),
                  onCheckout: _activeOrder.isEmpty || _checkingOut
                      ? null
                      : () => _checkout(context));
              return wide
                  ? Row(children: [
                      Expanded(child: catalog),
                      SizedBox(width: 360, child: cart)
                    ])
                  : Column(children: [
                      Expanded(child: catalog),
                      SizedBox(
                          height: (constraints.maxHeight * 0.42).clamp(360, 520),
                          child: cart)
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
    String managerPin = '';
    if (_settings.managerClosePin && !widget.session.isManagerLevel) {
      final pin =
          await _promptManagerPin(message: s.closeNeedsManagerPin);
      if (pin == null || !context.mounted) return;
      managerPin = pin;
    }
    try {
      final closed = await widget.apiClient.closeSession(
        widget.session,
        session.id,
        closingCashMinor: countedMinor,
        managerPin: managerPin,
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
    final tableId = _activeOrder.tableId;

    if (_settings.validateStockPayment) {
      for (final line in _activeOrder.items) {
        Product? product;
        for (final candidate in _products) {
          if (candidate.id == line.productId) {
            product = candidate;
            break;
          }
        }
        if (product != null &&
            product.stockQuantity >= 0 &&
            line.quantity > product.stockQuantity) {
          messenger.showSnackBar(SnackBar(content: Text(s.stockExceeded)));
          return;
        }
      }
    }

    final sessionPerms = widget.session.permissions;
    final effectiveCap = sessionPerms == null
        ? _settings.maxDiscountPct
        : sessionPerms.effectiveDiscountPct(_settings.maxDiscountPct);
    final sheetResult = await showModalBottomSheet<PaymentSheetResult>(
      context: context,
      isScrollControlled: true,
      builder: (_) => PaymentSheet(
        strings: s,
        currency: widget.session.currencyCode,
        totalMinor: _total,
        defaultMethod: _settings.defaultPaymentMethod,
        receiptFooter: _settings.receiptFooter,
        tableName: _activeOrder.tableName,
        discountMode: _settings.discountMode,
        effectiveMaxDiscountPct: effectiveCap,
        actorIsManagerLevel: widget.session.isManagerLevel,
        managerOverrideEnabled: _settings.managerDiscountOverride,
      ),
    );
    if (sheetResult == null || !mounted) return;

    final discountMinor = sheetResult.discountMinor;
    final decision = sheetResult.decision;
    if (decision != null &&
        (decision.kind == DiscountDecisionKind.invalidNegative ||
            decision.kind == DiscountDecisionKind.invalidAboveTotal ||
            decision.kind == DiscountDecisionKind.prohibited)) {
      messenger.showSnackBar(SnackBar(content: Text(s.discountProhibited)));
      return;
    }

    String managerPin = '';
    if (decision?.kind == DiscountDecisionKind.needsManagerPin) {
      final pin = await _promptManagerPin(message: s.discountNeedsPin);
      if (pin == null || !context.mounted) return;
      managerPin = pin;
    }

    final idempotencyKey = const Uuid().v4();
    final payments = sheetResult.payments;
    final items = _activeOrder.items
        .map((line) => SaleItemInput(
              productId: line.productId,
              quantity: line.quantity,
            ))
        .toList();
    final paidOrderIndex = _selectedOrder;
    setState(() => _checkingOut = true);
    try {
      final result = await widget.apiClient.createSale(
        widget.session,
        items,
        idempotencyKey: idempotencyKey,
        payments: payments,
        sessionId: _registerSession?.id,
        tableId: tableId,
        discountMinor: discountMinor,
        managerPin: managerPin,
      );
      if (!mounted) return;
      _closeOrder(paidOrderIndex);
      if (_registerSession != null) {
        await _loadRegisterSession();
      }
      final notice = decision?.kind == DiscountDecisionKind.capped
          ? s.discountCapped
          : decision?.kind == DiscountDecisionKind.warned
              ? s.discountWarning
              : '${s.saleCompleted}: ${result.id}';
      messenger.showSnackBar(SnackBar(
        content: Text(notice),
        action: tableId != null
            ? SnackBarAction(
                label: s.splitBill,
                onPressed: () => _openSplitBill(context, result.id),
              )
            : null,
      ));
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
              if (tableId != null) 'table_id': tableId,
              if (discountMinor > 0) 'discount_minor': discountMinor,
            }),
          ));
          if (!mounted) return;
          _closeOrder(paidOrderIndex);
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

Future<String?> _promptManagerPin({required String message}) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final controller = TextEditingController();
    String? error;
    while (true) {
      if (!mounted) return null;
      final pin = await showDialog<String>(
        context: context,
        builder: (_) => AlertDialog(
          title: Text(s.managerPin),
          content: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(message, style: Theme.of(context).textTheme.bodyMedium),
              const SizedBox(height: 16),
              _PinTextField(controller: controller, error: error),
            ],
          ),
          actions: [
            TextButton(
              onPressed: () => Navigator.of(context).pop(),
              child: Text(s.cancel),
            ),
            FilledButton(
              onPressed: () => Navigator.of(context).pop(controller.text.trim()),
              child: Text(s.ok),
            ),
          ],
        ),
      );
      if (pin == null || pin.isEmpty || !mounted) return null;
      final PinVerifyResult result;
      try {
        result = await widget.apiClient.verifyPin(widget.session, pin: pin);
      } on ApiException catch (e) {
        if (!mounted) return null;
        messenger.showSnackBar(SnackBar(content: Text(e.toString())));
        return null;
      }
      if (result.locked) {
        if (!mounted) return null;
        messenger.showSnackBar(SnackBar(content: Text(s.pinLocked)));
        return null;
      }
      if (result.valid) return pin;
      error = result.attemptsLeft > 0
          ? s.attemptsLeft(result.attemptsLeft)
          : s.pinWrong;
    }
  }

Future<void> _replayPending() async {
    final db = widget.localDatabase;
    if (db == null) return;
    final pending = await db.pendingCommands();
    final queued = pending.where((c) => c.operation == 'create_sale').toList();
    if (queued.isEmpty) return;
    var synced = 0;
    var attention = 0;
    for (final command in queued) {
      if (synced >= _pendingBatchLimit) break;
      try {
        final payload = jsonDecode(command.payload) as Map<String, dynamic>;
        final commands = <SyncPushCommand>[
          SyncPushCommand(
            commandId: command.id,
            operation: 'sale.create',
            payload: {
              ...payload,
              'idempotency_key': command.idempotencyKey,
            },
          ),
        ];
        final page = await widget.apiClient.syncPush(widget.session, commands);
        final result = page.results.isEmpty ? null : page.results.first;
        if (result != null && result.applied) {
          await db.markComplete(command.id);
          synced++;
        } else if (result != null && result.rejected) {
          await db.markComplete(command.id);
          attention++;
        } else if (result != null && result.conflicted) {
          await db.markComplete(command.id);
          attention++;
        } else {
          break;
        }
      } catch (_) {
        break;
      }
    }
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(attention > 0
              ? AppStrings.of(context).offlineNeedsAttention
              : AppStrings.of(context).offlineSynced),
        ),
      );
    }
  }
}

/// Owns the PIN controller so it is released only when the dialog — and its
/// exit animation — has fully unmounted.
class _PinTextField extends StatefulWidget {
  const _PinTextField({required this.controller, required this.error});

  final TextEditingController controller;
  final String? error;

  @override
  State<_PinTextField> createState() => _PinTextFieldState();
}

class _PinTextFieldState extends State<_PinTextField> {
  @override
  void dispose() {
    widget.controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return TextField(
      controller: widget.controller,
      autofocus: true,
      obscureText: true,
      keyboardType: TextInputType.number,
      maxLength: 8,
      decoration: InputDecoration(
        labelText: s.managerPin,
        hintText: s.managerPinHint,
        counterText: '',
        errorText: widget.error,
        border: const OutlineInputBorder(),
      ),
    );
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
      required this.lowStockOnly,
      required this.lowStockEnabled,
      required this.lowStockThreshold,
      required this.onToggleLowStock,
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
  final bool lowStockOnly;
  final bool lowStockEnabled;
  final int lowStockThreshold;
  final VoidCallback onToggleLowStock;
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
                  if (lowStockEnabled) ...[
                    const SizedBox(width: 12),
                    Padding(
                      padding: const EdgeInsets.only(right: 8),
                      child: FilterChip(
                        avatar: Icon(
                            lowStockOnly
                                ? Icons.filter_alt
                                : Icons.filter_alt_outlined,
                            size: 18),
                        label: Text(strings.lowStockOnly),
                        selected: lowStockOnly,
                        onSelected: (_) => onToggleLowStock(),
                      ),
                    ),
                  ],
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
                                            threshold: lowStockThreshold,
                                            strings: strings),
                                      ),
                                    Column(
                                        crossAxisAlignment:
                                            CrossAxisAlignment.start,
                                        mainAxisAlignment:
                                            MainAxisAlignment.spaceBetween,
                                        children: [
                                          _ProductImage(
                                              imageUrl: product.imageUrl),
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
  const _CartPanel({
    required this.orders,
    required this.selectedOrder,
    required this.strings,
    required this.currency,
    required this.tableName,
    required this.onPickTable,
    required this.onSelect,
    required this.onNewOrder,
    required this.onIncrease,
    required this.onDecrease,
    required this.onRemove,
    required this.onCheckout,
  });
  final List<_OpenOrder> orders;
  final int selectedOrder;
  final AppStrings strings;
  final String currency;
  final String? tableName;
  final VoidCallback onPickTable;
  final ValueChanged<int> onSelect;
  final VoidCallback onNewOrder;
  final ValueChanged<_CartLine> onIncrease;
  final ValueChanged<_CartLine> onDecrease;
  final ValueChanged<_CartLine> onRemove;
  final VoidCallback? onCheckout;

  @override
  Widget build(BuildContext context) {
    final active = orders[selectedOrder];
    return Container(
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border(left: BorderSide(color: Theme.of(context).dividerColor)),
      ),
      padding: const EdgeInsets.all(20),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: Row(
              children: [
                for (var i = 0; i < orders.length; i++)
                  Padding(
                    padding: const EdgeInsets.only(right: 6),
                    child: ChoiceChip(
                      label: Text(strings.orderLabel(orders[i].number)),
                      selected: i == selectedOrder,
                      onSelected: (_) => onSelect(i),
                      avatar: orders[i].items.isNotEmpty
                          ? CircleAvatar(
                              radius: 12,
                              child: Text('${orders[i].items.fold<int>(0, (s, l) => s + l.quantity)}',
                                  style: const TextStyle(fontSize: 11)),
                            )
                          : null,
                    ),
                  ),
                Padding(
                  padding: const EdgeInsets.only(left: 4),
                  child: ActionChip(
                    avatar: const Icon(Icons.add, size: 18),
                    label: Text(strings.newOrder),
                    onPressed: onNewOrder,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 12),
          Expanded(
            child: active.isEmpty
                ? Center(child: Text(strings.tapToAdd))
                : ListView(
                    children: active.items
                        .map(
                          (line) => ListTile(
                            title: Text(line.name),
                            subtitle: Text(
                              strings.formatMoney(line.price * line.quantity, currency),
                            ),
                            trailing: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                IconButton(
                                  visualDensity: VisualDensity.compact,
                                  onPressed: () => onDecrease(line),
                                  icon: const Icon(Icons.remove_circle_outline, size: 22),
                                ),
                                SizedBox(
                                  width: 28,
                                  child: Text(
                                    '${line.quantity}',
                                    textAlign: TextAlign.center,
                                    style: Theme.of(context).textTheme.titleSmall,
                                  ),
                                ),
                                IconButton(
                                  visualDensity: VisualDensity.compact,
                                  onPressed: () => onIncrease(line),
                                  icon: const Icon(Icons.add_circle_outline, size: 22),
                                ),
                                IconButton(
                                  visualDensity: VisualDensity.compact,
                                  onPressed: () => onRemove(line),
                                  icon: const Icon(Icons.delete_outline, size: 20),
                                ),
                              ],
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
                strings.formatMoney(active.total, currency),
                style: Theme.of(context).textTheme.headlineSmall,
              ),
            ],
          ),
          const SizedBox(height: 12),
          ActionChip(
            avatar: Icon(tableName != null
                ? Icons.table_restaurant
                : Icons.table_restaurant_outlined),
            label: Text(tableName ?? strings.pickTable),
            onPressed: onPickTable,
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
    this.tableName,
    this.discountMode = DiscountMode.cap,
    this.effectiveMaxDiscountPct = 0,
    this.actorIsManagerLevel = false,
    this.managerOverrideEnabled = true,
  });

  final AppStrings strings;
  final String currency;
  final int totalMinor;
  final PaymentMethod defaultMethod;
  final String receiptFooter;
  final String? tableName;
  final DiscountMode discountMode;
  final int effectiveMaxDiscountPct;
  final bool actorIsManagerLevel;
  final bool managerOverrideEnabled;

  @override
  State<PaymentSheet> createState() => _PaymentSheetState();
}

class PaymentSheetResult {
  const PaymentSheetResult({
    required this.payments,
    required this.discountMinor,
    this.decision,
  });

  final List<PaymentInput> payments;

  /// The discount to actually submit (already clamped in cap mode).
  final int discountMinor;

  /// Policy verdict for the entered discount; governs the checkout flow.
  final DiscountDecision? decision;
}

class _PaymentSheetState extends State<PaymentSheet> {
  late final TextEditingController _cashController;
  late final TextEditingController _cardController;
  late final TextEditingController _mobileController;
  late final TextEditingController _tipController;
  late final TextEditingController _discountController;
  String? _error;

  DiscountMode get _mode => widget.discountMode;

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
    _tipController = TextEditingController();
    _discountController = TextEditingController();
  }

  @override
  void dispose() {
    _cashController.dispose();
    _cardController.dispose();
    _mobileController.dispose();
    _tipController.dispose();
    _discountController.dispose();
    super.dispose();
  }

  DiscountDecision _evaluate(int discountMinor) => evaluateDiscount(
        mode: _mode,
        subtotalMinor: widget.totalMinor,
        discountMinor: discountMinor,
        effectiveMaxDiscountPct: widget.effectiveMaxDiscountPct,
        actorIsManagerLevel: widget.actorIsManagerLevel,
        managerOverrideEnabled: widget.managerOverrideEnabled,
      );

  int get _discountMinorInput =>
      minorFromInput(_discountController.text) ?? 0;

  int get _discountDecisionNet =>
      _discountMinorInput > 0 ? _evaluate(_discountMinorInput).effectiveMinor : 0;

  String? get _discountHint {
    if (_discountMinorInput <= 0) return null;
    final decision = _evaluate(_discountMinorInput);
    switch (decision.kind) {
      case DiscountDecisionKind.capped:
        return '${widget.strings.discountCapped} '
            '(${widget.strings.formatMoney(decision.effectiveMinor, widget.currency)})';
      case DiscountDecisionKind.warned:
        return widget.strings.discountWarning;
      case DiscountDecisionKind.needsManagerPin:
        return widget.strings.discountNeedsPin;
      case DiscountDecisionKind.prohibited:
        return widget.strings.discountProhibited;
      case DiscountDecisionKind.invalidAboveTotal:
        return widget.strings.paymentRequired;
      case DiscountDecisionKind.invalidNegative:
      case DiscountDecisionKind.allowed:
        return null;
    }
  }

  void _confirm() {
    final cashMinor = minorFromInput(_cashController.text);
    final cardMinor = minorFromInput(_cardController.text);
    final mobileMinor = minorFromInput(_mobileController.text);
    final tipMinor = minorFromInput(_tipController.text) ?? 0;
    if (cashMinor == null || cardMinor == null || mobileMinor == null) {
      setState(() => _error = widget.strings.paymentRequired);
      return;
    }
    final discountMinor = _discountMinorInput;
    final decision = _evaluate(discountMinor);
    switch (decision.kind) {
      case DiscountDecisionKind.invalidNegative:
      case DiscountDecisionKind.invalidAboveTotal:
        setState(() => _error = widget.strings.paymentRequired);
        return;
      case DiscountDecisionKind.prohibited:
        setState(() => _error = widget.strings.discountProhibited);
        return;
      case DiscountDecisionKind.capped:
      case DiscountDecisionKind.warned:
      case DiscountDecisionKind.needsManagerPin:
      case DiscountDecisionKind.allowed:
        break;
    }
    try {
      var payments = PaymentSplit.allocate(
        totalMinor: widget.totalMinor - decision.effectiveMinor,
        parts: {
          PaymentMethod.cash: cashMinor,
          PaymentMethod.card: cardMinor,
          PaymentMethod.mobile: mobileMinor,
        },
      );
      if (tipMinor > 0 && payments.isNotEmpty) {
        payments = [
          PaymentInput(
            method: payments.first.method,
            amountMinor: payments.first.amountMinor,
            tipMinor: tipMinor,
          ),
          ...payments.skip(1),
        ];
      }
      Navigator.of(context).pop(PaymentSheetResult(
        payments: payments,
        discountMinor: decision.effectiveMinor,
        decision: decision,
      ));
    } on ArgumentError {
      setState(() => _error = widget.strings.paymentRequired);
    }
  }

  Widget _methodRow(AppStrings s, String label, TextEditingController controller,
      {ValueChanged<String>? onChanged}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Row(
        children: [
          SizedBox(width: 120, child: Text(label)),
          Expanded(
            child: TextField(
              controller: controller,
              onChanged: onChanged,
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
              if (widget.tableName != null) ...[
                const SizedBox(height: 4),
                Row(
                  children: [
                    const Icon(Icons.table_restaurant, size: 16),
                    const SizedBox(width: 6),
                    Text(widget.tableName!,
                        style: Theme.of(context).textTheme.bodyMedium),
                  ],
                ),
              ],
              const SizedBox(height: 16),
              _methodRow(s, s.cash, _cashController),
              _methodRow(s, s.card, _cardController),
              _methodRow(s, s.mobilePayment, _mobileController),
              _methodRow(s, s.tip, _tipController,
                  onChanged: (_) => setState(() {})),
              _methodRow(s, s.discountAmount, _discountController,
                  onChanged: (_) => setState(() {})),
              if (_discountHint != null) ...[
                const SizedBox(height: 4),
                Text(_discountHint!,
                    style: Theme.of(context)
                        .textTheme
                        .bodySmall
                        ?.copyWith(color: Theme.of(context).colorScheme.tertiary)),
              ],
              Text(s.remainingLabel,
                  style: Theme.of(context).textTheme.bodySmall),
              Text(s.netTotal,
                  style: Theme.of(context).textTheme.bodySmall),
              Text(
                s.formatMoney(
                    widget.totalMinor - _discountDecisionNet + (minorFromInput(_tipController.text) ?? 0),
                    widget.currency),
                style: Theme.of(context).textTheme.titleMedium,
              ),
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
  const _StockBadge({
    required this.quantity,
    this.threshold = 5,
    required this.strings,
  });

  final int quantity;
  final int threshold;
  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final negative = quantity < 0;
    final out = quantity <= 0;
    final low = !out && quantity <= threshold;
    final (Color background, Color foreground, String label, IconData icon) =
        negative
            ? (colorScheme.errorContainer, colorScheme.onErrorContainer,
                '$quantity', Icons.remove_shopping_cart)
            : out
                ? (colorScheme.errorContainer, colorScheme.onErrorContainer,
                    strings.outOfStock, Icons.remove_shopping_cart)
                : low
                    ? (Colors.amber.shade100, Colors.brown.shade700,
                        '$quantity', Icons.inventory_2)
                    : (colorScheme.surfaceContainerHighest,
                        colorScheme.onSurfaceVariant, '$quantity',
                        Icons.inventory_2);
    return Tooltip(
      message: negative ? strings.backorderHint : '',
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
        decoration: BoxDecoration(
          color: background,
          borderRadius: BorderRadius.circular(999),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 14, color: foreground),
            const SizedBox(width: 4),
            Text(label,
                style: Theme.of(context)
                    .textTheme
                    .labelSmall
                    ?.copyWith(color: foreground)),
          ],
        ),
      ),
    );
  }
}

class _ProductImage extends StatelessWidget {
  const _ProductImage({required this.imageUrl});

  final String imageUrl;

  @override
  Widget build(BuildContext context) {
    if (imageUrl.isEmpty) {
      return _fallback(context);
    }
    return ClipRRect(
      borderRadius: BorderRadius.circular(8),
      child: Image.network(
        imageUrl,
        height: 44,
        width: 44,
        fit: BoxFit.cover,
        errorBuilder: (_, error, stackTrace) => _fallback(context),
        loadingBuilder: (context, child, progress) {
          if (progress == null) return child;
          return const SizedBox(
            height: 44,
            width: 44,
            child: Center(
              child: SizedBox(
                height: 18,
                width: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            ),
          );
        },
      ),
    );
  }

  Widget _fallback(BuildContext cx) {
    return Icon(Icons.inventory_2_outlined,
        size: 32, color: Theme.of(cx).colorScheme.primary);
  }
}