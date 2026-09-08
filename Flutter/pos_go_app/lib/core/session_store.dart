import 'package:flutter_secure_storage/flutter_secure_storage.dart';

class Session {
  const Session({
    required this.accessToken,
    required this.refreshToken,
    required this.userId,
    required this.displayName,
    required this.tenantId,
  });

  factory Session.fromJson(Map<String, dynamic> json) {
    final user = json['user'] as Map<String, dynamic>? ?? const {};
    final tenant = json['tenant'] as Map<String, dynamic>? ?? const {};
    return Session(
      accessToken: json['access_token'] as String,
      refreshToken: json['refresh_token'] as String,
      userId: user['id'] as String? ?? '',
      displayName: user['display_name'] as String? ?? 'Cashier',
      tenantId: tenant['id'] as String? ?? '',
    );
  }

  final String accessToken;
  final String refreshToken;
  final String userId;
  final String displayName;
  final String tenantId;

  Session copyWith({String? accessToken, String? refreshToken}) => Session(
        accessToken: accessToken ?? this.accessToken,
        refreshToken: refreshToken ?? this.refreshToken,
        userId: userId,
        displayName: displayName,
        tenantId: tenantId,
      );
}

class SessionStore {
  static const _accessTokenKey = 'access_token';
  static const _refreshTokenKey = 'refresh_token';
  static const _userIdKey = 'user_id';
  static const _displayNameKey = 'display_name';
  static const _tenantIdKey = 'tenant_id';

  final FlutterSecureStorage _storage = const FlutterSecureStorage();

  Future<void> save(Session session) async {
    await Future.wait([
      _storage.write(key: _accessTokenKey, value: session.accessToken),
      _storage.write(key: _refreshTokenKey, value: session.refreshToken),
      _storage.write(key: _userIdKey, value: session.userId),
      _storage.write(key: _displayNameKey, value: session.displayName),
      _storage.write(key: _tenantIdKey, value: session.tenantId),
    ]);
  }

  Future<void> clear() async {
    await _storage.deleteAll();
  }

  Future<Session?> read() async {
    final access = await _storage.read(key: _accessTokenKey);
    final refresh = await _storage.read(key: _refreshTokenKey);
    if (access == null || refresh == null) return null;
    return Session(
      accessToken: access,
      refreshToken: refresh,
      userId: await _storage.read(key: _userIdKey) ?? '',
      displayName: await _storage.read(key: _displayNameKey) ?? 'Cashier',
      tenantId: await _storage.read(key: _tenantIdKey) ?? '',
    );
  }
}
