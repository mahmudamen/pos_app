import 'printer_transport.dart';
import 'printers.dart';

/// A browser cannot open a raw TCP socket, so ESC/POS bytes never reach a
/// receipt printer from the web build. The transport reports itself offline
/// and [PrinterException]s on send, which the UI already surfaces as a print
/// failure; receipts are still generated and previewed on screen.
class UnavailablePrinterTransport implements PrinterTransport {
  const UnavailablePrinterTransport();

  @override
  Future<bool> reachable(PrinterDevice device, {Duration timeout = const Duration(seconds: 3)}) async =>
      false;

  @override
  Future<void> send(
    PrinterDevice device,
    List<int> bytes, {
    Duration timeout = const Duration(seconds: 8),
  }) async {
    throw const PrinterException(
      'receipt printing needs the Android app: the browser cannot open a raw '
      'socket to an ESC/POS printer',
    );
  }
}

PrinterTransport createDefaultPrinterTransport() => const UnavailablePrinterTransport();

/// No raw sockets in the browser, so there is nothing to sweep. Printers are
/// configured by hand (IP/port) instead, as on Android.
Future<List<PrinterDevice>> discoverNetworkPrinters() async => const [];

/// The browser has no access to the host's interfaces.
Future<String?> localIpv4Subnet() async => null;