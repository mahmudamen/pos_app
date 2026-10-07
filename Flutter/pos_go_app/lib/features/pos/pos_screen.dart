import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:uuid/uuid.dart';

import '../../core/api_client.dart';
import '../../core/app_exception.dart';
import '../../core/focus_mode.dart';
import '../../core/fonts.dart';
import '../../core/payments.dart';
import '../../core/printer_service.dart';
import '../../core/printers.dart';
import '../../core/registers.dart';
import '../../core/security.dart';
import '../../core/session_store.dart';
import '../../core/theme.dart';
import '../../core/storage/local_database.dart';
import '../../core/tour/tour_controller.dart';
import '../../core/tour/tour_target.dart';
import '../../l10n/strings.dart';
import '../customers/customers_screen.dart';
import '../dashboard/dashboard_screen.dart';
import '../community/community_hub_screen.dart';
import '../inventory/inventory_screen.dart';
import '../onboarding/onboarding_center_screen.dart';
import '../purchases/purchases_screen.dart';
import '../printers/printers_screen.dart';
import '../restaurants/split_bill_screen.dart';
import '../restaurants/table_picker_sheet.dart';
import '../sales/receipt_screen.dart';
import '../sales/sale_history_screen.dart';
import '../selforder/product_requests_screen.dart';
import '../selforder/qr_sheet.dart';
import '../selforder/self_orders_screen.dart';
import '../settings/settings_screen.dart';
import 'pending_sync_screen.dart';
import 'session_screen.dart';

class PosScreen extends StatefulWidget {
  /// Bounds on the two register calls that gate checkout. A stalled link must
  /// not leave the sale button spinning forever; a timed-out sale is safe to
  /// queue because the idempotency key lets the server deduplicate the replay.
  @visibleForTesting
  static Duration saleTimeout = const Duration(seconds: 20);

  @visibleForTesting
  static Duration sessionTimeout = const Duration(seconds: 20);

  const PosScreen(
      {super.key,
      required this.session,
      required this.apiClient,
      required this.onSignOut,
      this.localDatabase,
      this.sessionStore,
      this.fontSetting = const FontSetting(),
      this.onLanguageChanged,
      this.onFontModeChanged,
      this.onFontSizeChanged,
      this.themeSetting = const ThemeSetting(),
      this.onThemePreferenceChanged,
      this.onThemeAccentChanged,
      this.printerService,
      this.tourController});

  final Session session;
  final ApiClient apiClient;
  final VoidCallback onSignOut;
  final LocalDatabase? localDatabase;
  final SessionStore? sessionStore;
  final FontSetting fontSetting;
  final ValueChanged<String>? onLanguageChanged;
  final ValueChanged<FontMode>? onFontModeChanged;
  final ValueChanged<FontSize>? onFontSizeChanged;
  final ThemeSetting themeSetting;
  final ValueChanged<ThemePreference>? onThemePreferenceChanged;
  final ValueChanged<ThemeAccent>? onThemeAccentChanged;
  final PrinterService? printerService;
  final TourController? tourController;

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

  /// The shift running on this device because the server could not be reached.
  /// Mutually exclusive with [_registerSession] once reconciliation succeeds.
  LocalRegisterSession? _localSession;
  PrinterStatus _printerStatus = PrinterStatus.noPrinter;
  late final PrinterService _printerService =
      widget.printerService ?? PrinterService();
  bool _sessionLoading = true;
  bool _focusMode = false;
  bool _sessionMinimized = false;
  bool _sessionResumedAtStart = false;

  _OpenOrder get _activeOrder => _orders[_selectedOrder];

  @override
  void initState() {
    super.initState();
    _orders.add(_OpenOrder(++_orderCounter));
    _bootstrap();
    _restoreFocusMode();
    _restoreSessionMinimized();
    _loadPrinterStatus();
  }

  Future<void> _loadPrinterStatus() async {
    final store = widget.sessionStore;
    if (store == null) return;
    final config = await store.readPrinterConfig() ?? const PrinterConfig();
    if (!mounted) return;
    final status = config.defaultDevice == null
        ? PrinterStatus.noPrinter
        : await _printerService.statusOf(config);
    if (!mounted) return;
    setState(() => _printerStatus = status);
  }

  Future<void> _restoreFocusMode() async {
    final enabled = await widget.sessionStore?.readFocusMode() ?? false;
    if (!mounted || !enabled) return;
    await FocusMode.apply(enabled);
    if (mounted) setState(() => _focusMode = enabled);
  }

  Future<void> _restoreSessionMinimized() async {
    final minimized =
        await widget.sessionStore?.readSessionMinimized() ?? false;
    if (!mounted) return;
    setState(() => _sessionMinimized = minimized);
  }

  Future<void> _toggleSessionMinimized() async {
    final minimized = !_sessionMinimized;
    setState(() => _sessionMinimized = minimized);
    await widget.sessionStore?.saveSessionMinimized(minimized);
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
    await _restoreOrders();
    await _loadData();
    await _loadSettings();
    await _loadRegisterSession(resumedAtStart: true);
  }

  /// Rehydrates the in-progress order(s) so an app restart (or a crash)
  /// cannot drop what the cashier was ringing up. Malformed or empty state is
  /// ignored rather than allowed to block startup.
  Future<void> _restoreOrders() async {
    final store = widget.sessionStore;
    if (store == null) return;
    String? raw;
    try {
      raw = await store.readActiveOrders(widget.session.tenantId);
    } catch (_) {
      return;
    }
    if (raw == null || raw.isEmpty || !mounted) return;
    try {
      final data = jsonDecode(raw) as Map<String, dynamic>;
      final rawOrders = data['orders'];
      if (rawOrders is! List || rawOrders.isEmpty) return;
      final restored = <_OpenOrder>[];
      for (final rawOrder in rawOrders) {
        if (rawOrder is! Map<String, dynamic>) continue;
        final order = _OpenOrder((rawOrder['number'] as num?)?.toInt() ?? 0);
        final items = rawOrder['items'];
        if (items is List) {
          for (final item in items) {
            if (item is! Map<String, dynamic>) continue;
            final line = _CartLine(
              item['product_id'] as String? ?? '',
              item['name'] as String? ?? '',
              (item['price'] as num?)?.toInt() ?? 0,
            );
            line.quantity = (item['quantity'] as num?)?.toInt() ?? 1;
            if (line.productId.isNotEmpty && line.quantity > 0) {
              order.items.add(line);
            }
          }
        }
        order.tableId = rawOrder['table_id'] as String?;
        order.tableName = rawOrder['table_name'] as String?;
        restored.add(order);
      }
      if (restored.isEmpty) return;
      setState(() {
        _orders
          ..clear()
          ..addAll(restored);
        for (final order in _orders) {
          if (order.number > _orderCounter) _orderCounter = order.number;
        }
        var selected = (data['selected'] as num?)?.toInt() ?? 0;
        if (selected < 0 || selected >= _orders.length) selected = 0;
        _selectedOrder = selected;
      });
    } catch (_) {
      // A corrupt snapshot must never stop the app from starting.
    }
  }

  Future<void> _persistOrders() async {
    final store = widget.sessionStore;
    if (store == null) return;
    try {
      await store.saveActiveOrders(
        widget.session.tenantId,
        jsonEncode({
          'counter': _orderCounter,
          'selected': _selectedOrder,
          'orders': _orders
              .map((order) => {
                    'number': order.number,
                    'table_id': order.tableId,
                    'table_name': order.tableName,
                    'items': order.items
                        .map((line) => {
                              'product_id': line.productId,
                              'name': line.name,
                              'price': line.price,
                              'quantity': line.quantity,
                            })
                        .toList(),
                  })
              .toList(),
        }),
      );
    } catch (_) {
      // A cart that cannot be persisted still sells; it just will not
      // survive a restart.
    }
  }

  Future<void> _loadRegisterSession({bool resumedAtStart = false}) async {
    try {
      final current = await widget.apiClient
          .currentSession(widget.session)
          .timeout(PosScreen.sessionTimeout);
      if (!mounted) return;
      setState(() {
        _registerSession = current;
        // A server session wins: it is the one the Z-report will read.
        if (current != null) _localSession = null;
        if (resumedAtStart && current != null) _sessionResumedAtStart = true;
        _sessionLoading = false;
      });
    } catch (error) {
      if (!mounted) return;
      // The network cut or a stalled link: carry on with the shift this
      // device recorded, so the cashier can keep trading instead of being
      // locked out of their till.
      if (isTransportFailure(error) || error is TimeoutException) {
        final local = await widget.localDatabase?.openLocalSession();
        if (!mounted) return;
        setState(() {
          if (local != null) _localSession = local;
          if (resumedAtStart && local != null) _sessionResumedAtStart = true;
          _sessionLoading = false;
        });
        return;
      }
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
        _activeOrder.items.add(_CartLine(
            product.id,
            product.displayName(Localizations.localeOf(context)),
            product.priceMinor));
      } else {
        _bumpQuantity(existing, product, s);
      }
    });
    unawaited(_persistOrders());
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
    unawaited(_persistOrders());
  }

  void _decrease(_CartLine line) {
    setState(() {
      if (line.quantity > 1) {
        line.quantity--;
      } else {
        _activeOrder.items.remove(line);
      }
    });
    unawaited(_persistOrders());
  }

  void _newOrder() {
    setState(() {
      _orders.add(_OpenOrder(++_orderCounter));
      _selectedOrder = _orders.length - 1;
    });
    unawaited(_persistOrders());
  }

  void _selectOrder(int index) {
    setState(() => _selectedOrder = index);
    unawaited(_persistOrders());
  }

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
    unawaited(_persistOrders());
  }

  void _removeLine(_CartLine line) {
    setState(() => _activeOrder.items.remove(line));
    unawaited(_persistOrders());
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
    unawaited(_persistOrders());
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
          sessionStore: widget.sessionStore,
          fontSetting: widget.fontSetting,
          onFontModeChanged: widget.onFontModeChanged,
          onFontSizeChanged: widget.onFontSizeChanged,
          onLanguageChanged: widget.onLanguageChanged,
          themeSetting: widget.themeSetting,
          onThemePreferenceChanged: widget.onThemePreferenceChanged,
          onThemeAccentChanged: widget.onThemeAccentChanged,
          onSignOut: widget.onSignOut,
        ),
      ),
    )
        .then((_) {
      if (mounted) _loadSettings();
    });
  }

  void _openPrinters() {
    Navigator.of(context)
        .push(
      MaterialPageRoute<void>(
        builder: (_) => PrintersScreen(
          session: widget.session,
          sessionStore: widget.sessionStore ?? SessionStore(),
          printerService: _printerService,
        ),
      ),
    )
        .then((_) {
      if (mounted) _loadPrinterStatus();
    });
  }

  /// Best-effort receipt output after a successful sale. Never blocks or fails
  /// the checkout; the printed status is reflected by the app-bar icon.
  Future<void> _printAfterSale(String saleId) async {
    final store = widget.sessionStore;
    if (store == null) return;
    final config = await store.readPrinterConfig() ?? const PrinterConfig();
    if (!config.enabled) return;
    try {
      final outcome = await printSaleReceipt(
        config: config,
        service: _printerService,
        apiClient: widget.apiClient,
        session: widget.session,
        saleId: saleId,
        roleAllowsPrint: widget.session.isManagerLevel,
      );
      if (!mounted || outcome != PrintOutcome.sent) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(AppStrings.of(context).receiptSentToPrinter)),
      );
    } catch (_) {
      // Transport failure is expected when offline; keep the sale moving.
    }
  }

  /// Best-effort on-screen receipt after a sale. A failure here never surfaces:
  /// the sale already exists and the thermal print already ran, so the caller
  /// just falls back to the completion SnackBar. Returns whether the receipt
  /// route was actually opened, which the reprint action reports on.
  Future<bool> _openSaleReceipt(String saleId) async {
    try {
      // Bounded: the checkout stays in its busy state until this returns, so a
      // stalled link must not leave the sale button disabled forever.
      final receipt = await widget.apiClient
          .fetchReceipt(widget.session, saleId)
          .timeout(const Duration(seconds: 8));
      if (!mounted) return false;
      await Navigator.of(context).push(MaterialPageRoute<void>(
        builder: (_) => ReceiptScreen(
          receipt: receipt,
          apiClient: widget.apiClient,
          session: widget.session,
          sessionStore: widget.sessionStore,
          printerService: _printerService,
        ),
      ));
      return true;
    } catch (_) {
      // The receipt is a bonus; never let it fail a completed sale.
      return false;
    }
  }

  /// Remembers the sale so the app bar can reprint it later. Best effort: a
  /// failed write only degrades the reprint to "no receipt yet".
  Future<void> _recordLastSale(String saleId) async {
    final db = widget.localDatabase;
    if (db == null) return;
    try {
      await db.saveLastSale(widget.session.tenantId, saleId);
    } catch (_) {
      // Local persistence is opportunistic; the sale itself already exists.
    }
  }

  /// One-tap reprint of the newest sale completed on this device.
  Future<void> _reprintLastReceipt() async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final db = widget.localDatabase;
    String? saleId;
    if (db != null) {
      try {
        saleId = await db.readLastSale(widget.session.tenantId);
      } catch (_) {
        // Treated as "nothing to reprint" below.
      }
    }
    if (!mounted) return;
    if (saleId == null || saleId.isEmpty) {
      messenger.showSnackBar(SnackBar(content: Text(s.noLastReceipt)));
      return;
    }
    final opened = await _openSaleReceipt(saleId);
    if (!opened && mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(s.lastReceiptUnavailable)));
    }
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
                item.nameAr.toLowerCase().contains(query) ||
                item.sku.toLowerCase().contains(query) ||
                item.barcode.toLowerCase().contains(query)))
        .toList();
    return Scaffold(
      appBar: AppBar(
        title: Text(s.checkout),
        actions: [
          SizedBox(
            width: (MediaQuery.sizeOf(context).width - 130).clamp(200.0, 600.0),
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: Row(
                children: [
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
                    onPressed: () => showModalBottomSheet<void>(
                      context: context,
                      isScrollControlled: true,
                      builder: (_) => SelfOrderQrSheet(
                        session: widget.session,
                        apiClient: widget.apiClient,
                      ),
                    ),
                    tooltip: s.selfOrderQR,
                    icon: const Icon(Icons.qr_code_2),
                  ),
                  IconButton(
                    onPressed: () => Navigator.of(context).push(
                      MaterialPageRoute(
                        builder: (_) => SelfOrdersScreen(
                          session: widget.session,
                          apiClient: widget.apiClient,
                        ),
                      ),
                    ),
                    tooltip: s.selfOrdersQueue,
                    icon: const Icon(Icons.pending_actions),
                  ),
                  IconButton(
                    onPressed: () => Navigator.of(context).push(
                      MaterialPageRoute(
                        builder: (_) => ProductRequestsScreen(
                          session: widget.session,
                          apiClient: widget.apiClient,
                        ),
                      ),
                    ),
                    tooltip: s.productRequests,
                    icon: const Icon(Icons.rate_review_outlined),
                  ),
                  IconButton(
                    onPressed: () => Navigator.of(context).push(
                      MaterialPageRoute(
                        builder: (_) => SaleHistoryScreen(
                          session: widget.session,
                          apiClient: widget.apiClient,
                          sessionStore: widget.sessionStore,
                        ),
                      ),
                    ),
                    tooltip: s.salesHistory,
                    icon: const Icon(Icons.receipt_long),
                  ),
                  if (_settings.reprintLastReceipt)
                    IconButton(
                      onPressed: _reprintLastReceipt,
                      tooltip: s.reprintLastReceipt,
                      icon: const Icon(Icons.print_outlined),
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
                  IconButton(
                    onPressed: () => Navigator.of(context).push(
                      MaterialPageRoute(
                        builder: (_) => CommunityHubScreen(
                          session: widget.session,
                          apiClient: widget.apiClient,
                        ),
                      ),
                    ),
                    tooltip: s.community,
                    icon: const Icon(Icons.forum_outlined),
                  ),
                  if (widget.session.isManager)
                    TourTarget(
                      id: 'inventory_button',
                      child: IconButton(
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
                    ),
                  if (widget.session.isManager)
                    IconButton(
                      onPressed: () => Navigator.of(context).push(
                        MaterialPageRoute(
                          builder: (_) => PurchasesScreen(
                            session: widget.session,
                            apiClient: widget.apiClient,
                          ),
                        ),
                      ),
                      tooltip: s.purchases,
                      icon: const Icon(Icons.document_scanner_outlined),
                    ),
                  IconButton(
                    onPressed: () => _toggleFocusMode(context),
                    tooltip: s.focusMode,
                    icon: Icon(_focusMode
                        ? Icons.center_focus_weak
                        : Icons.center_focus_strong),
                  ),
                  TourTarget(
                    id: 'printers_button',
                    child: IconButton(
                      onPressed: _openPrinters,
                      tooltip: s.printers,
                      icon: Icon(
                        switch (_printerStatus) {
                          PrinterStatus.online => Icons.print,
                          PrinterStatus.offline => Icons.print_disabled,
                          PrinterStatus.noPrinter =>
                            Icons.local_printshop_outlined,
                        },
                        color: _printerStatus == PrinterStatus.offline
                            ? Theme.of(context).colorScheme.error
                            : null,
                      ),
                    ),
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
                    onPressed: widget.tourController == null
                        ? null
                        : () => Navigator.of(context).push(
                              MaterialPageRoute(
                                builder: (_) => OnboardingCenterScreen(
                                  session: widget.session,
                                  apiClient: widget.apiClient,
                                  tourController: widget.tourController!,
                                ),
                              ),
                            ),
                    tooltip: s.tourCenterEntry,
                    icon: const Icon(Icons.explore_outlined),
                  ),
                  TourTarget(
                    id: 'settings_button',
                    child: IconButton(
                      onPressed: () => _openSettings(context),
                      tooltip: s.settings,
                      icon: const Icon(Icons.settings_outlined),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
      body: Column(
        children: [
          TourTarget(
            id: 'session_bar',
            child: _SessionBar(
              session: _registerSession,
              localSession: _localSession,
              loading: _sessionLoading,
              strings: s,
              currency: widget.session.currencyCode,
              minimized: _sessionMinimized,
              resumed: _sessionResumedAtStart,
              onOpen: () => _openSessionFlow(context),
              onFinish: () => _finishSessionFlow(context),
              onHistory: () => _openSessionHistory(context),
              onToggleMinimize: _toggleSessionMinimized,
            ),
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
                  wide: wide,
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
              final cart = TourTarget(
                  id: 'cart_panel',
                  child: _CartPanel(
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
                      onRemove: _removeLine,
                      onCheckout: _activeOrder.isEmpty || _checkingOut
                          ? null
                          : () => _checkout(context)),
                );
              return wide
                  ? Row(children: [
                      Expanded(child: catalog),
                      SizedBox(
                          width:
                              (constraints.maxWidth * 0.30).clamp(360.0, 440.0),
                          child: cart)
                    ])
                  : Column(children: [
                      Expanded(child: catalog),
                      SizedBox(
                          height:
                              (constraints.maxHeight * 0.42).clamp(360, 520),
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
            TourTarget(
              id: 'session_starting_cash_field',
              child: TextField(
                controller: controller,
                autofocus: true,
                keyboardType:
                    const TextInputType.numberWithOptions(decimal: true),
                decoration: InputDecoration(
                  labelText: s.startingCash,
                  border: const OutlineInputBorder(),
                ),
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
      setState(() {
        _registerSession = opened;
        _sessionResumedAtStart = false;
      });
      messenger.showSnackBar(
          SnackBar(content: Text('${s.posSession} ${s.statusOpen}')));
      widget.tourController?.handleBusinessAction('open');
    } catch (error) {
      final db = widget.localDatabase;
      if (!isTransportFailure(error) || db == null) {
        if (!mounted) return;
        messenger.showSnackBar(SnackBar(content: Text(error.toString())));
        return;
      }
      // Offline: take the float on the record and open the shift locally. The
      // cashier can trade all shift; replay opens the server session after.
      final local = LocalRegisterSession.open(
        id: const Uuid().v4(),
        openingCashMinor: openingMinor,
      );
      await db.saveLocalSession(local);
      if (!mounted) return;
      setState(() {
        _localSession = local;
        _sessionResumedAtStart = false;
      });
      messenger.showSnackBar(SnackBar(content: Text(s.sessionOpenedOffline)));
      widget.tourController?.handleBusinessAction('open');
    }
  }

  Future<void> _finishSessionFlow(BuildContext context) async {
    final s = AppStrings.of(context);
    final session = _registerSession;
    final local = _localSession;
    if (session == null && local == null) return;
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
      final pin = await _promptManagerPin(message: s.closeNeedsManagerPin);
      if (pin == null || !context.mounted) return;
      managerPin = pin;
    }
    // A shift taken offline has no server session to close yet. Record the
    // count on the device; reconciliation closes the server session to match
    // once the link is back.
    if (session == null && local != null) {
      final db = widget.localDatabase;
      if (db != null) {
        await db.saveLocalSession(local.closed(countedCashMinor: countedMinor));
        if (!mounted) return;
        setState(() {
          _localSession = null;
          _sessionResumedAtStart = false;
        });
        messenger.showSnackBar(
            SnackBar(content: Text('${s.zReport} ${s.sessionClosedOffline}')));
        return;
      }
    }
    if (session == null) return;
    try {
      final closed = await widget.apiClient.closeSession(
        widget.session,
        session.id,
        closingCashMinor: countedMinor,
        managerPin: managerPin,
      );
      if (!context.mounted) return;
      setState(() {
        _registerSession = null;
        _sessionResumedAtStart = false;
      });
      messenger.showSnackBar(
          SnackBar(content: Text('${s.zReport} ${s.statusClosed}')));
      await Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) => SessionReportScreen(
            sessionData: closed,
            apiClient: widget.apiClient,
            currency: widget.session.currencyCode,
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
        roundingDenom: _settings.roundingDenom,
      ),
    );
    if (sheetResult == null || !mounted) return;

    final discountMinor = sheetResult.discountMinor;
    final roundingMinor = sheetResult.roundingMinor;
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
      final result = await widget.apiClient
          .createSale(
            widget.session,
            items,
            idempotencyKey: idempotencyKey,
            payments: payments,
            sessionId: _registerSession?.id,
            tableId: tableId,
            discountMinor: discountMinor,
            managerPin: managerPin,
            roundingMinor: roundingMinor,
          )
          .timeout(PosScreen.saleTimeout);
      if (!mounted) return;
      _closeOrder(paidOrderIndex);
      if (_registerSession != null) {
        await _loadRegisterSession();
      }
      unawaited(_printAfterSale(result.id));
      await _recordLastSale(result.id);
      await _openSaleReceipt(result.id);
      if (!mounted) return;
      widget.tourController?.handleBusinessAction('sale');
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
      } else if (isTransportFailure(error) || error is TimeoutException) {
        // The network cut — or the link stalled past the bound. Take the sale
        // locally and let the sync push carry it when the link returns.
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
              if (_registerSession != null) 'session_id': _registerSession!.id,
              // A shift that only exists on this device must not claim a
              // server session_id: the server would reject an unknown session
              // and the sale would be lost. The local id travels as its own
              // field and is swapped for the real one during reconciliation.
              if (_localSession != null)
                'local_session_id': _localSession!.id,
              if (tableId != null) 'table_id': tableId,
              if (discountMinor > 0) 'discount_minor': discountMinor,
              if (roundingMinor > 0) 'rounding_minor': roundingMinor,
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
      } else {
        // Not a transport failure and not an API rejection: a bug. Showing it
        // beats queueing a sale the server never accepted.
        if (!mounted) return;
        messenger.showSnackBar(SnackBar(content: Text(error.toString())));
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
              onPressed: () =>
                  Navigator.of(context).pop(controller.text.trim()),
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

  /// Resolves the server session a device-local shift belongs to, opening one
  /// when this is the first time we are back online.
  ///
  /// The pairing is persisted, so a crash mid-replay resumes against the same
  /// session instead of opening a second one for the same float.
  Future<String?> _resolveServerSession(
      LocalDatabase db, LocalRegisterSession local) async {
    final existing = local.serverSessionId;
    if (existing != null) return existing;
    try {
      final opened = await widget.apiClient.openSession(
        widget.session,
        openingCashMinor: local.openingCashMinor,
      );
      await db.saveLocalSession(local.pairedWith(opened.id));
      if (local.isOpen && mounted) {
        // The shift is still running, so it becomes the session the bar shows.
        setState(() {
          _registerSession = opened;
          _localSession = null;
        });
      }
      return opened.id;
    } catch (_) {
      // Best effort: the sales still replay, they just carry no session until
      // the next attempt, which beats stranding them in the queue.
      return null;
    }
  }

  /// Closes the server session for a shift the cashier finished while offline,
  /// using the cash they counted on the device.
  ///
  /// Returns false when the close did not go through — most often the manager
  /// close policy rejecting an unattended call. The shift then stays
  /// unreconciled so it is retried, and the user is told to finish it by hand.
  Future<bool> _closeReconciledSession(
      LocalDatabase db, LocalRegisterSession local, String serverSessionId) async {
    try {
      await widget.apiClient.closeSession(
        widget.session,
        serverSessionId,
        closingCashMinor: local.closingCashMinor,
      );
      await db.saveLocalSession(local.reconciled());
      return true;
    } catch (_) {
      return false;
    }
  }

  Future<void> _replayPending() async {
    final db = widget.localDatabase;
    if (db == null) return;
    final pending = await db.pendingCommands();
    final queued = pending.where((c) => c.operation == 'create_sale').toList();
    List<LocalRegisterSession> localSessions = const [];
    try {
      localSessions = await db.localSessions();
    } catch (_) {
      // A storage hiccup must not stop the sales from reaching the server.
    }
    List<PendingCommand> failed = const [];
    try {
      failed = await db.failedCommands();
    } catch (_) {
      // Surfacing failures is a nicety; replay must proceed without it.
    }
    // A shift finished offline still owes the server a session and a Z-report
    // even when it took no sales at all, so an empty queue is not the end.
    final owesZReport = localSessions.any((s) => !s.isOpen && !s.isReconciled);
    if (queued.isEmpty && !owesZReport) {
      _showReplayNotice(synced: 0, attention: 0, failed: failed);
      return;
    }

    // Group the queue by the shift each sale was taken on. Two offline shifts
    // have to land in two server sessions, replayed in the order they ran.
    final groups = <String?,
        List<({PendingCommand command, Map<String, dynamic> payload})>>{};
    for (final command in queued) {
      final payload = jsonDecode(command.payload) as Map<String, dynamic>;
      final localId = payload.remove('local_session_id') as String?;
      (groups[localId] ??= []).add((command: command, payload: payload));
    }

    final byId = {for (final s in localSessions) s.id: s};

    // Shifts first (chronological), then sales that belong to no shift at all.
    // A shift closed offline is included even with no queued sales, so its
    // Z-report still gets created and closed with the counted cash.
    final shiftIds = localSessions
        .where((s) => !s.isReconciled &&
            (s.serverSessionId == null || groups.containsKey(s.id)))
        .toList()
      ..sort((a, b) => a.openedAt.compareTo(b.openedAt));
    final order = <String?>[
      ...shiftIds.map((s) => s.id),
      if (groups.containsKey(null)) null,
    ];

    var synced = 0;
    var attention = 0;
    for (final localId in order) {
      if (synced >= _pendingBatchLimit) break;

      final group = groups[localId];
      final local = localId == null ? null : byId[localId];
      if ((groups[localId] == null || groups[localId]!.isEmpty) && local == null) continue;

      String? serverSessionId;
      if (local != null) {
        serverSessionId = await _resolveServerSession(db, local);
        if (serverSessionId == null) continue;
      }

      var applied = 0;
      for (final entry in group ?? const []) {
        if (synced + applied >= _pendingBatchLimit) break;
        try {
          final page = await widget.apiClient.syncPush(widget.session, [
            SyncPushCommand(
              commandId: entry.command.id,
              operation: 'sale.create',
              payload: {
                ...entry.payload,
                if (serverSessionId != null) 'session_id': serverSessionId,
                'idempotency_key': entry.command.idempotencyKey,
              },
            ),
          ]);
          final result = page.results.isEmpty ? null : page.results.first;
          if (result != null && result.applied) {
            await db.markComplete(entry.command.id);
            applied++;
          } else if (result != null && (result.rejected || result.conflicted)) {
            // Permanent for now: park it with its reason instead of dropping
            // it, so the failed-sync screen can show and replay it.
            await db.failCommand(entry.command.id, errorCode: result.errorCode);
            attention++;
          } else {
            break;
          }
        } catch (_) {
          break;
        }
      }
      synced += applied;
      final drained = applied == (group?.length ?? 0);
      if (local != null && drained && !local.isOpen && !local.isReconciled) {
        // Only close once every sale of the shift is on the server, otherwise
        // the Z-report would be short.
        if (!await _closeReconciledSession(db, local, serverSessionId!)) {
          attention++;
        }
      }
    }
    // Re-read after the loop: commands may have just moved into `failed`.
    try {
      failed = await db.failedCommands();
    } catch (_) {
      // Surfacing failures is a nicety; the notice must still be shown.
    }
    _showReplayNotice(synced: synced, attention: attention, failed: failed);
  }

  void _showReplayNotice({
    required int synced,
    required int attention,
    required List<PendingCommand> failed,
  }) {
    if (!mounted) return;
    if (synced == 0 && attention == 0 && failed.isEmpty) return;
    final s = AppStrings.of(context);
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(attention > 0 || failed.isNotEmpty
            ? s.offlineNeedsAttention
            : s.offlineSynced),
        action: failed.isEmpty
            ? null
            : SnackBarAction(
                label: s.viewFailed,
                onPressed: _openFailedSync,
              ),
      ),
    );
  }

  Future<void> _openFailedSync() async {
    final db = widget.localDatabase;
    if (db == null || !mounted) return;
    await Navigator.of(context).push(MaterialPageRoute<void>(
      builder: (_) => PendingSyncScreen(
        apiClient: widget.apiClient,
        session: widget.session,
        localDatabase: db,
        onReplay: _replayPending,
      ),
    ));
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
      required this.wide,
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
  final bool wide;
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
                          label: Text(
                              cat.displayName(Localizations.localeOf(context))),
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
                child: TourTarget(
                    id: 'product_grid',
                    child: GridView.builder(
                    gridDelegate: SliverGridDelegateWithMaxCrossAxisExtent(
                        maxCrossAxisExtent: wide ? 280 : 220,
                        childAspectRatio: wide ? 1.2 : 1.25,
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
                                    if (product.isRecentlyAdded(DateTime.now()))
                                      Positioned(
                                        top: 0,
                                        left: 0,
                                        child: _NewBadge(strings: strings),
                                      ),
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
                                          Text(
                                              product.displayName(Localizations
                                                  .localeOf(context)),
                                              maxLines: 1,
                                              overflow: TextOverflow.ellipsis,
                                              style: Theme.of(context)
                                                  .textTheme
                                                  .titleMedium),
                                          Text(strings.formatMoney(
                                              product.priceMinor, currency)),
                                          if (product.unit.isNotEmpty &&
                                              product.unit != 'piece')
                                            Text(
                                                strings.unitLabel(product.unit),
                                                style: Theme.of(context)
                                                    .textTheme
                                                    .bodySmall
                                                    ?.copyWith(
                                                        color: Theme.of(context)
                                                            .colorScheme
                                                            .onSurfaceVariant)),
                                        ]),
                                  ]))));
                    })))
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
    // Android 15 forces edge-to-edge at targetSdk 36 and the opt-out is ignored
    // at that target, so the bottom-pinned checkout button needs the inset
    // itself. `paddingOf` drops to zero with the keyboard up.
    final systemBottom = MediaQuery.paddingOf(context).bottom;
    return Container(
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border(left: BorderSide(color: Theme.of(context).dividerColor)),
      ),
      padding: EdgeInsets.fromLTRB(20, 20, 20, 20 + systemBottom),
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
                              child: Text(
                                  '${orders[i].items.fold<int>(0, (s, l) => s + l.quantity)}',
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
                              strings.formatMoney(
                                  line.price * line.quantity, currency),
                            ),
                            trailing: Row(
                              mainAxisSize: MainAxisSize.min,
                              children: [
                                IconButton(
                                  visualDensity: VisualDensity.compact,
                                  onPressed: () => onDecrease(line),
                                  icon: const Icon(Icons.remove_circle_outline,
                                      size: 22),
                                ),
                                SizedBox(
                                  width: 28,
                                  child: Text(
                                    '${line.quantity}',
                                    textAlign: TextAlign.center,
                                    style:
                                        Theme.of(context).textTheme.titleSmall,
                                  ),
                                ),
                                IconButton(
                                  visualDensity: VisualDensity.compact,
                                  onPressed: () => onIncrease(line),
                                  icon: const Icon(Icons.add_circle_outline,
                                      size: 22),
                                ),
                                IconButton(
                                  visualDensity: VisualDensity.compact,
                                  onPressed: () => onRemove(line),
                                  icon: const Icon(Icons.delete_outline,
                                      size: 20),
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
          TourTarget(
            id: 'checkout_button',
            child: FilledButton.icon(
              onPressed: onCheckout,
              icon: const Icon(Icons.payments_outlined),
              label: Text(strings.completeSale),
            ),
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
    this.roundingDenom = 0,
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

  /// Minor-unit cash denomination the payable is rounded to (0 = off).
  final int roundingDenom;

  @override
  State<PaymentSheet> createState() => _PaymentSheetState();
}

class PaymentSheetResult {
  const PaymentSheetResult({
    required this.payments,
    required this.discountMinor,
    this.decision,
    this.roundingMinor = 0,
  });

  final List<PaymentInput> payments;

  /// The discount to actually submit (already clamped in cap mode).
  final int discountMinor;

  /// Policy verdict for the entered discount; governs the checkout flow.
  final DiscountDecision? decision;

  /// Cash-change rounding delta to submit alongside the sale (0 when off).
  final int roundingMinor;
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
    final payable = widget.totalMinor +
        roundingDelta(widget.roundingDenom, widget.totalMinor);
    final full = (payable / 100).toStringAsFixed(2);
    _cashController = TextEditingController(
        text: widget.defaultMethod == PaymentMethod.cash ? full : '');
    _cardController = TextEditingController(
        text: widget.defaultMethod == PaymentMethod.card ? full : '');
    _mobileController = TextEditingController(
        text: widget.defaultMethod == PaymentMethod.mobile ? full : '');
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

  int get _discountMinorInput => minorFromInput(_discountController.text) ?? 0;

  int get _discountDecisionNet => _discountMinorInput > 0
      ? _evaluate(_discountMinorInput).effectiveMinor
      : 0;

  /// The nominal total after the (policy-applied) discount.
  int get _netTotal => widget.totalMinor - _discountDecisionNet;

  /// Cash-change rounding delta for the discounted total.
  int get _rounding =>
      widget.roundingDenom > 0 ? roundingDelta(widget.roundingDenom, _netTotal) : 0;

  /// The rounded payable the tender lines must sum to.
  int get _payable => _netTotal + _rounding;

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
        totalMinor: _payable,
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
        roundingMinor: _rounding,
      ));
    } on ArgumentError {
      setState(() => _error = widget.strings.paymentRequired);
    }
  }

  Widget _methodRow(
      AppStrings s, String label, TextEditingController controller,
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
      padding:
          EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(20),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Text(s.payment, style: Theme.of(context).textTheme.headlineSmall),
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
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: Theme.of(context).colorScheme.tertiary)),
              ],
              Text(s.remainingLabel,
                  style: Theme.of(context).textTheme.bodySmall),
              if (widget.roundingDenom > 0) ...[
                Text(s.netTotal, style: Theme.of(context).textTheme.bodySmall),
                Text(
                  s.formatMoney(
                      _netTotal + (minorFromInput(_tipController.text) ?? 0),
                      widget.currency),
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
                if (_rounding > 0) ...[
                  const SizedBox(height: 2),
                  Text(
                    '${s.rounding} +${s.formatMoney(_rounding, widget.currency)}',
                    style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: Theme.of(context).colorScheme.tertiary),
                  ),
                ],
                Text(s.payable, style: Theme.of(context).textTheme.bodySmall),
                Text(
                  s.formatMoney(
                      _payable + (minorFromInput(_tipController.text) ?? 0),
                      widget.currency),
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ] else ...[
                Text(s.netTotal, style: Theme.of(context).textTheme.bodySmall),
                Text(
                  s.formatMoney(
                      widget.totalMinor -
                          _discountDecisionNet +
                          (minorFromInput(_tipController.text) ?? 0),
                      widget.currency),
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ],
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
                    style:
                        TextStyle(color: Theme.of(context).colorScheme.error)),
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
    required this.localSession,
    required this.loading,
    required this.strings,
    required this.currency,
    required this.minimized,
    required this.resumed,
    required this.onOpen,
    required this.onFinish,
    required this.onHistory,
    required this.onToggleMinimize,
  });

  final RegisterSession? session;

  /// The offline shift, shown when there is no server session to report.
  final LocalRegisterSession? localSession;
  final bool loading;
  final AppStrings strings;
  final String currency;
  final bool minimized;
  final bool resumed;
  final VoidCallback onOpen;
  final VoidCallback onFinish;
  final VoidCallback onHistory;
  final VoidCallback onToggleMinimize;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final current = session;
    final local = localSession;
    final titleStart = resumed ? strings.resumeSession : strings.posSession;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 0),
      child: loading
          ? const SizedBox(height: 40)
          : Card(
              margin: EdgeInsets.zero,
              child: Padding(
                padding:
                    const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                child: current == null && local == null
                    ? Row(
                        children: [
                          Icon(Icons.play_circle_outline,
                              color: theme.colorScheme.primary),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Text(
                              strings.noOpenSession,
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                              style: theme.textTheme.bodyMedium,
                            ),
                          ),
                          const SizedBox(width: 4),
                          IconButton(
                            onPressed: onHistory,
                            tooltip: strings.sessionsHistory,
                            icon: const Icon(Icons.history),
                          ),
                          TourTarget(
                            id: 'session_bar_open_button',
                            child: FilledButton.icon(
                              onPressed: onOpen,
                              icon: const Icon(Icons.add),
                              label: Text(strings.openSession),
                            ),
                          ),
                        ],
                      )
                    : minimized
                        ? Row(
                            children: [
                              Icon(
                                current != null ? Icons.bolt : Icons.cloud_off,
                                color: current != null
                                    ? Colors.green.shade700
                                    : theme.colorScheme.tertiary,
                              ),
                              const SizedBox(width: 12),
                              Expanded(
                                child: Text(
                                  current != null
                                      ? '$titleStart ${_shortId(current.id)}'
                                      : '$titleStart ${_shortId(local!.id)}',
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: theme.textTheme.titleSmall,
                                ),
                              ),
                              const SizedBox(width: 8),
                              Text(
                                current != null
                                    ? strings.formatMoney(
                                        current.summary.expectedCashMinor,
                                        currency)
                                    : strings.formatMoney(
                                        local!.openingCashMinor, currency),
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: theme.textTheme.bodySmall,
                              ),
                              const SizedBox(width: 4),
                              IconButton(
                                onPressed: onToggleMinimize,
                                tooltip: strings.expandSession,
                                icon: const Icon(Icons.unfold_more),
                              ),
                            ],
                          )
                        : Row(
                        children: [
                          Icon(
                            current != null ? Icons.bolt : Icons.cloud_off,
                            color: current != null
                                ? Colors.green.shade700
                                : theme.colorScheme.tertiary,
                          ),
                          const SizedBox(width: 12),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Text(
                                  current != null
                                      ? '$titleStart ${_shortId(current.id)}'
                                      : '$titleStart ${_shortId(local!.id)} · ${strings.sessionOfflineBadge}',
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
                                  style: theme.textTheme.titleSmall,
                                ),
                                Text(
                                  current != null
                                      ? '${current.openedBy} · ${strings.formatMoney(current.summary.expectedCashMinor, currency)} ${strings.expectedCash}'
                                      : '${strings.startingCash} ${strings.formatMoney(local!.openingCashMinor, currency)}',
                                  maxLines: 1,
                                  overflow: TextOverflow.ellipsis,
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
                          IconButton(
                            onPressed: onToggleMinimize,
                            tooltip: strings.minimizeSession,
                            icon: const Icon(Icons.unfold_less),
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

  String _shortId(String id) =>
      id.isEmpty ? '—' : id.split('-').first.substring(0, 4).toUpperCase();
}

class _NewBadge extends StatelessWidget {
  const _NewBadge({required this.strings});

  final AppStrings strings;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
      decoration: BoxDecoration(
        color: colorScheme.primary,
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(
        strings.newBadge,
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
            color: colorScheme.onPrimary, fontWeight: FontWeight.bold),
      ),
    );
  }
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
            ? (
                colorScheme.errorContainer,
                colorScheme.onErrorContainer,
                '$quantity',
                Icons.remove_shopping_cart
              )
            : out
                ? (
                    colorScheme.errorContainer,
                    colorScheme.onErrorContainer,
                    strings.outOfStock,
                    Icons.remove_shopping_cart
                  )
                : low
                    ? (
                        Colors.amber.shade100,
                        Colors.brown.shade700,
                        '$quantity',
                        Icons.inventory_2
                      )
                    : (
                        colorScheme.surfaceContainerHighest,
                        colorScheme.onSurfaceVariant,
                        '$quantity',
                        Icons.inventory_2
                      );
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
