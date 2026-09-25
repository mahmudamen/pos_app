import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/community.dart';
import 'package:pos_go_app/core/session_store.dart';
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

NationalMember _member({
  required String userId,
  required String displayName,
  String role = 'member',
  String status = 'active',
  String level = 'gold',
  int expertiseScore = 100,
}) =>
    NationalMember(
      userId: userId,
      displayName: displayName,
      originTenantId: 'tenant-2',
      joinedVia: 'invite',
      invitedBy: 'user-manager',
      invitedByName: 'Store Manager',
      role: role,
      status: status,
      level: level,
      expertiseScore: expertiseScore,
      joinedAt: '2026-09-24 11:00:00',
    );

class _MemberApi extends ApiClient {
  _MemberApi(this.members);

  final List<NationalMember> members;

  @override
  Future<NationalMembersPage> listNationalMembers(
    Session session, {
    String query = '',
    String? level,
    String? role,
    int page = 1,
    int limit = 50,
  }) async {
    final filtering = query.isNotEmpty || (level != null && level.isNotEmpty);
    return NationalMembersPage(
      members: filtering ? const [] : members,
      total: filtering ? 0 : members.length,
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
    return members.first;
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

Widget _screen(List<NationalMember> members, {bool canModerate = false}) {
  return NationalMembersScreen(
    session: _manager,
    apiClient: _MemberApi(members),
    canModerate: canModerate,
    myUserId: 'user-manager',
  );
}

void main() {
  group('member directory redesign', () {
    testWidgets('count header shows the member total', (tester) async {
      await tester.pumpWidget(_wrap(_screen([
        _member(userId: 'u1', displayName: 'Adam'),
        _member(userId: 'u2', displayName: 'Bella'),
      ])));
      await tester.pumpAndSettle();

      expect(find.text('Members · 2'), findsOneWidget);
    });

    testWidgets('tier level pills render the tier icon and label',
        (tester) async {
      await tester.pumpWidget(_wrap(_screen([
        _member(userId: 'u1', displayName: 'Adam', level: 'bronze'),
        _member(userId: 'u2', displayName: 'Bella', level: 'silver'),
        _member(userId: 'u3', displayName: 'Carla', level: 'gold'),
        _member(userId: 'u4', displayName: 'Dalia', level: 'platinum'),
      ])));
      await tester.pumpAndSettle();

      expect(find.text('Bronze'), findsOneWidget);
      expect(find.text('Silver'), findsOneWidget);
      expect(find.text('Gold'), findsOneWidget);
      expect(find.text('Platinum'), findsOneWidget);
      expect(find.byIcon(Icons.military_tech_outlined), findsOneWidget);
      expect(find.byIcon(Icons.emoji_events_outlined), findsOneWidget);
      expect(find.byIcon(Icons.workspace_premium_outlined), findsOneWidget);
      expect(find.byIcon(Icons.diamond), findsOneWidget);
    });

    testWidgets('status pill appears only for non-active members',
        (tester) async {
      await tester.pumpWidget(_wrap(_screen([
        _member(userId: 'u1', displayName: 'Adam', status: 'active'),
        _member(userId: 'u2', displayName: 'Bella', status: 'suspended'),
        _member(userId: 'u3', displayName: 'Carla', status: 'left'),
      ])));
      await tester.pumpAndSettle();

      expect(find.text('Suspended'), findsOneWidget);
      expect(find.text('Left'), findsOneWidget);
      expect(find.text('Active'), findsNothing);
      expect(find.byIcon(Icons.block_outlined), findsOneWidget);
      expect(find.byIcon(Icons.person_off_outlined), findsOneWidget);
    });

    testWidgets('role pill appears inline for moderator and admin rows only',
        (tester) async {
      await tester.pumpWidget(_wrap(_screen([
        _member(
            userId: 'u1', displayName: 'Adam', role: 'admin', level: 'silver'),
        _member(
            userId: 'u2', displayName: 'Bella', role: 'moderator',
            level: 'silver'),
        _member(
            userId: 'u3', displayName: 'Carla', role: 'member', level: 'silver'),
      ])));
      await tester.pumpAndSettle();

      expect(find.text('Admin'), findsOneWidget);
      expect(find.text('Moderator'), findsOneWidget);
      expect(find.text('Member'), findsNothing);
      expect(find.byIcon(Icons.admin_panel_settings_outlined), findsOneWidget);
      expect(find.byIcon(Icons.shield_outlined), findsOneWidget);
      expect(find.byIcon(Icons.person_outline), findsNothing);
    });

    testWidgets('monogram avatar is deterministic per name', (tester) async {
      await tester.pumpWidget(_wrap(_screen([
        _member(userId: 'u1', displayName: 'Sara Chef'),
        _member(userId: 'u2', displayName: 'Sara Chef'),
      ])));
      await tester.pumpAndSettle();

      final avatars =
          tester.widgetList<CircleAvatar>(find.byType(CircleAvatar)).toList();
      expect(avatars, hasLength(2));
      expect(avatars[0].backgroundColor, avatars[1].backgroundColor);
      expect(find.text('S'), findsNWidgets(2));
    });

    testWidgets('empty state adds the invite hint; filtered adds the no-results hint',
        (tester) async {
      await tester.pumpWidget(_wrap(_screen(const [])));
      await tester.pumpAndSettle();

      expect(find.text('Invite peers you work with to join the community'),
          findsOneWidget);

      await tester.enterText(find.byType(TextField), 'zzz');
      await tester.testTextInput.receiveAction(TextInputAction.done);
      await tester.pumpAndSettle();

      expect(find.text('Try a different name or level'), findsOneWidget);
      expect(find.text('Invite peers you work with to join the community'),
          findsNothing);
    });
  });

  group('member detail redesign', () {
    testWidgets('hero shows status/role/level pills and joined-on line',
        (tester) async {
      await tester.pumpWidget(_wrap(_screen([
        _member(userId: 'u1', displayName: 'Sara Chef'),
      ])));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Sara Chef'));
      await tester.pumpAndSettle();

      expect(find.text('Active'), findsOneWidget);
      expect(find.byIcon(Icons.check_circle_outline), findsOneWidget);
      expect(find.text('Member'), findsOneWidget);
      expect(find.byIcon(Icons.person_outline), findsNWidgets(2));
      expect(find.text('Gold'), findsWidgets);
      expect(find.textContaining('Joined on'), findsOneWidget);
    });
  });

  group('moderation confirm styling', () {
    testWidgets('suspend confirm button uses the error color',
        (tester) async {
      await tester.pumpWidget(_wrap(_screen([
        _member(userId: 'u1', displayName: 'Sara Chef'),
      ], canModerate: true)));
      await tester.pumpAndSettle();

      await tester.tap(find.byIcon(Icons.more_vert));
      await tester.pumpAndSettle();
      await tester.tap(
          find.widgetWithText(PopupMenuItem<String>, 'Suspend member'));
      await tester.pumpAndSettle();

      final theme =
          Theme.of(tester.element(find.byType(NationalMembersScreen)));
      final button = tester.widget<FilledButton>(find.descendant(
        of: find.byType(AlertDialog),
        matching: find.widgetWithText(FilledButton, 'Suspend member'),
      ));
      expect(button.style?.backgroundColor?.resolve(const <WidgetState>{}),
          theme.colorScheme.error);

      await tester.tap(find.widgetWithText(TextButton, 'Cancel'));
      await tester.pumpAndSettle();
    });

    testWidgets('promote confirm button keeps the theme default',
        (tester) async {
      await tester.pumpWidget(_wrap(_screen([
        _member(userId: 'u1', displayName: 'Sara Chef'),
      ], canModerate: true)));
      await tester.pumpAndSettle();

      await tester.tap(find.byIcon(Icons.more_vert));
      await tester.pumpAndSettle();
      await tester.tap(find.widgetWithText(
          PopupMenuItem<String>, 'Promote to moderator'));
      await tester.pumpAndSettle();

      final button = tester.widget<FilledButton>(find.descendant(
        of: find.byType(AlertDialog),
        matching: find.widgetWithText(FilledButton, 'Promote to moderator'),
      ));
      expect(button.style?.backgroundColor, isNull);

      await tester.tap(find.widgetWithText(TextButton, 'Cancel'));
      await tester.pumpAndSettle();
    });
  });
}