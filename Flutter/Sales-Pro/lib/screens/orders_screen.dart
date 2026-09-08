// lib/screens/orders_screen.dart
import 'package:sales_pro/screens/invoice_screen.dart';
import 'package:sales_pro/screens/widgets/data_list_item.dart';
import 'package:flash/flash.dart';
import 'package:flash/flash_helper.dart';
import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../state/app_provider.dart';
import '../features/orders/dto/t_bill_dto.dart';
import '../theme/app_theme.dart';

class OrdersScreen extends StatefulWidget {
  const OrdersScreen({super.key});
  @override
  State<OrdersScreen> createState() => _OrdersScreenState();
}

class _OrdersScreenState extends State<OrdersScreen> {
  DateTime? _endDate;
  final ScrollController _scrollCtrl = ScrollController();

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _loadBills();
    });
  }

  Future<void> _loadBills() async {
    final app = context.read<AppProvider>();
    setState(() => app.busy = true);
    await app.loadBills(endDate: _endDate);
    if (mounted) setState(() => app.busy = false);
  }

  @override
  Widget build(BuildContext context) {
    final isSmall = MediaQuery.of(context).size.width < 600;

    return Padding(
      padding: const EdgeInsets.all(12.0),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Date bar
          _buildDatePicker(isSmall),
          const SizedBox(height: 12),

          // Invoices list
          Expanded(
            child: Consumer<AppProvider>(
              builder: (context, app, _) {
                if (app.busy) {
                  return const Center(child: CircularProgressIndicator());
                }

                final bills = app.bills;
                if (bills.isEmpty) {
                  return _buildEmptyState();
                }

                return RefreshIndicator(
                  onRefresh: _loadBills,
                  child: _buildBillsList(context, bills, app, isSmall),
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  // Date selection bar
  Widget _buildDatePicker(bool isSmall) {
    return Card(
      elevation: 2,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        child: Row(
          children: [
            Expanded(
              child: OutlinedButton.icon(
                onPressed: () async {
                  final now = DateTime.now();
                  final picked = await showDatePicker(
                    context: context,
                    initialDate: _endDate ?? now,
                    firstDate: DateTime(2020),
                    lastDate: DateTime(now.year + 1),
                    locale: const Locale('en'),
                  );
                  if (picked != null && picked != _endDate) {
                    setState(() => _endDate = picked);
                    await _loadBills();
                  }
                },
                icon: const Icon(Icons.date_range, size: 20),
                label: Text(
                  _endDate == null
                      ? 'Today'
                      : '${_endDate!.year}-${_endDate!.month.toString().padLeft(2, '0')}-${_endDate!.day.toString().padLeft(2, '0')}',
                  style: const TextStyle(fontSize: 14),
                ),
                style: OutlinedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(vertical: 12),
                ),
              ),
            ),
            if (_endDate != null) ...[
              const SizedBox(width: 8),
              IconButton(
                tooltip: 'Clear Date',
                onPressed: () {
                  setState(() => _endDate = null);
                  _loadBills();
                },
                icon: const Icon(Icons.clear, color: Colors.red),
              ),
            ],
          ],
        ),
      ),
    );
  }

  // Empty state
  Widget _buildEmptyState() {
    return Center(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(Icons.receipt_long_outlined, size: 80, color: Colors.grey[400]),
          const SizedBox(height: 16),
          const Text(
            'No invoices',
            style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          Text(
            'Start creating a new invoice',
            style: TextStyle(color: Colors.grey[600]),
          ),
        ],
      ),
    );
  }

  // قائمة الفواتير (ListView + Card)
  Widget _buildBillsList(
    BuildContext context,
    List<TBillDto> bills,
    AppProvider app,
    bool isSmall,
  ) {
    return Scrollbar(
      controller: _scrollCtrl,
      thumbVisibility: true,
      child: ListView.builder(
        controller: _scrollCtrl,
        padding: const EdgeInsets.symmetric(horizontal: 4),
        itemCount: bills.length,
        itemBuilder: (context, i) {
          final b = bills[i];
          final dt = app.parseServerDateToUi(b.date);
          final time = dt == null
              ? '-'
              : '${dt.hour.toString().padLeft(2, '0')}:${dt.minute.toString().padLeft(2, '0')}';

          return DataListItem(
            leading: b.id,
            title: b.customer.isEmpty ? 'Unspecified' : b.customer,
            subtitle: time,
            trailingTop: b.total == 0
                ? '-'
                : '${b.total.toStringAsFixed(2)} SAR',
            trailingColor: AppColors.tabActive,
            onTap: () => _openBill(context, app, b),
          );
        },
      ),
    );
  }

  // فتح الفاتورة
  void _openBill(BuildContext context, AppProvider app, TBillDto bill) {
    if (!app.connectivity.isOnline.value) {
      context.showFlash(
        barrierDismissible: true,
        duration: const Duration(seconds: 3),
        builder: (context, controller) => FlashBar(
          controller: controller,
          position: FlashPosition.bottom,
          indicatorColor: Colors.orange,
          icon: const Icon(Icons.wifi_off, color: Colors.orange),
          title: const Text('Offline'),
          content: const Text('You must be online to view invoice details'),
        ),
      );
      return;
    }

    Navigator.push(
      context,
      MaterialPageRoute(builder: (_) => InvoiceScreen(bill: bill)),
    );
  }

  @override
  void dispose() {
    _scrollCtrl.dispose();
    super.dispose();
  }
}
