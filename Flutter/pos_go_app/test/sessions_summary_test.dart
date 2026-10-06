import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/pos/session_screen.dart';
import 'package:pos_go_app/features/pos/sessions_summary_screen.dart';
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

const _summaryResult = RangeSummary(
  sessionsOpened: 2,
  sessionsClosed: 1,
  salesCount: 3,
  subtotalMinor: 3000,
  discountMinor: 100,
  taxMinor: 0,
  totalMinor: 2900,
  cashMinor: 1900,
  cardMinor: 700,
  mobileMinor: 300,
  cashDifferenceMinor: -50,
  averageSaleMinor: 966,
);

class _SummaryApi extends ApiClient {
  final List<String?> froms = [];
  bool fail = false;

  @override
  Future<RangeSummary> sessionSummary(
    Session session, {
    String? from,
    String? to,
  }) async {
    froms.add(from);
    if (fail) throw Exception('network down');
    return _summaryResult;
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
  testWidgets('the summary loads today and switches windows on the chips',
      (tester) async {
    final api = _SummaryApi();

    await tester.pumpWidget(_wrap(SessionsSummaryScreen(
      session: _session,
      apiClient: api,
    )));
    await tester.pumpAndSettle();

    expect(find.text('Revenue'), findsOneWidget);
    expect(find.text('E£29.00'), findsOneWidget);
    expect(find.text('Shifts opened'), findsOneWidget);
    expect(find.text('-E£0.50'), findsOneWidget,
        reason: 'the drawer shortage is shown signed');
    expect(api.froms, hasLength(1));
    expect(api.froms.single, isNotNull);

    await tester.tap(find.widgetWithText(FilterChip, 'Last 7 days'));
    await tester.pumpAndSettle();

    expect(api.froms, hasLength(2));
    expect(api.froms.last, isNot(api.froms.first),
        reason: 'a different window asks the server for a different range');
  });

  testWidgets('a failed summary offers a retry that recovers', (tester) async {
    final api = _SummaryApi()..fail = true;

    await tester.pumpWidget(_wrap(SessionsSummaryScreen(
      session: _session,
      apiClient: api,
    )));
    await tester.pumpAndSettle();

    expect(find.textContaining('network down'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Retry'), findsOneWidget);

    api.fail = false;
    await tester.tap(find.widgetWithText(FilledButton, 'Retry'));
    await tester.pumpAndSettle();

    expect(find.text('Revenue'), findsOneWidget);
    expect(api.froms, hasLength(2));
  });

  testWidgets('the Z-report prices in the tenant currency, not a hard-coded EGP',
      (tester) async {
    const report = RegisterSession(
      id: 'session-1',
      status: 'closed',
      openingCashMinor: 100,
      openedAt: '2026-10-06T09:00:00Z',
      openedBy: 'Cashier One',
      summary: SessionSummary(
        salesCount: 0,
        subtotalMinor: 0,
        discountMinor: 0,
        taxMinor: 0,
        totalMinor: 0,
        cashMinor: 0,
        cardMinor: 0,
        mobileMinor: 0,
        expectedCashMinor: 0,
      ),
    );

    await tester.pumpWidget(_wrap(SessionReportScreen(
      sessionData: report,
      apiClient: ApiClient(),
      currency: 'USD',
    )));
    await tester.pumpAndSettle();

    expect(find.textContaining('E£'), findsNothing,
        reason: 'the tenant is not Egyptian, so no Egyptian pound appears');
    expect(find.textContaining('\$USD1.00'), findsWidgets,
        reason: 'starting cash renders in the tenant currency');
  });
}
