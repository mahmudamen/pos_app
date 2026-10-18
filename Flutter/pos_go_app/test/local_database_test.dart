import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;
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

  test('failCommand parks a rejected command with its reason and attempts',
      () async {
    await db.enqueue(const PendingCommand(
      id: 'cmd-1',
      idempotencyKey: 'key-1',
      operation: 'create_sale',
      payload: '{}',
    ));

    await db.failCommand('cmd-1', errorCode: 'insufficient_stock');

    expect(await db.pendingCommands(), isEmpty,
        reason: 'a failed command must not block the replay queue');
    final failed = await db.failedCommands();
    expect(failed, hasLength(1));
    expect(failed.single.id, 'cmd-1');
    expect(failed.single.lastError, 'insufficient_stock');
    expect(failed.single.attempts, 1);
    expect(failed.single.isConflict, isFalse,
        reason: 'a plain failure is not flagged as a conflict');

    await db.failCommand('cmd-1', errorCode: 'insufficient_stock');
    expect((await db.failedCommands()).single.attempts, 2,
        reason: 'each refusal counts');
  });

  test('failCommand records a conflict with its server detail', () async {
    await db.enqueue(const PendingCommand(
      id: 'cmd-1',
      idempotencyKey: 'key-1',
      operation: 'create_sale',
      payload: '{}',
    ));

    await db.failCommand(
      'cmd-1',
      errorCode: 'insufficient_stock',
      errorDetail: 'insufficient stock',
      conflict: true,
    );

    final failed = (await db.failedCommands()).single;
    expect(failed.isConflict, isTrue,
        reason: 'a 409 clash with state elsewhere must stay recoverable');
    expect(failed.lastError, 'insufficient_stock');
    expect(failed.errorDetail, 'insufficient stock');
  });

  test('retryCommand moves a failed command back into the queue', () async {
    await db.enqueue(const PendingCommand(
      id: 'cmd-1',
      idempotencyKey: 'key-1',
      operation: 'create_sale',
      payload: '{}',
    ));
    await db.failCommand('cmd-1',
        errorCode: 'insufficient_stock',
        errorDetail: 'insufficient stock',
        conflict: true);

    await db.retryCommand('cmd-1');

    expect(await db.failedCommands(), isEmpty);
    final pending = await db.pendingCommands();
    expect(pending.single.id, 'cmd-1');
    expect(pending.single.lastError, isNull,
        reason: 'the stale reason must not resurface');
    expect(pending.single.errorDetail, isNull);
    expect(pending.single.isConflict, isFalse,
        reason: 'the conflict flag is cleared for the retry');
    expect(pending.single.attempts, 1, reason: 'history is kept');
  });

  test('dropCommand discards a failed command for good', () async {
    await db.enqueue(const PendingCommand(
      id: 'cmd-1',
      idempotencyKey: 'key-1',
      operation: 'create_sale',
      payload: '{}',
    ));
    await db.failCommand('cmd-1', errorCode: 'command_conflict');

    await db.dropCommand('cmd-1');

    expect(await db.failedCommands(), isEmpty);
    expect(await db.pendingCommands(), isEmpty);
  });

  // Local register sessions: the offline half of a shift, recorded on the
  // device when POST /v1/registers/open cannot reach the server.
  group('last sale', () {
    test('a completed sale is remembered per tenant', () async {
      await db.saveLastSale('tenant-1', 'sale-1');

      expect(await db.readLastSale('tenant-1'), 'sale-1');
      expect(await db.readLastSale('tenant-2'), isNull,
          reason: 'a device switched to another tenant must not reprint it');
    });

    test('a newer sale replaces the previous one', () async {
      await db.saveLastSale('tenant-1', 'sale-1');
      await db.saveLastSale('tenant-1', 'sale-2');

      expect(await db.readLastSale('tenant-1'), 'sale-2');
    });

    test('nothing is remembered before the first sale', () async {
      expect(await db.readLastSale('tenant-1'), isNull);
    });
  });

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

  test('opens when the v4 columns exist but user_version is still 3',
      () async {
    // An open interrupted before sqflite persisted user_version leaves the
    // pairing columns in place. Replaying the upgrade must not raise
    // `duplicate column name`, which used to brick every later launch.
    final dir = await Directory.systemTemp.createTemp('pos_go_upgrade');
    final path = p.join(dir.path, 'bricked.db');

    final seeded = await databaseFactoryFfi.openDatabase(
      path,
      options: OpenDatabaseOptions(
        version: 3,
        onCreate: (db, _) async {
          await db.execute('''
            CREATE TABLE local_sessions (
              id TEXT PRIMARY KEY,
              opening_cash_minor INTEGER NOT NULL,
              opened_at INTEGER NOT NULL,
              closing_cash_minor INTEGER,
              closed_at INTEGER,
              created_at INTEGER NOT NULL,
              status TEXT NOT NULL DEFAULT 'open',
              server_session_id TEXT,
              reconciled_at INTEGER
            )
          ''');
          await db.execute('''
            CREATE TABLE pending_commands (
              id TEXT PRIMARY KEY,
              idempotency_key TEXT NOT NULL UNIQUE,
              operation TEXT NOT NULL,
              payload TEXT NOT NULL,
              status TEXT NOT NULL DEFAULT 'pending',
              attempts INTEGER NOT NULL DEFAULT 0,
              created_at INTEGER NOT NULL
            )
          ''');
        },
      ),
    );
    await seeded.execute('PRAGMA user_version = 3');
    await seeded.close();

    final upgraded = LocalDatabase(
      factory: databaseFactoryFfi,
      databasePath: path,
    );
    addTearDown(() async {
      await upgraded.close();
      await dir.delete(recursive: true);
    });

    final columns = await (await upgraded.database)
        .rawQuery('PRAGMA table_info(local_sessions)');
    final names = columns.map((row) => row['name'] as String).toSet();
    expect(names, containsAll(<String>['server_session_id', 'reconciled_at']));

    final commandColumns = await (await upgraded.database)
        .rawQuery('PRAGMA table_info(pending_commands)');
    final commandNames =
        commandColumns.map((row) => row['name'] as String).toSet();
    expect(commandNames, containsAll(<String>['attempts', 'last_error']));
  });

  test('cachedProducts keeps the category and Arabic name', () async {
    await db.cacheProducts(const [
      Product(
        id: 'p1',
        name: 'Coffee',
        sku: 'COF',
        barcode: '',
        priceMinor: 100,
        currency: 'EGP',
        stockQuantity: 3,
        categoryId: 'cat-9',
        nameAr: 'قهوة',
      ),
    ]);

    final cached = await db.cachedProducts();
    expect(cached.single.categoryId, 'cat-9');
    expect(cached.single.nameAr, 'قهوة');
  });

  test('cacheCategories stores and retrieves categories', () async {
    await db.cacheCategories(const [
      Category(id: 'c1', name: 'Drinks', slug: 'drinks', nameAr: 'مشروبات'),
    ]);

    final cached = await db.cachedCategories();
    expect(cached, hasLength(1));
    expect(cached.single.name, 'Drinks');
    expect(cached.single.nameAr, 'مشروبات');
  });

  test('applyProductChanges upserts by id and deletes without a full wipe',
      () async {
    await db.cacheProducts(const [
      Product(
        id: 'p1',
        name: 'Old',
        sku: 'OLD',
        barcode: '',
        priceMinor: 100,
        currency: 'EGP',
        stockQuantity: 1,
      ),
      Product(
        id: 'p2',
        name: 'Keep',
        sku: 'KEEP',
        barcode: '',
        priceMinor: 100,
        currency: 'EGP',
        stockQuantity: 1,
      ),
    ]);

    await db.applyProductChanges(
      upserts: const [
        Product(
          id: 'p1',
          name: 'New',
          sku: 'NEW',
          barcode: '',
          priceMinor: 200,
          currency: 'EGP',
          stockQuantity: 9,
        ),
        Product(
          id: 'p3',
          name: 'Added',
          sku: 'ADD',
          barcode: '',
          priceMinor: 100,
          currency: 'EGP',
          stockQuantity: 1,
        ),
      ],
      deletes: ['p2'],
    );

    final cached = await db.cachedProducts();
    expect(cached.map((p) => p.id), unorderedEquals(['p1', 'p3']));
    expect(cached.firstWhere((p) => p.id == 'p1').name, 'New');
    expect(cached.firstWhere((p) => p.id == 'p1').stockQuantity, 9);
  });

  test('applyCategoryChanges upserts by id and deletes', () async {
    await db.cacheCategories(const [
      Category(id: 'c1', name: 'Old', slug: 'c1'),
      Category(id: 'c2', name: 'Gone', slug: 'c2'),
    ]);

    await db.applyCategoryChanges(
      upserts: const [Category(id: 'c1', name: 'New', slug: 'c1')],
      deletes: ['c2'],
    );

    final cached = await db.cachedCategories();
    expect(cached, hasLength(1));
    expect(cached.single.name, 'New');
  });

  test('clearCatalogCache empties both cached tables', () async {
    await db.cacheProducts(const [
      Product(
        id: 'p1',
        name: 'Coffee',
        sku: 'COF',
        barcode: '',
        priceMinor: 100,
        currency: 'EGP',
        stockQuantity: 1,
      ),
    ]);
    await db.cacheCategories(const [
      Category(id: 'c1', name: 'Drinks', slug: 'drinks'),
    ]);

    await db.clearCatalogCache();

    expect(await db.cachedProducts(), isEmpty);
    expect(await db.cachedCategories(), isEmpty);
  });
}
