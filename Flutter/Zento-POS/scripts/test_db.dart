import 'dart:io';
import 'package:path/path.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

void main() async {
  try {
    print('Checking environment...');
    sqfliteFfiInit();
    final databaseFactory = databaseFactoryFfi;
    
    final dbPath = join(Directory.current.path, '.dart_tool', 'sqflite_common_ffi', 'databases', 'pos_system_v2.db');
    print('DB Path: $dbPath');

    if (!await File(dbPath).exists()) {
      print('❌ DB Not Found');
      return;
    }

    final db = await databaseFactory.openDatabase(dbPath);
    print('DB Opened.');

    await db.transaction((txn) async {
      print('Testing minimal insert...');
      await txn.insert('categories', {
         'name': 'Test Category',
         'orderIndex': 0,
         'colorValue': 0xFF000000,
         'isEnabled': 1,
      });
    });

    print('✅ Minimal test success!');
    await db.close();
  } catch (e, stack) {
    print('❌ ERROR: $e');
    print(stack);
  }
}
