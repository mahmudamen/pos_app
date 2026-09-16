import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/printer_service.dart';
import 'package:pos_go_app/core/printers.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/pos/pos_screen.dart';
import 'package:pos_go_app/features/printers/printers_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

/// Instant transport so widget tests never touch real sockets.
class _FakeTransport implements PrinterTransport {
  const _FakeTransport({this.isReachable = false});
  final bool isReachable;

  @override
  Future<bool> reachable(PrinterDevice device,
      {Duration timeout = const Duration(seconds: 3)}) async {
    return isReachable;
  }

  @override
  Future<void> send(
    PrinterDevice device,
    List<int> bytes, {
    Duration timeout = const Duration(seconds: 8),
  }) async {}
}

class _StubPrinterService extends PrinterService {
  _StubPrinterService({this.status = PrinterStatus.noPrinter})
      : super(transport: const _FakeTransport(isReachable: true));
  PrinterStatus status;
  List<PrinterDevice> scanned = const [
    PrinterDevice(name: 'EPSON 10.0.0.42', ip: '10.0.0.42'),
  ];
  int printReceiptCalls = 0;

  @override
  Future<PrinterStatus> statusOf(PrinterConfig config) async => status;

  @override
  Future<List<PrinterDevice>> scanNetwork() async => scanned;

  @override
  Future<void> printReceipt({
    required PrinterDevice device,
    required int cols,
    required int copies,
    required bool cut,
    required Future<Uint8List> Function() fetchBytes,
  }) async {
    printReceiptCalls++;
    await fetchBytes();
  }
}

const _owner = Session(
  accessToken: 't',
  refreshToken: 'r',
  userId: 'u',
  displayName: 'Owner',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'owner',
  currencyCode: 'EGP',
);

const _cashier = Session(
  accessToken: 't',
  refreshToken: 'r',
  userId: 'u2',
  displayName: 'Cashier',
  tenantId: 'tenant-1',
  deviceId: 'device-1',
  role: 'cashier',
  currencyCode: 'EGP',
);

const _category = Category(id: 'category-1', name: 'Drinks', slug: 'drinks');

class _MinimalPosApi extends ApiClient {
  @override
  Future<List<Product>> products(Session session, {String search = ''}) async =>
      const [];

  @override
  Future<List<Category>> categories(Session session) async => const [_category];

  @override
  Future<RegisterSession?> currentSession(Session session) async => null;

  @override
  Future<TenantSettings> settings(Session session) async =>
      const TenantSettings();
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

PrinterConfig _enabledWithDefault() => const PrinterConfig(
      enabled: true,
      devices: [PrinterDevice(name: 'EPSON', ip: '10.0.0.42')],
      defaultIp: '10.0.0.42',
      copies: 1,
    );

void main() {
  setUp(() {
    FlutterSecureStorage.setMockInitialValues(<String, String>{});
  });

  testWidgets('fresh config: owner sees the enable switch, cashiers cannot',
      (tester) async {
    await tester.pumpWidget(_wrap(PrintersScreen(
      session: _owner,
      sessionStore: SessionStore(),
      printerService: _StubPrinterService(),
    )));
    await tester.pumpAndSettle();
    expect(find.text('Printing enabled'), findsOneWidget);
    expect(find.byType(Switch), findsOneWidget);

    await tester.pumpWidget(_wrap(PrintersScreen(
      session: _cashier,
      sessionStore: SessionStore(),
      printerService: _StubPrinterService(),
    )));
    await tester.pumpAndSettle();
    final switchWidget = tester.widget<Switch>(find.byType(Switch));
    expect(switchWidget.onChanged, isNull);
  });

  void tallViewport(WidgetTester tester) {
    tester.view.physicalSize = const Size(1200, 2200);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
  }

  testWidgets('owner enables, scans and adopts a found printer',
      (tester) async {
    tallViewport(tester);
    await tester.pumpWidget(_wrap(PrintersScreen(
      session: _owner,
      sessionStore: SessionStore(),
      printerService: _StubPrinterService(),
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Printing enabled'));
    await tester.pumpAndSettle();
    expect(find.text('No printers configured yet'), findsOneWidget);

    await tester.tap(find.text('Scan printers'));
    await tester.pumpAndSettle();
    expect(find.text('10.0.0.42'), findsOneWidget);

    await tester.tap(find.text('Add found'));
    await tester.pumpAndSettle();
    expect(find.text('10.0.0.42:9100'), findsWidgets);
    expect(find.text('Default'), findsOneWidget);

    final saved = await SessionStore().readPrinterConfig();
    expect(saved, isNotNull);
    expect(saved!.hasDefault, true);
  });

  testWidgets('test print without a default printer shows the hint',
      (tester) async {
    tallViewport(tester);
    FlutterSecureStorage.setMockInitialValues({
      'printer_config': jsonEncode(const PrinterConfig(enabled: true).toJson()),
    });
    await tester.pumpWidget(_wrap(PrintersScreen(
      session: _owner,
      sessionStore: SessionStore(),
      printerService: _StubPrinterService(),
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Test print'));
    await tester.pumpAndSettle();
    expect(find.text('No default printer configured'), findsWidgets);
  });

  testWidgets('test print reaches a configured printer', (tester) async {
    tallViewport(tester);
    FlutterSecureStorage.setMockInitialValues({
      'printer_config': jsonEncode(_enabledWithDefault().toJson()),
    });
    final service = _StubPrinterService();
    await tester.pumpWidget(_wrap(PrintersScreen(
      session: _owner,
      sessionStore: SessionStore(),
      printerService: service,
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Test print'));
    await tester.pumpAndSettle();
    expect(service.printReceiptCalls, 1);
    expect(find.text('Test page sent'), findsOneWidget);
  });

  testWidgets('POS app bar reflects an offline default printer',
      (tester) async {
    FlutterSecureStorage.setMockInitialValues({
      'printer_config': jsonEncode(_enabledWithDefault().toJson()),
    });
    await tester.pumpWidget(_wrap(PosScreen(
      session: _owner,
      apiClient: _MinimalPosApi(),
      onSignOut: () {},
      sessionStore: SessionStore(),
      printerService: _StubPrinterService(status: PrinterStatus.offline),
    )));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.print_disabled), findsOneWidget);

    await tester.tap(find.byIcon(Icons.print_disabled));
    await tester.pumpAndSettle();
    expect(find.text('Printers'), findsOneWidget);
    expect(find.textContaining('Offline · 10.0.0.42'), findsOneWidget);
  });
}