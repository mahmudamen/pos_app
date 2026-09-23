import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/community.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/community/community_hub_screen.dart';
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

const _chef = StaffProfile(
  userId: 'user-chef',
  displayName: 'Sara Chef',
  email: 'sara@example.com',
  headline: 'Head Chef',
  bio: 'Team lead & grill master',
  location: 'Cairo',
  yearsExperience: 12,
  isChief: true,
  skills: ['grill', 'sous'],
  resume: {},
  hasProfile: true,
);

const _lineCook = StaffProfile(
  userId: 'user-cook',
  displayName: 'Omar Cook',
  email: 'omar@example.com',
  hasProfile: false,
);

const _bakery = Company(
  id: 'company-1',
  name: 'Bakery X',
  description: 'Fresh sourdough',
  industry: 'food',
  city: 'Cairo',
  membersCount: 1,
);

class _FakeCommunityApi extends ApiClient {
  _FakeCommunityApi();

  int updateProfileCalls = 0;
  int addMemberCalls = 0;
  int removeMemberCalls = 0;
  int createCompanyCalls = 0;
  String? lastRoleFilter;

  @override
  Future<ProfilesPage> listProfiles(
    Session session, {
    String query = '',
    String? role,
    int page = 1,
    int limit = 50,
  }) async {
    lastRoleFilter = role;
    final all = [_chef, _lineCook];
    return ProfilesPage(
      profiles: role == 'chief' ? [all.first] : all,
      total: role == 'chief' ? 1 : all.length,
      page: page,
      limit: limit,
    );
  }

  @override
  Future<StaffProfile> myProfile(Session session) async => _chef;

  @override
  Future<StaffProfile> getProfile(Session session, String userId) async =>
      userId == _chef.userId ? _chef : _lineCook;

  @override
  Future<StaffProfile> updateMyProfile(
    Session session, {
    required String headline,
    required String bio,
    required String location,
    required int yearsExperience,
    required bool isChief,
    required List<String> skills,
    Map<String, dynamic>? resume,
  }) async {
    updateProfileCalls++;
    return _chef;
  }

  @override
  Future<CompaniesPage> listCompanies(
    Session session, {
    String query = '',
    int page = 1,
    int limit = 50,
  }) async {
    return CompaniesPage(
      companies: const [_bakery],
      total: 1,
      page: page,
      limit: limit,
    );
  }

  @override
  Future<Company> getCompany(Session session, String companyId) async =>
      const Company(
        id: 'company-1',
        name: 'Bakery X',
        description: 'Fresh sourdough',
        industry: 'food',
        city: 'Cairo',
        members: [
          CompanyMember(
            id: 'member-1',
            companyId: 'company-1',
            userId: 'user-chef',
            displayName: 'Sara Chef',
            role: 'staff',
            title: 'Line Cook',
            createdAt: '',
          ),
        ],
      );

  @override
  Future<Company> createCompany(
    Session session, {
    required String name,
    String slug = '',
    String description = '',
    String industry = '',
    String website = '',
    String city = '',
  }) async {
    createCompanyCalls++;
    return _bakery;
  }

  @override
  Future<CompanyMember> addCompanyMember(
    Session session,
    String companyId, {
    required String userId,
    String role = 'staff',
    String title = '',
  }) async {
    addMemberCalls++;
    return const CompanyMember(
      id: 'member-2',
      companyId: 'company-1',
      userId: 'user-cook',
      displayName: 'Omar Cook',
      role: 'staff',
      title: '',
      createdAt: '',
    );
  }

  @override
  Future<void> removeCompanyMember(
      Session session, String companyId, String userId) async {
    removeMemberCalls++;
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
  testWidgets('hub shows profiles and companies entries', (tester) async {
    await tester.pumpWidget(
        _wrap(CommunityHubScreen(session: _manager, apiClient: _FakeCommunityApi())));
    await tester.pumpAndSettle();

    expect(find.text('Community'), findsOneWidget);
    expect(find.text('Profiles'), findsOneWidget);
    expect(find.text('Companies'), findsOneWidget);
  });

  testWidgets('profiles list shows profiles, chief chip filters',
      (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(
        _wrap(CommunityHubScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Profiles'));
    await tester.pumpAndSettle();

    expect(find.text('Sara Chef'), findsOneWidget);
    expect(find.text('Omar Cook'), findsOneWidget);
    expect(find.textContaining('Head Chef'), findsOneWidget);

    await tester.tap(find.text('Chief'));
    await tester.pumpAndSettle();

    expect(api.lastRoleFilter, 'chief');
    expect(find.text('Sara Chef'), findsOneWidget);
    expect(find.text('Omar Cook'), findsNothing);
  });

  testWidgets('profile detail shows skills and memberships', (tester) async {
    await tester.pumpWidget(
        _wrap(CommunityHubScreen(session: _manager, apiClient: _FakeCommunityApi())));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Profiles'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Sara Chef'));
    await tester.pumpAndSettle();

    expect(find.text('Team lead & grill master'), findsOneWidget);
    expect(find.text('grill'), findsOneWidget);
    expect(find.text('sous'), findsOneWidget);
    expect(find.text('Cairo'), findsOneWidget);
  });

  testWidgets('my profile edit loads and saves', (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(
        _wrap(CommunityHubScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.byIcon(Icons.badge_outlined));
    await tester.pumpAndSettle();

    expect(find.text('Sara Chef'), findsNothing);
    expect(find.widgetWithText(TextField, 'Head Chef'), findsOneWidget);
    expect(find.widgetWithText(TextField, 'grill, sous'), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, 'Save'));
    await tester.pumpAndSettle();

    expect(api.updateProfileCalls, 1);
  });

  testWidgets('companies list opens detail and manages members',
      (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(
        _wrap(CommunityHubScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Companies'));
    await tester.pumpAndSettle();

    expect(find.text('Bakery X'), findsOneWidget);
    expect(find.textContaining('food'), findsOneWidget);

    await tester.tap(find.text('Bakery X'));
    await tester.pumpAndSettle();

    expect(find.text('Sara Chef'), findsOneWidget);
    expect(find.textContaining('Line Cook'), findsOneWidget);

    await tester.tap(find.byIcon(Icons.person_remove_outlined));
    await tester.pumpAndSettle();

    expect(api.removeMemberCalls, 1);
  });

  testWidgets('companies add-company form posts', (tester) async {
    final api = _FakeCommunityApi();
    await tester.pumpWidget(
        _wrap(CommunityHubScreen(session: _manager, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Companies'));
    await tester.pumpAndSettle();
    await tester.tap(find.byIcon(Icons.add_business_outlined));
    await tester.pumpAndSettle();

    await tester.enterText(find.widgetWithText(TextField, 'Company name'), 'New Bakery');
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(api.createCompanyCalls, 1);
  });
}