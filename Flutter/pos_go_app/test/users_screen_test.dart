import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/security.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/core/users.dart';
import 'package:pos_go_app/features/settings/settings_screen.dart';
import 'package:pos_go_app/features/users/users_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

class _FakeApiClient extends ApiClient {
  _FakeApiClient({List<TenantUser>? users}) : _users = [...?users];

  List<TenantUser> _users;
  final List<Map<String, String>> created = [];
  Object? createUserError;

  @override
  Future<TenantUsersPage> listUsers(
    Session session, {
    int page = 1,
    int limit = 50,
  }) async =>
      TenantUsersPage(
        users: _users,
        total: _users.length,
        page: page,
        limit: limit,
      );

  @override
  Future<TenantUser> createUser(
    Session session, {
    required String email,
    required String displayName,
    required String password,
    required String role,
    String accountType = 'standard',
  }) async {
    final error = createUserError;
    if (error != null) throw error;
    created.add({'email': email, 'role': role, 'password': password});
    final user = TenantUser(
      id: 'user-${_users.length + 1}',
      email: email,
      displayName: displayName,
      role: role,
      accountType: accountType,
    );
    _users = [..._users, user];
    return user;
  }

  @override
  Future<TenantSettings> settings(Session session) async =>
      const TenantSettings();

  final List<Map<String, Object?>> securityUpdates = [];
  Object? updateSecurityError;

  @override
  Future<UserPermissions> updateUserSecurity(
    Session session, {
    required String userId,
    required UserSecurityProfile security,
  }) async {
    final error = updateSecurityError;
    if (error != null) throw error;
    securityUpdates.add({'userId': userId, 'profile': security});
    return UserPermissions.fromJson(security.toJson());
  }
}

const _managerSession = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-1',
  displayName: 'Store Manager',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'manager',
);

const _cashierSession = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-2',
  displayName: 'Cashier',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'cashier',
);

const _ownerSession = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-0',
  displayName: 'Owner',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'owner',
);

Widget _wrap(Widget child) => MaterialApp(
      locale: const Locale('en'),
      supportedLocales: AppStrings.supportedLocales,
      localizationsDelegates: const [
        AppStrings.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      home: child,
    );

const _seedUsers = [
  TenantUser(
    id: 'user-1',
    email: 'manager@demo.com',
    displayName: 'Store Manager',
    role: 'manager',
    isActive: true,
  ),
  TenantUser(
    id: 'user-2',
    email: 'old@demo.com',
    displayName: 'Former Cashier',
    role: 'cashier',
    isActive: false,
  ),
];

Future<void> _fillAddSheet(WidgetTester tester,
    {String password = 'secret123'}) async {
  await tester.tap(find.byTooltip('Add user'));
  await tester.pumpAndSettle();
  final fields = find.byType(TextField);
  await tester.enterText(fields.at(0), 'new@demo.com');
  await tester.enterText(fields.at(1), 'New Cashier');
  await tester.enterText(fields.at(2), password);
  await tester.pump();
}

void main() {
  testWidgets('lists the tenant users with role and inactive badge',
      (tester) async {
    final api = _FakeApiClient(users: _seedUsers);

    await tester.pumpWidget(
        _wrap(UsersScreen(session: _managerSession, apiClient: api)));
    await tester.pumpAndSettle();

    expect(find.text('Store Manager'), findsOneWidget);
    expect(find.text('manager@demo.com'), findsOneWidget);
    expect(find.text('Former Cashier'), findsOneWidget);
    expect(find.text('Inactive'), findsOneWidget,
        reason: 'deactivated accounts are marked on the row');
    expect(find.text('Manager'), findsOneWidget);
    expect(find.text('Cashier'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('creates a user through the add sheet', (tester) async {
    final api = _FakeApiClient(users: _seedUsers);

    await tester.pumpWidget(
        _wrap(UsersScreen(session: _managerSession, apiClient: api)));
    await tester.pumpAndSettle();

    await _fillAddSheet(tester);
    await tester.tap(find.widgetWithText(FilledButton, 'Save'));
    await tester.pumpAndSettle();

    expect(api.created, hasLength(1));
    expect(api.created.single['email'], 'new@demo.com');
    expect(api.created.single['role'], 'cashier',
        reason: 'a manager may only create cashiers (server hierarchy)');
    expect(find.text('User created'), findsOneWidget);
    expect(find.text('New Cashier'), findsOneWidget,
        reason: 'the list reloads with the new row');
  });

  testWidgets('a duplicate email surfaces the localized conflict',
      (tester) async {
    final api = _FakeApiClient(users: _seedUsers)
      ..createUserError =
          const ApiException('email is already in use', code: 'user_conflict');

    await tester.pumpWidget(
        _wrap(UsersScreen(session: _managerSession, apiClient: api)));
    await tester.pumpAndSettle();

    await _fillAddSheet(tester);
    await tester.tap(find.widgetWithText(FilledButton, 'Save'));
    await tester.pumpAndSettle();

    expect(api.created, isEmpty);
    expect(find.text('Email already in use'), findsOneWidget,
        reason: 'the error stays inline in the sheet');
    expect(find.byType(UsersScreen), findsOneWidget);
  });

  testWidgets('save stays disabled until the password is long enough',
      (tester) async {
    final api = _FakeApiClient(users: _seedUsers);

    await tester.pumpWidget(
        _wrap(UsersScreen(session: _managerSession, apiClient: api)));
    await tester.pumpAndSettle();

    await _fillAddSheet(tester, password: 'short');
    expect(
      tester.widget<FilledButton>(find.widgetWithText(FilledButton, 'Save'))
          .onPressed,
      isNull,
    );

    await tester.enterText(find.byType(TextField).at(2), 'secret123');
    await tester.pump();
    expect(
      tester.widget<FilledButton>(find.widgetWithText(FilledButton, 'Save'))
          .onPressed,
      isNotNull,
    );
  });

  testWidgets('settings offers the Users tile to managers', (tester) async {
    final api = _FakeApiClient(users: _seedUsers);

    await tester.pumpWidget(
        _wrap(SettingsScreen(session: _managerSession, apiClient: api)));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(find.text('Users'), 200,
        scrollable: find.byType(Scrollable).first);
    await tester.pumpAndSettle();
    expect(find.text('Users'), findsOneWidget);
    await tester.tap(find.text('Users'));
    await tester.pumpAndSettle();
    expect(find.byType(UsersScreen), findsOneWidget,
        reason: 'the tile pushes the staff directory');
  });

  testWidgets('settings hides the Users tile from cashiers', (tester) async {
    final api = _FakeApiClient(users: _seedUsers);

    await tester.pumpWidget(
        _wrap(SettingsScreen(session: _cashierSession, apiClient: api)));
    await tester.pumpAndSettle();
    await tester.scrollUntilVisible(find.text('Account'), 200,
        scrollable: find.byType(Scrollable).first);
    await tester.pumpAndSettle();
    expect(find.text('Users'), findsNothing,
        reason: 'cashiers must not see the staff directory entry');
    expect(find.byType(UsersScreen), findsNothing);
  });

  testWidgets('a manager gets the security editor only on cashier rows',
      (tester) async {
    final api = _FakeApiClient(users: _seedUsers);

    await tester.pumpWidget(
        _wrap(UsersScreen(session: _managerSession, apiClient: api)));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.security), findsOneWidget,
        reason: 'the seed has one manager row (no edit) + one cashier row');
    await tester.tap(find.byIcon(Icons.security));
    await tester.pumpAndSettle();
    expect(find.text('old@demo.com'), findsNWidgets(2),
        reason: 'the open sheet echoes the cashier row it was opened from');
    expect(
        find.byType(DropdownButtonFormField<PosAccessLevel>), findsOneWidget,
        reason: 'the security sheet is up (only it has a level dropdown)');
  });

  testWidgets('owners edit anyone; cashiers get no edit action',
      (tester) async {
    final api = _FakeApiClient(users: _seedUsers);

    await tester.pumpWidget(
        _wrap(UsersScreen(session: _ownerSession, apiClient: api)));
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.security), findsNWidgets(2));

    await tester.pumpWidget(
        _wrap(UsersScreen(session: _cashierSession, apiClient: api)));
    await tester.pumpAndSettle();
    expect(find.byIcon(Icons.security), findsNothing);
  });

  testWidgets('saving the sheet PUTs the edited profile', (tester) async {
    final api = _FakeApiClient(users: _seedUsers);

    await tester.pumpWidget(
        _wrap(UsersScreen(session: _managerSession, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(Icons.security));
    await tester.pumpAndSettle();
    expect(
        find.widgetWithText(
            DropdownButtonFormField<PosAccessLevel>, 'Cashier'),
        findsOneWidget,
        reason: 'a cashier-role target seeds the cashier level by default');

    await tester.tap(find.byType(DropdownButtonFormField<PosAccessLevel>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Advanced').last);
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), '10');
    await tester.pump();

    await tester.tap(find.widgetWithText(SwitchListTile, 'Custom permissions'));
    await tester.pumpAndSettle();

    final refunds = find.widgetWithText(SwitchListTile, 'Refunds');
    await tester.ensureVisible(refunds);
    await tester.pumpAndSettle();
    await tester.tap(refunds);
    await tester.pumpAndSettle();

    final save = find.widgetWithText(FilledButton, 'Save');
    await tester.ensureVisible(save);
    await tester.pumpAndSettle();
    await tester.tap(save);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    await tester.pumpAndSettle();

    expect(api.securityUpdates, hasLength(1));
    final update = api.securityUpdates.single;
    expect(update['userId'], 'user-2');
    final profile = update['profile'] as UserSecurityProfile;
    expect(profile.accessLevel, PosAccessLevel.advanced);
    expect(profile.maxDiscountPct, 10);
    expect(profile.useCustomPermissions, isTrue);
    expect(profile.canRefund, isTrue,
        reason: 'the toggled custom flag round-trips');
    expect(profile.canDeleteOrder, isFalse);
    expect(find.text('Security profile saved'), findsOneWidget);
  });

  testWidgets('an out-of-range cap blocks the save until it is fixed',
      (tester) async {
    final api = _FakeApiClient(users: _seedUsers);

    await tester.pumpWidget(
        _wrap(UsersScreen(session: _managerSession, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(Icons.security));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), '150');
    await tester.pump();
    final save = find.widgetWithText(FilledButton, 'Save');
    await tester.ensureVisible(save);
    await tester.pumpAndSettle();
    await tester.tap(save);
    await tester.pumpAndSettle();

    expect(api.securityUpdates, isEmpty);
    expect(find.text('Enter a number from 0 to 100'), findsOneWidget);

    final field = find.byType(TextField);
    await tester.ensureVisible(field);
    await tester.pumpAndSettle();
    await tester.enterText(field, '50');
    await tester.pump();
    await tester.ensureVisible(save);
    await tester.pumpAndSettle();
    await tester.tap(save);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 300));
    await tester.pumpAndSettle();

    expect(api.securityUpdates, hasLength(1));
    final profile = api.securityUpdates.single['profile'] as UserSecurityProfile;
    expect(profile.maxDiscountPct, 50);
  });
}
