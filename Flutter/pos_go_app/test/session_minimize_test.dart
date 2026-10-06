import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

const _session = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-1',
  displayName: 'Restaurant Admin',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'manager',
  currencyCode: 'EGP',
);

const _summary = SessionSummary(
  salesCount: 0,
  subtotalMinor: 0,
  discountMinor: 0,
  taxMinor: 0,
  totalMinor: 0,
  cashMinor: 0,
  cardMinor: 0,
  mobileMinor: 0,
  expectedCashMinor: 1250,
);

RegisterSession _openSession() => const RegisterSession(
      id: 'aaaa1111-2222-3333-4444-555566667777',
      status: 'open',
      openingCashMinor: 5000,
      openedAt: '2026-10-06T09:00:00Z',
      openedBy: 'Restaurant Admin',
      summary: _summary,
    );

class _SessionBarApi extends ApiClient {
  _SessionBarApi({required this.openRegister});

  RegisterSession? openRegister;

  @override
  Future<List<Product>> products(Session session, {String search = ''}) async =>
      const [];

  @override
  Future<List<Category>> categories(Session session) async => const [];

  @override
  Future<TenantSettings> settings(Session session) async =>
      const TenantSettings();

  @override
  Future<RegisterSession?> currentSession(Session session) async =>
      openRegister;
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
  setUp(() {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
  });

  testWidgets('the open session bar collapses to one line and remembers it',
      (tester) async {
    final api = _SessionBarApi(openRegister: _openSession());
    final store = SessionStore();

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      sessionStore: store,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(OutlinedButton, 'Finish session'), findsOneWidget,
        reason: 'the bar starts expanded');

    await tester.tap(find.byTooltip('Minimize session'));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(OutlinedButton, 'Finish session'), findsNothing,
        reason: 'the collapsed card drops the actions');
    expect(find.byTooltip('Expand session'), findsOneWidget);
    expect(find.textContaining('AAAA'), findsOneWidget,
        reason: 'the session id stays visible on the single line');
    expect(await store.readSessionMinimized(), isTrue,
        reason: 'the choice is persisted');

    // Relaunch: a fresh widget instance restores the collapsed state.
    await tester.pumpWidget(const SizedBox());
    await tester.pumpAndSettle();
    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      sessionStore: store,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(OutlinedButton, 'Finish session'), findsNothing,
        reason: 'a restart comes back minimized');
    expect(find.byTooltip('Expand session'), findsOneWidget);

    await tester.tap(find.byTooltip('Expand session'));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(OutlinedButton, 'Finish session'), findsOneWidget);
    expect(await store.readSessionMinimized(), isFalse);
  });

  testWidgets('a session restored at cold start is titled Resume session',
      (tester) async {
    final api = _SessionBarApi(openRegister: _openSession());

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    expect(find.textContaining('Resume session'), findsOneWidget,
        reason: 'a session found at startup was resumed, not opened here');
    expect(find.textContaining('POS session'), findsNothing);
  });

  testWidgets('minimizing never hides the no-session open action',
      (tester) async {
    final api = _SessionBarApi(openRegister: null);

    await tester.pumpWidget(_wrap(PosScreen(
      session: _session,
      apiClient: api,
      onSignOut: () {},
    )));
    await tester.pumpAndSettle();

    expect(find.text('Open session'), findsOneWidget);
    expect(find.byTooltip('Minimize session'), findsNothing,
        reason: 'there is nothing to collapse without an open session');
  });
}
