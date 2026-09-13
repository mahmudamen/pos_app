import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/restaurants.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/restaurants/split_bill_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

void main() {
  test('Floor parses the Go item', () {
    final floor =
        Floor.fromJson(const {'id': 'f1', 'name': 'Ground', 'sort_order': 3});
    expect(floor.id, 'f1');
    expect(floor.name, 'Ground');
    expect(floor.sortOrder, 3);
  });

  test('RestaurantTable parses status and floor name', () {
    final table = RestaurantTable.fromJson(const {
      'id': 't1',
      'floor_id': 'f1',
      'name': 'T01',
      'seats': 4,
      'status': 'occupied',
      'floor_name': 'Ground',
    });
    expect(table.id, 't1');
    expect(table.floorId, 'f1');
    expect(table.name, 'T01');
    expect(table.seats, 4);
    expect(table.isAvailable, isFalse);
  });

  test('RestaurantTable free and reserved are available; closed is not', () {
    const free = RestaurantTable(id: 'a', floorId: 'f', name: 'A', status: 'free');
    const reserved =
        RestaurantTable(id: 'b', floorId: 'f', name: 'B', status: 'reserved');
    const closed =
        RestaurantTable(id: 'c', floorId: 'f', name: 'C', status: 'closed');
    expect(free.isAvailable, isTrue);
    expect(reserved.isAvailable, isTrue);
    expect(closed.isAvailable, isFalse);
  });

  test('SplitBillLine serializes sale_item_id and quantity', () {
    const line = SplitBillLine(saleItemId: 'line-1', quantity: 2);
    expect(line.toJson(), {'sale_item_id': 'line-1', 'quantity': 2});
  });

  test('SplitBillResult parses children ids and currency', () {
    final result = SplitBillResult.fromJson(const {
      'parent_sale_id': 'sale-1',
      'children': ['child-1', 'child-2'],
      'currency': 'EGP',
    });
    expect(result.parentSaleId, 'sale-1');
    expect(result.children, ['child-1', 'child-2']);
    expect(result.currency, 'EGP');
  });

  test('SaleDetail parses items with sale_item ids', () {
    final sale = SaleDetail.fromJson({
      'id': 'sale-1',
      'status': 'completed',
      'currency': 'EGP',
      'items': [
        {'id': 'si-1', 'product_name': 'Pizza', 'quantity': 2, 'unit_price_minor': 10000, 'total_minor': 20000},
        {'id': 'si-2', 'product_name': 'Cola', 'quantity': 1, 'unit_price_minor': 1500, 'total_minor': 1500},
      ],
    });
    expect(sale.id, 'sale-1');
    expect(sale.items, hasLength(2));
    expect(sale.items.first.id, 'si-1');
    expect(sale.items.first.productName, 'Pizza');
    expect(sale.items.first.quantity, 2);
    expect(sale.items.first.unitPriceMinor, 10000);
  });

  group('SplitBillScreen', () {
    testWidgets('allocates items to covers and submits the split', (tester) async {
      tester.view.physicalSize = const Size(1080, 2400);
      tester.view.devicePixelRatio = 2.0;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final client = _FakeRestaurantClient();
      await tester.pumpWidget(_SplitApp(client: client));
      await tester.pumpAndSettle();

      expect(find.text('Pizza'), findsWidgets);
      expect(find.text('Cola'), findsWidgets);

      await tester.tap(find.byKey(const Key('cover-0-item-0-minus')));
      await tester.pump();
      await tester.tap(find.byKey(const Key('cover-0-item-0-minus')));
      await tester.pump();
      await tester.tap(find.byKey(const Key('cover-1-item-0-plus')));
      await tester.pump();
      await tester.tap(find.byKey(const Key('cover-1-item-0-plus')));
      await tester.pump();
      await tester.tap(find.byKey(const Key('split-action')));
      await tester.pumpAndSettle();

      expect(client.splitCalls, hasLength(1));
      final groups = client.splitCalls.single;
      expect(groups, hasLength(2));
      expect(groups[0].single.saleItemId, 'si-2');
      expect(groups[0].single.quantity, 1);
      expect(groups[1].single.saleItemId, 'si-1');
      expect(groups[1].single.quantity, 2);
      expect(find.textContaining('child-a'), findsOneWidget);
    });

    testWidgets('blocks split when a cover would be empty', (tester) async {
      tester.view.physicalSize = const Size(1080, 2400);
      tester.view.devicePixelRatio = 2.0;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final client = _FakeRestaurantClient();
      await tester.pumpWidget(_SplitApp(client: client));
      await tester.pumpAndSettle();

      await tester.tap(find.byKey(const Key('split-action')));
      await tester.pumpAndSettle();

      expect(client.splitCalls, isEmpty);
    });
  });
}

class _SplitApp extends StatelessWidget {
  const _SplitApp({required this.client});

  final ApiClient client;

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
      home: SplitBillScreen(
        session: const Session(
          accessToken: 'at',
          refreshToken: 'rt',
          userId: 'u1',
          displayName: 'Cashier',
          tenantId: 't1',
        ),
        apiClient: client,
        saleId: 'sale-1',
        currencyCode: 'EGP',
      ),
    );
  }
}

class _UnusedClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async =>
      throw UnimplementedError();
}

class _FakeRestaurantClient extends ApiClient {
  _FakeRestaurantClient() : super(client: _UnusedClient());

  final List<List<List<SplitBillLine>>> splitCalls = [];

  @override
  Future<SaleDetail> saleDetail(Session session, String saleId) async =>
      const SaleDetail(
        id: 'sale-1',
        status: 'completed',
        currency: 'EGP',
        items: [
          SaleDetailItem(
              id: 'si-1', productName: 'Pizza', quantity: 2, unitPriceMinor: 10000, totalMinor: 20000),
          SaleDetailItem(
              id: 'si-2', productName: 'Cola', quantity: 1, unitPriceMinor: 1500, totalMinor: 1500),
        ],
      );

  @override
  Future<SplitBillResult> splitSale(
    Session session,
    String saleId,
    List<List<SplitBillLine>> children,
  ) async {
    splitCalls.add(children);
    return const SplitBillResult(
      parentSaleId: 'sale-1',
      children: ['child-a', 'child-b'],
      currency: 'EGP',
    );
  }
}