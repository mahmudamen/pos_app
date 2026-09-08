import 'dart:convert';

import 'package:sales_pro/services/notification_service.dart';
import 'package:drift/drift.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import '../features/orders/dto/table_bill_line_dto.dart' show TableBillLineDto;
import '../repository/api_repository.dart';
import '../features/auth/dto/user_dto.dart';
import '../features/catalog/dto/customer_dto.dart';
import '../features/catalog/dto/store_dto.dart';
import '../features/catalog/dto/price_dto.dart';
import '../features/invoice/dto/table_qu_dto.dart';
import '../features/orders/dto/t_bill_dto.dart';
import '../services/prefs.dart';
import '../services/prefs_tz.dart';
import '../db/app_database.dart';
import '../services/connectivity_service.dart';
import '../services/offline_cache_policy.dart';
import '../services/offline_sync_service.dart';

class AppProvider extends ChangeNotifier {
  final ApiRepository api;
  final AppDatabase _db = AppDatabase();
  final ConnectivityService connectivity = ConnectivityService();
  OfflineSyncService? _sync;
  OfflineSyncService? get syncService => _sync;

  ClsUserDto? currentUser;
  bool get isOnline => connectivity.isOnline.value;
  String? loginName;
  int? saleMan;
  List<TCustomerDto> customers = [];
  List<TableStoreDto> stores = [];
  List<TablePriceDto> prices = [];
  double balance = 0.0;
  bool busy = false;

  List<TBillDto> bills = [];
  List<InvoiceDraftData> draftInvoices = [];
  int get draftCount => draftInvoices.length;
  DateTime? billsEndDate;
  String? serverTimeZoneName;
  Duration? serverOffset;

  // ==== Language ====
  Locale _locale = const Locale('en');
  Locale get locale => _locale;

  AppProvider(this.api) {
    _loadLocale();
  }

  Future<void> _loadLocale() async {
    final code = await Prefs.getLanguageCode();
    _locale = Locale(code);
    notifyListeners();
  }

  Future<void> setLocale(Locale locale) async {
    if (_locale == locale) return;
    _locale = locale;
    await Prefs.setLanguageCode(locale.languageCode);
    notifyListeners();
  }

  // Private variable
  bool _initialized = false;

  bool get initialized => _initialized;

  Future<void> initialize() async {
    if (!_initialized) {
      try {
        // Initialize Firebase and NotificationService with timeout
        await NotificationService().init().timeout(
          const Duration(seconds: 5),
          onTimeout: () {
            debugPrint('⏰ NotificationService init timeout');
          },
        );

        // ================== Register callbacks ==================
        debugPrint('🔔 Registering notification callbacks...');

        NotificationService().onCustomerUpdate = (message) async {
          debugPrint(
            '🔔 Customer update notification received: ${message.data}',
          );
          await loadCustomers();
          notifyListeners();
          debugPrint('✅ Customer update processed');
        };

        NotificationService().onItemUpdate = (message) async {
          debugPrint('🔔 Item update notification received: ${message.data}');

          // ✅ Check for currentUser first
          if (currentUser == null) {
            debugPrint('⚠️ No current user, skipping item update');
            return;
          }

          debugPrint(
            '🔔 Current user: ${currentUser!.name}, Store: ${currentUser!.idTStore}',
          );

          try {
            final defaultStoreId = await _ensureValidStoreId();
            debugPrint('🔔 Default store ID resolved: $defaultStoreId');

            if (defaultStoreId != null && defaultStoreId > 0) {
              final defaultStoreIdStr = defaultStoreId.toString();
              debugPrint(
                '🔔 Comparing store IDs - Current: ${currentUser!.idTStore}, Default: $defaultStoreIdStr',
              );

              if (currentUser!.idTStore != defaultStoreIdStr) {
                debugPrint(
                  '🔔 Updating user store from ${currentUser!.idTStore} to $defaultStoreIdStr',
                );
                currentUser = currentUser!.copyWith(
                  idTStore: defaultStoreIdStr,
                );
              }

              debugPrint(
                '🔔 Starting to load prices for store: $defaultStoreId',
              );

              // ✅ Avoid calling loadPrices if busy
              if (!busy) {
                await loadPrices(defaultStoreId);
                debugPrint(
                  '✅ Prices loaded successfully for store: $defaultStoreId',
                );
              } else {
                debugPrint('⏳ Prices loading already in progress, skipping');
              }
            } else {
              debugPrint('⚠️ No valid default store ID found');
            }
          } catch (e, stackTrace) {
            debugPrint('❌ Error in onItemUpdate: $e');
            debugPrint('❌ StackTrace: $stackTrace');
          }
        };

        NotificationService().onWarehouseUpdate = (message) async {
          debugPrint(
            '🔔 Warehouse update notification received: ${message.data}',
          );
          await loadStores();
          notifyListeners();
          debugPrint('✅ Warehouse update processed');
        };

        debugPrint('✅ Notification callbacks registered successfully');

        // Here you can load data or other settings
        _initialized = true;
        notifyListeners();
      } catch (e) {
        debugPrint('❌ Error during initialization: $e');
        // Initialize the app basically even if some parts fail
        _initialized = true;
        notifyListeners();
      }
    }
  }

  /// Check notification service status
  void checkNotificationStatus() {
    debugPrint('🔍 Checking notification service status...');
    NotificationService().checkCallbacksStatus();
  }

  Future<void> login(
    String name,
    String password, {
    bool rememberMe = true,
  }) async {
    _setBusy(true);
    try {
      loginName = name; // Save entered name

      currentUser = await api.auth.selectUser(name: name, password: password);

      if (rememberMe) {
        await Prefs.saveUser(currentUser!); // Will save loginName as well
        await Prefs.saveLoginName(name);
        await Prefs.saveSaleMan(currentUser!.saleMan);
      } else {
        await Prefs.clearUser();
      }

      // Ensure store ID is saved as default if not exists
      final defaultStoreId = await Prefs.getDefaultStoreIdWithFallback();
      if (defaultStoreId == null || defaultStoreId <= 0) {
        final userStoreId = int.tryParse(currentUser!.idTStore);
        if (userStoreId != null && userStoreId > 0) {
          await Prefs.setDefaultStoreId(userStoreId);
          debugPrint('✅ Set user store as default during login: $userStoreId');
        }
      }

      await connectivity.start();
      connectivity.isOnline.addListener(() => notifyListeners());
      _sync ??= OfflineSyncService(
        api: api,
        db: _db,
        connectivity: connectivity,
      );
      await _sync!.start();
    } finally {
      _setBusy(false);
    }
  }

  // ================= LOGOUT =================
  Future<void> logout() async {
    _setBusy(true);
    try {
      currentUser = null;
      loginName = null;
      saleMan = null;
      customers.clear();
      stores.clear();
      prices.clear();
      bills.clear();
      draftInvoices.clear();
      balance = 0.0;
      billsEndDate = null;

      await Prefs.clearUser();

      notifyListeners();
    } finally {
      _setBusy(false);
    }
  }

  Future<void> restoreSession() async {
    try {
      final savedUser = await Prefs.getUser();
      if (savedUser == null || savedUser.idUser.isEmpty) return;

      currentUser = savedUser;

      // ✅ Use saved name from login()
      final savedLoginName = await Prefs.getLoginName();
      loginName = savedLoginName ?? savedUser.name;
      saleMan = savedUser.saleMan;

      // Start connectivity service with timeout
      await connectivity.start().timeout(
        const Duration(seconds: 3),
        onTimeout: () {
          debugPrint('⏰ Connectivity service timeout');
        },
      );
      connectivity.isOnline.addListener(() => notifyListeners());

      // Initialize sync service with error handling
      try {
        _sync ??= OfflineSyncService(
          api: api,
          db: _db,
          connectivity: connectivity,
        );
        await _sync!.start().timeout(const Duration(seconds: 5));
      } catch (e) {
        debugPrint('⚠️ OfflineSync service failed to start: $e');
      }

      // Load data with error handling
      try {
        if (stores.isEmpty) {
          await loadStores().timeout(const Duration(seconds: 8));
        }
      } catch (e) {
        debugPrint('⚠️ Failed to load stores: $e');
      }

      try {
        if (customers.isEmpty) {
          await loadCustomers().timeout(const Duration(seconds: 8));
        }
      } catch (e) {
        debugPrint('⚠️ Failed to load customers: $e');
      }

      // Load prices carefully
      try {
        final defaultStoreId = await _ensureValidStoreId();
        if (defaultStoreId != null && defaultStoreId > 0) {
          final defaultStoreIdStr = defaultStoreId.toString();
          if (currentUser!.idTStore != defaultStoreIdStr) {
            currentUser = currentUser!.copyWith(idTStore: defaultStoreIdStr);
          }
          if (prices.isEmpty) {
            await loadPrices(
              defaultStoreId,
            ).timeout(const Duration(seconds: 8));
          }
        } else {
          debugPrint(
            '⚠️ No valid default store ID found during session restore',
          );
        }
      } catch (e) {
        debugPrint('⚠️ Failed to load prices: $e');
      }

      // Refresh balance carefully
      try {
        await refreshBalance().timeout(const Duration(seconds: 5));
      } catch (e) {
        debugPrint('⚠️ Failed to refresh balance: $e');
      }
      notifyListeners();
    } catch (e) {
      debugPrint('restoreSession error: $e');
    }
  }

  Stream<List<InvoiceDraftData>> watchDraftInvoices() {
    return (_db.select(_db.invoiceDraft)
          ..where((t) => t.status.equals('draft'))
          ..orderBy([
            (t) =>
                OrderingTerm(expression: t.createdAt, mode: OrderingMode.desc),
          ]))
        .watch();
  }

  Future<void> loadCustomers() async {
    if (currentUser == null) return;
    _setBusy(true);
    try {
      customers = await api.catalog.getCustomers(currentUser!.saleMan);
    } finally {
      _setBusy(false);
    }
  }

  Future<void> loadStores() async {
    _setBusy(true);
    try {
      if (connectivity.isOnline.value) {
        try {
          stores = await api.catalog.getStores();
          await _db.upsertStores(
            stores
                .map(
                  (s) => {
                    'id': s.id,
                    'name': s.name,
                    'updatedAt': DateTime.now(),
                  },
                )
                .toList(),
          );
        } catch (_) {
          // Fallback to cache
          final cached = await _db.getAllStores();
          if (cached.isNotEmpty) {
            stores = cached
                .map((e) => TableStoreDto(id: e.id, name: e.name))
                .toList();
          }
        }
      } else {
        // offline
        final last = await _db.storesLastUpdate();
        if (last != null &&
            DateTime.now().difference(last) <= CachePolicy.storesTtl) {
          final cached = await _db.getAllStores();
          stores = cached
              .map((e) => TableStoreDto(id: e.id, name: e.name))
              .toList();
        } else {
          final cached = await _db.getAllStores();
          stores = cached
              .map((e) => TableStoreDto(id: e.id, name: e.name))
              .toList();
        }
      }
    } finally {
      _setBusy(false);
    }
  }

  Future<void> loadPrices(int storeId) async {
    debugPrint('🏪 Starting loadPrices for store: $storeId');
    debugPrint('🌐 Online status: ${connectivity.isOnline.value}');

    // ✅ Prevent concurrent calls
    if (busy) {
      debugPrint('⏳ loadPrices already in progress, skipping duplicate call');
      return;
    }

    _setBusy(true);
    try {
      if (connectivity.isOnline.value) {
        try {
          debugPrint('🌐 Fetching prices from API for store: $storeId');
          final fetchedPrices = await api.catalog.getPrice(storeId);
          debugPrint('📦 Fetched ${fetchedPrices.length} prices from API');

          // Save prices to Drift
          await _db.upsertPrices(
            fetchedPrices
                .map(
                  (p) => {
                    'id': p.id,
                    'barcode': p.barcode,
                    'name': p.name,
                    'price': p.price,
                    'availableQty': p.availableQty,
                    'storeId': storeId,
                    'updatedAt': DateTime.now(),
                  },
                )
                .toList(),
          );
          debugPrint('💾 Prices saved to local database');
        } catch (e) {
          debugPrint('⚠️ API failed, fallback to local cache: $e');
        }
      } else {
        // offline fallback
        final last = await _db.pricesLastUpdate(storeId);
        if (last != null) {
          debugPrint(
            '⚠️ Offline mode: using cached prices (last updated: $last)',
          );
        } else {
          debugPrint('⚠️ Offline mode: no cache available');
        }
      }

      // Update price list from database after fetch or update
      debugPrint('📊 Loading prices from local database for store: $storeId');
      final localPrices = await _db.getPricesByStore(storeId);
      debugPrint('📊 Found ${localPrices.length} prices in local database');

      prices = localPrices
          .map(
            (p) => TablePriceDto(
              id: p.id,
              barcode: p.barcode,
              name: p.name,
              price: p.price,
              availableQty: p.availableQty,
            ),
          )
          .toList();

      debugPrint('✅ Prices list updated with ${prices.length} items');
      debugPrint('🔔 Calling notifyListeners() from loadPrices');
      notifyListeners();
      debugPrint('✅ loadPrices completed successfully');
    } catch (e, stackTrace) {
      debugPrint('❌ Error in loadPrices: $e');
      debugPrint('❌ StackTrace: $stackTrace');
    } finally {
      _setBusy(false);
    }
  }

  Future<void> refreshBalance() async {
    if (currentUser == null && (loginName == null || loginName!.isEmpty))
      return;
    _setBusy(true);
    try {
      final target = currentUser?.saleMan ?? 0;

      balance = await api.balance.getBalance(target.toString());
    } finally {
      _setBusy(false);
    }
  }

  Future<void> loadBills({DateTime? endDate}) async {
    if (currentUser == null && ((loginName ?? '').isEmpty)) return;

    _setBusy(true);
    try {
      billsEndDate = endDate ?? billsEndDate;
      final dynamic rawId = currentUser?.idUser;
      final int idUser = rawId is int
          ? rawId
          : int.tryParse(rawId?.toString() ?? '0') ?? 0;
      if (connectivity.isOnline.value) {
        try {
          bills = await api.orders.getTBills(
            idUser: idUser,
            date1: billsEndDate,
          );
          await _db.upsertBills(
            bills
                .map(
                  (b) => {
                    'id': b.id,
                    'date': b.date,
                    'total': b.total,
                    'customer': b.customer,
                    'updatedAt': DateTime.now(),
                  },
                )
                .toList(),
          );
        } catch (_) {
          final cached = await _db.getAllBills();
          if (cached.isNotEmpty) {
            bills = cached
                .map(
                  (e) => TBillDto(
                    id: e.id,
                    date: e.date,
                    total: e.total,
                    customer: e.customer,
                  ),
                )
                .toList();
          }
        }
      } else {
        final last = await _db.billsLastUpdate();
        if (last != null &&
            DateTime.now().difference(last) <= CachePolicy.billsTtl) {
          final cached = await _db.getAllBills();
          bills = cached
              .map(
                (e) => TBillDto(
                  id: e.id,
                  date: e.date,
                  total: e.total,
                  customer: e.customer,
                ),
              )
              .toList();
        } else {
          final cached = await _db.getAllBills();
          bills = cached
              .map(
                (e) => TBillDto(
                  id: e.id,
                  date: e.date,
                  total: e.total,
                  customer: e.customer,
                ),
              )
              .toList();
        }
      }
      notifyListeners();
    } finally {
      _setBusy(false);
    }
  }

  Future<String> sendInvoice({
    required List<TableQuDto> lines,
    required String idCu,
    required String customer,
    required String credit,
  }) async {
    if (currentUser == null) throw Exception('No user');
    _setBusy(true);
    try {
      if (connectivity.isOnline.value) {
        try {
          final res = await api.invoice.setQu(
            lines: lines,
            idCu: idCu,
            idUser: currentUser!.idUser,
            credit: credit,
          );
          final dynamic rawId = currentUser!.idUser;
          final int idUser = rawId is int
              ? rawId
              : int.tryParse(rawId.toString()) ?? 0;
          bills = await api.orders.getTBills(
            idUser: idUser,
            date1: billsEndDate,
          );
          notifyListeners();
          return res;
        } catch (e) {
          return 'error in internet';
        }
      } else {
        return 'internet not connected';
      }
    } finally {
      _setBusy(false);
    }
  }

  Future<void> saveInvoiceDraft({
    required List<TableQuDto> lines,
    required String idCu,
    required String credit,
    required String customer,
  }) async {
    final int storeId =
        int.tryParse(currentUser?.idTStore?.toString() ?? '0') ?? 0;

    // Convert invoice lines to Map<String, dynamic>
    final linesMaps = lines
        .map(
          (e) => {
            'id': e.id,
            'idMat': e.idMat,
            'barcode': e.barcode,
            'name': e.name,
            'qty': e.qty,
            'idStore': e.idStore,
            'note': e.note,
            'idList': e.idList,
            'price_': e.price_,
            'stock': e.stock,
          },
        )
        .toList();

    // Save draft to database
    await _db
        .into(_db.invoiceDraft)
        .insert(
          InvoiceDraftCompanion.insert(
            customerId: idCu,
            storeId: storeId,
            credit: credit,
            customerName: Value(customer),
            note: Value(""),
            linesJson: jsonEncode(linesMaps), // Pass linesMaps not lines
            status: const Value('draft'), // Save as draft
          ),
        );
    getDraftInvoices();
    notifyListeners();
  }

  /// Update existing draft (edit)
  Future<void> updateInvoiceDraft({
    required int id,
    required List<TableQuDto> lines,
    required String idCu,
    required String credit,
    required String customer,
    String note = "",
  }) async {
    final linesMaps = lines
        .map(
          (e) => {
            'id': e.id,
            'idMat': e.idMat,
            'barcode': e.barcode,
            'name': e.name,
            'qty': e.qty,
            'idStore': e.idStore,
            'note': e.note,
            'idList': e.idList,
            'price_': e.price_,
            'stock': e.stock,
          },
        )
        .toList();

    await (_db.update(_db.invoiceDraft)..where((t) => t.id.equals(id))).write(
      InvoiceDraftCompanion(
        linesJson: Value(jsonEncode(linesMaps)),
        credit: Value(credit),
        customerId: Value(idCu),
        customerName: Value(customer),
        createdAt: Value(DateTime.now()),
      ),
    );
    // Database update completed (debug info removed for production)
    notifyListeners();
  }

  /// Retrieve all drafts

  Future<void> getDraftInvoices() async {
    draftInvoices =
        await (_db.select(_db.invoiceDraft)
              ..where((t) => t.status.equals('draft'))
              ..orderBy([(t) => OrderingTerm(expression: t.createdAt)]))
            .get();
    notifyListeners();
  }

  /// Delete invoice draft by ID
  Future<void> deleteInvoiceDraft(int id) async {
    // Delete draft from database
    await (_db.delete(_db.invoiceDraft)..where((t) => t.id.equals(id))).go();

    // Update drafts list after deletion
    await getDraftInvoices();

    notifyListeners();
  }

  Future<void> _enqueueInvoice({
    required List<TableQuDto> lines,
    required String idCu,
    required String customer,
    required String credit,
  }) async {
    final int storeId =
        int.tryParse(currentUser?.idTStore?.toString() ?? '0') ?? 0;
    final linesMaps = lines
        .map(
          (e) => {
            'id': e.id,
            'idMat': e.idMat,
            'barcode': e.barcode,
            'name': e.name,
            'qty': e.qty,
            'idStore': e.idStore,
            'note': e.note,
            'idList': e.idList,
            'price_': e.price_,
            'stock': e.stock,
          },
        )
        .toList();
    await _db.enqueueInvoice(
      customerId: idCu,
      storeId: storeId,
      customerName: customer,
      credit: credit,
      lines: linesMaps,
    );
  }

  void _setBusy(bool v) {
    busy = v;
    notifyListeners();
  }

  /// Ensure valid store ID with fallback to first available store
  Future<int?> _ensureValidStoreId() async {
    debugPrint('🔍 _ensureValidStoreId: Starting...');

    int? storeId = await Prefs.getDefaultStoreIdWithFallback();
    debugPrint('🔍 _ensureValidStoreId: Got store ID from prefs: $storeId');

    // If no valid store ID, try first available store
    if (storeId == null || storeId <= 0) {
      debugPrint(
        '🔍 _ensureValidStoreId: No valid store ID, checking stores list (${stores.length} stores)',
      );

      if (stores.isEmpty) {
        debugPrint(
          '🔍 _ensureValidStoreId: Stores list empty, loading stores...',
        );
        await loadStores();
        debugPrint(
          '🔍 _ensureValidStoreId: After loadStores, have ${stores.length} stores',
        );
      }

      if (stores.isNotEmpty) {
        storeId = stores.first.id;
        debugPrint(
          '🔍 _ensureValidStoreId: Selected first store: $storeId (${stores.first.name})',
        );
        await Prefs.setDefaultStoreId(storeId);
        debugPrint(
          '✅ Auto-selected first available store as default: $storeId',
        );
      } else {
        debugPrint('❌ _ensureValidStoreId: No stores available!');
      }
    }

    debugPrint('🔍 _ensureValidStoreId: Returning store ID: $storeId');
    return (storeId != null && storeId > 0) ? storeId : null;
  }

  Future<bool> changeDefaultStore(int newStoreId) async {
    if (currentUser == null) return false;
    _setBusy(true);
    try {
      final dynamic rawId = currentUser!.idUser;
      final int idUser = rawId is int
          ? rawId
          : int.tryParse(rawId.toString()) ?? 0;
      final ok = await api.auth.updateUserStore(
        idUser: idUser,
        newIdTStore: newStoreId,
      );
      if (ok) {
        await Prefs.setDefaultStoreId(newStoreId);
        prices = await api.catalog.getPrice(newStoreId);
        bills = await api.orders.getTBills(idUser: idUser, date1: billsEndDate);
        notifyListeners();
      }
      return ok;
    } finally {
      _setBusy(false);
    }
  }

  Future<void> _ensureServerTimeZone() async {
    if (serverOffset != null && serverTimeZoneName != null) return;

    final cachedName = await PrefsTz.getServerTzName();
    final cachedMin = await PrefsTz.getServerTzOffsetMinutes();
    if (cachedName != null && cachedMin != null) {
      serverTimeZoneName = cachedName;
      serverOffset = Duration(minutes: cachedMin);
      return;
    }

    try {
      final name = await api.time.getTimeZoneName();
      serverTimeZoneName = name;
      serverOffset = _mapWindowsTzToOffset(name);
      await PrefsTz.setServerTzName(name ?? '');
      await PrefsTz.setServerTzOffsetMinutes(serverOffset!.inMinutes);
    } catch (_) {
      serverOffset = DateTime.now().timeZoneOffset;
    }
  }

  DateTime _toServerLocal(DateTime dt) {
    final off = serverOffset ?? DateTime.now().timeZoneOffset;
    final utc = dt.toUtc();
    final s = utc.add(off);
    return DateTime(s.year, s.month, s.day, s.hour, s.minute, s.second);
  }

  Future<void> refreshServerTimeZone() async {
    try {
      final name = await api.time.getTimeZoneName();
      serverTimeZoneName = name;
      serverOffset = _mapWindowsTzToOffset(name);
      await PrefsTz.setServerTzName(name ?? '');
      await PrefsTz.setServerTzOffsetMinutes(serverOffset!.inMinutes);
      notifyListeners();
    } catch (_) {}
  }

  Duration _mapWindowsTzToOffset(String? name) {
    const Map<String, Duration> map = {
      'Arab Standard Time': Duration(hours: 3),
      'Arabian Standard Time': Duration(hours: 4),
      'Egypt Standard Time': Duration(hours: 2),
      'Syria Standard Time': Duration(hours: 3),
      'Jordan Standard Time': Duration(hours: 3),
      'GTB Standard Time': Duration(hours: 2),
      'E. Europe Standard Time': Duration(hours: 2),
      'Turkey Standard Time': Duration(hours: 3),
      'Israel Standard Time': Duration(hours: 2),
    };
    return map[name ?? ''] ?? DateTime.now().timeZoneOffset;
  }

  Future<List<TableBillLineDto>> loadBillLinesByIdBC(int idBC) async {
    _setBusy(true);
    try {
      final items = await api.orders.getTBillByIdBC(idBC: idBC);
      return items;
    } finally {
      _setBusy(false);
    }
  }

  DateTime? parseServerDateToUi(String s) {
    try {
      DateTime dtUtc = DateTime.parse(s + "Z");

      return dtUtc.toLocal();
    } catch (_) {
      return null;
    }
  }
}
