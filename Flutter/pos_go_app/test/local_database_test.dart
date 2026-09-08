import 'package:flutter_test/flutter_test.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

import 'package:pos_go_app/core/storage/local_database.dart';

void main() {
  sqfliteFfiInit();
  test('pending commands are durable and ordered', () async {
    final database = LocalDatabase(
      factory: databaseFactoryFfi,
      databasePath: ':memory:',
    );
    await database.enqueue(const PendingCommand(
      id: 'command-1',
      idempotencyKey: 'sale-1',
      operation: 'create_sale',
      payload: '{"items":[]}',
    ));

    final commands = await database.pendingCommands();
    expect(commands.single.id, 'command-1');
    expect(commands.single.operation, 'create_sale');

    await database.markComplete('command-1');
    expect(await database.pendingCommands(), isEmpty);
    await database.close();
  });
}
