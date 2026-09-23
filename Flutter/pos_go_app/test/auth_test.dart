import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/fonts.dart';
import 'package:pos_go_app/core/security.dart';
import 'package:pos_go_app/core/session_store.dart';

void main() {
  test('sync cursor persists per tenant and defaults to zero', () async {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
    final store = SessionStore();

    expect(await store.readSyncCursor('tenant-1'), 0);

    await store.saveSyncCursor('tenant-1', 130);
    await store.saveSyncCursor('tenant-2', 42);

    expect(await store.readSyncCursor('tenant-1'), 130);
    expect(await store.readSyncCursor('tenant-2'), 42);
  });

  test('session restores identity fields from the Go login envelope', () {
    final session = Session.fromJson({
      'access_token': 'access',
      'refresh_token': 'refresh',
      'user': {
        'id': 'user-1',
        'display_name': 'Manager',
        'role': 'manager',
      },
      'tenant': {'id': 'tenant-1'},
    });

    expect(session.accessToken, 'access');
    expect(session.refreshToken, 'refresh');
    expect(session.userId, 'user-1');
    expect(session.displayName, 'Manager');
    expect(session.tenantId, 'tenant-1');
  });

  test('session parses the SaaS and i18n fields from the login envelope',
      () {
    final session = Session.fromJson({
      'access_token': 'access',
      'refresh_token': 'refresh',
      'user': {
        'id': 'user-1',
        'display_name': 'Admin',
        'role': 'saas_admin',
        'account_type': 'standard',
      },
      'tenant': {
        'id': 'tenant-1',
        'business_type': 'restaurant',
        'country_code': 'EG',
        'currency_code': 'EGP',
        'default_language': 'ar',
        'plan': 'trial',
        'trial_ends_at': '2026-09-29T00:00:00Z',
      },
    });

    expect(session.role, 'saas_admin');
    expect(session.accountType, 'standard');
    expect(session.businessType, 'restaurant');
    expect(session.countryCode, 'EG');
    expect(session.currencyCode, 'EGP');
    expect(session.defaultLanguage, 'ar');
    expect(session.plan, 'trial');
    expect(session.trialEndsAt, '2026-09-29T00:00:00Z');
    expect(session.isTrial, isTrue);
    expect(session.isPlatformAdmin, isTrue);
  });

  test('session defaults are Arabic-first and Egypt-only', () {
    const session = Session(
      accessToken: 't',
      refreshToken: 'r',
      userId: 'u',
      displayName: 'Cashier',
      tenantId: 't1',
    );
    expect(session.countryCode, 'EG');
    expect(session.currencyCode, 'EGP');
    expect(session.defaultLanguage, 'ar');
    expect(session.role, '');
    expect(session.isPlatformAdmin, isFalse);
  });

  test('session parses the resolved permissions payload from login', () {
    final session = Session.fromJson({
      'access_token': 'access',
      'refresh_token': 'refresh',
      'user': {
        'id': 'user-1',
        'display_name': 'Manager',
        'role': 'manager',
        'permissions': {
          'access_level': 'manager',
          'max_discount_pct': 50,
          'can_delete_order': true,
          'can_delete_line': true,
          'can_change_qty': true,
          'can_negative_qty': false,
          'can_price_change': true,
          'can_discount': true,
          'can_open_session': true,
          'can_close_session': true,
          'can_payment_modification': true,
          'can_refund': true,
          'can_negative_stock': false,
        },
      },
      'tenant': {'id': 'tenant-1'},
    });

    expect(session.permissions, isNotNull);
    expect(session.permissions!.accessLevel, PosAccessLevel.manager);
    expect(session.permissions!.maxDiscountPct, 50);
    expect(session.permissions!.canRefund, isTrue);
    expect(session.permissions!.canNegativeStock, isFalse);
    expect(session.isManagerLevel, isTrue);
    expect(session.permissions!.effectiveDiscountPct(100), 50);
  });

  test('session falls back to the role when no permissions are present',
      () {
    final cashier = Session.fromJson({
      'access_token': 'access',
      'refresh_token': 'refresh',
      'user': {'id': 'user-1', 'display_name': 'Cashier', 'role': 'cashier'},
      'tenant': {'id': 'tenant-1'},
    });

    expect(cashier.permissions, isNull);
    expect(cashier.isManagerLevel, isFalse);

    final manager = Session.fromJson({
      'access_token': 'access',
      'refresh_token': 'refresh',
      'user': {'id': 'user-2', 'display_name': 'Manager', 'role': 'manager'},
      'tenant': {'id': 'tenant-1'},
    });
    expect(manager.isManagerLevel, isTrue);
  });

  test('session copyWith keeps permissions across a token refresh', () {
    final session = Session.fromJson({
      'access_token': 'old',
      'refresh_token': 'old-refresh',
      'user': {
        'id': 'user-1',
        'display_name': 'Cashier',
        'role': 'cashier',
        'permissions': {'access_level': 'advanced', 'max_discount_pct': 10},
      },
      'tenant': {'id': 'tenant-1'},
    });

    final refreshed = session.copyWith(
      accessToken: 'new',
      refreshToken: 'new-refresh',
    );

    expect(refreshed.permissions?.accessLevel, PosAccessLevel.advanced);
    expect(refreshed.accessToken, 'new');
    expect(refreshed.permissions!.effectiveDiscountPct(20), 10);
  });

  test('session permissions round-trip through secure storage', () async {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
    final store = SessionStore();

    final session = Session.fromJson({
      'access_token': 'access',
      'refresh_token': 'refresh',
      'user': {
        'id': 'user-1',
        'display_name': 'Cashier',
        'role': 'cashier',
        'permissions': {'access_level': 'advanced', 'max_discount_pct': 10},
      },
      'tenant': {'id': 'tenant-1'},
    });
    await store.save(session);

    final restored = await store.read();
    expect(restored, isNotNull);
    expect(restored!.permissions, isNotNull);
    expect(restored.permissions!.accessLevel, PosAccessLevel.advanced);
    expect(restored.permissions!.maxDiscountPct, 10);

    await store.clear();
    expect(await store.read(), isNull);
  });

  test('session exposes the owner/manager/cashier role hierarchy', () {
    const owner = Session(
      accessToken: 't',
      refreshToken: 'r',
      userId: 'u',
      displayName: 'Owner',
      tenantId: 't1',
      role: 'owner',
    );
    const manager = Session(
      accessToken: 't',
      refreshToken: 'r',
      userId: 'u',
      displayName: 'Manager',
      tenantId: 't1',
      role: 'manager',
    );
    const cashier = Session(
      accessToken: 't',
      refreshToken: 'r',
      userId: 'u',
      displayName: 'Cashier',
      tenantId: 't1',
      role: 'cashier',
    );

    expect(owner.isOwner, isTrue);
    expect(owner.isManager, isTrue);
    expect(owner.canManageSettings, isTrue);
    expect(manager.isOwner, isFalse);
    expect(manager.isManager, isTrue);
    expect(manager.canManageSettings, isTrue);
    expect(cashier.isOwner, isFalse);
    expect(cashier.isManager, isFalse);
    expect(cashier.canManageSettings, isFalse);
    expect(cashier.isPlatformAdmin, isFalse);
  });

  test('remembered login holds the prefilled form fields', () {
    const remembered = RememberedLogin(
      tenantId: 'demo-restaurant',
      email: 'admin@demo-restaurant.com',
      password: 'admin',
      deviceName: 'Counter 1',
    );

    expect(remembered.tenantId, 'demo-restaurant');
    expect(remembered.email, 'admin@demo-restaurant.com');
    expect(remembered.password, 'admin');
    expect(remembered.deviceName, 'Counter 1');
  });

  test('session copyWith only rotates token fields', () {
    const session = Session(
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final refreshed = session.copyWith(
      accessToken: 'new-access',
      refreshToken: 'new-refresh',
    );

    expect(refreshed.accessToken, 'new-access');
    expect(refreshed.refreshToken, 'new-refresh');
    expect(refreshed.userId, session.userId);
    expect(refreshed.tenantId, session.tenantId);
  });

  test('session round-trips through secure storage', () async {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
    final store = SessionStore();

    expect(await store.read(), isNull);

    const session = Session(
      accessToken: 'access',
      refreshToken: 'refresh',
      userId: 'user-1',
      displayName: 'Manager',
      tenantId: 'tenant-1',
      deviceId: 'dev-abc',
      role: 'manager',
      accountType: 'standard',
      businessType: 'grocery',
      countryCode: 'EG',
      currencyCode: 'EGP',
      defaultLanguage: 'ar',
      plan: 'trial',
      trialEndsAt: '2026-09-29T00:00:00Z',
    );
    await store.save(session);

    final restored = await store.read();
    expect(restored, isNotNull);
    expect(restored!.accessToken, 'access');
    expect(restored.refreshToken, 'refresh');
    expect(restored.userId, 'user-1');
    expect(restored.displayName, 'Manager');
    expect(restored.tenantId, 'tenant-1');
    expect(restored.deviceId, 'dev-abc');
    expect(restored.role, 'manager');
    expect(restored.accountType, 'standard');
    expect(restored.businessType, 'grocery');
    expect(restored.defaultLanguage, 'ar');
    expect(restored.plan, 'trial');
    expect(restored.trialEndsAt, '2026-09-29T00:00:00Z');
    expect(restored.isTrial, isTrue);
  });

  test('session clear drops tokens but keeps device, language and fonts',
      () async {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
    final store = SessionStore();
    await store.save(const Session(
      accessToken: 'access',
      refreshToken: 'refresh',
      userId: 'u',
      displayName: 'Cashier',
      tenantId: 't',
      deviceId: 'dev-keep',
    ));
    await store.saveDeviceId('dev-keep');
    await store.applyTenantLanguage('en');
    await store.saveFontFamily(FontMode.kufi.wire);
    await store.saveFontSize(FontSize.large.wire);
    await store.saveSoundEnabled(false);

    await store.clear();

    expect(await store.read(), isNull);
    expect(await store.readDeviceId(), 'dev-keep');
    expect(await store.readLanguage(), 'en');
    expect(await store.readFontFamily(), FontMode.kufi.wire);
    expect(await store.readFontSize(), FontSize.large.wire);
    expect(await store.readSoundEnabled(), isFalse);
  });

  test('font family/size round-trip through secure storage with defaults',
      () async {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
    final store = SessionStore();

    // Defaults: prompt sans at medium, sound on.
    expect(await store.readFontFamily(), FontMode.sans.wire);
    expect(await store.readFontSize(), FontSize.medium.wire);
    expect(await store.readSoundEnabled(), isTrue);

    await store.saveFontFamily(FontMode.naskh.wire);
    await store.saveFontSize(FontSize.small.wire);
    expect(await store.readFontFamily(), FontMode.naskh.wire);
    expect(await store.readFontSize(), FontSize.small.wire);

    await store.saveFontFamily(FontMode.sans.wire);
    await store.saveFontSize(FontSize.medium.wire);
    expect(await store.readFontFamily(), FontMode.sans.wire);
    expect(await store.readFontSize(), FontSize.medium.wire);
  });

  test('sound enabled flag round-trips through secure storage', () async {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
    final store = SessionStore();

    await store.saveSoundEnabled(false);
    expect(await store.readSoundEnabled(), isFalse);
    await store.saveSoundEnabled(true);
    expect(await store.readSoundEnabled(), isTrue);
  });

  test('remembered login round-trips through secure storage', () async {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
    final store = SessionStore();

    expect(await store.readRememberedLogin(), isNull);

    await store.saveRememberedLogin(const RememberedLogin(
      tenantId: 'demo-restaurant',
      email: 'admin@demo-restaurant.com',
      password: 'admin',
      deviceName: 'Counter 1',
    ));

    final remembered = await store.readRememberedLogin();
    expect(remembered, isNotNull);
    expect(remembered!.tenantId, 'demo-restaurant');
    expect(remembered.email, 'admin@demo-restaurant.com');
    expect(remembered.password, 'admin');
    expect(remembered.deviceName, 'Counter 1');

    await store.clearRememberedLogin();
    expect(await store.readRememberedLogin(), isNull);
  });

  test('remembered login falls back to defaults for partial writes', () async {
    FlutterSecureStorage.setMockInitialValues(<String, String>{
      'remember_login': '1',
      'remember_tenant': 'demo-grocery',
      'remember_email': 'cashier@demo-grocery.com',
    });
    final store = SessionStore();

    final remembered = await store.readRememberedLogin();
    expect(remembered, isNotNull);
    expect(remembered!.tenantId, 'demo-grocery');
    expect(remembered.email, 'cashier@demo-grocery.com');
    expect(remembered.password, '');
    expect(remembered.deviceName, 'Counter 1');
  });

  test('language defaults to Arabic until customized', () async {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
    final store = SessionStore();

    expect(await store.readLanguage(), 'ar');
    expect(await store.hasCustomizedLanguage(), isFalse);

    await store.applyTenantLanguage('en');
    expect(await store.readLanguage(), 'en');
    expect(await store.hasCustomizedLanguage(), isFalse);

    await store.saveLanguage('ar');
    expect(await store.readLanguage(), 'ar');
    expect(await store.hasCustomizedLanguage(), isTrue);
  });
}
