import 'package:path/path.dart' as path;
import 'package:sqflite/sqflite.dart';

class PendingCommand {
  const PendingCommand({
    required this.id,
    required this.idempotencyKey,
    required this.operation,
    required this.payload,
  });

  final String id;
  final String idempotencyKey;
  final String operation;
  final String payload;
}

class LocalDatabase {
  LocalDatabase({DatabaseFactory? factory, this.databasePath})
      : _factory = factory ?? databaseFactory;

  final DatabaseFactory _factory;
  final String? databasePath;
  Database? _database;

  Future<Database> get database async {
    if (_database != null) return _database!;
    final databasesPath = databasePath ??
        path.join(await _factory.getDatabasesPath(), 'pos_go.db');
    _database = await _factory.openDatabase(
      databasesPath,
      options: OpenDatabaseOptions(
        version: 1,
        onCreate: (db, _) async {
          await db.execute('''
            CREATE TABLE cached_products (
              id TEXT PRIMARY KEY,
              name TEXT NOT NULL,
              sku TEXT NOT NULL,
              barcode TEXT,
              price_minor INTEGER NOT NULL,
              currency TEXT NOT NULL,
              stock_quantity INTEGER NOT NULL
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
    return _database!;
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

  Future<List<PendingCommand>> pendingCommands() async {
    final rows = await (await database).query(
      'pending_commands',
      where: 'status = ?',
      whereArgs: ['pending'],
      orderBy: 'created_at ASC',
    );
    return rows
        .map(
          (row) => PendingCommand(
            id: row['id']! as String,
            idempotencyKey: row['idempotency_key']! as String,
            operation: row['operation']! as String,
            payload: row['payload']! as String,
          ),
        )
        .toList();
  }

  Future<void> markComplete(String id) async {
    await (await database).update(
      'pending_commands',
      {'status': 'complete'},
      where: 'id = ?',
      whereArgs: [id],
    );
  }

  Future<void> close() async {
    await _database?.close();
    _database = null;
  }
}
