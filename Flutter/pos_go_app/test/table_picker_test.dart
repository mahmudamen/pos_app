import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/restaurants.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/restaurants/table_picker_sheet.dart';
import 'package:pos_go_app/l10n/strings.dart';

const _manager = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-1',
  displayName: 'Restaurant Admin',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'manager',
  currencyCode: 'EGP',
);

const _cashier = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-2',
  displayName: 'Cashier One',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'cashier',
  currencyCode: 'EGP',
);

/// One occupied and one free table; freeing the occupied one flips its status
/// so the reload after the PATCH shows the new state.
class _PickerApi extends ApiClient {
  String occupiedStatus = 'occupied';
  final List<String> updates = [];

  @override
  Future<List<RestaurantTable>> tables(Session session) async => [
        RestaurantTable(
          id: 'table-1',
          floorId: 'floor-1',
          name: 'T1',
          seats: 4,
          status: occupiedStatus,
          floorName: 'Main',
        ),
        const RestaurantTable(
          id: 'table-2',
          floorId: 'floor-1',
          name: 'T2',
          seats: 2,
          status: 'free',
          floorName: 'Main',
        ),
      ];

  @override
  Future<void> updateTable(
    Session session,
    String tableId, {
    String? status,
    int? seats,
    String? name,
  }) async {
    updates.add('$tableId:${status ?? ''}');
    if (status != null) occupiedStatus = status;
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
    // The sheet lives inside a modal bottom sheet in the app, which supplies
    // the Material ancestor ListTiles need.
    home: Scaffold(body: child),
  );
}

void main() {
  testWidgets('a manager can free an occupied table from the picker',
      (tester) async {
    final api = _PickerApi();

    await tester.pumpWidget(_wrap(TablePickerSheet(
      apiClient: api,
      session: _manager,
      currentTableId: null,
    )));
    await tester.pumpAndSettle();

    expect(find.text('Occupied'), findsOneWidget);
    expect(find.byIcon(Icons.lock_open), findsOneWidget);

    await tester.tap(find.byIcon(Icons.lock_open));
    await tester.pumpAndSettle();

    expect(api.updates, ['table-1:free']);
    expect(find.text('Table freed'), findsOneWidget);
    expect(find.text('Occupied'), findsNothing,
        reason: 'the reload shows the table as free again');
    expect(find.text('Free'), findsWidgets);
  });

  testWidgets('a cashier gets no free action', (tester) async {
    final api = _PickerApi();

    await tester.pumpWidget(_wrap(TablePickerSheet(
      apiClient: api,
      session: _cashier,
      currentTableId: null,
    )));
    await tester.pumpAndSettle();

    expect(find.text('Occupied'), findsOneWidget);
    expect(find.byIcon(Icons.lock_open), findsNothing);
  });
}
