import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/receipts.dart';
import 'package:pos_go_app/features/sales/receipt_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

void main() {
  const receipt = SaleReceipt(
    tenantName: 'Demo Store',
    tenantAddress: 'Cairo, Egypt',
    saleId: 'sale-123',
    status: 'completed',
    createdAt: '2026-09-12 12:00:00',
    cashier: 'Demo Manager',
    device: 'register-1',
    tableName: 'T1',
    floorName: 'Ground',
    customerName: 'Ahmed',
    currency: 'EGP',
    items: [
      ReceiptLine(
        name: 'Flat White',
        sku: 'FW',
        quantity: 2,
        unitPriceMinor: 1000,
        totalMinor: 2000,
      ),
    ],
    subtotalMinor: 2500,
    discountMinor: 500,
    totalMinor: 2000,
    payments: [ReceiptPayment(method: 'card', amountMinor: 2000)],
    loyaltyPointsEarned: 20,
  );

  testWidgets('receipt preview renders the printer layout', (tester) async {
    await tester.pumpWidget(const _App(receipt: receipt));
    await tester.pump();

    expect(find.text('Receipt'), findsOneWidget);
    expect(find.textContaining('Demo Store'), findsOneWidget);
    expect(find.textContaining('Flat White'), findsOneWidget);
    expect(find.textContaining('TOTAL E£20.00'), findsOneWidget);
    expect(find.textContaining('E£20.00'), findsWidgets);
    expect(find.textContaining('Paid(card)'), findsOneWidget);
    expect(find.textContaining('Loyalty points earned: 20'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('print action surfaces the hardware hint', (tester) async {
    await tester.pumpWidget(const _App(receipt: receipt));
    await tester.pump();
    await tester.tap(find.text('Print'));
    await tester.pump();
    expect(
        find.text('Enable printing from the printer settings, then try again'),
        findsOneWidget);
  });

  testWidgets('rounding receipt shows the Rounding line and payable total',
      (tester) async {
    const rounded = SaleReceipt(
      tenantName: 'Demo Store',
      cashier: 'Demo Manager',
      saleId: 'sale-124',
      status: 'completed',
      createdAt: '2026-09-12 12:00:00',
      currency: 'EGP',
      items: [
        ReceiptLine(
          name: 'Flat White',
          quantity: 1,
          unitPriceMinor: 2513,
          totalMinor: 2513,
        ),
      ],
      subtotalMinor: 2513,
      totalMinor: 2513,
      roundingMinor: 12,
      payments: [ReceiptPayment(method: 'cash', amountMinor: 2525)],
    );
    await tester.pumpWidget(const _App(receipt: rounded));
    await tester.pump();

    expect(find.textContaining('Rounding'), findsOneWidget);
    expect(find.textContaining('TOTAL E£25.25'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}

class _App extends StatelessWidget {
  const _App({required this.receipt});

  final SaleReceipt receipt;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      locale: const Locale('en'),
      supportedLocales: AppStrings.supportedLocales,
      localizationsDelegates: const [
        AppStrings.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      home: ReceiptScreen(receipt: receipt),
    );
  }
}