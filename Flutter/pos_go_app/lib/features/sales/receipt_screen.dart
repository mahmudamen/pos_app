import 'package:flutter/material.dart';

import '../../core/receipts.dart';
import '../../l10n/strings.dart';

/// A printer-style on-screen receipt. It mirrors the ESC/POS layout served by
/// the backend: dashed rules at the thermal width, item rows, totals, and a
/// QR block. Actual printing is hardware-installation dependent; the wire
/// contract (JSON here, ESC/POS bytes at /sales/:id/receipt/print) is what is
/// stable across devices.
class ReceiptScreen extends StatelessWidget {
  const ReceiptScreen({super.key, required this.receipt});

  final SaleReceipt receipt;

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
                      onPressed: () {
                        ScaffoldMessenger.of(context).showSnackBar(
                          SnackBar(
                            content: Text(
                              s.receiptPrintHint,
                            ),
                          ),
                        );
                      },
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