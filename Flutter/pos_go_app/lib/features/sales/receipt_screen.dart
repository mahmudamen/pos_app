import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/printer_service.dart';
import '../../core/printers.dart';
import '../../core/session_store.dart';
import '../../core/receipts.dart';
import '../../l10n/strings.dart';

/// A printer-style on-screen receipt. It mirrors the ESC/POS layout served by
/// the backend: dashed rules at the thermal width, item rows, totals, and a
/// QR block. When [apiClient], [session], [sessionStore] and [printerService]
/// are supplied, the print button emits the same layout to a configured
/// thermal printer; otherwise it falls back to a hint.
class ReceiptScreen extends StatelessWidget {
  const ReceiptScreen({
    super.key,
    required this.receipt,
    this.apiClient,
    this.session,
    this.sessionStore,
    this.printerService,
  });

  final SaleReceipt receipt;
  final ApiClient? apiClient;
  final Session? session;
  final SessionStore? sessionStore;
  final PrinterService? printerService;

  Future<void> _print(BuildContext context) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final apiClient = this.apiClient;
    final session = this.session;
    final store = sessionStore;
    if (apiClient == null || session == null || store == null) {
      messenger.showSnackBar(SnackBar(content: Text(s.receiptPrintHint)));
      return;
    }
    final config = await store.readPrinterConfig() ?? const PrinterConfig();
    if (!config.enabled || !config.hasDefault) {
      if (!context.mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(s.receiptPrintHint)));
      return;
    }
    try {
      await printSaleReceipt(
        config: config,
        service: printerService ?? PrinterService(),
        apiClient: apiClient,
        session: session,
        saleId: receipt.saleId,
        roleAllowsPrint: session.isManagerLevel,
      );
      if (!context.mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(s.receiptSentToPrinter)));
    } catch (e) {
      if (!context.mounted) return;
      messenger.showSnackBar(
          SnackBar(content: Text('${s.printFailed}: $e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    const style = TextStyle(
      fontFamily: 'monospace',
      fontSize: 12,
      height: 1.35,
    );
    return Scaffold(
      appBar: AppBar(title: Text(s.receipt)),
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 420),
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Container(
                  color: Theme.of(context).colorScheme.surfaceContainerHighest,
                  padding: const EdgeInsets.all(12),
                  child: Text(
                    _render(receipt),
                    style: style,
                  ),
                ),
                const SizedBox(height: 16),
                OverflowBar(
                  alignment: MainAxisAlignment.center,
                  children: [
                    TextButton.icon(
                      onPressed: () => _print(context),
                      icon: const Icon(Icons.local_printshop_outlined),
                      label: Text(s.printReceipt),
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

  String _render(SaleReceipt r) {
    const dash = '--------------------------------';
    final buffer = StringBuffer()
      ..writeln(r.tenantName)
      ..writeln(r.tenantAddress)
      ..writeln(dash)
      ..writeln('Sale     ${_short(r.saleId)}')
      ..writeln('Date     ${r.createdAt}')
      ..writeln('Cashier  ${r.cashier}');
    if (r.device.isNotEmpty) buffer.writeln('Device   ${r.device}');
    if (r.floorName.isNotEmpty || r.tableName.isNotEmpty) {
      buffer.writeln('Table    ${r.floorName}/${r.tableName}');
    }
    if (r.customerName.isNotEmpty) {
      buffer.writeln('Customer ${r.customerName}');
    }
    buffer.writeln(dash);
    for (final line in r.items) {
      buffer
        ..writeln(line.name)
        ..writeln('${line.quantity} x ${_money(line.unitPriceMinor, r.currency)}'
            ' ... ${_money(line.totalMinor, r.currency)}');
    }
    buffer.writeln(dash);
    buffer.writeln('Subtotal     ${_money(r.subtotalMinor, r.currency)}');
    if (r.discountMinor > 0) {
      buffer.writeln('Discount     -${_money(r.discountMinor, r.currency)}');
    }
    if (r.tipsMinor > 0) {
      buffer.writeln('Tip          ${_money(r.tipsMinor, r.currency)}');
    }
    for (final p in r.payments) {
      buffer.writeln(
          'Paid(${p.method})  ${_money(p.amountMinor, r.currency)}');
    }
    buffer
      ..writeln(dash)
      ..writeln('     TOTAL ${_money(r.totalMinor, r.currency)}');
    if (r.loyaltyPointsEarned > 0) {
      buffer.writeln('Loyalty points earned: ${r.loyaltyPointsEarned}');
    }
    return buffer.toString().trimRight();
  }

  String _short(String id) =>
      id.length > 26 ? id.substring(0, 26) : id;

  String _money(int minor, String currency) {
    final sign = minor.isNegative ? '-' : '';
    final abs = minor.abs();
    final whole = abs ~/ 100;
    final frac = abs % 100;
    final fracStr = frac.toString().padLeft(2, '0');
    return '$sign${currency == 'EGP' ? 'E£' : '$currency '}$whole.$fracStr';
  }
}