import 'package:shared_preferences/shared_preferences.dart';

class PrefsTz {
  static const _kServerTzName = 'server_tz_name';
  static const _kServerTzOffsetMin = 'server_tz_offset_min';

  static Future<String?> getServerTzName() async {
    final p = await SharedPreferences.getInstance();
    return p.getString(_kServerTzName);
  }


  static Future<void> setServerTzName(String name) async {
    final p = await SharedPreferences.getInstance();
    await p.setString(_kServerTzName, name);
  }

  static Future<int?> getServerTzOffsetMinutes() async {
    final p = await SharedPreferences.getInstance();
    return p.getInt(_kServerTzOffsetMin);
  }

  static Future<void> setServerTzOffsetMinutes(int minutes) async {
    final p = await SharedPreferences.getInstance();
    await p.setInt(_kServerTzOffsetMin, minutes);
  }

  static Future<void> clearServerTimeZone() async {
    final p = await SharedPreferences.getInstance();
    await p.remove(_kServerTzName);
    await p.remove(_kServerTzOffsetMin);
  }
}
