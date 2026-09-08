import 'dart:convert';

import 'package:sales_pro/db/app_database.dart';
import 'package:sales_pro/features/catalog/dto/customer_dto.dart';
import 'package:sales_pro/features/catalog/dto/price_dto.dart';
import 'package:sales_pro/features/catalog/dto/store_dto.dart';
import 'package:sales_pro/features/invoice/dto/table_qu_dto.dart';
import 'package:sales_pro/features/orders/dto/t_bill_dto.dart';
import 'package:sales_pro/features/orders/dto/table_bill_line_dto.dart';
import 'package:sales_pro/core/utils/responsive_extensions.dart';
import 'package:sales_pro/screens/widgets/invoice_row.dart';
import 'package:sales_pro/screens/widgets/invoice_table_body.dart';
import 'package:sales_pro/screens/widgets/invoice_table_header.dart';
import 'package:sales_pro/services/flash_service.dart';
import 'package:sales_pro/state/app_provider.dart';
import 'package:sales_pro/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:loading_animation_widget/loading_animation_widget.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';

class InvoiceScreen extends StatefulWidget {
  final TBillDto? bill;
  final InvoiceDraftData? billDraft;

  const InvoiceScreen({this.bill, this.billDraft, Key? key}) : super(key: key);

  @override
  State<InvoiceScreen> createState() => _InvoiceScreenState();
}

class _InvoiceScreenState extends State<InvoiceScreen> {
  final Color headerColor = AppColors.tabActive;
  final Color background = AppColors.background;
  final Color tableHeader = AppColors.tableHeader;
  final Color cardColor = AppColors.card;
  final Color gridCardColor = AppColors.gridCard;
  final Color totalDarkGreen = AppColors.lineTotal;
  final Color warnAmber = AppColors.warnAmber;
  final Color focusedBorderColor = AppColors.tabActive;
  final Color lowStockColor = Colors.red;

  final TextEditingController customerCtrl = TextEditingController();
  final TextEditingController itemCtrl = TextEditingController();
  final TextEditingController receivedCtrl = TextEditingController(text: '0');
  final TextEditingController noteCtrl = TextEditingController();
  final ScrollController _vertCtrl = ScrollController();
  final ScrollController _horizCtrl = ScrollController();
  final ScrollController _horizHeaderCtrl = ScrollController();
  bool _syncingHoriz = false;
  bool _isBottomFieldFocused = false;
  String currency = 'SAR';
  InvoiceDraftData? billDraft;
  int? selectedIndex;
  bool _isSending = false;
  final List<InvoiceRow> rows = [];
  TCustomerDto? selectedCustomer;
  TableStoreDto? selectedStore;

  // Item search field variables
  final FocusNode _itemFocusNode = FocusNode();
  bool _showItemsList = false;
  OverlayEntry? _itemsOverlay;
  final LayerLink _itemsLayerLink = LayerLink();
  List<TablePriceDto> _filteredItems = [];

  double get grandTotal => rows.fold(0, (a, b) => a + (b.price * b.qty));
  bool _isLoading = false;

  @override
  void initState() {
    super.initState();
    billDraft = widget.billDraft;
    _horizCtrl.addListener(() {
      if (_syncingHoriz) return;
      _syncingHoriz = true;
      if (_horizHeaderCtrl.hasClients) {
        try {
          _horizHeaderCtrl.jumpTo(_horizCtrl.offset);
        } catch (_) {}
      }
      _syncingHoriz = false;
    });
    _horizHeaderCtrl.addListener(() {
      if (_syncingHoriz) return;
      _syncingHoriz = true;
      if (_horizCtrl.hasClients) {
        try {
          _horizCtrl.jumpTo(_horizHeaderCtrl.offset);
        } catch (_) {}
      }
      _syncingHoriz = false;
    });

    // Setup FocusNode for items field
    _itemFocusNode.addListener(() {
      if (_itemFocusNode.hasFocus && !_showItemsList) {
        _showFullItemsList();
      }
    });

    WidgetsBinding.instance.addPostFrameCallback((_) {
      _initialize();
      if (widget.bill != null) {
        _initializeInvoice();
      } else if (billDraft != null) {
        _initializedraft();
      }
    });
  }

  Future<void> _initialize() async {
    final app = context.read<AppProvider>();

    if (app.stores.isEmpty) await app.loadStores();
    if (app.customers.isEmpty) await app.loadCustomers();

    int defaultStoreId = int.tryParse(app.currentUser?.idTStore ?? '') ?? 0;
    try {
      final sp = await SharedPreferences.getInstance();
      final saved = sp.getInt('default_store_id');
      if (saved != null) defaultStoreId = saved;
    } catch (_) {}

    // Set default store
    if (app.stores.isNotEmpty) {
      selectedStore = app.stores.firstWhere(
        (s) => s.id == defaultStoreId,
        orElse: () => app.stores.first,
      );
      await app.loadPrices(selectedStore!.id);
    }
  }

  Future<void> _initializedraft() async {
    setState(() => _isLoading = true);
    List<TableBillLineDto> lines = [];

    lines = getDraftInvoiceLines(billDraft!.linesJson);

    // ✅ Create temporary list of rows
    final tempRows = <InvoiceRow>[];
    for (var line in lines) {
      tempRows.add(
        InvoiceRow(
          itemId: line.idMat,
          barcode: line.barcode,
          name: line.name,
          qty: double.tryParse(line.quantity) ?? 1,
          stock: line.stock,
          price: line.price,
          note: line.note,
        ),
      );
    }

    // ✅ Add rows in reverse order (newest on top)
    rows.clear();
    rows.addAll(tempRows.reversed);

    // Set customer text only if it exists
    if (billDraft!.customerName.isNotEmpty) {
      customerCtrl.text = billDraft!.customerName;
      selectedCustomer = TCustomerDto(
        id: int.tryParse(billDraft!.customerId) ?? 0,
        barcode: billDraft!.customerId,
        name: billDraft!.customerName,
        credit: double.tryParse(billDraft!.credit) ?? 0.0,
      );
    }

    receivedCtrl.text = billDraft!.credit.toString();
    if (mounted) setState(() => _isLoading = false);

    if (!mounted) return;
    setState(() {});
  }

  Future<void> _initializeInvoice() async {
    final app = context.read<AppProvider>();

    setState(() => _isLoading = true);
    final IdBC = int.tryParse(widget.bill!.id) ?? 0;
    final lines = await app.loadBillLinesByIdBC(IdBC);

    // ✅ Create temporary list of rows
    final tempRows = <InvoiceRow>[];
    for (var line in lines) {
      tempRows.add(
        InvoiceRow(
          itemId: line.id,
          barcode: line.idMat,
          name: line.name,
          qty: double.tryParse(line.quantity) ?? 1,
          stock: 0,
          price: line.price,
          note: line.note,
        ),
      );
    }

    // ✅ Add rows in reverse order (newest on top)
    rows.clear();
    rows.addAll(tempRows.reversed);

    // Set customer text only if it exists
    if (widget.bill!.customer.isNotEmpty) {
      customerCtrl.text = widget.bill!.customer;
    }

    receivedCtrl.text = widget.bill!.total.toString();
    if (mounted) setState(() => _isLoading = false);

    if (!mounted) return;
    setState(() {});
  }

  List<TableBillLineDto> getDraftInvoiceLines(String linesJson) {
    try {
      final List<dynamic> decoded = jsonDecode(linesJson);

      return decoded.map<TableBillLineDto>((e) {
        return TableBillLineDto(
          id: e['id']?.toString() ?? '',
          idBill: e['idBill']?.toString() ?? '',
          idMat: e['idMat']?.toString() ?? '',
          barcode: e['barcode']?.toString() ?? '',
          name: e['name']?.toString() ?? '',
          quantity: e['qty']?.toString() ?? '',
          dateEnd: e['dateEnd']?.toString() ?? '',
          idCustomer: e['idCustomer']?.toString() ?? '',
          note: e['note']?.toString() ?? '',
          idStore: e['idStore']?.toString() ?? '',
          price: (e['price_'] != null)
              ? double.tryParse(e['price_'].toString()) ?? 0.0
              : 0.0,
          stock: (e['stock'] != null)
              ? double.tryParse(e['stock'].toString()) ?? 0.0
              : 0.0,
        );
      }).toList();
    } catch (ex) {
      // Draft conversion error handled silently (debug info removed for production)
      return [];
    }
  }

  @override
  Widget build(BuildContext context) {
    return Directionality(
      textDirection: TextDirection.ltr,
      child: Scaffold(
        resizeToAvoidBottomInset: _isBottomFieldFocused,
        backgroundColor: background,
        appBar: AppBar(
          backgroundColor: headerColor,

          title: Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Text(
                'Sales Invoice',
                style: TextStyle(
                  color: Colors.white,
                  fontWeight: FontWeight.bold,
                ),
              ),
              if (billDraft != null) ...[
                const SizedBox(width: 8),
                Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 4,
                  ),
                  decoration: BoxDecoration(
                    color: Colors.orangeAccent,
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: const Text(
                    'Draft',
                    style: TextStyle(
                      color: Colors.white,
                      fontWeight: FontWeight.bold,
                      fontSize: 12,
                    ),
                  ),
                ),
              ],
            ],
          ),

          centerTitle: true,
          actions: [
            PopupMenuButton<String>(
              onSelected: (v) async {
                if (v == 'settings') {
                  final before = selectedStore?.id;
                  final res = await Navigator.pushNamed(context, '/settings');
                  if (res is int && res != before) {
                    final app = context.read<AppProvider>();
                    final st = app.stores.firstWhere(
                      (s) => s.id == res,
                      orElse: () => app.stores.first,
                    );
                    setState(() => selectedStore = st);
                    await context.read<AppProvider>().loadPrices(st.id);
                    setState(() {});
                  }
                }
              },
              itemBuilder: (_) => [
                const PopupMenuItem(value: 'settings', child: Text('Settings')),
              ],
            ),
          ],
        ),
        body: SafeArea(
          child: Padding(
            padding: const EdgeInsets.all(2.0),
            child: Stack(
              children: [
                Column(
                  children: [
                    Card(
                      color: cardColor,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                      elevation: 2,
                      child: Padding(
                        padding: const EdgeInsets.all(12),
                        child: Column(
                          children: [
                            SizedBox(
                              height: MediaQuery.of(context).size.width < 700
                                  ? 48
                                  : 52,
                              child: Autocomplete<TCustomerDto>(
                                displayStringForOption: (c) => c.name.isEmpty
                                    ? '(${c.id}) — ${c.barcode}'
                                    : '${c.name} — ${c.barcode}',
                                optionsBuilder: (text) =>
                                    _customerOptions(text.text),
                                onSelected: (c) {
                                  setState(() => selectedCustomer = c);
                                  customerCtrl.text = c.name.isEmpty
                                      ? '(${c.id}) — ${c.barcode}'
                                      : '${c.name} — ${c.barcode}';
                                },
                                fieldViewBuilder:
                                    (
                                      context,
                                      controller,
                                      focusNode,
                                      onFieldSubmitted,
                                    ) {
                                      controller.text = customerCtrl.text;
                                      return TextField(
                                        controller: controller,
                                        style: TextStyle(
                                          fontSize: 14 * context.scale,
                                        ),
                                        focusNode: focusNode,
                                        decoration: InputDecoration(
                                          labelText: 'Customer',
                                          contentPadding:
                                              const EdgeInsets.symmetric(
                                                vertical: 10,
                                                horizontal: 12,
                                              ),
                                          prefixIcon: const Icon(
                                            Icons.person_search,
                                            size: 20,
                                          ),
                                          border: OutlineInputBorder(
                                            borderRadius: BorderRadius.circular(
                                              16,
                                            ),
                                          ),
                                          focusedBorder: OutlineInputBorder(
                                            borderRadius: BorderRadius.circular(
                                              16,
                                            ),
                                            borderSide: BorderSide(
                                              color: focusedBorderColor,
                                              width: 2,
                                            ),
                                          ),
                                        ),
                                        onSubmitted: (v) =>
                                            _selectCustomerByText(v),
                                        onTap: () => setState(
                                          () => _isBottomFieldFocused = false,
                                        ),
                                        onEditingComplete: () => setState(
                                          () => _isBottomFieldFocused = true,
                                        ),
                                      );
                                    },
                                optionsViewBuilder: (context, onSelected, options) {
                                  final list = options.toList();
                                  return Align(
                                    alignment: Alignment.topRight,
                                    child: Material(
                                      elevation: 8,
                                      borderRadius: BorderRadius.circular(12),
                                      child: ConstrainedBox(
                                        constraints: BoxConstraints(
                                          maxHeight: 320,
                                          minWidth:
                                              MediaQuery.of(
                                                context,
                                              ).size.width *
                                              0.85,
                                          maxWidth: 500,
                                        ),
                                        child: ListView.builder(
                                          padding: const EdgeInsets.symmetric(
                                            vertical: 8,
                                          ),
                                          itemCount: list.length,
                                          itemBuilder: (context, i) {
                                            final c = list[i];
                                            return InkWell(
                                              onTap: () => onSelected(c),
                                              child: Container(
                                                padding:
                                                    const EdgeInsets.symmetric(
                                                      horizontal: 16,
                                                      vertical: 12,
                                                    ),
                                                child: Row(
                                                  children: [
                                                    Container(
                                                      padding:
                                                          const EdgeInsets.all(
                                                            8,
                                                          ),
                                                      decoration: BoxDecoration(
                                                        color:
                                                            focusedBorderColor
                                                                .withOpacity(
                                                                  0.1,
                                                                ),
                                                        borderRadius:
                                                            BorderRadius.circular(
                                                              8,
                                                            ),
                                                      ),
                                                      child: Icon(
                                                        Icons.person,
                                                        color:
                                                            focusedBorderColor,
                                                        size: 20,
                                                      ),
                                                    ),
                                                    const SizedBox(width: 12),
                                                    Expanded(
                                                      child: Column(
                                                        crossAxisAlignment:
                                                            CrossAxisAlignment
                                                                .start,
                                                        children: [
                                                          Text(
                                                            c.name.isEmpty
                                                                ? '(${c.id})'
                                                                : c.name,
                                                            style:
                                                                const TextStyle(
                                                                  fontWeight:
                                                                      FontWeight
                                                                          .w600,
                                                                  fontSize: 15,
                                                                ),
                                                            textAlign:
                                                                TextAlign.right,
                                                          ),
                                                          const SizedBox(
                                                            height: 4,
                                                          ),
                                                          Text(
                                                            'Barcode: ${c.barcode}',
                                                            style: TextStyle(
                                                              fontSize: 13,
                                                              color: Colors
                                                                  .grey[600],
                                                            ),
                                                            textAlign:
                                                                TextAlign.left,
                                                          ),
                                                        ],
                                                      ),
                                                    ),
                                                  ],
                                                ),
                                              ),
                                            );
                                          },
                                        ),
                                      ),
                                    ),
                                  );
                                },
                              ),
                            ),
                            const SizedBox(height: 8),
                            SizedBox(
                              height: MediaQuery.of(context).size.width < 700
                                  ? 48
                                  : 52,
                              child: CompositedTransformTarget(
                                link: _itemsLayerLink,
                                child: TextField(
                                  controller: itemCtrl,
                                  focusNode: _itemFocusNode,
                                  readOnly: false,
                                  style: TextStyle(
                                    fontSize: 14 * context.scale,
                                  ),
                                  decoration: InputDecoration(
                                    labelText: 'Search Items',
                                    contentPadding: const EdgeInsets.symmetric(
                                      vertical: 10,
                                      horizontal: 12,
                                    ),
                                    prefixIcon: const Icon(
                                      Icons.search,
                                      size: 20,
                                    ),
                                    suffixIcon: Row(
                                      mainAxisSize: MainAxisSize.min,
                                      children: [
                                        if (itemCtrl.text.isNotEmpty)
                                          IconButton(
                                            icon: const Icon(
                                              Icons.clear,
                                              size: 20,
                                            ),
                                            onPressed: () {
                                              setState(() {
                                                itemCtrl.clear();
                                                _filterItems('');
                                              });
                                            },
                                          ),
                                        IconButton(
                                          icon: const Icon(
                                            Icons.qr_code_scanner,
                                            color: AppColors.tabActive,
                                            size: 20,
                                          ),
                                          onPressed: () async {
                                            final code =
                                                await Navigator.pushNamed(
                                                  context,
                                                  '/scan',
                                                );
                                            if (code is String &&
                                                code.isNotEmpty) {
                                              _addItemByText(code);
                                              _hideItemsList();
                                              itemCtrl.clear();
                                            }
                                          },
                                          tooltip: 'Scan Barcode',
                                        ),
                                      ],
                                    ),
                                    border: OutlineInputBorder(
                                      borderRadius: BorderRadius.circular(16),
                                    ),
                                    focusedBorder: OutlineInputBorder(
                                      borderRadius: BorderRadius.circular(16),
                                      borderSide: BorderSide(
                                        color: focusedBorderColor,
                                        width: 2,
                                      ),
                                    ),
                                  ),
                                  onChanged: (value) {
                                    _filterItems(value);
                                  },
                                  onTap: () {
                                    setState(
                                      () => _isBottomFieldFocused = false,
                                    );
                                    if (!_showItemsList) {
                                      _showFullItemsList();
                                    }
                                  },
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 6),
                    Expanded(
                      child: Card(
                        color: gridCardColor,
                        shape: RoundedRectangleBorder(
                          borderRadius: BorderRadius.circular(16),
                        ),
                        elevation: 1,
                        child: Padding(
                          padding: const EdgeInsets.all(0.0),
                          child: rows.isEmpty
                              ? _buildEmptyState()
                              : Column(
                                  children: [
                                    SingleChildScrollView(
                                      scrollDirection: Axis.horizontal,
                                      controller: _horizHeaderCtrl,
                                      child: InvoiceTableHeader(
                                        columnWidths: _colWidths,
                                        isSmallScreen:
                                            MediaQuery.of(context).size.width <
                                            700,
                                      ),
                                    ),
                                    const SizedBox(height: 1),
                                    Expanded(
                                      child: Scrollbar(
                                        controller: _horizCtrl,
                                        thumbVisibility: true,
                                        notificationPredicate: (notif) =>
                                            notif.metrics.axis ==
                                            Axis.horizontal,
                                        child: SingleChildScrollView(
                                          scrollDirection: Axis.horizontal,
                                          controller: _horizCtrl,
                                          child: Scrollbar(
                                            controller: _vertCtrl,
                                            thumbVisibility: true,
                                            child: SingleChildScrollView(
                                              controller: _vertCtrl,
                                              child: InvoiceTableBody(
                                                rows: rows,
                                                selectedIndex: selectedIndex,
                                                columnWidths: _colWidths,
                                                isSmallScreen:
                                                    MediaQuery.of(
                                                      context,
                                                    ).size.width <
                                                    700,
                                                totalDarkGreen: totalDarkGreen,
                                                warnAmber: warnAmber,
                                                onDelete: (i) => setState(
                                                  () => rows.removeAt(i),
                                                ),
                                                onQtyChange: (row, qty) {
                                                  if (qty > 0) {
                                                    setState(
                                                      () => row.updateQty(qty),
                                                    );
                                                  } else {
                                                    showAppFlash(
                                                      context,
                                                      type: FlashMessageType
                                                          .warning,
                                                      message:
                                                          'Quantity must be greater than zero',
                                                    );
                                                  }
                                                },
                                                onPriceChange: (row, price) =>
                                                    setState(
                                                      () => row.updatePrice(
                                                        price,
                                                      ),
                                                    ),
                                                onNoteChange: (row, note) =>
                                                    row.note = note,
                                              ),
                                            ),
                                          ),
                                        ),
                                      ),
                                    ),
                                  ],
                                ),
                        ),
                      ),
                    ),
                    const SizedBox(height: 10),
                    Card(
                      color: cardColor,
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(12),
                      ),
                      elevation: 2,
                      child: Padding(
                        padding: const EdgeInsets.all(12),
                        child: Column(
                          children: [
                            Row(
                              children: [
                                Expanded(
                                  flex: 3,
                                  child: SizedBox(
                                    height: 56,
                                    child: TextField(
                                      controller: receivedCtrl,
                                      decoration: InputDecoration(
                                        labelText: 'Received Amount',
                                        border: OutlineInputBorder(
                                          borderRadius: BorderRadius.circular(
                                            8,
                                          ),
                                        ),
                                        prefixIcon: const Icon(Icons.payments),
                                      ),
                                      keyboardType: TextInputType.number,
                                      onTap: () {
                                        setState(
                                          () => _isBottomFieldFocused = true,
                                        );
                                      },
                                      onEditingComplete: () {
                                        setState(
                                          () => _isBottomFieldFocused = false,
                                        );
                                      },
                                    ),
                                  ),
                                ),
                                const SizedBox(width: 12),
                                Expanded(
                                  flex: 2,
                                  child: Container(
                                    constraints: const BoxConstraints(
                                      minHeight: 56,
                                    ),
                                    decoration: BoxDecoration(
                                      color: selectedCustomer != null
                                          ? AppColors.tabActive.withOpacity(0.1)
                                          : Colors.grey.withOpacity(0.1),
                                      borderRadius: BorderRadius.circular(8),
                                      border: Border.all(
                                        color: selectedCustomer != null
                                            ? AppColors.tabActive.withOpacity(
                                                0.3,
                                              )
                                            : Colors.grey.withOpacity(0.3),
                                        width: 1,
                                      ),
                                    ),
                                    padding: const EdgeInsets.symmetric(
                                      horizontal: 12,
                                      vertical: 8,
                                    ),
                                    child: selectedCustomer != null
                                        ? Row(
                                            children: [
                                              Icon(
                                                Icons.account_balance_wallet,
                                                color: AppColors.tabActive,
                                                size: 20,
                                              ),
                                              const SizedBox(width: 8),
                                              Expanded(
                                                child: Column(
                                                  crossAxisAlignment:
                                                      CrossAxisAlignment.start,
                                                  mainAxisAlignment:
                                                      MainAxisAlignment.center,
                                                  children: [
                                                    Text(
                                                      'Customer Balance',
                                                      style: TextStyle(
                                                        fontSize: 11,
                                                        color: Colors.grey[600],
                                                      ),
                                                    ),
                                                    const SizedBox(height: 2),
                                                    Text(
                                                      '${selectedCustomer!.credit.toStringAsFixed(2)} $currency',
                                                      style: const TextStyle(
                                                        fontSize: 14,
                                                        fontWeight:
                                                            FontWeight.bold,
                                                        color: AppColors
                                                            .grandTotal,
                                                      ),
                                                    ),
                                                  ],
                                                ),
                                              ),
                                            ],
                                          )
                                        : Row(
                                            mainAxisAlignment:
                                                MainAxisAlignment.center,
                                            children: [
                                              Icon(
                                                Icons.person_off,
                                                color: Colors.grey[400],
                                                size: 18,
                                              ),
                                              const SizedBox(width: 8),
                                              Text(
                                                'No Customer',
                                                style: TextStyle(
                                                  fontSize: 13,
                                                  color: Colors.grey[600],
                                                ),
                                              ),
                                            ],
                                          ),
                                  ),
                                ),
                              ],
                            ),
                            const SizedBox(height: 12),
                            Row(
                              mainAxisAlignment: MainAxisAlignment.spaceBetween,
                              children: [
                                SizedBox(
                                  height: 30,
                                  child: const Text(
                                    'Grand Total:',
                                    style: TextStyle(
                                      fontSize: 18,
                                      fontWeight: FontWeight.bold,
                                    ),
                                  ),
                                ),
                                Text(
                                  '${grandTotal.toStringAsFixed(2)} $currency',
                                  style: const TextStyle(
                                    fontSize: 18,
                                    color: AppColors.grandTotal,
                                    fontWeight: FontWeight.bold,
                                  ),
                                ),
                              ],
                            ),
                          ],
                        ),
                      ),
                    ),
                    const SizedBox(height: 10),

                    Row(
                      children: [
                        if (widget.bill == null)
                          Expanded(
                            child: ElevatedButton.icon(
                              style: ElevatedButton.styleFrom(
                                backgroundColor: AppColors.send,
                                padding: const EdgeInsets.symmetric(
                                  vertical: 14,
                                ),
                                shape: RoundedRectangleBorder(
                                  borderRadius: BorderRadius.circular(8),
                                ),
                              ),
                              icon: _isSending
                                  ? SizedBox(
                                      width: 24,
                                      height: 24,
                                      child: LoadingAnimationWidget.fallingDot(
                                        color: Colors.white,
                                        size: 24,
                                      ),
                                    )
                                  : const Icon(Icons.send),
                              label: Text(
                                _isSending ? 'Sending...' : 'Send Invoice',
                              ),
                              onPressed: _isSending ? null : _sendInvoice,
                            ),
                          ),

                        const SizedBox(width: 12),

                        // If it's a draft invoice, show edit and delete buttons
                        if (billDraft != null) ...[
                          Expanded(
                            child: ElevatedButton.icon(
                              style: ElevatedButton.styleFrom(
                                backgroundColor: Colors.orange, // Edit color
                                padding: const EdgeInsets.symmetric(
                                  vertical: 14,
                                ),
                                shape: RoundedRectangleBorder(
                                  borderRadius: BorderRadius.circular(8),
                                ),
                              ),
                              icon: const Icon(Icons.edit),
                              label: const Text('Edit'),
                              onPressed: _editDraft, // Edit function
                            ),
                          ),
                          // const SizedBox(width: 12),
                          // Expanded(
                          //   child: ElevatedButton.icon(
                          //     style: ElevatedButton.styleFrom(
                          //       backgroundColor: Colors.red, // Delete color
                          //       padding: const EdgeInsets.symmetric(vertical: 14),
                          //       shape: RoundedRectangleBorder(
                          //         borderRadius: BorderRadius.circular(8),
                          //       ),
                          //     ),
                          //     icon: const Icon(Icons.delete),
                          //     label: const Text('Delete'),
                          //     onPressed: _deleteDraft, // Delete function
                          //   ),
                          // ),
                        ] else ...[
                          if (widget.bill == null)
                            Expanded(
                              child: ElevatedButton.icon(
                                style: ElevatedButton.styleFrom(
                                  backgroundColor: AppColors.send,
                                  padding: const EdgeInsets.symmetric(
                                    vertical: 14,
                                  ),
                                  shape: RoundedRectangleBorder(
                                    borderRadius: BorderRadius.circular(8),
                                  ),
                                ),
                                icon: const Icon(Icons.save),
                                label: const Text('Save as Draft'),
                                onPressed: _saveDraft,
                              ),
                            ),
                        ],
                      ],
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Iterable<TCustomerDto> _customerOptions(String text) {
    final app = context.read<AppProvider>();
    final q = text.trim();
    if (q.isEmpty) return const Iterable<TCustomerDto>.empty();
    final List<TCustomerDto> all = app.customersList;
    final res = <TCustomerDto>[];
    for (final c in all) {
      if (c.name.contains(q) ||
          c.barcode.contains(q) ||
          c.id.toString().contains(q)) {
        res.add(c);
        if (res.length >= 10) break;
      }
    }
    return res;
  }

  void _selectCustomerByText(String text) {
    final app = context.read<AppProvider>();
    final q = text.trim();
    if (q.isEmpty) return;
    TCustomerDto? match;
    // barcode exact
    for (final c in app.customers) {
      if (c.barcode == q) {
        match = c;
        break;
      }
    }
    // id numeric
    if (match == null) {
      final id = int.tryParse(q);
      if (id != null) {
        for (final c in app.customers) {
          if (c.id == id) {
            match = c;
            break;
          }
        }
      }
    }
    // name contains
    if (match == null) {
      for (final c in app.customers) {
        if (c.name.contains(q)) {
          match = c;
          break;
        }
      }
    }
    if (match != null) {
      setState(() {
        selectedCustomer = match;
      });
    } else {
      showAppFlash(
        context,
        type: FlashMessageType.error,
        message: 'Item not found',
      );
    }
  }

  void _addItemByText(String text) {
    final app = context.read<AppProvider>();
    final q = text.trim();
    if (q.isEmpty) return;
    TablePriceDto? p;
    for (final e in app.prices) {
      if (e.id == q || e.barcode == q) {
        p = e;
        break;
      }
    }
    p ??= app.prices.firstWhere(
      (e) => e.name.contains(q),
      orElse: () =>
          TablePriceDto(id: q, barcode: q, name: '', price: 0, availableQty: 0),
    );
    _addItemFromPrice(p);
  }

  void _addItemFromPrice(TablePriceDto p) {
    setState(() {
      // Find existing row with same itemId
      final existingRow = rows.firstWhere(
        (r) => r.itemId == p.id,
        orElse: () => InvoiceRow(
          itemId: '',
          barcode: '',
          name: '',
          qty: 0,
          stock: 0,
          price: 0,
          note: '',
        ),
      );

      if (existingRow.itemId.isNotEmpty) {
        // If same item found, just increase quantity
        existingRow.qty += 1;
      } else {
        // ✅ Add new row at the top of the table (first position)
        rows.insert(
          0,
          InvoiceRow(
            itemId: p.id,
            barcode: p.barcode,
            name: p.name,
            qty: 1,
            stock: p.availableQty,
            price: p.price,
            note: '',
          ),
        );
      }
    });
  }

  Future<void> _saveDraft() async {
    if (!await validateSelection()) {
      setState(() => _isSending = false);
      return;
    }
    if (rows.isEmpty) {
      showAppFlash(
        context,
        type: FlashMessageType.success,
        message: 'Invoice saved as draft',
      );
      return;
    }

    final app = context.read<AppProvider>();

    final lines = <TableQuDto>[];
    for (int i = 0; i < rows.length; i++) {
      final r = rows[i];
      lines.add(
        TableQuDto(
          id: i + 1,
          idMat: r.itemId,
          barcode: r.barcode,
          name: r.name,
          qty: r.qty.toString(),
          idStore: (selectedStore!.id).toString(),
          note: r.note,
          idList: i + 1,
          price_: r.price,
          stock: r.stock,
        ),
      );
    }

    final credit = receivedCtrl.text.trim().isEmpty
        ? '0'
        : receivedCtrl.text.trim();

    await app.saveInvoiceDraft(
      lines: lines,
      idCu: selectedCustomer?.barcode ?? 'unknown',
      customer: selectedCustomer?.name ?? 'unknown',
      credit: credit,
    );
    setState(() {
      customerCtrl.text = '';
      rows.clear();
      receivedCtrl.text = '0';
      noteCtrl.clear();
      selectedIndex = null;
      billDraft = null;
    });
    showAppFlash(
      context,
      type: FlashMessageType.success,
      message: 'Invoice saved as draft',
    );
  }

  Future<void> _showOfflineAlert() async {
    await showDialog<void>(
      context: context,
      barrierDismissible: true, // Can be closed by clicking outside
      builder: (ctx) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        title: Row(
          children: const [
            Icon(Icons.wifi_off, color: Colors.orange),
            SizedBox(width: 8),
            Text('No Internet Connection'),
          ],
        ),
        content: const Text(
          'Cannot send invoice now.\n\nPlease check your internet connection.',
          textAlign: TextAlign.left,
        ),
        actions: [
          TextButton(
            onPressed: () {
              Navigator.pop(ctx);
            },
            child: const Text('OK'),
          ),
        ],
      ),
    );
  }

  Future<bool?> _showOfflineConfirmationDialog() async {
    return await showDialog<bool>(
      context: context,
      barrierDismissible: false,
      builder: (ctx) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
        title: Row(
          children: const [
            Icon(Icons.wifi_off, color: Colors.orange),
            SizedBox(width: 8),
            Text('No Internet Connection'),
          ],
        ),
        content: const Text(
          'Cannot send invoice now.\n\n'
          'Do you want to save it as a draft to send later?',
          textAlign: TextAlign.left,
        ),
        actions: [
          TextButton(
            onPressed: () {
              Navigator.pop(ctx, false);
              FocusScope.of(ctx).unfocus();
            },
            child: const Text('No, Cancel'),
          ),
          ElevatedButton(
            style: ElevatedButton.styleFrom(backgroundColor: Colors.orange),
            onPressed: () {
              Navigator.pop(ctx, true);
              FocusScope.of(ctx).unfocus();
            },
            child: const Text('Yes, Save as Draft'),
          ),
        ],
      ),
    );
  }

  Future<bool> validateSelection() async {
    if (selectedCustomer == null) {
      showAppFlash(
        context,
        type: FlashMessageType.error,
        message: 'Select Customer',
      );

      return false;
    }

    if (selectedStore == null) {
      showAppFlash(
        context,
        type: FlashMessageType.error,
        message: 'Select Store',
      );

      return false;
    }

    if (rows.isEmpty) {
      showAppFlash(
        context,
        type: FlashMessageType.warning,
        message: 'Add at least one item to send the invoice',
      );

      return false;
    }

    return true;
  }

  Future<void> _sendInvoice() async {
    if (_isSending) return; // 🚨 Prevent double click
    setState(() => _isSending = true);
    final app = context.read<AppProvider>();
    if (!app.connectivity.isOnline.value) {
      // No internet → Ask user
      final Object? saveAsDraft = widget.billDraft != null
          ? _showOfflineAlert()
          : await _showOfflineConfirmationDialog();

      if (saveAsDraft == true) {
        _saveDraft();
      }
      setState(() => _isSending = false);
      return;
    }
    if (!await validateSelection()) {
      setState(() => _isSending = false);
      return;
    }

    final lines = <TableQuDto>[];
    for (int i = 0; i < rows.length; i++) {
      final r = rows[i];
      lines.add(
        TableQuDto(
          id: i + 1,
          idMat: r.itemId,
          barcode: r.barcode,
          name: r.name,
          qty: r.qty.toString(),
          idStore: (selectedStore!.id).toString(),
          note: r.note,
          idList: i + 1,
          price_: r.price,
          stock: r.stock,
        ),
      );
    }
    final credit = receivedCtrl.text.trim().isEmpty
        ? '0'
        : receivedCtrl.text.trim();
    final res = await app.sendInvoice(
      lines: lines,
      customer: selectedCustomer!.name,
      idCu: selectedCustomer!.barcode,
      credit: credit,
    );
    if (!mounted) return;
    if (res.contains('\u062A\u0645 \u0627\u0631\u0633\u0627\u0644')) {
      showAppFlash(
        context,
        type: FlashMessageType.success,
        message: 'Invoice sent successfully',
      );
    } else {
      if (res.contains('internet not connected')) {
        final Object? saveAsDraft = widget.billDraft != null
            ? _showOfflineAlert()
            : await _showOfflineConfirmationDialog();

        if (saveAsDraft == true) {
          await _saveDraft(); // Make sure this is async if it does I/O
        }

        setState(() => _isSending = false);
        return;
      }
      showAppFlash(context, type: FlashMessageType.error, message: res);
    }

    if (res.contains('\u062A\u0645 \u0627\u0631\u0633\u0627\u0644')) {
      setState(() {
        rows.clear();
        receivedCtrl.text = '0';
        customerCtrl.text = '';
        noteCtrl.clear();
        selectedIndex = null;
        billDraft = null;
      });
      if (widget.billDraft != null) {
        app.deleteInvoiceDraft(widget.billDraft!.id);
      }
    }
    setState(() => _isSending = false);
  }

  Map<int, TableColumnWidth> get _colWidths {
    final width = MediaQuery.of(context).size.width;
    final isSmallScreen = width < 700;

    if (isSmallScreen) {
      return const {
        0: FixedColumnWidth(25), // Number
        1: FixedColumnWidth(100), // Name
        2: FixedColumnWidth(110), // Quantity
        3: FixedColumnWidth(70), // Price
        4: FixedColumnWidth(110), // Total
        5: FixedColumnWidth(90), // Stock Balance
        6: FixedColumnWidth(180), // Note (Scrollable)
        7: FixedColumnWidth(60), // Delete
      };
    }

    return const {
      0: FixedColumnWidth(50), // Number
      1: FixedColumnWidth(150), // Name
      2: FixedColumnWidth(130), // Quantity
      3: FixedColumnWidth(100), // Price
      4: FixedColumnWidth(120), // Total
      5: FixedColumnWidth(100), // Stock Balance
      6: FixedColumnWidth(80), // Code
      7: FixedColumnWidth(220), // Note
      8: FixedColumnWidth(64), // Delete
    };
  }

  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.receipt_long_outlined, size: 72, color: Colors.grey),
          const SizedBox(height: 12),
          Text(
            _isLoading ? 'Loading invoice data...' : 'Invoice is empty',
            style: const TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w600,
              color: Colors.black54,
            ),
          ),
          const SizedBox(height: 6),
          _isLoading
              ? LoadingAnimationWidget.fallingDot(
                  color: AppColors.tabActive,
                  size: 50,
                )
              : const Text(
                  'Add items from search or use scanner',
                  style: TextStyle(color: Colors.black45),
                ),
        ],
      ),
    );
  }

  @override
  void dispose() {
    for (var row in rows) row.dispose();
    customerCtrl.dispose();
    itemCtrl.dispose();
    receivedCtrl.dispose();
    noteCtrl.dispose();
    _vertCtrl.dispose();
    _horizCtrl.dispose();
    _horizHeaderCtrl.dispose();
    _itemFocusNode.dispose();
    _itemsOverlay?.remove();
    _itemsOverlay = null;
    super.dispose();
  }

  void _showFullItemsList() {
    final app = context.read<AppProvider>();
    setState(() {
      _filteredItems = app.priceList;
      _showItemsList = true;
    });
    _itemsOverlay?.remove();
    _itemsOverlay = _buildItemsOverlay();
    Overlay.of(context).insert(_itemsOverlay!);
  }

  void _filterItems(String query) {
    final app = context.read<AppProvider>();
    setState(() {
      if (query.trim().isEmpty) {
        _filteredItems = app.priceList;
      } else {
        final q = query.trim();
        _filteredItems = app.priceList.where((p) {
          return p.name.contains(q) ||
              p.barcode.contains(q) ||
              p.id.contains(q);
        }).toList();
      }
    });
    if (_showItemsList) {
      _itemsOverlay?.remove();
      _itemsOverlay = _buildItemsOverlay();
      Overlay.of(context).insert(_itemsOverlay!);
    }
  }

  void _hideItemsList() {
    _itemsOverlay?.remove();
    _itemsOverlay = null;
    setState(() {
      _showItemsList = false;
    });
  }

  OverlayEntry _buildItemsOverlay() {
    return OverlayEntry(
      builder: (context) => GestureDetector(
        onTap: () {
          _hideItemsList();
          _itemFocusNode.unfocus();
        },
        behavior: HitTestBehavior.translucent,
        child: Stack(
          children: [
            Positioned.fill(child: Container(color: Colors.transparent)),
            Positioned(
              width: MediaQuery.of(context).size.width - 20,
              child: CompositedTransformFollower(
                link: _itemsLayerLink,
                showWhenUnlinked: false,
                offset: const Offset(0, 56),
                child: Material(
                  elevation: 8,
                  borderRadius: BorderRadius.circular(12),
                  child: Container(
                    constraints: const BoxConstraints(maxHeight: 480),
                    decoration: BoxDecoration(
                      color: Colors.white,
                      borderRadius: BorderRadius.circular(12),
                      boxShadow: [
                        BoxShadow(
                          color: Colors.black.withOpacity(0.1),
                          blurRadius: 10,
                          offset: const Offset(0, 4),
                        ),
                      ],
                    ),
                    child: _filteredItems.isEmpty
                        ? const Padding(
                            padding: EdgeInsets.all(16.0),
                            child: Center(
                              child: Text(
                                'No items found',
                                style: TextStyle(
                                  fontSize: 15,
                                  color: Colors.grey,
                                ),
                              ),
                            ),
                          )
                        : ListView.separated(
                            padding: const EdgeInsets.symmetric(vertical: 4),
                            shrinkWrap: true,
                            itemCount: _filteredItems.length,
                            separatorBuilder: (context, index) => Divider(
                              height: 1,
                              thickness: 0.5,
                              color: Colors.grey[300],
                              indent: 12,
                              endIndent: 12,
                            ),
                            itemBuilder: (context, i) {
                              final p = _filteredItems[i];
                              return InkWell(
                                onTap: () {
                                  _addItemFromPrice(p);
                                  itemCtrl.clear();
                                  _hideItemsList();
                                  _itemFocusNode.unfocus();
                                },
                                child: Container(
                                  padding: const EdgeInsets.symmetric(
                                    horizontal: 12,
                                    vertical: 10,
                                  ),
                                  child: Row(
                                    children: [
                                      Container(
                                        padding: const EdgeInsets.all(6),
                                        decoration: BoxDecoration(
                                          color: focusedBorderColor.withOpacity(
                                            0.1,
                                          ),
                                          borderRadius: BorderRadius.circular(
                                            6,
                                          ),
                                        ),
                                        child: Icon(
                                          Icons.inventory_2,
                                          color: focusedBorderColor,
                                          size: 18,
                                        ),
                                      ),
                                      const SizedBox(width: 10),
                                      Expanded(
                                        child: Text(
                                          p.name.isEmpty ? '(${p.id})' : p.name,
                                          style: const TextStyle(
                                            fontWeight: FontWeight.bold,
                                            fontSize: 15,
                                            height: 1.2,
                                          ),
                                          maxLines: 1,
                                          overflow: TextOverflow.ellipsis,
                                          textAlign: TextAlign.right,
                                        ),
                                      ),
                                    ],
                                  ),
                                ),
                              );
                            },
                          ),
                  ),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  void _editDraft() async {
    if (!await validateSelection()) {
      setState(() => _isSending = false);
      return;
    }
    if (rows.isEmpty) {
      showAppFlash(
        context,
        type: FlashMessageType.warning,
        message: 'Add at least one item to save draft',
      );
      return;
    }

    final app = context.read<AppProvider>();
    final draftId = billDraft!.id;
    final lines = <TableQuDto>[];
    for (int i = 0; i < rows.length; i++) {
      final r = rows[i];
      lines.add(
        TableQuDto(
          id: i + 1,
          idMat: r.itemId,
          barcode: r.barcode,
          name: r.name,
          qty: r.qty.toString(),
          idStore: selectedStore!.id.toString(),
          note: r.note,
          idList: i + 1,
          price_: r.price,
          stock: r.stock,
        ),
      );
    }

    final credit = receivedCtrl.text.trim().isEmpty
        ? '0'
        : receivedCtrl.text.trim();

    await app.updateInvoiceDraft(
      id: draftId, // make sure you have this
      lines: lines,
      idCu: selectedCustomer?.barcode ?? 'unknown',
      customer: selectedCustomer?.name ?? 'unknown',
      credit: credit,
      note: noteCtrl.text.trim(),
    );

    setState(() {
      customerCtrl.text = '';
      rows.clear();
      receivedCtrl.text = '0';
      noteCtrl.clear();
      selectedIndex = null;
      billDraft = null; // Reset the draft after editing
    });

    // ScaffoldMessenger.of(
    //   context,
    // ).showSnackBar(const SnackBar(content: Text('Invoice saved as draft')));

    Navigator.pop(context);
    showAppFlash(
      context,
      type: FlashMessageType.success,
      message: 'Draft updated',
    );
  }
}

extension _AppLists on AppProvider {
  List<TCustomerDto> get customersList => customers;
  List<TablePriceDto> get priceList => prices;
}
