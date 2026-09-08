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
