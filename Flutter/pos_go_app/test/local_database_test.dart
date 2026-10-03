import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/registers.dart';
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

  // Local register sessions: the offline half of a shift, recorded on the
  // device when POST /v1/registers/open cannot reach the server.
  group('local session storage', () {
    test('an offline shift is readable as the open session', () async {
      final db = LocalDatabase(
        factory: databaseFactoryFfi,
        databasePath: inMemoryDatabasePath,
      );
      addTearDown(db.close);

      expect(await db.openLocalSession(), isNull);

      await db.saveLocalSession(LocalRegisterSession.open(
        id: 'local-1',
        openingCashMinor: 5000,
        openedAt: DateTime.utc(2026, 3, 1, 9),
      ));

      final open = await db.openLocalSession();
      expect(open, isNotNull);
      expect(open!.id, 'local-1');
      expect(open.openingCashMinor, 5000);
    });

    test('closing the shift clears the open session but keeps the record',
        () async {
      final db = LocalDatabase(
        factory: databaseFactoryFfi,
        databasePath: inMemoryDatabasePath,
      );
      addTearDown(db.close);

      final opened = LocalRegisterSession.open(
        id: 'local-1',
        openingCashMinor: 5000,
        openedAt: DateTime.utc(2026, 3, 1, 9),
      );
      await db.saveLocalSession(opened);
      await db.saveLocalSession(
        opened.closed(countedCashMinor: 4200, at: DateTime.utc(2026, 3, 1, 17)),
      );

      expect(await db.openLocalSession(), isNull);
      final all = await db.localSessions();
      expect(all, hasLength(1));
      expect(all.single.isOpen, isFalse);
      expect(all.single.closingCashMinor, 4200);
    });

    test('a terminal keeps at most one open shift, the newest winning',
        () async {
      final db = LocalDatabase(
        factory: databaseFactoryFfi,
        databasePath: inMemoryDatabasePath,
      );
      addTearDown(db.close);

      await db.saveLocalSession(LocalRegisterSession.open(
        id: 'local-1',
        openingCashMinor: 100,
        openedAt: DateTime.utc(2026, 3, 1, 9),
      ));
      await db.saveLocalSession(LocalRegisterSession.open(
        id: 'local-2',
        openingCashMinor: 200,
        openedAt: DateTime.utc(2026, 3, 1, 17),
      ));

      expect((await db.openLocalSession())!.id, 'local-2');
      expect(await db.localSessions(), hasLength(2),
          reason: 'both shifts stay on record for reconciliation');
    });
  });

  group('LocalRegisterSession', () {
    test('an open session has no closing facts', () {
      final session = LocalRegisterSession.open(
        id: 'local-1',
        openingCashMinor: 5000,
        openedAt: DateTime.utc(2026, 3, 1, 9),
      );

      expect(session.isOpen, isTrue);
      expect(session.closingCashMinor, isNull);
      expect(session.closedAt, isNull);
    });

    test('closing records the counted cash and stamps the time', () {
      final opened = LocalRegisterSession.open(
        id: 'local-1',
        openingCashMinor: 5000,
        openedAt: DateTime.utc(2026, 3, 1, 9),
      );

      final closed = opened.closed(
        countedCashMinor: 4200,
        at: DateTime.utc(2026, 3, 1, 17),
      );

      expect(closed.isOpen, isFalse);
      expect(closed.closingCashMinor, 4200);
      expect(closed.openingCashMinor, 5000, reason: 'float must survive');
      expect(closed.id, 'local-1');
      expect(opened.isOpen, isTrue, reason: 'closing returns a new instance');
    });

    test('survives a database round trip', () {
      final session = LocalRegisterSession.open(
        id: 'local-1',
        openingCashMinor: 1234,
        openedAt: DateTime.utc(2026, 3, 1, 9),
      ).closed(countedCashMinor: 900, at: DateTime.utc(2026, 3, 1, 17));

      final restored = LocalRegisterSession.fromRow(session.toRow());

      expect(restored.id, session.id);
      expect(restored.openingCashMinor, 1234);
      expect(restored.closingCashMinor, 900);
      expect(restored.isOpen, isFalse);
      expect(
        DateTime.parse(restored.closedAt!),
        DateTime.utc(2026, 3, 1, 17),
      );
    });
  });
}
