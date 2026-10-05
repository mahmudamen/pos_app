import 'dart:typed_data';

import 'api_client.dart';
import 'printer_network.dart';
import 'printer_transport.dart';
import 'printers.dart';
import 'session_store.dart';

export 'printer_transport.dart';

/// ESC/POS full paper cut (`GS V 0`). The backend honours a `?cut=` query so
/// multi-copy jobs avoid cutting between copies; this client appends one cut
/// after the final copy instead.
const List<int> escposCut = [0x1D, 0x56, 0x00];

/// Builds a tiny pure-Dart test page (no backend round-trip) used by the
/// troubleshooting / test-print action. [cols] wraps the centre-justified text;
/// [cut] appends the paper-cut command if configured.
Uint8List buildTestPrint({int cols = 32, bool cut = true}) {
  final b = BytesBuilder();
  b.add([
    0x1B, 0x40, // ESC @ reset
    0x1B, 0x64, 0x01, // ESC d 1 feed
    0x1B, 0x61, 0x01, // ESC a center
    0x1D, 0x21, 0x01, // GS ! double height
  ]);
  String centered(String s) {
    final runes = s.runes.toList();
    if (runes.length >= cols) return s;
    final pad = (cols - runes.length) ~/ 2;
    return (' ' * pad) + s + (' ' * (cols - runes.length - pad));
  }

  b.add(centered('TEST PRINT').codeUnits);
  b.add([0x1B, 0x21, 0x00]); // GS ! off
  b.add('\n\n\n'.codeUnits);
  if (cut) b.add(escposCut);
  return b.takeBytes();
}

/// Discover and print receipts. All networking funnels through the injected
/// [PrinterTransport] so unit tests can inject a fake transport and never touch
/// real hardware.
class PrinterService {
  PrinterService({PrinterTransport? transport})
      : _transport = transport ?? createDefaultPrinterTransport();

  final PrinterTransport _transport;

  /// Resolves the overall printer status shown on the POS app bar:
  /// disabled/no default → [PrinterStatus.noPrinter], otherwise online/offline.
  Future<PrinterStatus> statusOf(PrinterConfig config) async {
    final device = config.defaultDevice;
    if (!config.enabled || device == null) return PrinterStatus.noPrinter;
    final ok = await _transport.reachable(device);
    return ok ? PrinterStatus.online : PrinterStatus.offline;
  }

  /// Scans the local /24 for printers listening on :9100. The sweep runs in
  /// small parallel batches so the UI stays responsive and the network isn't
  /// hammered; any host that accepts a connection is reported as a network
  /// printer. Returns nothing on platforms without raw sockets (the browser).
  Future<List<PrinterDevice>> scanNetwork() => discoverNetworkPrinters();

  /// Prints [copies] copies of the receipt produced by [fetchBytes]. Fetches
  /// the byte stream once (with the backend cut disabled when >1 copy), then
  /// re-sends the same bytes per copy and cuts after the final one per [cut].
  Future<void> printReceipt({
    required PrinterDevice device,
    required int cols,
    required int copies,
    required bool cut,
    required Future<Uint8List> Function() fetchBytes,
  }) async {
    if (!device.ip.isNotEmpty) {
      throw const PrinterException('no printer selected');
    }
    copies = copies < 1 ? 1 : copies;
    final backendCut = copies == 1 && cut;
    final bytes = await fetchBytes();
    for (var i = 0; i < copies; i++) {
      var out = bytes;
      if (cut && i == copies - 1 && !backendCut) {
        out = Uint8List.fromList([...bytes, ...escposCut]);
      }
      await _transport.send(device, out);
    }
  }

}

/// Backend request gating for a print job: printing must be enabled and either
/// the acting role always prints (manager/owner) or the cashier grant is on.
class PrintGate {
  const PrintGate({required this.enabled, required this.allowed});

  final bool enabled;
  final bool allowed;

  bool get mayPrint => enabled && allowed;
}

PrintGate gateFor({required PrinterConfig config, required bool roleAllowsPrint}) {
  return PrintGate(enabled: config.enabled, allowed: config.cashiersPrint || roleAllowsPrint);
}

/// Outcome of a print request, used by UI to decide the snackbar message.
enum PrintOutcome { sent, notConfigured, denied }

/// Fetches and prints a sale's receipt, honouring the per-device config and
/// role gating. Throws [PrinterException] on transport failures.
Future<PrintOutcome> printSaleReceipt({
  required PrinterConfig config,
  required PrinterService service,
  required ApiClient apiClient,
  required Session session,
  required String saleId,
  required bool roleAllowsPrint,
}) async {
  final gate = gateFor(config: config, roleAllowsPrint: roleAllowsPrint);
  if (!gate.enabled || !gate.allowed) {
    return !config.enabled ? PrintOutcome.notConfigured : PrintOutcome.denied;
  }
  final device = config.defaultDevice;
  if (device == null) return PrintOutcome.notConfigured;
  await service.printReceipt(
    device: device,
    cols: config.paper.cols,
    copies: config.copies,
    cut: config.cut,
    fetchBytes: () => apiClient.fetchReceiptPrint(
      session,
      saleId,
      cols: config.paper.cols,
      cut: config.copies == 1 && config.cut,
      compact: config.theme == ReceiptTheme.compact,
    ),
  );
  return PrintOutcome.sent;
}