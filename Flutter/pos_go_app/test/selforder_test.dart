import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:qr_flutter/qr_flutter.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/selforder.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/selforder/qr_sheet.dart';
import 'package:pos_go_app/l10n/strings.dart';

const _session = Session(
  accessToken: 'at',
  refreshToken: 'rt',
  userId: 'u1',
  displayName: 'Cashier',
  tenantId: 'tenant-123',
);

class _RecordingClient extends http.BaseClient {
  _RecordingClient(this.response);

  final http.Response response;
  final List<http.BaseRequest> calls = [];

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    calls.add(request);
    return http.StreamedResponse(
      Stream.value(response.bodyBytes),
      response.statusCode,
      headers: response.headers,
    );
  }
}

http.Response _envelope(Object data, [int status = 200]) =>
    http.Response(jsonEncode({'data': data}), status,
        headers: {'content-type': 'application/json'});

void main() {
  const itemJson = {
    'product_id': 'p1',
    'name': 'Cappuccino',
    'sku': 'CAP-001',
    'quantity': 2,
    'unit_price_minor': 350,
    'total_minor': 700,
  };

  test('SelfOrder parses the wire payload and status helpers', () {
    final order = SelfOrder.fromJson(const {
      'id': 'o1',
      'reference': 'SO-ABC123DE',
      'customer_name': 'Ahmed',
      'table_name': 'T5',
      'status': 'pending',
      'subtotal_minor': 700,
      'total_minor': 700,
      'currency': 'EGP',
      'items': [itemJson],
      'approved_sale_id': null,
      'created_at': '2026-09-20T00:00:00+03:00',
    });
    expect(order.reference, 'SO-ABC123DE');
    expect(order.isPending, isTrue);
    expect(order.isApproved, isFalse);
    expect(order.totalMinor, 700);
    expect(order.items.single.name, 'Cappuccino');
    expect(order.items.single.quantity, 2);
    expect(order.items.single.totalMinor, 700);
    expect(order.tableName, 'T5');
  });

  test('SelfOrdersPage parses items/total/page/limit', () {
    final page = SelfOrdersPage.fromJson(const {
      'items': [
        {'id': 'o1', 'reference': 'SO-1', 'status': 'approved'},
        {'id': 'o2', 'reference': 'SO-2', 'status': 'cancelled'},
      ],
      'total': 2,
      'page': 1,
      'limit': 50,
    });
    expect(page.orders.length, 2);
    expect(page.total, 2);
    expect(page.hasMore, isFalse);
  });

  test('SelfOrderApproval parses sale id', () {
    final approval = SelfOrderApproval.fromJson(const {
      'order_id': 'o1',
      'status': 'approved',
      'sale': {'id': 'sale-9'},
    });
    expect(approval.orderId, 'o1');
    expect(approval.saleId, 'sale-9');
  });

  test('ProductRequest parses webpush and status', () {
    final request = ProductRequest.fromJson(const {
      'id': 'r1',
      'product_id': null,
      'product_name': 'Mango Smoothie',
      'note': '2L please',
      'contact': '010123',
      'status': 'open',
      'has_webpush': true,
      'notified_at': null,
      'created_at': '2026-09-20',
    });
    expect(request.isOpen, isTrue);
    expect(request.hasWebPush, isTrue);
    expect(request.productId, isNull);
    expect(request.productName, 'Mango Smoothie');
  });

  test('ProductRequestPage parses items', () {
    final page = ProductRequestsPage.fromJson(const {
      'items': [
        {'id': 'r1', 'product_name': 'X', 'status': 'open'},
      ],
      'total': 1,
      'page': 1,
      'limit': 50,
    });
    expect(page.requests.single.productName, 'X');
  });

  test('Product parses selforder_enabled and publishedForSelfOrder', () {
    final off = Product.fromJson(const {
      'id': 'p1',
      'name': 'X',
      'sku': 'S',
      'barcode': '',
      'price_minor': 100,
      'currency': 'EGP',
      'stock_quantity': 0,
      'selforder_enabled': true,
    });
    expect(off.selforderEnabled, isTrue);
    expect(off.publishedForSelfOrder, isFalse);

    final inStock = Product.fromJson(const {
      'id': 'p2',
      'name': 'Y',
      'sku': 'S2',
      'price_minor': 100,
      'currency': 'EGP',
      'stock_quantity': 4,
      'selforder_enabled': true,
    });
    expect(inStock.publishedForSelfOrder, isTrue);

    final unlisted = Product.fromJson(const {
      'id': 'p3',
      'name': 'Z',
      'sku': 'S3',
      'price_minor': 100,
      'currency': 'EGP',
      'stock_quantity': 4,
      'selforder_enabled': false,
    });
    expect(unlisted.publishedForSelfOrder, isFalse);
  });

  test('selfOrders lists with status filter and parses the page', () async {
    final client = _RecordingClient(_envelope(const {
      'items': [
        {'id': 'o1', 'reference': 'SO-1', 'status': 'pending'},
      ],
      'total': 1,
      'page': 1,
      'limit': 50,
    }));
    final api = ApiClient(client: client);
    final page = await api
        .selfOrders(_session, status: 'pending', page: 2, limit: 25);
    expect(page.orders.single.status, 'pending');
    expect(page.total, 1);
    expect(
        client.calls.single.url.queryParameters,
        containsPair('status', 'pending'));
    expect(client.calls.single.url.queryParameters,
        containsPair('page', '2'));
    expect(client.calls.single.url.queryParameters,
        containsPair('limit', '25'));
    expect(client.calls.single.url.path, '/v1/self-orders');
  });

  test('approveSelfOrder POSTs and maps the approval', () async {
    final client = _RecordingClient(_envelope(const {
      'order_id': 'o1',
      'status': 'approved',
      'sale': {'id': 'sale-9'},
    }));
    final api = ApiClient(client: client);
    final approval = await api.approveSelfOrder(_session, 'o1');
    expect(approval.saleId, 'sale-9');
    expect(client.calls.single.method, 'POST');
    expect(client.calls.single.url.path, '/v1/self-orders/o1/approve');
  });

  test('cancelSelfOrder POSTs', () async {
    final client =
        _RecordingClient(_envelope({'ok': true}));
    final api = ApiClient(client: client);
    await api.cancelSelfOrder(_session, 'o1');
    expect(client.calls.single.method, 'POST');
    expect(client.calls.single.url.path, '/v1/self-orders/o1/cancel');
  });

  test('productRequests lists and parses', () async {
    final client = _RecordingClient(_envelope(const {
      'items': [
        {'id': 'r1', 'product_name': 'X', 'status': 'open'},
      ],
      'total': 1,
      'page': 1,
      'limit': 50,
    }));
    final api = ApiClient(client: client);
    final page = await api.productRequests(_session);
    expect(page.requests.single.id, 'r1');
    expect(client.calls.single.url.path, '/v1/product-requests');
  });

  test('fulfillProductRequest / closeProductRequest POST', () async {
    final client = _RecordingClient(_envelope({'ok': true}));
    final api = ApiClient(client: client);
    await api.fulfillProductRequest(_session, 'r1');
    expect(client.calls.single.url.path, '/v1/product-requests/r1/fulfill');
    await api.closeProductRequest(_session, 'r1');
    expect(client.calls.last.url.path, '/v1/product-requests/r1/close');
  });

  test('updateProduct forwards selforder_enabled (and omits when null)',
      () async {
    final client = _RecordingClient(_envelope(const {
      'id': 'p1',
      'name': 'X',
      'sku': 'S',
      'price_minor': 100,
      'currency': 'EGP',
      'stock_quantity': 2,
      'selforder_enabled': true,
    }));
    final api = ApiClient(client: client);
    final updated =
        await api.updateProduct(_session, 'p1', selforderEnabled: true);
    expect(updated.selforderEnabled, isTrue);
    final body = jsonDecode((client.calls.single as http.Request).body)
        as Map<String, dynamic>;
    expect(body, contains('selforder_enabled'));
    expect(body['selforder_enabled'], isTrue);

    final client2 = _RecordingClient(_envelope(const {
      'id': 'p1',
      'name': 'X',
      'sku': 'S',
      'price_minor': 100,
      'currency': 'EGP',
      'stock_quantity': 2,
      'selforder_enabled': false,
    }));
    final api2 = ApiClient(client: client2);
    await api2.updateProduct(_session, 'p1', isActive: true);
    final body2 = jsonDecode((client2.calls.single as http.Request).body)
        as Map<String, dynamic>;
    expect(body2, isNot(contains('selforder_enabled')));
  });

  group('SelfOrderQrSheet', () {
    Widget harness() => MaterialApp(
          locale: const Locale('en'),
          supportedLocales: AppStrings.supportedLocales,
          localizationsDelegates: const [
            AppStrings.delegate,
            GlobalMaterialLocalizations.delegate,
            GlobalWidgetsLocalizations.delegate,
            GlobalCupertinoLocalizations.delegate,
          ],
          home: Scaffold(
            body: SelfOrderQrSheet(session: _session, apiClient: ApiClient()),
          ),
        );

    testWidgets('renders a QR for the tenant and table name', (tester) async {
      await tester.pumpWidget(harness());
      await tester.pumpAndSettle();
      expect(find.byType(QrImageView), findsOneWidget);

      await tester.enterText(find.byType(TextField), 'Table 7');
      await tester.pumpAndSettle();
      expect(
        find.textContaining('/selforder?tenant=tenant-123&table=Table%207'),
        findsOneWidget,
      );
    });

    testWidgets('Regenerate re-renders with a fresh cache-buster',
        (tester) async {
      await tester.pumpWidget(harness());
      await tester.pumpAndSettle();
      expect(find.textContaining('&v='), findsNothing);
      await tester.tap(find.text('Regenerate'));
      await tester.pumpAndSettle();
      expect(find.textContaining('&v=1'), findsOneWidget);
    });
  });
}