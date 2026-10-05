@TestOn('browser')
library;

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:pos_go_app/core/storage/local_database.dart';

/// Runs only in a real browser (`flutter test --platform chrome`), because it
/// exercises the web offline-database path for real: `LocalDatabase` with no
/// injected factory resolves to `databaseFactoryFfiWeb`, which runs sqlite3
/// compiled to WebAssembly and persists to IndexedDB.
///
/// This is the one assertion the whole web support rests on -- a POS whose
/// product cache and pending-sale queue cannot open is a POS that cannot sell
/// offline. The VM tests inject `sqflite_common_ffi` instead and therefore say
/// nothing about it.
void main() {
  test('the web offline database opens, queues a sale and reads it back', () async {
    // `flutter test --platform chrome` serves only the generated test bundle,
    // not `web/`, so sqlite3.wasm is unreachable there and the failure would be
    // the harness rather than the app. Check first and skip with the reason, so
    // this file never reports a false failure.
    final wasm = await http.get(Uri.base.resolve('sqlite3.wasm'));
    if (wasm.statusCode != 200) {
      markTestSkipped(
        'the flutter web test harness does not serve web/ (sqlite3.wasm is '
        '404). Verify in a real browser: build web with '
        '--no-web-resources-cdn and open it over https or localhost.',
      );
      return;
    }

    final db = LocalDatabase();
    // Unique per run: the database lives in IndexedDB and survives reloads, so a
    // fixed key would be swallowed by the UNIQUE conflict-ignore on the second
    // run and the test would silently assert nothing.
    final key = 'web-${DateTime.now().microsecondsSinceEpoch}';

    await db.enqueue(
      PendingCommand(
        id: key,
        idempotencyKey: key,
        operation: 'create_sale',
        payload: '{"total_minor":1250}',
      ),
    );

    final pending = await db.pendingCommands();
    expect(
      pending.any((c) => c.idempotencyKey == key),
      isTrue,
      reason: 'the queued sale should come back out of IndexedDB',
    );

    await db.markComplete(key);
    expect(
      await db.pendingCommands(),
      isEmpty,
      reason: 'a completed sale leaves the replay queue',
    );

    await db.close();
  });
}