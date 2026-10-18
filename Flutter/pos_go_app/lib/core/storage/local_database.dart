import 'package:path/path.dart' as path;
import 'package:sqflite/sqflite.dart';

import '../api_client.dart';
import '../registers.dart';
import 'database_target_resolver.dart';

class PendingCommand {
  const PendingCommand({
    required this.id,
    required this.idempotencyKey,
    required this.operation,
    required this.payload,
    this.attempts = 0,
    this.lastError,
    this.errorDetail,
    this.isConflict = false,
  });

  final String id;
  final String idempotencyKey;
  final String operation;
  final String payload;
  final int attempts;
  final String? lastError;

  /// The server's human-readable explanation ("insufficient stock"), when it
  /// sent one. [lastError] is the stable machine code the UI maps to a reason.
  final String? errorDetail;

  /// True when the server answered a *conflict* (the command clashed with
  /// state created elsewhere, e.g. another device sold the last units) rather
  /// than a plain rejection. Conflicts are recoverable and surface separately.
  final bool isConflict;
}

class LocalDatabase {
  LocalDatabase({DatabaseFactory? factory, this.databasePath})
      : _factory = factory;

  // Left null unless injected. Reading sqflite's global `databaseFactory`
  // throws where no plugin registered it (the browser), so it must not be
  // touched before [resolveDatabaseTarget] has picked the platform factory.
  final DatabaseFactory? _factory;
  final String? databasePath;
  Database? _database;

  Future<Database> get database async {
    if (_database != null) return _database!;
    final target = await _target();
    _database = await target.factory.openDatabase(
      target.path,
      options: OpenDatabaseOptions(
        version: 8,
        onCreate: (db, _) async {
          await db.execute('''
            CREATE TABLE cached_products (
              id TEXT PRIMARY KEY,
              name TEXT NOT NULL,
              sku TEXT NOT NULL,
              barcode TEXT,
              price_minor INTEGER NOT NULL,
              cost_minor INTEGER NOT NULL DEFAULT 0,
              currency TEXT NOT NULL,
              stock_quantity INTEGER NOT NULL,
              image_url TEXT NOT NULL DEFAULT '',
              category_id TEXT NOT NULL DEFAULT '',
              name_ar TEXT NOT NULL DEFAULT ''
            )
          ''');
          await _createCachedCategories(db);
          await db.execute('''
            CREATE TABLE pending_commands (
              id TEXT PRIMARY KEY,
              idempotency_key TEXT NOT NULL UNIQUE,
              operation TEXT NOT NULL,
              payload TEXT NOT NULL,
              status TEXT NOT NULL DEFAULT 'pending',
              attempts INTEGER NOT NULL DEFAULT 0,
              last_error TEXT,
              error_detail TEXT,
              is_conflict INTEGER NOT NULL DEFAULT 0,
              created_at INTEGER NOT NULL
            )
          ''');
          await _createLocalSessions(db);
          await _createLastSale(db);
        },
        onUpgrade: (db, oldVersion, newVersion) async {
          if (oldVersion < 2) {
            await db.execute(
                'ALTER TABLE cached_products ADD COLUMN image_url TEXT NOT NULL DEFAULT \'\'');
          }
          if (oldVersion < 3) {
            await _createLocalSessions(db);
          }
          if (oldVersion < 4) {
            // Pairing columns: added after the first offline-shift release so
            // an interrupted replay can resume against the same session.
            await _addColumnIfMissing(
                db, 'local_sessions', 'server_session_id', 'TEXT');
            await _addColumnIfMissing(
                db, 'local_sessions', 'reconciled_at', 'INTEGER');
          }
          if (oldVersion < 5) {
            // Why a queued command was rejected, surfaced by the failed-sync
            // screen so a sale stuck in the queue is visible instead of silent.
            await _addColumnIfMissing(
                db, 'pending_commands', 'last_error', 'TEXT');
          }
          if (oldVersion < 6) {
            await _createLastSale(db);
          }
          if (oldVersion < 7) {
            // Distinguish a recoverable conflict (state changed on another
            // device) from a plain rejection, and keep the server's own
            // explanation for the conflict-review screen.
            await _addColumnIfMissing(
                db, 'pending_commands', 'error_detail', 'TEXT');
            await _addColumnIfMissing(db, 'pending_commands', 'is_conflict',
                'INTEGER NOT NULL DEFAULT 0');
          }
          if (oldVersion < 8) {
            // The incremental change feed turns the cache into the POS's
            // primary catalog, so a cached row must keep enough to render the
            // grid offline: the category it filters under and its Arabic name.
            await _addColumnIfMissing(
                db, 'cached_products', 'category_id', "TEXT NOT NULL DEFAULT ''");
            await _addColumnIfMissing(
                db, 'cached_products', 'name_ar', "TEXT NOT NULL DEFAULT ''");
            await _createCachedCategories(db);
          }
        },
      ),
    );
    return _database!;
  }

  /// Where the database lives. An injected factory (tests, or a caller that
  /// wants its own database) and an explicit [databasePath] win over the
  /// platform default, so existing callers keep their behaviour on every
  /// platform.
  Future<DatabaseTarget> _target() async {
    final injected = _factory;
    if (injected != null) {
      return DatabaseTarget(
        factory: injected,
        path: databasePath ??
            path.join(await injected.getDatabasesPath(), 'pos_go.db'),
      );
    }
    final resolved = await resolveDatabaseTarget();
    return DatabaseTarget(
      factory: resolved.factory,
      path: databasePath ?? resolved.path,
    );
  }

  /// Shifts taken while the register session could not be opened on the server.
  /// Created by [_createLocalSessions] for both fresh installs and upgrades so
  /// the two paths cannot diverge.
  static Future<void> _createLocalSessions(Database db) async {
    await db.execute('''
      CREATE TABLE IF NOT EXISTS local_sessions (
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
  }

  /// The id of the newest sale completed on this device for a tenant.
  /// Created by [_createLastSale] for both fresh installs and upgrades so the
  /// two paths cannot diverge.
  static Future<void> _createLastSale(Database db) async {
    await db.execute('''
      CREATE TABLE IF NOT EXISTS last_sale (
        tenant_id TEXT PRIMARY KEY,
        sale_id TEXT NOT NULL,
        saved_at INTEGER NOT NULL
      )
    ''');
  }

  /// Categories the POS filters its grid by, cached so a warm start can render
  /// the chips before the change feed answers. Created for both fresh installs
  /// and upgrades so the two paths cannot diverge.
  static Future<void> _createCachedCategories(Database db) async {
    await db.execute('''
      CREATE TABLE IF NOT EXISTS cached_categories (
        id TEXT PRIMARY KEY,
        name TEXT NOT NULL,
        slug TEXT NOT NULL,
        name_ar TEXT NOT NULL DEFAULT ''
      )
    ''');
  }

  /// Adds [column] to [table] only when it is not already present.
  ///
  /// `ALTER TABLE ... ADD COLUMN` is not idempotent, and `onUpgrade` re-runs
  /// whenever an earlier open was interrupted before sqflite persisted the new
  /// `user_version`. Replaying it raises `duplicate column name`, the upgrade
  /// transaction rolls back, and every later launch fails the same way — the
  /// database never opens, so the app cannot reach its UI.
  static Future<void> _addColumnIfMissing(
    DatabaseExecutor db,
    String table,
    String column,
    String type,
  ) async {
    final info = await db.rawQuery('PRAGMA table_info($table)');
    // A missing table means there is nothing to widen yet (the create path
    // owns it); ALTERing it here would raise `no such table` and roll back the
    // whole upgrade, which is exactly the bricking this helper exists to stop.
    if (info.isEmpty) return;
    final existing = info.map((row) => row['name'] as String).toSet();
    if (existing.contains(column)) return;
    await db.execute('ALTER TABLE $table ADD COLUMN $column $type');
  }

  Future<void> cacheProducts(List<Product> products) async {
    final db = await database;
    final batch = db.batch();
    await db.delete('cached_products');
    for (final p in products) {
      batch.insert(
          'cached_products', _productRow(p),
          conflictAlgorithm: ConflictAlgorithm.replace);
    }
    await batch.commit(noResult: true);
  }

  /// Applies the product half of a change-feed page on top of the cache:
  /// [upserts] replace/insert by id and [deletes] drop the row. Kept separate
  /// from [cacheProducts] so an incremental pull never has to wipe the table
  /// first (which would blank the grid between the delete and the insert).
  Future<void> applyProductChanges({
    List<Product> upserts = const [],
    List<String> deletes = const [],
  }) async {
    final db = await database;
    final batch = db.batch();
    for (final id in deletes) {
      batch.delete('cached_products', where: 'id = ?', whereArgs: [id]);
    }
    for (final p in upserts) {
      batch.insert(
          'cached_products', _productRow(p),
          conflictAlgorithm: ConflictAlgorithm.replace);
    }
    await batch.commit(noResult: true);
  }

  static Map<String, Object?> _productRow(Product p) => {
        'id': p.id,
        'name': p.name,
        'sku': p.sku,
        'barcode': p.barcode,
        'price_minor': p.priceMinor,
        'cost_minor': p.costMinor,
        'currency': p.currency,
        'stock_quantity': p.stockQuantity,
        'image_url': p.imageUrl,
        'category_id': p.categoryId,
        'name_ar': p.nameAr,
      };

  Future<List<Product>> cachedProducts() async {
    final rows = await (await database).query('cached_products', orderBy: 'name');
    return rows
        .map((row) => Product(
              id: row['id'] as String,
              name: row['name'] as String,
              sku: row['sku'] as String,
              barcode: row['barcode'] as String? ?? '',
              priceMinor: row['price_minor'] as int,
              costMinor: row['cost_minor'] as int,
              currency: row['currency'] as String,
              stockQuantity: row['stock_quantity'] as int,
              imageUrl: row['image_url'] as String? ?? '',
              categoryId: row['category_id'] as String? ?? '',
              nameAr: row['name_ar'] as String? ?? '',
            ))
        .toList();
  }

  Future<void> cacheCategories(List<Category> categories) async {
    final db = await database;
    final batch = db.batch();
    await db.delete('cached_categories');
    for (final c in categories) {
      batch.insert('cached_categories', _categoryRow(c),
          conflictAlgorithm: ConflictAlgorithm.replace);
    }
    await batch.commit(noResult: true);
  }

  /// Applies the category half of a change-feed page on top of the cache.
  Future<void> applyCategoryChanges({
    List<Category> upserts = const [],
    List<String> deletes = const [],
  }) async {
    final db = await database;
    final batch = db.batch();
    for (final id in deletes) {
      batch.delete('cached_categories', where: 'id = ?', whereArgs: [id]);
    }
    for (final c in upserts) {
      batch.insert('cached_categories', _categoryRow(c),
          conflictAlgorithm: ConflictAlgorithm.replace);
    }
    await batch.commit(noResult: true);
  }

  static Map<String, Object?> _categoryRow(Category c) => {
        'id': c.id,
        'name': c.name,
        'slug': c.slug,
        'name_ar': c.nameAr,
      };

  Future<List<Category>> cachedCategories() async {
    final rows =
        await (await database).query('cached_categories', orderBy: 'name');
    return rows
        .map((row) => Category(
              id: row['id'] as String,
              name: row['name'] as String,
              slug: row['slug'] as String,
              nameAr: row['name_ar'] as String? ?? '',
            ))
        .toList();
  }

  /// Drops the cached catalog so a full resynchronization can rebuild it. Used
  /// when the change feed reports an expired cursor and the bridge is gone.
  Future<void> clearCatalogCache() async {
    final db = await database;
    final batch = db.batch();
    batch.delete('cached_products');
    batch.delete('cached_categories');
    await batch.commit(noResult: true);
  }

  Future<void> enqueue(PendingCommand command) async {
    final db = await database;
    await db.insert(
      'pending_commands',
      {
        'id': command.id,
        'idempotency_key': command.idempotencyKey,
        'operation': command.operation,
        'payload': command.payload,
        'created_at': DateTime.now().millisecondsSinceEpoch,
      },
      conflictAlgorithm: ConflictAlgorithm.ignore,
    );
  }

  static PendingCommand _commandFromRow(Map<String, Object?> row) =>
      PendingCommand(
        id: row['id']! as String,
        idempotencyKey: row['idempotency_key']! as String,
        operation: row['operation']! as String,
        payload: row['payload']! as String,
        attempts: (row['attempts'] as num?)?.toInt() ?? 0,
        lastError: row['last_error'] as String?,
        errorDetail: row['error_detail'] as String?,
        isConflict: ((row['is_conflict'] as num?)?.toInt() ?? 0) != 0,
      );

  Future<List<PendingCommand>> pendingCommands() async {
    final rows = await (await database).query(
      'pending_commands',
      where: 'status = ?',
      whereArgs: ['pending'],
      orderBy: 'created_at ASC',
    );
    return rows.map(_commandFromRow).toList();
  }

  /// Commands the server refused that never entered the feed: kept in their
  /// own status so they neither block the queue nor vanish silently. Each row
  /// carries [PendingCommand.isConflict] so callers can tell a recoverable
  /// conflict from a plain rejection.
  Future<List<PendingCommand>> failedCommands() async {
    final rows = await (await database).query(
      'pending_commands',
      where: 'status = ?',
      whereArgs: ['failed'],
      orderBy: 'created_at ASC',
    );
    return rows.map(_commandFromRow).toList();
  }

  Future<void> markComplete(String id) async {
    await (await database).update(
      'pending_commands',
      {'status': 'complete'},
      where: 'id = ?',
      whereArgs: [id],
    );
  }

  /// Parks a command the server rejected, recording why and how often.
  ///
  /// [conflict] marks a recoverable clash with state created elsewhere, so the
  /// review screen can tell it apart from a hard rejection.
  Future<void> failCommand(
    String id, {
    String? errorCode,
    String? errorDetail,
    bool conflict = false,
  }) async {
    await (await database).rawUpdate(
      'UPDATE pending_commands SET status = ?, attempts = attempts + 1, '
      'last_error = ?, error_detail = ?, is_conflict = ? WHERE id = ?',
      ['failed', errorCode, errorDetail, conflict ? 1 : 0, id],
    );
  }

  /// Moves a failed command back into the replay queue, clearing the previous
  /// failure so the next attempt starts clean.
  Future<void> retryCommand(String id) async {
    await (await database).update(
      'pending_commands',
      {
        'status': 'pending',
        'last_error': null,
        'error_detail': null,
        'is_conflict': 0,
      },
      where: 'id = ?',
      whereArgs: [id],
    );
  }

  /// Discards a failed command for good.
  Future<void> dropCommand(String id) async {
    await (await database).delete(
      'pending_commands',
      where: 'id = ?',
      whereArgs: [id],
    );
  }

  /// Records a shift taken offline. Re-saving the same [session] id updates it,
  /// so closing a shift is just a second write.
  Future<void> saveLocalSession(LocalRegisterSession session) async {
    await (await database).insert(
      'local_sessions',
      session.toRow(),
      conflictAlgorithm: ConflictAlgorithm.replace,
    );
  }

  /// The shift currently open on this device, or null when there is none.
  ///
  /// A terminal has at most one open shift, so the newest open row wins; this
  /// is also what makes the value safe to call on every screen entry.
  Future<LocalRegisterSession?> openLocalSession() async {
    final rows = await (await database).query(
      'local_sessions',
      where: 'status = ?',
      whereArgs: ['open'],
      orderBy: 'created_at DESC',
      limit: 1,
    );
    return rows.isEmpty ? null : LocalRegisterSession.fromRow(rows.first);
  }

  /// Every locally recorded shift, oldest first, for reconciliation on
  /// reconnect.
  Future<List<LocalRegisterSession>> localSessions() async {
    final rows = await (await database).query(
      'local_sessions',
      orderBy: 'created_at ASC',
    );
    return rows.map(LocalRegisterSession.fromRow).toList();
  }

  /// Remembers the newest sale completed on this device for [tenantId], so the
  /// POS can offer a one-tap reprint without paging through sale history.
  /// Scoped per tenant: a device that switches businesses must not reprint
  /// another tenant's receipt.
  Future<void> saveLastSale(String tenantId, String saleId) async {
    await (await database).insert(
      'last_sale',
      {
        'tenant_id': tenantId,
        'sale_id': saleId,
        'saved_at': DateTime.now().millisecondsSinceEpoch,
      },
      conflictAlgorithm: ConflictAlgorithm.replace,
    );
  }

  /// The sale [saveLastSale] recorded for [tenantId], or null when this device
  /// has not completed one for that tenant yet.
  Future<String?> readLastSale(String tenantId) async {
    final rows = await (await database).query(
      'last_sale',
      columns: ['sale_id'],
      where: 'tenant_id = ?',
      whereArgs: [tenantId],
      limit: 1,
    );
    return rows.isEmpty ? null : rows.first['sale_id'] as String;
  }

  Future<void> close() async {
    await _database?.close();
    _database = null;
  }
}
