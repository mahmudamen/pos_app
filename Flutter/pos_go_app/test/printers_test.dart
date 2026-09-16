import 'dart:typed_data';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/printer_service.dart';
import 'package:pos_go_app/core/printers.dart';
import 'package:pos_go_app/core/session_store.dart';

class _FakeTransport implements PrinterTransport {
  _FakeTransport({this.reachableHosts = const {}});
  final Set<String> reachableHosts;
  final List<List<int>> sent = [];

  @override
  Future<bool> reachable(PrinterDevice device, {Duration timeout = const Duration(seconds: 3)}) async {
    return reachableHosts.contains(device.ip);
  }

  @override
  Future<void> send(
    PrinterDevice device,
    List<int> bytes, {
    Duration timeout = const Duration(seconds: 8),
  }) async {
    sent.add(bytes);
  }
}

/// Serves ESC/POS bytes for the receipt/print route and records the request so
/// tests can inspect the cols/cut/compact knobs.
class _BytesClient extends http.BaseClient {
  _BytesClient({this.onRequest});
  final void Function(http.BaseRequest request)? onRequest;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    onRequest?.call(request);
    return http.StreamedResponse(
      Stream.value(const [7]),
      200,
      headers: {'content-type': 'application/vnd.escpos'},
    );
  }
}

ApiClient _printApi({void Function(http.BaseRequest)? onRequest}) {
  return ApiClient(client: _BytesClient(onRequest: onRequest));
}

const _session = Session(
  accessToken: 't',
  refreshToken: 'r',
  userId: 'u',
  displayName: 'Owner',
  tenantId: 'ten',
  role: 'owner',
);

void main() {
  group('PrinterConfig', () {
    test('round-trips through JSON', () {
      const config = PrinterConfig(
        enabled: true,
        autoScan: false,
        devices: [PrinterDevice(name: 'EPSON', ip: '192.168.1.50', port: 9100)],
        defaultIp: '192.168.1.50',
        paper: PaperSize.mm58,
        theme: ReceiptTheme.compact,
        copies: 2,
        cut: false,
        cashiersPrint: true,
      );
      final restored = PrinterConfig.fromJson(config.toJson());
      expect(restored.enabled, true);
      expect(restored.autoScan, false);
      expect(restored.devices, hasLength(1));
      expect(restored.devices.first.ip, '192.168.1.50');
      expect(restored.defaultDevice?.address, '192.168.1.50:9100');
      expect(restored.paper, PaperSize.mm58);
      expect(restored.paper.cols, 24);
      expect(restored.theme, ReceiptTheme.compact);
      expect(restored.copies, 2);
      expect(restored.cut, false);
      expect(restored.cashiersPrint, true);
      expect(restored.hasDefault, true);
    });

    test('missing default resolves to null', () {
      const config = PrinterConfig(enabled: true, devices: [
        PrinterDevice(name: 'a', ip: '192.168.1.9'),
      ]);
      expect(config.hasDefault, false);
      expect(config.defaultDevice, isNull);
    });

    test('defaults are disabled 80mm single copy', () {
      const config = PrinterConfig();
      expect(config.enabled, false);
      expect(config.defaultDevice, isNull);
      expect(config.paper, PaperSize.mm80);
      expect(config.copies, 1);
      expect(config.cut, true);
    });
  });

  group('SessionStore printer config', () {
    test('round-trips through secure storage', () async {
      FlutterSecureStorage.setMockInitialValues({});
      const config = PrinterConfig(
        enabled: true,
        devices: [PrinterDevice(name: 'EPSON', ip: '192.168.1.50')],
        defaultIp: '192.168.1.50',
        paper: PaperSize.mm58,
        theme: ReceiptTheme.compact,
        copies: 2,
        cut: false,
        cashiersPrint: true,
      );
      final store = SessionStore();
      expect(await store.readPrinterConfig(), isNull);
      await store.savePrinterConfig(config);
      final restored = await store.readPrinterConfig();
      expect(restored, isNotNull);
      expect(restored!.enabled, true);
      expect(restored.defaultDevice?.address, '192.168.1.50:9100');
      expect(restored.paper, PaperSize.mm58);
      expect(restored.theme, ReceiptTheme.compact);
      expect(restored.copies, 2);
      expect(restored.cashiersPrint, true);
    });
  });

  group('PaperSize', () {
    test('wire code maps to columns', () {
      expect(PaperSize.fromWire('24'), PaperSize.mm58);
      expect(PaperSize.fromWire('32'), PaperSize.mm80);
      expect(PaperSize.fromWire(null), PaperSize.mm80);
      expect(PaperSize.mm80.wire, '32');
      expect(PaperSize.mm58.wire, '24');
    });
  });

  group('buildTestPrint', () {
    test('embeds a centered label and optional cut', () {
      final withCut = buildTestPrint(cols: 32, cut: true);
      expect(withCut.take(2), [0x1B, 0x40]);
      expect(String.fromCharCodes(withCut), contains('TEST PRINT'));
      expect(withCut.sublist(withCut.length - 3), [0x1D, 0x56, 0x00]);
    });

    test('omits cut when disabled', () {
      final noCut = buildTestPrint(cols: 24, cut: false);
      expect(noCut.sublist(noCut.length - 3), isNot([0x1D, 0x56, 0x00]));
    });
  });

  group('PrinterService.statusOf', () {
    test('noPrinter when disabled or without default', () async {
      final service = PrinterService(transport: _FakeTransport(reachableHosts: {'192.168.1.50'}));
      const config = PrinterConfig();
      expect(await service.statusOf(config), PrinterStatus.noPrinter);
      const noDefault = PrinterConfig(enabled: true);
      expect(await service.statusOf(noDefault), PrinterStatus.noPrinter);
    });

    test('online when default reachable, offline otherwise', () async {
      final service = PrinterService(transport: _FakeTransport(reachableHosts: {'192.168.1.50'}));
      const online = PrinterConfig(
        enabled: true,
        devices: [PrinterDevice(name: 'p', ip: '192.168.1.50')],
        defaultIp: '192.168.1.50',
      );
      expect(await service.statusOf(online), PrinterStatus.online);
      const offline = PrinterConfig(
        enabled: true,
        devices: [PrinterDevice(name: 'p', ip: '192.168.1.99')],
        defaultIp: '192.168.1.99',
      );
      expect(await service.statusOf(offline), PrinterStatus.offline);
    });
  });

  group('PrinterService.printReceipt', () {
    test('single copy passes backend bytes straight through', () async {
      final transport = _FakeTransport();
      final service = PrinterService(transport: transport);
      const device = PrinterDevice(name: 'p', ip: '192.168.1.50');
      await service.printReceipt(
        device: device,
        cols: 32,
        copies: 1,
        cut: true,
        fetchBytes: () async => Uint8List.fromList([1, 2, 3]),
      );
      expect(transport.sent, hasLength(1));
      expect(transport.sent.single, [1, 2, 3]);
    });

    test('multi-copy appends a single cut after the last copy', () async {
      final transport = _FakeTransport();
      final service = PrinterService(transport: transport);
      await service.printReceipt(
        device: const PrinterDevice(name: 'p', ip: '192.168.1.50'),
        cols: 24,
        copies: 3,
        cut: true,
        fetchBytes: () async => Uint8List.fromList([9]),
      );
      expect(transport.sent, hasLength(3));
      expect(transport.sent[0], [9]);
      expect(transport.sent[1], [9]);
      expect(transport.sent[2], [9, ...escposCut]);
    });

    test('multi-copy without cut never appends one', () async {
      final transport = _FakeTransport();
      final service = PrinterService(transport: transport);
      await service.printReceipt(
        device: const PrinterDevice(name: 'p', ip: '192.168.1.50'),
        cols: 32,
        copies: 2,
        cut: false,
        fetchBytes: () async => Uint8List.fromList([9]),
      );
      expect(transport.sent, hasLength(2));
      expect(transport.sent[0], [9]);
      expect(transport.sent[1], [9]);
    });

    test('rejects an empty device', () async {
      final service = PrinterService(transport: _FakeTransport());
      await expectLater(
        service.printReceipt(
          device: const PrinterDevice(name: 'p', ip: ''),
          cols: 32,
          copies: 1,
          cut: true,
          fetchBytes: () async => Uint8List.fromList([1]),
        ),
        throwsA(isA<PrinterException>()),
      );
    });
  });

  group('printSaleReceipt gating', () {
    PrinterConfig enabledConfig({bool cashiersPrint = false, int copies = 1}) =>
        PrinterConfig(
          enabled: true,
          devices: const [PrinterDevice(name: 'p', ip: '192.168.1.50')],
          defaultIp: '192.168.1.50',
          paper: PaperSize.mm58,
          theme: ReceiptTheme.compact,
          copies: copies,
          cut: true,
          cashiersPrint: cashiersPrint,
        );

    test('manager always prints when enabled', () async {
      final transport = _FakeTransport();
      final service = PrinterService(transport: transport);
      final outcome = await printSaleReceipt(
        config: enabledConfig(),
        service: service,
        apiClient: _printApi(),
        session: _session,
        saleId: 'sale-1',
        roleAllowsPrint: true,
      );
      expect(outcome, PrintOutcome.sent);
      expect(transport.sent, hasLength(1));
      expect(transport.sent.single, [7]);
    });

    test('single-copy compact job asks for backend cut+compact', () async {
      http.BaseRequest? seen;
      final transport = _FakeTransport();
      final service = PrinterService(transport: transport);
      await printSaleReceipt(
        config: enabledConfig(),
        service: service,
        apiClient: _printApi(onRequest: (r) => seen = r),
        session: _session,
        saleId: 'sale-1',
        roleAllowsPrint: true,
      );
      expect(seen!.url.path, '/v1/sales/sale-1/receipt/print');
      expect(seen!.url.queryParameters['cols'], '24');
      expect(seen!.url.queryParameters['cut'], '1');
      expect(seen!.url.queryParameters['compact'], '1');
    });

    test('cashier without grant is denied', () async {
      final transport = _FakeTransport();
      final service = PrinterService(transport: transport);
      final outcome = await printSaleReceipt(
        config: enabledConfig(cashiersPrint: false),
        service: service,
        apiClient: _printApi(),
        session: _session,
        saleId: 'sale-1',
        roleAllowsPrint: false,
      );
      expect(outcome, PrintOutcome.denied);
      expect(transport.sent, isEmpty);
    });

    test('cashier with grant prints', () async {
      final transport = _FakeTransport();
      final service = PrinterService(transport: transport);
      final outcome = await printSaleReceipt(
        config: enabledConfig(cashiersPrint: true),
        service: service,
        apiClient: _printApi(),
        session: _session,
        saleId: 'sale-1',
        roleAllowsPrint: false,
      );
      expect(outcome, PrintOutcome.sent);
      expect(transport.sent, hasLength(1));
    });

    test('multi-copy disables backend cut and appends one client cut',
        () async {
      http.BaseRequest? seen;
      final transport = _FakeTransport();
      final service = PrinterService(transport: transport);
      final outcome = await printSaleReceipt(
        config: enabledConfig(copies: 2),
        service: service,
        apiClient: _printApi(onRequest: (r) => seen = r),
        session: _session,
        saleId: 'sale-1',
        roleAllowsPrint: true,
      );
      expect(outcome, PrintOutcome.sent);
      expect(seen!.url.queryParameters['cut'], '0');
      expect(transport.sent, hasLength(2));
      expect(transport.sent[0], [7]);
      expect(transport.sent[1], [7, ...escposCut]);
    });

    test('disabled printer reports notConfigured', () async {
      final service = PrinterService(transport: _FakeTransport());
      final outcome = await printSaleReceipt(
        config: const PrinterConfig(),
        service: service,
        apiClient: _printApi(),
        session: _session,
        saleId: 'sale-1',
        roleAllowsPrint: true,
      );
      expect(outcome, PrintOutcome.notConfigured);
    });
  });

  group('PrintGate', () {
    test('manager gate opens automatically', () {
      expect(gateFor(config: const PrinterConfig(enabled: true), roleAllowsPrint: true).mayPrint, true);
      expect(gateFor(config: const PrinterConfig(enabled: true), roleAllowsPrint: false).mayPrint, false);
      expect(gateFor(config: const PrinterConfig(enabled: true, cashiersPrint: true), roleAllowsPrint: false).mayPrint, true);
      expect(gateFor(config: const PrinterConfig(), roleAllowsPrint: true).mayPrint, false);
    });
  });
}