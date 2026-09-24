import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/community.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/community/community_hub_screen.dart';
import 'package:pos_go_app/features/community/national_hub_screen.dart';
import 'package:pos_go_app/features/community/national_invitations_screen.dart';
import 'package:pos_go_app/features/community/national_members_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

const _manager = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-manager',
  displayName: 'Store Manager',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'manager',
  currencyCode: 'EGP',
);

const _owner = NationalMember(
  userId: 'user-manager',
  displayName: 'Store Manager',
  originTenantId: 'tenant-1',
  joinedVia: 'invite',
  invitedBy: 'user-saas',
  invitedByName: 'POS.Go Team',
  role: 'moderator',
  status: 'active',
  level: 'silver',
  expertiseScore: 300,
  joinedAt: '2026-09-24 10:00:00',
);

const _chef = NationalMember(
  userId: 'user-chef',
  displayName: 'Sara Chef',
  originTenantId: 'tenant-2',
  joinedVia: 'invite',
  invitedBy: 'user-manager',
  invitedByName: 'Store Manager',
  role: 'member',
  status: 'active',
  level: 'gold',
  expertiseScore: 640,
  joinedAt: '2026-09-24 11:00:00',
);

const _invitation = NationalInvitation(
  id: 'inv-1',
  code: 'ABCDEFGHJKMNPQ',
  inviterId: 'user-manager',
  email: 'peer@example.com',
  note: 'co-founder',
  maxUses: 3,
  usedCount: 1,
  status: 'active',
  expiresAt: '',
  createdAt: '2026-09-24 12:00:00',
);

class _FakeNationalApi extends ApiClient {
  _FakeNationalApi();

  NationalMember? me = _owner;
  int joinCalls = 0;
  int createInvitationCalls = 0;
  int revokeCalls = 0;
  String? lastRole;
  String? lastStatus;

  @override
  Future<NationalMember?> nationalMe(Session session) async => me;

  @override
  Future<NationalMember> nationalJoin(Session session, String code) async {
    joinCalls++;
    me = _chef;
    return _chef;
  }

  @override
  Future<List<NationalInvitation>> listNationalInvitations(
          Session session) async =>
      const [_invitation];

  @override
  Future<NationalInvitation> createNationalInvitation(
    Session session, {
    String email = '',
    String note = '',
    int maxUses = 1,
    String? expiresAt,
  }) async {
    createInvitationCalls++;
    return _invitation;
  }

  @override
  Future<void> revokeNationalInvitation(
      Session session, String invitationId) async {
    revokeCalls++;
  }

  @override
  Future<NationalMembersPage> listNationalMembers(
    Session session, {
    String query = '',
    String? level,
    String? role,
    int page = 1,
    int limit = 50,
  }) async {
    return NationalMembersPage(
      members: const [_owner, _chef],
      total: 2,
      page: page,
      limit: limit,
    );
  }

  @override
  Future<NationalMember> updateNationalMember(
    Session session,
    String userId, {
    String? status,
    String? role,
  }) async {
    lastStatus = status;
    lastRole = role;
    return _chef;
  }
}

Widget _wrap(Widget child) {
  return MaterialApp(
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
}

void main() {
  testWidgets('hub links to the national community entry', (tester) async {
    await tester.pumpWidget(
        _wrap(CommunityHubScreen(session: _manager, apiClient: _FakeNationalApi())));
    await tester.pumpAndSettle();

    expect(find.text('Egypt national community'), findsOneWidget);
  });

  testWidgets('member hub shows membership + national tiles', (tester) async {
    await tester.pumpWidget(
        _wrap(NationalHubScreen(session: _manager, apiClient: _FakeNationalApi())));
    await tester.pumpAndSettle();

    expect(find.text('Store Manager'), findsOneWidget);
    expect(find.text('Silver'), findsOneWidget);
    expect(find.text('Invite peers'), findsOneWidget);
    expect(find.text('Member directory'), findsOneWidget);
  });

  testWidgets('non-member join flow submits the code', (tester) async {
    final api = _FakeNationalApi()..me = null;
    await tester.pumpWidget(
        _wrap(NationalHubScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    expect(find.text('You are not a member yet'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, 'Join now'));
    await tester.pumpAndSettle();

    expect(find.text('Join the national community'), findsOneWidget);

    await tester.enterText(find.byType(TextField), 'EG-ABCDEFGHJKMNPQ');
    await tester.tap(find.descendant(
      of: find.byType(AlertDialog),
      matching: find.widgetWithText(FilledButton, 'Join now'),
    ));
    await tester.pumpAndSettle();

    expect(api.joinCalls, 1);
    expect(find.text('Joined'), findsOneWidget);
    expect(find.text('Sara Chef'), findsOneWidget);
  });

  testWidgets('invitations list creates a code and revokes', (tester) async {
    final api = _FakeNationalApi();
    await tester.pumpWidget(
        _wrap(NationalInvitationsScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    expect(find.text('peer@example.com'), findsOneWidget);

    await tester.tap(find.byTooltip('New invitation'));
    await tester.pumpAndSettle();

    await tester.enterText(
        find.widgetWithText(TextField, 'Recipient email (optional)'),
        'new-peer@example.com');
    await tester.tap(find.descendant(
      of: find.byType(AlertDialog),
      matching: find.widgetWithText(FilledButton, 'Create invitation'),
    ));
    await tester.pumpAndSettle();

    expect(api.createInvitationCalls, 1);
    expect(find.text('EG-ABCDEFGHJKMNPQ'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, 'OK'));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Revoke invitation'));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilledButton, 'Revoke invitation'));
    await tester.pumpAndSettle();

    expect(api.revokeCalls, 1);
    expect(find.text('Invitation revoked'), findsOneWidget);
  });

  testWidgets('member directory lists and moderates', (tester) async {
    final api = _FakeNationalApi();
    await tester.pumpWidget(
        _wrap(NationalMembersScreen(
      session: _manager,
      apiClient: api,
      canModerate: true,
      myUserId: _owner.userId,
    )));
    await tester.pumpAndSettle();

    expect(find.text('Store Manager'), findsOneWidget);
    expect(find.text('Sara Chef'), findsOneWidget);

    await tester.tap(find.text('Sara Chef'));
    await tester.pumpAndSettle();

    expect(find.text('Promote to moderator'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, 'Promote to moderator'));
    await tester.pumpAndSettle();

    expect(find.text('Confirm action'), findsOneWidget);
    await tester.tap(find.widgetWithText(FilledButton, 'OK'));
    await tester.pumpAndSettle();

    expect(api.lastRole, 'moderator');
    expect(find.text('Member updated'), findsOneWidget);
  });

  testWidgets('directory moderation menu promotes without opening detail',
      (tester) async {
    final api = _FakeNationalApi();
    await tester
        .pumpWidget(_wrap(NationalMembersScreen(
      session: _manager,
      apiClient: api,
      canModerate: true,
      myUserId: _owner.userId,
    )));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.more_vert), findsOneWidget);
    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();

    expect(find.text('Promote to moderator'), findsOneWidget);
    expect(find.text('Suspend member'), findsOneWidget);

    await tester.tap(find.widgetWithText(PopupMenuItem<String>, 'Promote to moderator'));
    await tester.pumpAndSettle();

    expect(find.text('Confirm action'), findsOneWidget);
    await tester.tap(find.widgetWithText(FilledButton, 'OK'));
    await tester.pumpAndSettle();

    expect(api.lastRole, 'moderator');
    expect(find.text('Member updated'), findsOneWidget);
  });

  testWidgets('cancelling the confirm dialog does not moderate',
      (tester) async {
    final api = _FakeNationalApi();
    await tester
        .pumpWidget(_wrap(NationalMembersScreen(
      session: _manager,
      apiClient: api,
      canModerate: true,
      myUserId: _owner.userId,
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(Icons.more_vert));
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(PopupMenuItem<String>, 'Suspend member'));
    await tester.pumpAndSettle();

    expect(find.text('Confirm action'), findsOneWidget);
    await tester.tap(find.widgetWithText(TextButton, 'Cancel'));
    await tester.pumpAndSettle();

    expect(api.lastStatus, isNull);
    expect(find.text('Member updated'), findsNothing);
  });
}