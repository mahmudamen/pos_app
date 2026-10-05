import 'package:sqflite_common_ffi_web/sqflite_ffi_web.dart';

import 'database_target.dart';

export 'database_target.dart' show DatabaseTarget;

/// The browser has no filesystem: `sqflite` runs sqlite3 compiled to
/// WebAssembly and persists it in IndexedDB under the page's origin, so [path]
/// is only a name for the database inside that IndexedDB store.
///
/// `dart run sqflite_common_ffi_web:setup` copies `sqlite3.wasm` and
/// `sqflite_sw.js` into `web/`, where they are served next to the app.
Future<DatabaseTarget> resolveDatabaseTarget() async {
  await sqfliteFfiWebLoadSqlite3Wasm(SqfliteFfiWebOptions());
  return DatabaseTarget(
    factory: databaseFactoryFfiWeb,
    path: 'pos_go.db',
  );
}