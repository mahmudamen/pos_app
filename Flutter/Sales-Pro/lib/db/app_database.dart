import 'dart:io';
import 'dart:convert';
import 'package:drift/drift.dart';
import 'package:drift/native.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';

part 'app_database.g.dart';

class Stores extends Table {
  IntColumn get id => integer()();
  TextColumn get name => text()();
  DateTimeColumn get updatedAt => dateTime()();
  @override
  Set<Column> get primaryKey => {id};
}

class Prices extends Table {
  TextColumn get id => text()();
  TextColumn get barcode => text()();
  TextColumn get name => text()();
  RealColumn get price => real()();
  RealColumn get availableQty => real()();
  IntColumn get storeId => integer()();
  DateTimeColumn get updatedAt => dateTime()();
  @override
  Set<Column> get primaryKey => {id, storeId};
}

class Bills extends Table {
  TextColumn get id => text()();
  TextColumn get date => text()();
  RealColumn get total => real()();
  TextColumn get customer => text()();
  DateTimeColumn get updatedAt => dateTime()();
  @override
  Set<Column> get primaryKey => {id};
}

class InvoiceDraft extends Table {
  IntColumn get id => integer().autoIncrement()();
  TextColumn get customerId => text()();
  TextColumn get customerName =>
      text().withDefault(const Constant(''))(); // 👈 اسم الزبون المضاف
  IntColumn get storeId => integer()();
  TextColumn get credit => text()();
  TextColumn get note => text().withDefault(const Constant(''))();
  TextColumn get linesJson => text()();
  TextColumn get status => text().withDefault(
    const Constant('pending'),
  )(); // pending | sent | failed | draft
  IntColumn get tryCount => integer().withDefault(const Constant(0))();
  TextColumn get lastError => text().withDefault(const Constant(''))();
  DateTimeColumn get createdAt => dateTime().withDefault(currentDateAndTime)();
}

LazyDatabase _open() => LazyDatabase(() async {
  final dir = await getApplicationDocumentsDirectory();
  final file = File(p.join(dir.path, 'sales_pro_offline.sqlite'));
  // if (await file.exists()) {
  //   await file.delete();
  //   print('✅ تم حذف قاعدة البيانات مؤقتًا.');
  // } else {
  //   print('⚠️ لا يوجد ملف قاعدة بيانات لحذفه.');
  // }
  return NativeDatabase.createInBackground(file);
});

@DriftDatabase(tables: [Stores, Prices, Bills, InvoiceDraft])
class AppDatabase extends _$AppDatabase {
  AppDatabase() : super(_open());
  @override
  int get schemaVersion => 1;

  // Stores
  Future<void> upsertStores(List<Map<String, dynamic>> rows) async {
    await batch((b) {
      b.insertAllOnConflictUpdate(
        stores,
        rows.map(
          (e) => StoresCompanion(
            id: Value(e['id'] as int),
            name: Value(e['name'] as String),
            updatedAt: Value(e['updatedAt'] as DateTime),
          ),
        ),
      );
    });
  }

  Future<List<Store>> getAllStores() => select(stores).get();
  Future<DateTime?> storesLastUpdate() async {
    final q =
        await (select(stores)
              ..orderBy([
                (t) => OrderingTerm(
                  expression: t.updatedAt,
                  mode: OrderingMode.desc,
                ),
              ])
              ..limit(1))
            .getSingleOrNull();
    return q?.updatedAt;
  }

  // Prices
  Future<void> upsertPrices(List<Map<String, dynamic>> rows) async {
    await batch((b) {
      b.insertAllOnConflictUpdate(
        prices,
        rows.map(
          (e) => PricesCompanion(
            id: Value(e['id'] as String),
            barcode: Value(e['barcode'] as String),
            name: Value(e['name'] as String),
            price: Value((e['price'] as num).toDouble()),
            availableQty: Value((e['availableQty'] as num).toDouble()),
            storeId: Value(e['storeId'] as int),
            updatedAt: Value(e['updatedAt'] as DateTime),
          ),
        ),
      );
    });
  }

  Future<List<Price>> getPricesByStore(int storeId) {
    return (select(prices)..where((t) => t.storeId.equals(storeId))).get();
  }

  Future<DateTime?> pricesLastUpdate(int storeId) async {
    final q =
        await (select(prices)
              ..where((t) => t.storeId.equals(storeId))
              ..orderBy([
                (t) => OrderingTerm(
                  expression: t.updatedAt,
                  mode: OrderingMode.desc,
                ),
              ])
              ..limit(1))
            .getSingleOrNull();
    return q?.updatedAt;
  }

  // Bills
  Future<void> upsertBills(List<Map<String, dynamic>> rows) async {
    await batch((b) {
      b.insertAllOnConflictUpdate(
        bills,
        rows.map(
          (e) => BillsCompanion(
            id: Value(e['id'] as String),
            date: Value(e['date'] as String),
            total: Value((e['total'] as num).toDouble()),
            customer: Value(e['customer'] as String),
            updatedAt: Value(e['updatedAt'] as DateTime),
          ),
        ),
      );
    });
  }

  Future<List<Bill>> getAllBills() => select(bills).get();
  Future<DateTime?> billsLastUpdate() async {
    final q =
        await (select(bills)
              ..orderBy([
                (t) => OrderingTerm(
                  expression: t.updatedAt,
                  mode: OrderingMode.desc,
                ),
              ])
              ..limit(1))
            .getSingleOrNull();
    return q?.updatedAt;
  }

  // Queue
  Future<int> enqueueInvoice({
    required String customerId,
    required String customerName, // 👈 أضف هذا الباراميتر
    required int storeId,
    required String credit,
    String note = '',
    required List<Map<String, dynamic>> lines,
  }) {
    return into(invoiceDraft).insert(
      InvoiceDraftCompanion.insert(
        customerId: customerId,
        customerName: Value(customerName), // 👈 تخزين الاسم في قاعدة البيانات
        storeId: storeId,
        credit: credit,
        note: Value(note),
        linesJson: jsonEncode(lines),
      ),
    );
  }

  Future<List<InvoiceDraftData>> pendingInvoices() {
    return (select(invoiceDraft)
          ..where((t) => t.status.equals('pending'))
          ..orderBy([(t) => OrderingTerm(expression: t.createdAt)]))
        .get();
  }

  Future<void> markInvoiceSent(int id) async {
    await (update(invoiceDraft)..where((t) => t.id.equals(id))).write(
      const InvoiceDraftCompanion(status: Value('sent')),
    );
  }

  Future<void> markInvoiceFailed(int id, String err) async {
    final item = await (select(
      invoiceDraft,
    )..where((t) => t.id.equals(id))).getSingle();
    await (update(invoiceDraft)..where((t) => t.id.equals(id))).write(
      InvoiceDraftCompanion(
        status: const Value('failed'),
        tryCount: Value(item.tryCount + 1),
        lastError: Value(err),
      ),
    );
  }

  Future<int> saveInvoiceDraft({
    required String customerId,
    required String customerName, // 👈 أضف هذا
    required int storeId,
    required String credit,
    String note = '',
    required List<Map<String, dynamic>> lines,
  }) {
    return into(invoiceDraft).insert(
      InvoiceDraftCompanion.insert(
        customerId: customerId,
        customerName: Value(customerName), // 👈 أضف هذا
        storeId: storeId,
        credit: credit,
        note: Value(note),
        linesJson: jsonEncode(lines),
        status: const Value('draft'),
      ),
    );
  }
}
