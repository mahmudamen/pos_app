import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/community.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/community/community_hub_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

/// Real-backend national-community E2E on the device.
///
/// Logs in as the demo restaurant manager, opens the community hub on the
/// real NationalHubScreen, joins the Egypt national community through the
/// invite-code dialog (code minted by the `saas_admin` bootstrap inviter),
/// asserts the membership view, then opens the member directory.
///
///   flutter test integration_test/national_e2e_test.dart -d <device> \
///     --dart-define=API_BASE_URL=http://<host>:<port>
///
/// Idempotent: if the demo manager is already a national member it skips the
/// join step and just asserts the member UI. Needs the seeded demo tenants
/// (admin@demo-restaurant.com / admin and admin@posgo.saas / admin).
void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  const saasTenantId = '78da200f-7f9c-4d90-b07d-3cae30315ade';

  Future<void> pumpUntil(
    WidgetTester tester,
    Finder finder, {
    Duration timeout = const Duration(seconds: 45),
  }) async {
    final end = DateTime.now().add(timeout);
    while (DateTime.now().isBefore(end)) {
      await tester.pump(const Duration(milliseconds: 200));
      if (finder.evaluate().isNotEmpty) return;
    }
    fail('Timed out waiting for $finder');
  }

  testWidgets('national community: hub -> join/invite -> membership -> directory',
      (tester) async {
    final api = ApiClient();
    final store = SessionStore();
    await store.saveLanguage('en');

    final session = await api.login(
      tenantId: 'demo-restaurant',
      email: 'admin@demo-restaurant.com',
      password: 'admin',
      deviceId: 'national-e2e',
      deviceName: 'National E2E',
    );
    expect(session.accessToken, isNotEmpty);

    NationalMember? me;
    try {
      me = await api.nationalMe(session);
    } on ApiException {
      me = null;
    }

    String? inviteCode;
    if (me == null) {
      // Mint a single-use code through the saas_admin bootstrap inviter.
      Session saas;
      try {
        saas = await api.login(
          tenantId: 'saas',
          email: 'admin@posgo.saas',
          password: 'admin',
          deviceId: 'national-e2e-inviter',
          deviceName: 'National E2E',
        );
      } on ApiException {
        saas = await api.login(
          tenantId: saasTenantId,
          email: 'admin@posgo.saas',
          password: 'admin',
          deviceId: 'national-e2e-inviter',
          deviceName: 'National E2E',
        );
      }
      final invite = await api.createNationalInvitation(
        saas,
        email: 'admin@demo-restaurant.com',
        note: 'posTest on-device deploy verification',
      );
      expect(invite.code.length, greaterThan(6));
      inviteCode = invite.code;
    }

    final harness = MaterialApp(
      debugShowCheckedModeBanner: false,
      locale: const Locale('en'),
      supportedLocales: AppStrings.supportedLocales,
      localizationsDelegates: const [
        AppStrings.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      theme: ThemeData(useMaterial3: true),
      home: CommunityHubScreen(session: session, apiClient: api),
    );
    await tester.pumpWidget(harness);
    await tester.pump();

    // Hub -> national tile (real navigation through the hub screen).
    await pumpUntil(tester, find.text('Egypt national community'));
    await tester.tap(find.text('Egypt national community'));
    await tester.pump(const Duration(milliseconds: 400));

    final joinNow = find.text('Join now');
    if (inviteCode != null) {
      // Not a member yet: drive the join dialog end to end.
      await pumpUntil(tester, joinNow, timeout: const Duration(seconds: 30));
      await tester.tap(joinNow);
      await tester.pump(const Duration(milliseconds: 300));

      await tester.enterText(find.byType(TextField).last, inviteCode);
      await tester.pump(const Duration(milliseconds: 300));
      // The dialog's confirm button shared the "Join now" label.
      await tester.tap(find.text('Join now').last);
    }

    // Member view: invitation + directory tiles, level chip, no join prompt.
    await pumpUntil(tester, find.text('Invite peers'));
    await tester.pump(const Duration(milliseconds: 400));
    expect(find.text('Member directory'), findsOneWidget);
    expect(find.text('Join now'), findsNothing);

    // The persisted membership is real on the backend.
    final after = await api.nationalMe(session);
    expect(after, isNotNull);
    expect(after!.userId, session.userId);
    expect(after.displayName, isNotEmpty);

    // Directory lists the just-joined member.
    final members = await api.listNationalMembers(session,
        query: after.displayName);
    expect(members.members, isNotEmpty);
    expect(members.members.first.userId, after.userId);

    // On-screen directory works too.
    await tester.tap(find.text('Member directory'));
    await tester.pump(const Duration(milliseconds: 400));
    await pumpUntil(tester, find.byType(TextField));
    expect(find.text('Member directory'), findsWidgets);
  });
}