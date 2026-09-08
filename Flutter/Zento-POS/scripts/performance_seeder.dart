import 'dart:io';
import 'dart:math';
import 'package:path/path.dart';
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

void main() async {
  sqfliteFfiInit();
  final databaseFactory = databaseFactoryFfi;
  
  final dbPath = join(Directory.current.path, '.dart_tool', 'sqflite_common_ffi', 'databases', 'pos_system_v2.db');
  
  print('🚀 --- THE ULTIMATE POS STRESS SEEDER --- 🚀');
  print('Target database: $dbPath');
  
  if (!await File(dbPath).exists()) {
    print('❌ Error: Database file not found. Please run the app first to initialize the DB.');
    return;
  }

  final db = await databaseFactory.openDatabase(dbPath);
  final random = Random();

  try {
    await db.transaction((txn) async {
      print('🗑️ Clearing existing data for a fresh Super-Test...');
      await txn.delete('order_items');
      await txn.delete('orders');
      await txn.delete('product_addons');
      await txn.delete('product_ingredients');
      await txn.delete('products');
      await txn.delete('categories');
      await txn.delete('tables');
      await txn.delete('expenses');
      await txn.delete('shifts');
      await txn.delete('stock_history');

      print('📂 Seeding 15 Categories...');
      final categoriesData = [
        {'name': '🔥 Signature Burgers', 'color': 0xFFE74C3C},
        {'name': '🍕 Artisanal Pizzas', 'color': 0xFFF39C12},
        {'name': '🥗 Fresh Salads', 'color': 0xFF2ECC71},
        {'name': '🍝 Italian Pasta', 'color': 0xFF3498DB},
        {'name': '🍟 Sides & Snacks', 'color': 0xFF9B59B6},
        {'name': '🥤 Soft Drinks', 'color': 0xFF1ABC9C},
        {'name': '🍺 Craft Beers', 'color': 0xFFD35400},
        {'name': '☕ Specialty Coffee', 'color': 0xFF795548},
        {'name': '🍰 Gourmet Desserts', 'color': 0xFFE91E63},
        {'name': '🌮 Mexican Street Food', 'color': 0xFFFFEB3B},
        {'name': '🍣 Sushi Rolls', 'color': 0xFF607D8B},
        {'name': '🍜 Asian Fusion', 'color': 0xFFE67E22},
        {'name': '🥪 Breakfast Club', 'color': 0xFFCDDC39},
        {'name': '🍹 Cocktails', 'color': 0xFF9C27B0},
        {'name': '🥩 Steaks & Grill', 'color': 0xFF212121},
      ];

      final categoryIds = [];
      for (int i = 0; i < categoriesData.length; i++) {
        final id = await txn.insert('categories', {
          'name': categoriesData[i]['name'],
          'orderIndex': i,
          'colorValue': categoriesData[i]['color'],
          'isEnabled': 1,
        });
        categoryIds.add(id);
      }

      print('🍱 Seeding 450 Products...');
      final productIds = [];
      for (final catId in categoryIds) {
        for (int i = 1; i <= 30; i++) {
          final price = (200 + random.nextInt(2000));
          final cost = (price * 0.3).toInt();
          final id = await txn.insert('products', {
            'categoryId': catId,
            'name': 'Item #${catId}-${i}',
            'price': price,
            'cost': cost,
            'isEnabled': 1,
            'stockQuantity': 100 + random.nextInt(500),
            'trackStock': 1,
            'alertLevel': 10,
          });
          productIds.add({'id': id, 'price': price, 'name': 'Item #${catId}-${i}'});

          await txn.insert('product_addons', {
            'productId': id,
            'name': 'Extra Sauce',
            'price': 50,
            'isEnabled': 1,
          });
        }
      }

      print('🪑 Seeding 20 Tables...');
      for (int i = 1; i <= 20; i++) {
        await txn.insert('tables', {
          'name': 'Table $i',
          'is_active': 1, // Fixed column name
        });
      }

      print('🕒 Seeding 120 Shifts...');
      final now = DateTime.now();
      final twoYearsAgo = now.subtract(const Duration(days: 730));
      final shiftIds = [];
      for (int i = 0; i < 120; i++) {
        final start = twoYearsAgo.add(Duration(days: i * 6));
        final id = await txn.insert('shifts', {
          'userId': 1, // Fixed column name
          'startTime': start.toIso8601String(), // Fixed type
          'endTime': start.add(const Duration(hours: 8)).toIso8601String(),
          'openingCash': 500.0,
          'totalSales': 1200.0 + random.nextInt(1000),
          'expectedCash': 1700.0,
          'status': 'CLOSED',
        });
        shiftIds.add(id);
      }

      print('⌛ Seeding 8,000 Orders (2 years history)...');
      for (int i = 0; i < 8000; i++) {
        if (i % 2000 == 0 && i > 0) print('📈 Progress: $i orders seeded...');
        
        final timestamp = twoYearsAgo.add(Duration(minutes: (i * 131.4).toInt())).millisecondsSinceEpoch;
        
        final orderId = await txn.insert('orders', {
          'cashierId': 1,
          'shiftId': shiftIds[random.nextInt(shiftIds.length)],
          'totalAmount': 0.0, 
          'orderType': random.nextInt(2),
          'timestamp': timestamp,
          'status': 1, 
          'paymentMethod': random.nextInt(3),
        });

        int totalAmountCents = 0;
        final itemCount = random.nextInt(4) + 1;
        for (int j = 0; j < itemCount; j++) {
          final p = productIds[random.nextInt(productIds.length)];
          final qty = random.nextInt(3) + 1;
          
          await txn.insert('order_items', {
            'orderId': orderId,
            'productId': p['id'],
            'productName': p['name'],
            'quantity': qty,
            'priceAtTime': (p['price'] as int) / 100.0,
          });
          totalAmountCents += (p['price'] as int) * qty;
        }

        await txn.update('orders', {'totalAmount': totalAmountCents / 100.0}, where: 'id = ?', whereArgs: [orderId]);
      }

      print('💰 Seeding 50 Expenses...');
      for (int i = 1; i <= 50; i++) {
        await txn.insert('expenses', {
          'description': 'Expense $i',
          'amount': (500 + random.nextInt(5000)) / 100.0,
          'category': ['Rent', 'Utilities', 'Salary', 'Supplies'][random.nextInt(4)],
          'timestamp': DateTime.now().subtract(Duration(days: random.nextInt(30))).millisecondsSinceEpoch,
          'wasPaidFromDrawer': 1,
        });
      }
    });

    print('✅ SUCCESS! App is now loaded with 8,000 orders and 450 products.');
  } catch (e) {
    print('❌ Seeding failed: $e');
  } finally {
    await db.close();
  }
}
