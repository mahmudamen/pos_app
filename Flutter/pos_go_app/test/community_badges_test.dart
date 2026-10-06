import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/community.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/community/badges_screen.dart';
import 'package:pos_go_app/features/community/community_hub_screen.dart';
import 'package:pos_go_app/features/community/profiles_screen.dart';
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

const _cashier = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-cashier',
  displayName: 'Cashier One',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'cashier',
  currencyCode: 'EGP',
);

const _peerProfile = StaffProfile(
  userId: 'user-9',
  displayName: 'Omar Grill',
  email: 'omar@example.com',
  skills: ['grill', 'stock'],
);

const _topBadge = BadgeDefinition(
  id: 'badge-1',
  code: 'top-seller',
  name: 'Top Seller',
  description: 'Ten sales in a day',
  icon: 'star',
  earnedCount: 4,
);

const _oldBadge = BadgeDefinition(
  id: 'badge-2',
  code: 'old-badge',
  name: 'Old Badge',
  isActive: false,
);

const _awardedBadge = UserBadge(
  badgeId: 'badge-1',
  code: 'top-seller',
  name: 'Top Seller',
  note: 'Beat target',
  awardedByName: 'Store Manager',
);

const _grillGroup = EndorsementGroup(
  skill: 'grill',
  count: 3,
  givers: [
    EndorsementGiver(
      id: 'end-1',
      giverId: 'user-2',
      giverName: 'Sara Chef',
    ),
    EndorsementGiver(
      id: 'end-2',
      giverId: 'user-3',
      giverName: 'Omar Second',
    ),
  ],
);

const _stockGroup = EndorsementGroup(
  skill: 'stock',
  count: 1,
  myEndorsementId: 'end-9',
  givers: [
    EndorsementGiver(
      id: 'end-9',
      giverId: 'user-manager',
      giverName: 'Store Manager',
    ),
  ],
);

class _FakeCommunityApi extends ApiClient {
  String? lastQuery;
  bool lastIncludeInactive = false;
  String? lastCreatedCode;
  String? lastCreatedName;
  String? lastCreatedDescription;
  String? lastUpdatedName;
  bool? lastUpdatedActive;
  String? lastAwardBadgeId;
  String? lastAwardNote;
  String? lastRevokedBadgeId;
  String? lastEndorsedSkill;
  String? lastEndorseNote;
  String? lastWithdrawnId;

  @override
  Future<List<BadgeDefinition>> listBadges(
    Session session, {
    String query = '',
    bool includeInactive = false,
    int page = 1,
    int limit = 100,
  }) async {
    lastQuery = query;
    lastIncludeInactive = includeInactive;
    return const [_topBadge, _oldBadge];
  }

  @override
  Future<BadgeDefinition> createBadge(
    Session session, {
    required String code,
    required String name,
    String description = '',
    String icon = 'military_tech',
    Map<String, dynamic> criteria = const {},
  }) async {
    lastCreatedCode = code;
    lastCreatedName = name;
    lastCreatedDescription = description;
    return _topBadge;
  }

  @override
  Future<BadgeDefinition> updateBadge(
    Session session,
    String badgeId, {
    String? name,
    String? description,
    String? icon,
    bool? isActive,
  }) async {
    lastUpdatedName = name;
    lastUpdatedActive = isActive;
    return _topBadge;
  }

  @override
  Future<StaffProfile> getProfile(Session session, String userId) async =>
      _peerProfile;

  @override
  Future<List<UserBadge>> listProfileBadges(
          Session session, String userId) async =>
      const [_awardedBadge];

  @override
  Future<UserBadge> awardProfileBadge(
    Session session,
    String userId, {
    required String badgeId,
    String note = '',
  }) async {
    lastAwardBadgeId = badgeId;
    lastAwardNote = note;
    return _awardedBadge;
  }

  @override
  Future<void> revokeProfileBadge(
      Session session, String userId, String badgeId) async {
    lastRevokedBadgeId = badgeId;
  }

  @override
  Future<List<EndorsementGroup>> listProfileEndorsements(
          Session session, String userId) async =>
      const [_grillGroup, _stockGroup];

  @override
  Future<Endorsement> endorsePeer(
    Session session, {
    required String receiverId,
    required String skill,
    String note = '',
  }) async {
    lastEndorsedSkill = skill;
    lastEndorseNote = note;
    return const Endorsement(id: 'end-1', skill: 'grill', receiverId: 'user-9');
  }

  @override
  Future<void> withdrawEndorsement(
      Session session, String endorsementId) async {
    lastWithdrawnId = endorsementId;
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

/// Lets an already-scheduled SnackBar reach the screen. Advancing 5s here
/// would run its 4s lifetime out and dismiss it before the assertion.
Future<void> _flushSnackbar(WidgetTester tester) async {
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 300));
  await tester.pumpAndSettle();
}

void main() {
  testWidgets('hub shows the badges entry and opens the badge list',
      (tester) async {
    await tester.pumpWidget(_wrap(CommunityHubScreen(
        session: _manager, apiClient: _FakeCommunityApi())));
    await tester.pumpAndSettle();

    expect(find.text('Badges'), findsOneWidget);
    expect(
        find.text('Team recognitions and achievements'), findsOneWidget);

    await tester.tap(find.text('Badges'));
    await tester.pumpAndSettle();

    expect(find.byType(BadgesScreen), findsOneWidget);
    expect(find.text('Top Seller'), findsOneWidget);
    expect(find.text('Old Badge'), findsOneWidget);
    expect(find.textContaining('top-seller'), findsOneWidget);
  });

  testWidgets('badge search and include-inactive reach the API',
      (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(
        _wrap(BadgesScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    expect(api.lastIncludeInactive, isTrue);

    await tester.enterText(find.byType(TextField).first, 'top');
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pumpAndSettle();

    expect(api.lastQuery, 'top');
  });

  testWidgets('cashier does not see the add-badge action', (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(
        _wrap(BadgesScreen(session: _cashier, apiClient: api)));
    await tester.pumpAndSettle();

    expect(find.byType(FloatingActionButton), findsNothing);
    expect(api.lastIncludeInactive, isFalse);
  });

  testWidgets('manager creates a badge through the add sheet',
      (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(
        _wrap(BadgesScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.byType(FloatingActionButton));
    await tester.pumpAndSettle();
    expect(find.text('Add badge'), findsOneWidget);

    // Address fields by their labels: the screen's own search TextField
    // also matches byType and would shift the indices.
    await tester.enterText(
        find.widgetWithText(TextField, 'Badge code'), 'top-seller');
    await tester.enterText(
        find.widgetWithText(TextField, 'Badge name'), 'Top Seller');
    await tester.enterText(
        find.widgetWithText(TextField, 'Description'), 'Ten sales');
    await tester.tap(find.widgetWithText(FilledButton, 'Save'));
    await tester.pumpAndSettle();
    await _flushSnackbar(tester);

    expect(api.lastCreatedCode, 'top-seller');
    expect(api.lastCreatedName, 'Top Seller');
    expect(api.lastCreatedDescription, 'Ten sales');
    expect(find.text('Badge saved'), findsOneWidget);
  });

  testWidgets('manager edits a badge from the list', (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(
        _wrap(BadgesScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Top Seller'));
    await tester.pumpAndSettle();
    expect(find.widgetWithText(FilledButton, 'Save'), findsOneWidget);

    await tester.enterText(
        find.widgetWithText(TextField, 'Badge name'), 'Elite Seller');
    await tester.tap(find.widgetWithText(FilledButton, 'Save'));
    await tester.pumpAndSettle();
    await _flushSnackbar(tester);

    expect(api.lastUpdatedName, 'Elite Seller');
    expect(api.lastUpdatedActive, isTrue);
    expect(find.text('Badge saved'), findsOneWidget);
  });

  testWidgets('profile detail shows badges and endorsements',
      (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(_wrap(ProfileDetailScreen(
        session: _manager, apiClient: api, profile: _peerProfile)));
    await tester.pumpAndSettle();

    expect(find.text('Badges'), findsOneWidget);
    expect(find.text('Top Seller'), findsOneWidget);
    expect(find.text('Beat target · Store Manager'), findsOneWidget);
    expect(find.text('Endorsements'), findsOneWidget);
    expect(find.text('Sara Chef, Omar Second'), findsOneWidget);
    expect(find.text('Withdraw'), findsOneWidget);
    // Both skills also render as profile chips, so only their presence
    // (not a count) is asserted here.
    expect(find.text('grill'), findsWidgets);
    expect(find.text('stock'), findsWidgets);
  });

  testWidgets('peer endorses a skill through the dialog', (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(_wrap(ProfileDetailScreen(
        session: _cashier, apiClient: api, profile: _peerProfile)));
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(OutlinedButton, 'Endorse a skill'));
    await tester.pumpAndSettle();

    expect(
        find.widgetWithText(FilledButton, 'Add endorsement'), findsOneWidget);
    await tester.enterText(find.byType(TextField).last, 'solid');
    await tester.tap(find.widgetWithText(FilledButton, 'Add endorsement'));
    await tester.pumpAndSettle();
    await _flushSnackbar(tester);

    expect(api.lastEndorsedSkill, 'grill');
    expect(api.lastEndorseNote, 'solid');
    expect(find.text('Endorsement added'), findsOneWidget);
  });

  testWidgets('manager revokes a badge from the profile', (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(_wrap(ProfileDetailScreen(
        session: _manager, apiClient: api, profile: _peerProfile)));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Badge removed'));
    await tester.pumpAndSettle();
    await _flushSnackbar(tester);

    expect(api.lastRevokedBadgeId, 'badge-1');
    expect(find.text('Badge removed'), findsOneWidget);
  });

  testWidgets('manager awards a badge from the profile', (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(_wrap(ProfileDetailScreen(
        session: _manager, apiClient: api, profile: _peerProfile)));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Award badge'));
    await tester.pumpAndSettle();

    expect(find.text('Top Seller (top-seller)'), findsOneWidget);
    await tester.enterText(find.byType(TextField).last, 'Beat target');
    await tester.tap(find.widgetWithText(FilledButton, 'Award badge'));
    await tester.pumpAndSettle();
    await _flushSnackbar(tester);

    expect(api.lastAwardBadgeId, 'badge-1');
    expect(api.lastAwardNote, 'Beat target');
    expect(find.text('Badge awarded'), findsOneWidget);
  });

  testWidgets('viewer withdraws their own endorsement', (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(_wrap(ProfileDetailScreen(
        session: _manager, apiClient: api, profile: _peerProfile)));
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(TextButton, 'Withdraw'));
    await tester.pumpAndSettle();
    await _flushSnackbar(tester);

    expect(api.lastWithdrawnId, 'end-9');
    expect(find.text('Endorsement withdrawn'), findsOneWidget);
  });
}
