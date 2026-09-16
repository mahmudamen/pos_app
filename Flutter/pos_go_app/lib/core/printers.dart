/// Thermal-printer domain models for POS receipt printing.
///
/// Printing is deliberately printer-agnostic: the backend renders the ESC/POS
/// byte stream (`GET /v1/sales/:id/receipt/print`), and this app only has to
/// find a printer, check it is online, and push the bytes to it. Configuration
/// is a per-device local concern (which printer, how many copies...) persisted
/// through [PrinterConfig], not a server-side tenant setting.
library;

/// How a receipt printer is connected. Network (TCP :9100) is the only
/// transport implemented today; USB is reserved for hot-pluggable adapters.
enum PrinterKind { network, usb }

/// Receipt paper width. PAPER_80 is the default 32-column width; PAPER_58 is
/// the narrow 24-column bezel used on small handheld thermal printers.
enum PaperSize {
  mm58(24),
  mm80(32);

  const PaperSize(this.cols);

  /// ESC/POS column width for this paper size.
  final int cols;

  /// Wire code understood by the backend `?cols=` query parameter.
  String get wire => '$cols';

  static PaperSize fromWire(String? value) =>
      value == '24' ? PaperSize.mm58 : PaperSize.mm80;
}

/// Online/offline state of the whole printer setup, used for the POS app-bar
/// indicator and the settings troubleshooting page.
enum PrinterStatus { noPrinter, online, offline }

/// Receipt template. Standard is the printer's readable layout; compact trims
/// whitespace for narrow paper (future themes can be added without breaking
/// the wire format, which is backend-rendered anyway).
enum ReceiptTheme { standard, compact }

/// A discovered or manually-added printer on the local network.
class PrinterDevice {
  const PrinterDevice({
    required this.name,
    required this.ip,
    this.port = 9100,
    this.kind = PrinterKind.network,
  });

  factory PrinterDevice.fromJson(Map<String, dynamic> json) => PrinterDevice(
        name: json['name'] as String? ?? json['ip'] as String? ?? 'Printer',
        ip: json['ip'] as String? ?? '',
        port: (json['port'] as num?)?.toInt() ?? 9100,
        kind: json['kind'] == 'usb' ? PrinterKind.usb : PrinterKind.network,
      );

  final String name;
  final String ip;
  final int port;
  final PrinterKind kind;

  Map<String, dynamic> toJson() => {
        'name': name,
        'ip': ip,
        'port': port,
        'kind': kind == PrinterKind.usb ? 'usb' : 'network',
      };

  String get address => '$ip:$port';

  @override
  bool operator ==(Object other) =>
      other is PrinterDevice && other.ip == ip && other.port == port;

  @override
  int get hashCode => Object.hash(ip, port);
}

/// Per-device, admin-editable printing configuration.
class PrinterConfig {
  const PrinterConfig({
    this.enabled = false,
    this.autoScan = true,
    this.devices = const [],
    this.defaultIp = '',
    this.paper = PaperSize.mm80,
    this.theme = ReceiptTheme.standard,
    this.copies = 1,
    this.cut = true,
    this.cashiersPrint = false,
  });

  factory PrinterConfig.fromJson(Map<String, dynamic> json) => PrinterConfig(
        enabled: json['enabled'] as bool? ?? false,
        autoScan: json['auto_scan'] as bool? ?? true,
        devices: (json['devices'] as List<dynamic>? ?? [])
            .map((e) => PrinterDevice.fromJson(e as Map<String, dynamic>))
            .toList(),
        defaultIp: json['default_ip'] as String? ?? '',
        paper: PaperSize.fromWire((json['paper'] as String?)),
        theme: (json['theme'] as String?) == 'compact'
            ? ReceiptTheme.compact
            : ReceiptTheme.standard,
        copies: (json['copies'] as num?)?.toInt() ?? 1,
        cut: json['cut'] as bool? ?? true,
        cashiersPrint: json['cashiersPrint'] as bool? ?? false,
      );

  /// Whether receipt printing is turned on for this device.
  final bool enabled;

  /// Whether the app auto-scans the LAN for printers on launch.
  final bool autoScan;

  /// All printers known to this device (discovered or manually added).
  final List<PrinterDevice> devices;

  /// IP of the default printer; empty means 'ask the cashier / no default'.
  final String defaultIp;

  /// Paper width used when requesting the backend ESC/POS layout.
  final PaperSize paper;

  /// Visual template for the printed receipt.
  final ReceiptTheme theme;

  /// Number of copies per receipt (printed by re-sending the same bytes).
  final int copies;

  /// Whether to emit a full paper-cut command after the receipt.
  final bool cut;

  /// Managers/owners may also grant cashiers the right to print.
  final bool cashiersPrint;

  bool get hasDefault => defaultIp.isNotEmpty && devices.any((d) => d.ip == defaultIp);

  PrinterDevice? get defaultDevice {
    for (final d in devices) {
      if (d.ip == defaultIp) return d;
    }
    return null;
  }

  PrinterConfig copyWith({
    bool? enabled,
    bool? autoScan,
    List<PrinterDevice>? devices,
    String? defaultIp,
    PaperSize? paper,
    ReceiptTheme? theme,
    int? copies,
    bool? cut,
    bool? cashiersPrint,
  }) =>
      PrinterConfig(
        enabled: enabled ?? this.enabled,
        autoScan: autoScan ?? this.autoScan,
        devices: devices ?? this.devices,
        defaultIp: defaultIp ?? this.defaultIp,
        paper: paper ?? this.paper,
        theme: theme ?? this.theme,
        copies: copies ?? this.copies,
        cut: cut ?? this.cut,
        cashiersPrint: cashiersPrint ?? this.cashiersPrint,
      );

  Map<String, dynamic> toJson() => {
        'enabled': enabled,
        'auto_scan': autoScan,
        'devices': devices.map((d) => d.toJson()).toList(),
        'default_ip': defaultIp,
        'paper': paper.wire,
        'theme': theme == ReceiptTheme.compact ? 'compact' : 'standard',
        'copies': copies,
        'cut': cut,
        'cashiersPrint': cashiersPrint,
      };
}