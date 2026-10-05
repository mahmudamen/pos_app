import 'package:path/path.dart' as p;
import 'package:sqflite/sqflite.dart';

import 'database_target.dart';

export 'database_target.dart' show DatabaseTarget;

/// Android/iOS (and desktop) keep the database on the filesystem, inside the
/// app's private data directory.
Future<DatabaseTarget> resolveDatabaseTarget() async {
  final dir = await databaseFactory.getDatabasesPath();
  return DatabaseTarget(factory: databaseFactory, path: p.join(dir, 'pos_go.db'));
}