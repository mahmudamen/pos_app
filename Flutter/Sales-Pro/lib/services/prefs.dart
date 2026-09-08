import 'package:shared_preferences/shared_preferences.dart';
import '../features/auth/dto/user_dto.dart';

class Prefs {
  static const _kDefaultStoreId = 'default_store_id';
  static const _kLoginName = 'login_name_saved';
  static const _kUserId = 'user_id';
  static const _kSaleMan = 'sale_man';
  static const _kIdTStore = 'id_t_store';
  static const _kNameStore = 'name_store';
  static const _kName = 'name';
  static const _kLanguageCode = 'language_code';

  // ==== Language ====
  static Future<String> getLanguageCode() async {
    final sp = await SharedPreferences.getInstance();
    return sp.getString(_kLanguageCode) ?? 'en';
  }

  static Future<void> setLanguageCode(String code) async {
    final sp = await SharedPreferences.getInstance();
    await sp.setString(_kLanguageCode, code);
  }

  // ==== Default Store ====
  static Future<int?> getDefaultStoreId() async {
    final sp = await SharedPreferences.getInstance();
    return sp.getInt(_kDefaultStoreId);
  }

  /// Ensure valid store ID with fallback
  static Future<int?> getDefaultStoreIdWithFallback() async {
    final sp = await SharedPreferences.getInstance();
    int? storeId = sp.getInt(_kDefaultStoreId);

    // If no store ID is saved, try getting it from user data
    if (storeId == null || storeId <= 0) {
      final userStoreId = sp.getString(_kIdTStore);
      if (userStoreId != null && userStoreId.isNotEmpty) {
        storeId = int.tryParse(userStoreId);
        // Save value for next time if correct
        if (storeId != null && storeId > 0) {
          await setDefaultStoreId(storeId);
        }
      }
    }

    return (storeId != null && storeId > 0) ? storeId : null;
  }

  static Future<void> setDefaultStoreId(int id) async {
    final sp = await SharedPreferences.getInstance();
    await sp.setInt(_kDefaultStoreId, id);
  }

  // ==== Retrieve User ====
  static Future<ClsUserDto?> getUser() async {
    final sp = await SharedPreferences.getInstance();
    final userId = sp.getString(_kUserId);
    final saleMan = sp.getInt(_kSaleMan);
    final idTStore = sp.getString(_kIdTStore);
    final nameStore = sp.getString(_kNameStore);
    final name = sp.getString(_kName);

    if (userId == null ||
        saleMan == null ||
        idTStore == null ||
        nameStore == null) {
      return null;
    }

    return ClsUserDto(
      idUser: userId,
      saleMan: saleMan,
      idTStore: idTStore,
      nameStore: nameStore,
      name: name ?? '',
    );
  }

  // ==== Save Login Name ====
  static Future<void> saveLoginName(String name) async {
    final sp = await SharedPreferences.getInstance();
    await sp.setString(_kLoginName, name);
  }

  static Future<void> saveSaleMan(int saleMan) async {
    final sp = await SharedPreferences.getInstance();
    await sp.setInt(_kSaleMan, saleMan);
  }

  static Future<String?> getLoginName() async {
    final sp = await SharedPreferences.getInstance();
    return sp.getString(_kLoginName);
  }

  static Future<void> clearLoginName() async {
    final sp = await SharedPreferences.getInstance();
    await sp.remove(_kLoginName);
  }

  // ==== Save User ====
  static Future<void> saveUser(ClsUserDto user) async {
    final sp = await SharedPreferences.getInstance();
    await Future.wait([
      sp.setString(_kUserId, user.idUser),
      sp.setInt(_kSaleMan, user.saleMan),
      sp.setString(_kIdTStore, user.idTStore),
      sp.setString(_kNameStore, user.nameStore),
      sp.setString(_kName, user.name),
    ]);
  }

  // ==== Clear User Data ====
  static Future<void> clearUser() async {
    final sp = await SharedPreferences.getInstance();
    await Future.wait([
      sp.remove(_kLoginName),
      sp.remove(_kUserId),
      sp.remove(_kSaleMan),
      sp.remove(_kIdTStore),
      sp.remove(_kNameStore),
      sp.remove(_kName),
      sp.remove(_kDefaultStoreId),
    ]);
  }
}
