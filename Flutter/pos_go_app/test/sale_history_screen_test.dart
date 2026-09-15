import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/sales/sale_history_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

void main() {
  testWidgets('manager sees a refund action on a completed sale',
      (tester) async {
    final api = _FakeApi(sales: [_sale('sale-1', 'completed')]);
    await tester.pumpWidget(_App(api: api, role: 'manager'));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.undo), findsOneWidget);
  });

  testWidgets('cashier sees no refund action', (tester) async {
    final api = _FakeApi(sales: [_sale('sale-1', 'completed')]);
    await tester.pumpWidget(_App(api: api, role: 'cashier'));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.undo), findsNothing);
  });

  testWidgets('a refunded sale is not refundable again', (tester) async {
    final api = _FakeApi(sales: [_sale('sale-1', 'refunded')]);
    await tester.pumpWidget(_App(api: api, role: 'manager'));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.undo), findsNothing);
  });

  testWidgets('refund dialog submits reason and refreshes', (tester) async {
    final api = _FakeApi(sales: [_sale('sale-1', 'completed')]);
    await tester.pumpWidget(_App(api: api, role: 'manager'));
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(Icons.undo));
    await tester.pumpAndSettle();

    expect(find.text('Refund sale'), findsOneWidget);
    expect(find.text('Reason (optional)'), findsOneWidget);
    expect(find.text('Manager PIN (optional)'), findsOneWidget);

    await tester.enterText(find.byType(TextField).first, 'defective item');
    await tester.tap(find.text('Refund'));
    await tester.pumpAndSettle();

    expect(api.refunds, hasLength(1));
    expect(api.refunds.single.saleId, 'sale-1');
    expect(api.refunds.single.reason, 'defective item');
    expect(find.text('Sale refunded and stock restored'), findsOneWidget);
  });

  testWidgets('completed sale offers a split action that opens the screen',
      (tester) async {
    final api = _FakeApi(sales: [_sale('sale-1', 'completed')]);
    await tester.pumpWidget(_App(api: api, role: 'cashier'));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.call_split), findsOneWidget);

    await tester.tap(find.byIcon(Icons.call_split));
    await tester.pumpAndSettle();

    expect(find.text('Split bill'), findsWidgets);
    expect(api.splitDetailRequests, hasLength(1));
    expect(api.splitDetailRequests.single, 'sale-1');
  });

  testWidgets('a refunded sale has no split action', (tester) async {
    final api = _FakeApi(sales: [_sale('sale-1', 'refunded')]);
    await tester.pumpWidget(_App(api: api, role: 'cashier'));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.call_split), findsNothing);
  });
}

SaleSummary _sale(String id, String status, {int totalMinor = 2000}) =>
    SaleSummary(
      id: id,
      status: status,
      subtotalMinor: totalMinor,
      totalMinor: totalMinor,
      currency: 'EGP',
      createdAt: '2026-09-13 12:00:00',
      paymentMethod: PaymentMethod.cash,
    );

class _FakeApi extends ApiClient {
  _FakeApi({required this.sales});

  final List<SaleSummary> sales;
  final List<RefundResult> refunds = [];
  final List<String> splitDetailRequests = [];

  @override
  Future<SalesPage> listSales(
    Session session, {
    int page = 1,
    int limit = 50,
  }) async =>
      SalesPage(sales: sales, total: sales.length, page: page, limit: limit);

  @override
  Future<SaleDetail> saleDetail(Session session, String saleId) async {
    splitDetailRequests.add(saleId);
    return SaleDetail(
      id: saleId,
      status: 'completed',
      currency: 'EGP',
      items: const [
        SaleDetailItem(
            id: 'line-1',
            productName: 'Cappuccino',
            quantity: 1,
            unitPriceMinor: 350,
            totalMinor: 350),
      ],
    );
  }

  @override
  Future<RefundResult> refundSale(
    Session session,
    String saleId, {
    String reason = '',
    String managerPin = '',
    String? idempotencyKey,
  }) async {
    final result = RefundResult(
      id: 'refund-1',
      saleId: saleId,
      refundMinor: 2000,
      status: 'completed',
      reason: reason,
    );
    refunds.add(result);
    return result;
  }
}

class _App extends StatelessWidget {
  const _App({required this.api, required this.role});

  final ApiClient api;
  final String role;

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
      home: SaleHistoryScreen(
        session: Session(
          accessToken: 'access',
          refreshToken: 'refresh',
          userId: 'user-1',
          displayName: 'Demo User',
          tenantId: 'tenant-1',
          role: role,
        ),
        apiClient: api,
      ),
    );
  }
}