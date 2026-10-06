import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/security.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/pos/session_screen.dart';
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
  expectedCashMinor: 0,
);

SessionListItem _row({
  required String id,
  required String status,
  String deviceId = 'device-1',
  String deviceName = 'Front register',
  int salesCount = 0,
  int totalMinor = 0,
}) =>
    SessionListItem(
      id: id,
      status: status,
      openingCashMinor: 100,
      openedAt: '2026-10-06T09:00:00Z',
      openedBy: 'Cashier One',
      deviceId: deviceId,
      deviceName: deviceName,
      salesCount: salesCount,
      totalMinor: totalMinor,
    );

class _SessionsApi extends ApiClient {
  _SessionsApi({required this.rows, this.managerClosePin = false});

  List<SessionListItem> rows;
  final bool managerClosePin;
  String? lastStatus;
  int closeCalls = 0;
  int? lastClosingCash;
  String lastClosedId = '';
  final List<String> pinAttempts = [];

  @override
  Future<SessionsPage> sessions(
    Session session, {
    int page = 1,
    int limit = 50,
    String? status,
    String? deviceId,
    String? from,
    String? to,
  }) async {
    lastStatus = status;
    final filtered =
        rows.where((r) => status == null || r.status == status).toList();
    return SessionsPage(
      sessions: filtered,
      total: filtered.length,
      page: page,
      limit: limit,
    );
  }

  @override
  Future<TenantSettings> settings(Session session) async =>
      TenantSettings(managerClosePin: managerClosePin);

  @override
  Future<PinVerifyResult> verifyPin(
    Session session, {
    required String pin,
  }) async {
    pinAttempts.add(pin);
    return PinVerifyResult(
      valid: pin == '1234',
      hasPin: true,
      attemptsLeft: 4,
      locked: false,
    );
  }

  @override
  Future<RegisterSession> closeSession(
    Session session,
    String sessionId, {
    int? closingCashMinor,
    String managerPin = '',
  }) async {
    closeCalls++;
    lastClosedId = sessionId;
    lastClosingCash = closingCashMinor;
    rows = [
      for (final row in rows)
        if (row.id == sessionId)
          _row(
            id: row.id,
            status: 'closed',
            deviceId: row.deviceId,
            deviceName: row.deviceName,
            salesCount: row.salesCount,
            totalMinor: row.totalMinor,
          )
        else
          row,
    ];
    return RegisterSession(
      id: sessionId,
      status: 'closed',
      openingCashMinor: 100,
      closingCashMinor: closingCashMinor,
      expectedCashMinor: 100,
      cashDifferenceMinor: closingCashMinor == null ? null : closingCashMinor - 100,
      openedAt: '2026-10-06T09:00:00Z',
      closedAt: '2026-10-06T10:00:00Z',
      openedBy: 'Cashier One',
      summary: _summary,
    );
  }

  @override
  Future<RegisterSession> sessionDetail(Session session, String sessionId) async {
    final row = rows.firstWhere((r) => r.id == sessionId);
    return RegisterSession(
      id: sessionId,
      status: row.status,
      openingCashMinor: 100,
      openedAt: row.openedAt,
      openedBy: row.openedBy,
      summary: _summary,
    );
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
  testWidgets('an open session on this terminal offers close with device name',
      (tester) async {
    final api = _SessionsApi(
      rows: [
        _row(id: 'session-1', status: 'open'),
      ],
    );

    await tester.pumpWidget(_wrap(SessionHistoryScreen(
      session: _session,
      apiClient: api,
    )));
    await tester.pumpAndSettle();

    expect(find.textContaining('Front register'), findsOneWidget,
        reason: 'the row names the terminal the shift runs on');
    expect(find.widgetWithText(OutlinedButton, 'Finish session'), findsOneWidget);

    await tester
        .tap(find.widgetWithText(OutlinedButton, 'Finish session'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), '50');
    await tester.tap(find.widgetWithText(FilledButton, 'Finish session'));
    await tester.pumpAndSettle();

    expect(api.closeCalls, 1, reason: 'the row close posts the close');
    expect(api.lastClosedId, 'session-1');
    expect(api.lastClosingCash, 5000, reason: 'E£50 counted = 5000 minor');
    expect(find.byType(SessionReportScreen), findsOneWidget,
        reason: 'the Z-report opens for the closed session');
  });

  testWidgets('a cashier may close own terminal but not a foreign one',
      (tester) async {
    final api = _SessionsApi(
      rows: [
        _row(id: 'session-own', status: 'open', deviceName: 'Front register'),
        _row(
            id: 'session-other',
            status: 'open',
            deviceId: 'device-9',
            deviceName: 'Back register'),
      ],
    );
    const cashier = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier One',
      tenantId: 'tenant-1',
      deviceId: 'device-1',
      role: 'cashier',
      currencyCode: 'EGP',
    );

    await tester.pumpWidget(_wrap(SessionHistoryScreen(
      session: cashier,
      apiClient: api,
    )));
    await tester.pumpAndSettle();

    expect(find.widgetWithText(OutlinedButton, 'Finish session'), findsOneWidget,
        reason: 'only the own-device row gets a close action');
    expect(find.textContaining('Back register'), findsOneWidget);
    expect(find.text('Open'), findsWidgets,
        reason: 'the foreign open row stays a status chip');
  });

  testWidgets('status chips reload the list with the matching filter',
      (tester) async {
    final api = _SessionsApi(
      rows: [
        _row(id: 'session-1', status: 'open'),
        _row(id: 'session-2', status: 'closed'),
      ],
    );

    await tester.pumpWidget(_wrap(SessionHistoryScreen(
      session: _session,
      apiClient: api,
    )));
    await tester.pumpAndSettle();
    expect(api.lastStatus, isNull, reason: 'All is the default view');

    await tester.tap(find.widgetWithText(FilterChip, 'Open'));
    await tester.pumpAndSettle();
    expect(api.lastStatus, 'open');
    expect(find.textContaining('Front register'), findsOneWidget,
        reason: 'only the open shift is listed');

    await tester.tap(find.widgetWithText(FilterChip, 'Closed'));
    await tester.pumpAndSettle();
    expect(api.lastStatus, 'closed');

    await tester.tap(find.widgetWithText(FilterChip, 'All'));
    await tester.pumpAndSettle();
    expect(api.lastStatus, isNull);
  });

  testWidgets('close demands the manager PIN when the setting requires it',
      (tester) async {
    final api = _SessionsApi(
      rows: [
        _row(id: 'session-1', status: 'open'),
      ],
      managerClosePin: true,
    );

    await tester.pumpWidget(_wrap(SessionHistoryScreen(
      session: _session,
      apiClient: api,
    )));
    await tester.pumpAndSettle();

    await tester
        .tap(find.widgetWithText(OutlinedButton, 'Finish session'));
    await tester.pumpAndSettle();
    await tester.enterText(find.byType(TextField), '50');
    await tester.tap(find.widgetWithText(FilledButton, 'Finish session'));
    await tester.pumpAndSettle();

    expect(api.closeCalls, 0, reason: 'the PIN gates the close request');
    expect(find.byType(TextField), findsOneWidget,
        reason: 'the PIN prompt is the only field left');

    await tester.enterText(find.byType(TextField), '9999');
    await tester.tap(find.widgetWithText(FilledButton, 'OK'));
    await tester.pumpAndSettle();
    expect(api.pinAttempts, ['9999']);
    expect(api.closeCalls, 0);
    expect(find.text('4 attempts left'), findsOneWidget,
        reason: 'a wrong PIN stays in the loop with the remaining count');

    await tester.enterText(find.byType(TextField), '1234');
    await tester.tap(find.widgetWithText(FilledButton, 'OK'));
    await tester.pumpAndSettle();

    expect(api.pinAttempts, ['9999', '1234']);
    expect(api.closeCalls, 1, reason: 'the right PIN releases the close');
    expect(api.lastClosingCash, 5000);
    expect(find.byType(SessionReportScreen), findsOneWidget);
  });
}
