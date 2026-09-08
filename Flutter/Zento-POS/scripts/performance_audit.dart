import 'dart:io';
import 'package:path/path.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

void main() async {
  sqfliteFfiInit();
  final databaseFactory = databaseFactoryFfi;
  final dbPath = join(Directory.current.path, '.dart_tool', 'sqflite_common_ffi', 'databases', 'pos_system_v2.db');
  
  print('--- POS Performance Audit ---');
  final db = await databaseFactory.openDatabase(dbPath);
  
  // Ensure indices exist for the test
  await db.execute('CREATE INDEX IF NOT EXISTS idx_orders_timestamp ON orders (timestamp)');
  await db.execute('CREATE INDEX IF NOT EXISTS idx_order_items_orderId ON order_items (orderId)');
  await db.execute('CREATE INDEX IF NOT EXISTS idx_stock_history_productId ON stock_history (productId)');
  await db.execute('CREATE INDEX IF NOT EXISTS idx_shifts_endTime ON shifts (endTime)');
  await db.execute('CREATE INDEX IF NOT EXISTS idx_products_stock ON products (trackStock, stockQuantity)');

  final stopwatch = Stopwatch();

  // Test 1: Today Revenue (Filtered by date range)
  stopwatch.start();
  final todayStart = DateTime.now().subtract(const Duration(hours: 24)).millisecondsSinceEpoch;
  final todayEnd = DateTime.now().millisecondsSinceEpoch;
  final revenue = await db.rawQuery(
    'SELECT SUM(totalAmount) as total FROM orders WHERE timestamp BETWEEN ? AND ? AND status = 0',
    [todayStart, todayEnd]
  );
  stopwatch.stop();
  print('Query: Today Revenue (Data scan) | Time: ${stopwatch.elapsedMilliseconds}ms | Result: ${revenue.first['total']}');

  // Test 2: Monthly Revenue (Complex aggregation)
  stopwatch.reset();
  stopwatch.start();
  final monthStart = DateTime(2023, 1, 1).millisecondsSinceEpoch;
  final monthEnd = DateTime(2023, 1, 31).millisecondsSinceEpoch;
  final monthly = await db.rawQuery('''
    SELECT strftime('%d', datetime(timestamp / 1000, 'unixepoch')) as day, SUM(totalAmount) as total
    FROM orders
    WHERE timestamp BETWEEN ? AND ? AND status = 0
    GROUP BY day
  ''', [monthStart, monthEnd]);
  stopwatch.stop();
  print('Query: Monthly Revenue (Aggregation) | Time: ${stopwatch.elapsedMilliseconds}ms | Rows: ${monthly.length}');

  // Test 3: Low Stock Alerts (Join + Filter)
  stopwatch.reset();
  stopwatch.start();
  final lowStock = await db.query(
    'products',
    where: 'trackStock = 1 AND stockQuantity <= alertLevel',
    orderBy: 'stockQuantity ASC',
    limit: 5
  );
  stopwatch.stop();
  print('Query: Low Stock Alerts (Filter/Sort) | Time: ${stopwatch.elapsedMilliseconds}ms | Items: ${lowStock.length}');

  // Test 4: Order history with Items (Joins)
  stopwatch.reset();
  stopwatch.start();
  final orders = await db.rawQuery('''
    SELECT o.*, oi.productName, oi.quantity 
    FROM orders o 
    JOIN order_items oi ON o.id = oi.orderId 
    WHERE o.timestamp > ? 
    LIMIT 100
  ''', [monthStart]);
  stopwatch.stop();
  print('Query: Order Items Join (Join/Limit) | Time: ${stopwatch.elapsedMilliseconds}ms | Records: ${orders.length}');

  await db.close();
}
