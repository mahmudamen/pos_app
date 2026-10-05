import 'printers.dart';

/// Raised when a print job cannot complete (offline, unreachable, timeout).
class PrinterException implements Exception {
  const PrinterException(this.message);
  final String message;

  @override
  String toString() => 'PrinterException: $message';
}

/// Raw transport for poking receipt ESC/POS bytes at a printer. Kept as an
/// abstraction so a USB/Bluetooth transport can be added later without
/// touching the printing pipeline (the backend already renders the byte
/// stream; this app only ships bytes to hardware).
///
/// The TCP implementation lives in `printer_network_io.dart` and is only
/// reachable on platforms with raw sockets; the browser gets an
/// `UnavailablePrinterTransport` so the rest of the printing pipeline still
/// compiles and renders.
abstract class PrinterTransport {
  /// Whether the device accepts a connection (online status probe).
  Future<bool> reachable(PrinterDevice device, {Duration timeout});

  /// Writes [bytes] to the device and flushes them before returning.
  Future<void> send(PrinterDevice device, List<int> bytes, {Duration timeout});
}