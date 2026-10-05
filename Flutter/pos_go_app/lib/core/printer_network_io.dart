import 'dart:io';

import 'printer_transport.dart';
import 'printers.dart';

/// Network transport over the classic ESC/POS RAW socket (port 9100).
class SocketPrinterTransport implements PrinterTransport {
  const SocketPrinterTransport();

  @override
  Future<bool> reachable(PrinterDevice device, {Duration timeout = const Duration(seconds: 3)}) async {
    if (device.kind != PrinterKind.network) return false;
    try {
      final socket = await Socket.connect(device.ip, device.port, timeout: timeout);
      socket.destroy();
      return true;
    } catch (_) {
      return false;
    }
  }

  @override
  Future<void> send(
    PrinterDevice device,
    List<int> bytes, {
    Duration timeout = const Duration(seconds: 8),
  }) async {
    if (device.kind != PrinterKind.network) {
      throw const PrinterException('unsupported printer transport');
    }
    Socket? socket;
    try {
      socket = await Socket.connect(device.ip, device.port, timeout: timeout);
      socket.add(bytes);
      await socket.flush();
    } catch (e) {
      throw PrinterException('could not reach ${device.address}: $e');
    } finally {
      socket?.destroy();
    }
  }
}

PrinterTransport createDefaultPrinterTransport() => const SocketPrinterTransport();

/// Sweeps the local /24 for printers listening on :9100. Runs in small
/// parallel batches so the UI stays responsive and the network isn't
/// hammered; any host that accepts a connection is reported as a network
/// printer.
Future<List<PrinterDevice>> discoverNetworkPrinters() async {
  final subnet = await localIpv4Subnet();
  if (subnet == null) return const [];
  const port = 9100;
  const batchSize = 16;
  const connectTimeout = Duration(milliseconds: 350);
  final found = <PrinterDevice>[];
  for (var start = 0; start < 254; start += batchSize) {
    final upper = (start + batchSize).clamp(0, 254);
    final hosts = List.generate(upper - start, (i) => '$subnet.${start + i + 1}');
    await Future.wait(hosts.map((host) async {
      try {
        final socket = await Socket.connect(host, port, timeout: connectTimeout);
        socket.destroy();
        found.add(PrinterDevice(name: host, ip: host, port: port));
      } catch (_) {
        // Unreachable host; keep scanning.
      }
    }));
  }
  return found;
}

/// First non-loopback IPv4 subnet of this device, e.g. `192.168.1` for
/// `192.168.1.42`. Null when the device has no usable IPv4 address.
Future<String?> localIpv4Subnet() async {
  try {
    final interfaces = await NetworkInterface.list(
      type: InternetAddressType.IPv4,
      includeLinkLocal: false,
    );
    for (final iface in interfaces) {
      for (final addr in iface.addresses) {
        if (addr.isLoopback) continue;
        final parts = addr.address.split('.');
        if (parts.length == 4) return '${parts[0]}.${parts[1]}.${parts[2]}';
      }
    }
  } catch (_) {}
  return null;
}