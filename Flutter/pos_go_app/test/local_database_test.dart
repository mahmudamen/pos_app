import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/storage/local_database.dart';

void main() {
  sqfliteFfiInit();

  late LocalDatabase db;

  setUp(() {
    db = LocalDatabase(
      factory: databaseFactoryFfi,
      databasePath: inMemoryDatabasePath,
    );
  });

  tearDown(() async {
    await db.close();
  });

  test('cachedProducts stores and retrieves products', () async {
    final products = [
      const Product(
        id: 'p1',
        name: 'Espresso',
        sku: 'ESP-1',
        barcode: '0123',
        priceMinor: 280,
        costMinor: 100,
        currency: 'USD',
        stockQuantity: 10,
      ),
      const Product(
        id: 'p2',
        name: 'Latte',
        sku: 'LAT-1',
        barcode: '0456',
        priceMinor: 350,
        costMinor: 120,
        currency: 'USD',
        stockQuantity: 5,
      ),
    ];

    await db.cacheProducts(products);
    final cached = await db.cachedProducts();

    expect(cached, hasLength(2));
    expect(cached[0].name, 'Espresso');
    expect(cached[0].priceMinor, 280);
    expect(cached[1].name, 'Latte');
    expect(cached[1].stockQuantity, 5);
  });

  test('cacheProducts replaces previous cache', () async {
    final original = [
      const Product(
        id: 'p1',
        name: 'Old Product',
        sku: 'OLD-1',
        barcode: '',
        priceMinor: 100,
        currency: 'USD',
        stockQuantity: 1,
      ),
    ];
    await db.cacheProducts(original);
    expect(await db.cachedProducts(), hasLength(1));

    final updated = [
      const Product(
        id: 'p2',
        name: 'New Product',
        sku: 'NEW-1',
        barcode: '',
        priceMinor: 200,
        currency: 'USD',
        stockQuantity: 2,
      ),
      const Product(
        id: 'p3',
        name: 'Another Product',
        sku: 'AN-1',
        barcode: '',
        priceMinor: 300,
        currency: 'USD',
        stockQuantity: 3,
      ),
    ];
    await db.cacheProducts(updated);
    final cached = await db.cachedProducts();

    expect(cached, hasLength(2));
    expect(cached[0].name, 'Another Product');
    expect(cached[1].name, 'New Product');
  });

  test('cachedProducts returns empty list when no cache', () async {
    final cached = await db.cachedProducts();
    expect(cached, isEmpty);
  });

  test('cacheProducts preserves cost_minor field', () async {
    final products = [
      const Product(
        id: 'p1',
        name: 'Widget',
        sku: 'W-1',
        barcode: '',
        priceMinor: 500,
        costMinor: 200,
        currency: 'USD',
        stockQuantity: 10,
      ),
    ];
    await db.cacheProducts(products);
    final cached = await db.cachedProducts();
    expect(cached.single.costMinor, 200);
  });

  test('pending sale command is queued and retrieved in order', () async {
    await db.enqueue(const PendingCommand(
      id: 'cmd-1',
      idempotencyKey: 'key-1',
      operation: 'create_sale',
      payload: '{"items":[{"product_id":"p1","quantity":2}]}',
    ));
    await db.enqueue(const PendingCommand(
      id: 'cmd-2',
      idempotencyKey: 'key-2',
      operation: 'create_sale',
      payload: '{"items":[{"product_id":"p2","quantity":1}]}',
    ));

    final pending = await db.pendingCommands();
    expect(pending, hasLength(2));
    expect(pending[0].id, 'cmd-1');
    expect(pending[0].idempotencyKey, 'key-1');
    expect(pending[1].id, 'cmd-2');
  });

  test('queued split-payment payload round-trips intact', () async {
    const payload =
        '{"items":[{"product_id":"p1","quantity":2}],"payments":[{"method":"card","amount_minor":400},{"method":"mobile","amount_minor":160}]}';
    await db.enqueue(const PendingCommand(
      id: 'cmd-split',
      idempotencyKey: 'key-split',
      operation: 'create_sale',
      payload: payload,
    ));

    final pending = await db.pendingCommands();
    expect(pending.single.payload, payload);
  });

  test('duplicate idempotency keys are not enqueued twice', () async {
    await db.enqueue(const PendingCommand(
      id: 'cmd-1',
      idempotencyKey: 'same-key',
      operation: 'create_sale',
      payload: '{}',
    ));
    await db.enqueue(const PendingCommand(
      id: 'cmd-2',
      idempotencyKey: 'same-key',
      operation: 'create_sale',
      payload: '{}',
    ));

    expect(await db.pendingCommands(), hasLength(1));
  });

  test('markComplete removes a command from the pending queue', () async {
    await db.enqueue(const PendingCommand(
      id: 'cmd-1',
      idempotencyKey: 'key-1',
      operation: 'create_sale',
      payload: '{}',
    ));
    expect(await db.pendingCommands(), hasLength(1));

    await db.markComplete('cmd-1');
    expect(await db.pendingCommands(), isEmpty);
  });
}
