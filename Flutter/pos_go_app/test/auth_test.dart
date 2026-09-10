import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/session_store.dart';

void main() {
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
      },
    });

    expect(session.role, 'saas_admin');
    expect(session.accountType, 'standard');
    expect(session.businessType, 'restaurant');
    expect(session.countryCode, 'EG');
    expect(session.currencyCode, 'EGP');
    expect(session.defaultLanguage, 'ar');
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
}
