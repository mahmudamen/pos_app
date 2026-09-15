import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'security.dart';

class Session {
  const Session({
    required this.accessToken,
    required this.refreshToken,
    required this.userId,
    required this.displayName,
    required this.tenantId,
    this.deviceId = '',
    this.role = '',
    this.accountType = '',
    this.businessType = '',
    this.countryCode = 'EG',
    this.currencyCode = 'EGP',
    this.defaultLanguage = 'ar',
    this.plan = '',
    this.trialEndsAt,
    this.permissions,
  });

  factory Session.fromJson(Map<String, dynamic> json) {
    final user = json['user'] as Map<String, dynamic>? ?? const {};
    final tenant = json['tenant'] as Map<String, dynamic>? ?? const {};
    final permissions = user['permissions'];
    return Session(
      accessToken: json['access_token'] as String,
      refreshToken: json['refresh_token'] as String,
      userId: user['id'] as String? ?? '',
      displayName: user['display_name'] as String? ?? 'Cashier',
      role: user['role'] as String? ?? '',
      accountType: user['account_type'] as String? ?? '',
      tenantId: tenant['id'] as String? ?? '',
      deviceId: json['device_id'] as String? ?? '',
      businessType: tenant['business_type'] as String? ?? '',
      countryCode: tenant['country_code'] as String? ?? 'EG',
      currencyCode: tenant['currency_code'] as String? ?? 'EGP',
      defaultLanguage: tenant['default_language'] as String? ?? 'ar',
      plan: tenant['plan'] as String? ?? '',
      trialEndsAt: tenant['trial_ends_at'] as String?,
      permissions: permissions is Map<String, dynamic>
          ? UserPermissions.fromJson(permissions)
          : null,
    );
  }

  final String accessToken;
  final String refreshToken;
  final String userId;
  final String displayName;
  final String tenantId;
  final String deviceId;
  final String role;
  final String accountType;
  final String businessType;
  final String countryCode;
  final String currencyCode;
  final String defaultLanguage;
  final String plan;
  final String? trialEndsAt;
  final UserPermissions? permissions;

  bool get isPlatformAdmin => role == 'saas_admin';

  bool get isOwner => role == 'owner';

  bool get isManager => role == 'owner' || role == 'manager';

  /// True when the user's resolved POS access level can act as a manager
  /// (e.g. overriding discounts or closing sessions without a manager PIN).
  bool get isManagerLevel => permissions?.isManagerLevel ?? isManager;

  bool get canManageSettings => isManager || isPlatformAdmin;

  bool get isTrial => plan == 'trial';

  Session copyWith({
    String? accessToken,
    String? refreshToken,
    String? displayName,
    String? role,
    String? accountType,
    String? businessType,
    String? countryCode,
    String? currencyCode,
    String? defaultLanguage,
    String? plan,
    String? trialEndsAt,
    UserPermissions? permissions,
  }) =>
      Session(
        accessToken: accessToken ?? this.accessToken,
        refreshToken: refreshToken ?? this.refreshToken,
        userId: userId,
        displayName: displayName ?? this.displayName,
        tenantId: tenantId,
        deviceId: deviceId,
        role: role ?? this.role,
        accountType: accountType ?? this.accountType,
        businessType: businessType ?? this.businessType,
        countryCode: countryCode ?? this.countryCode,
        currencyCode: currencyCode ?? this.currencyCode,
        defaultLanguage: defaultLanguage ?? this.defaultLanguage,
        plan: plan ?? this.plan,
        trialEndsAt: trialEndsAt ?? this.trialEndsAt,
        permissions: permissions ?? this.permissions,
      );
}

class RememberedLogin {
  const RememberedLogin({
    required this.tenantId,
    required this.email,
    required this.deviceName,
    required this.password,
  });

  final String tenantId;
  final String email;
  final String deviceName;
  final String password;
}

class SessionStore {
  static const _accessTokenKey = 'access_token';
  static const _refreshTokenKey = 'refresh_token';
  static const _userIdKey = 'user_id';
  static const _displayNameKey = 'display_name';
  static const _tenantIdKey = 'tenant_id';
  static const _deviceIdKey = 'device_id';
  static const _roleKey = 'role';
  static const _accountTypeKey = 'account_type';
  static const _businessTypeKey = 'business_type';
  static const _countryCodeKey = 'country_code';
  static const _currencyCodeKey = 'currency_code';
  static const _defaultLanguageKey = 'default_language';
  static const _planKey = 'tenant_plan';
  static const _trialEndsAtKey = 'tenant_trial_ends_at';
  static const _permissionsKey = 'user_permissions';

  static const _languageKey = 'app_language';
  static const _languageCustomizedKey = 'app_language_customized';

  static const _rememberLoginKey = 'remember_login';
  static const _rememberTenantKey = 'remember_tenant';
  static const _rememberEmailKey = 'remember_email';
  static const _rememberDeviceNameKey = 'remember_device_name';
  static const _rememberPasswordKey = 'remember_password';

  static const _focusModeKey = 'focus_mode';

  final FlutterSecureStorage _storage = const FlutterSecureStorage();

  Future<void> saveRememberedLogin(RememberedLogin login) async {
    await Future.wait([
      _storage.write(key: _rememberLoginKey, value: '1'),
      _storage.write(key: _rememberTenantKey, value: login.tenantId),
      _storage.write(key: _rememberEmailKey, value: login.email),
      _storage.write(key: _rememberDeviceNameKey, value: login.deviceName),
      _storage.write(key: _rememberPasswordKey, value: login.password),
    ]);
  }

  Future<RememberedLogin?> readRememberedLogin() async {
    final enabled = await _storage.read(key: _rememberLoginKey);
    if (enabled != '1') return null;
    final tenant = await _storage.read(key: _rememberTenantKey);
    final email = await _storage.read(key: _rememberEmailKey);
    final deviceName = await _storage.read(key: _rememberDeviceNameKey);
    final password = await _storage.read(key: _rememberPasswordKey);
    if (tenant == null || email == null) return null;
    return RememberedLogin(
      tenantId: tenant,
      email: email,
      deviceName: deviceName ?? 'Counter 1',
      password: password ?? '',
    );
  }

  Future<void> clearRememberedLogin() async {
    await Future.wait([
      _storage.delete(key: _rememberLoginKey),
      _storage.delete(key: _rememberTenantKey),
      _storage.delete(key: _rememberEmailKey),
      _storage.delete(key: _rememberDeviceNameKey),
      _storage.delete(key: _rememberPasswordKey),
    ]);
  }

  Future<void> save(Session session) async {
    await Future.wait([
      _storage.write(key: _accessTokenKey, value: session.accessToken),
      _storage.write(key: _refreshTokenKey, value: session.refreshToken),
      _storage.write(key: _userIdKey, value: session.userId),
      _storage.write(key: _displayNameKey, value: session.displayName),
      _storage.write(key: _tenantIdKey, value: session.tenantId),
      _storage.write(key: _roleKey, value: session.role),
      _storage.write(key: _accountTypeKey, value: session.accountType),
      _storage.write(key: _businessTypeKey, value: session.businessType),
      _storage.write(key: _countryCodeKey, value: session.countryCode),
      _storage.write(key: _currencyCodeKey, value: session.currencyCode),
      _storage.write(key: _defaultLanguageKey, value: session.defaultLanguage),
      _storage.write(key: _planKey, value: session.plan),
      _storage.write(key: _trialEndsAtKey, value: session.trialEndsAt ?? ''),
      if (session.permissions != null)
        _storage.write(
            key: _permissionsKey,
            value: jsonEncode(session.permissions!.toJson())),
      if (session.deviceId.isNotEmpty)
        _storage.write(key: _deviceIdKey, value: session.deviceId),
    ]);
  }

  Future<void> clear() async {
    await Future.wait([
      _storage.delete(key: _accessTokenKey),
      _storage.delete(key: _refreshTokenKey),
      _storage.delete(key: _userIdKey),
      _storage.delete(key: _displayNameKey),
      _storage.delete(key: _tenantIdKey),
      _storage.delete(key: _roleKey),
      _storage.delete(key: _accountTypeKey),
      _storage.delete(key: _businessTypeKey),
      _storage.delete(key: _countryCodeKey),
      _storage.delete(key: _currencyCodeKey),
      _storage.delete(key: _defaultLanguageKey),
      _storage.delete(key: _planKey),
      _storage.delete(key: _trialEndsAtKey),
      _storage.delete(key: _permissionsKey),
      // Device id and app language preferences intentionally survive sign-out.
    ]);
  }

  Future<String?> readDeviceId() async {
    return _storage.read(key: _deviceIdKey);
  }

  Future<void> saveDeviceId(String deviceId) async {
    await _storage.write(key: _deviceIdKey, value: deviceId);
  }

  Future<String> readLanguage() async {
    return await _storage.read(key: _languageKey) ?? 'ar';
  }

  Future<void> saveLanguage(String code) async {
    await Future.wait([
      _storage.write(key: _languageKey, value: code),
      _storage.write(key: _languageCustomizedKey, value: '1'),
    ]);
  }

  Future<void> applyTenantLanguage(String code) async {
    await _storage.write(key: _languageKey, value: code);
  }

  Future<bool> hasCustomizedLanguage() async {
    return await _storage.read(key: _languageCustomizedKey) == '1';
  }

  Future<Session?> read() async {
    final access = await _storage.read(key: _accessTokenKey);
    final refresh = await _storage.read(key: _refreshTokenKey);
    if (access == null || refresh == null) return null;
    final permissionsRaw = await _storage.read(key: _permissionsKey);
    UserPermissions? permissions;
    if (permissionsRaw != null) {
      try {
        permissions = UserPermissions.fromJson(
            jsonDecode(permissionsRaw) as Map<String, dynamic>);
      } catch (_) {
        permissions = null;
      }
    }
    return Session(
      accessToken: access,
      refreshToken: refresh,
      userId: await _storage.read(key: _userIdKey) ?? '',
      displayName: await _storage.read(key: _displayNameKey) ?? 'Cashier',
      tenantId: await _storage.read(key: _tenantIdKey) ?? '',
      deviceId: await _storage.read(key: _deviceIdKey) ?? '',
      role: await _storage.read(key: _roleKey) ?? '',
      accountType: await _storage.read(key: _accountTypeKey) ?? '',
      businessType: await _storage.read(key: _businessTypeKey) ?? '',
      countryCode: await _storage.read(key: _countryCodeKey) ?? 'EG',
      currencyCode: await _storage.read(key: _currencyCodeKey) ?? 'EGP',
      defaultLanguage: await _storage.read(key: _defaultLanguageKey) ?? 'ar',
      plan: await _storage.read(key: _planKey) ?? '',
      trialEndsAt: await _storage.read(key: _trialEndsAtKey).then((v) =>
          (v == null || v.isEmpty) ? null : v),
      permissions: permissions,
    );
  }

  String _syncKey(String tenantId) => 'sync_cursor_$tenantId';

  Future<int> readSyncCursor(String tenantId) async {
    final raw = await _storage.read(key: _syncKey(tenantId));
    return int.tryParse(raw ?? '') ?? 0;
  }

  Future<void> saveSyncCursor(String tenantId, int cursor) async {
    await _storage.write(key: _syncKey(tenantId), value: '$cursor');
  }

  Future<bool> readFocusMode() async {
    return await _storage.read(key: _focusModeKey) == '1';
  }

  Future<void> saveFocusMode(bool enabled) async {
    await _storage.write(
        key: _focusModeKey, value: enabled ? '1' : '0');
  }
}