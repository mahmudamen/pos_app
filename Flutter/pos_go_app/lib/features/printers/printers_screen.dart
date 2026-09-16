import 'package:flutter/material.dart';

import '../../core/printer_service.dart';
import '../../core/printers.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

/// Per-device printer configuration: what to print, to which printer, in what
/// layout. Editing is owner/manager-gated like the POS settings; cashiers only
/// see the live status and the troubleshooting test page.
class PrintersScreen extends StatefulWidget {
  const PrintersScreen({
    super.key,
    required this.sessionStore,
    required this.session,
    this.printerService,
  });

  final SessionStore sessionStore;
  final Session session;
  final PrinterService? printerService;

  @override
  State<PrintersScreen> createState() => _PrintersScreenState();
}

class _PrintersScreenState extends State<PrintersScreen> {
  late final PrinterService _service =
      widget.printerService ?? PrinterService();
  PrinterConfig? _config;
  PrinterStatus? _status;
  bool _loading = true;
  bool _scanning = false;
  List<PrinterDevice> _scanResults = const [];
  String? _addError;

  bool get _canEdit => widget.session.canManageSettings;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
    });
    final config = await widget.sessionStore.readPrinterConfig();
    if (!mounted) return;
    setState(() {
      _config = config ?? const PrinterConfig();
      _loading = false;
    });
    await _refreshStatus();
    if (_config?.enabled ?? false) {
      await _autoScanIfNeeded();
    }
  }

  Future<void> _refreshStatus() async {
    final config = _config;
    if (config == null) return;
    final status = await _service.statusOf(config);
    if (!mounted) return;
    setState(() => _status = status);
  }

  Future<void> _autoScanIfNeeded() async {
    if (!(_config?.autoScan ?? false)) return;
    if (_config?.devices.isNotEmpty ?? false) return;
    await _scan();
  }

  Future<void> _persist(PrinterConfig next) async {
    await widget.sessionStore.savePrinterConfig(next);
    if (mounted) setState(() => _config = next);
  }

  Future<void> _scan() async {
    setState(() {
      _scanning = true;
    });
    final found = await _service.scanNetwork();
    if (!mounted) return;
    // Merge any already-known printers (manual entries) in front.
    final known = _config?.devices.where((d) => d.kind == PrinterKind.network).toList() ?? [];
    for (final d in known) {
      if (!found.any((f) => f.ip == d.ip)) found.insert(0, d);
    }
    setState(() {
      _scanResults = found;
      _scanning = false;
    });
  }

  Future<void> _applyScanResults() async {
    final config = _config;
    if (config == null || _scanResults.isEmpty) return;
    var devices = [...config.devices];
    for (final d in _scanResults) {
      if (!devices.any((e) => e.ip == d.ip)) devices.add(d);
    }
    var next = config.copyWith(devices: devices, enabled: true);
    if (next.defaultIp.isEmpty && devices.isNotEmpty) {
      next = next.copyWith(defaultIp: devices.first.ip);
    }
    await _persist(next);
  }

  Future<void> _removeDevice(PrinterDevice device) async {
    final config = _config;
    if (config == null) return;
    final devices = config.devices.where((d) => d.ip != device.ip).toList();
    var next = config.copyWith(
      devices: devices,
      defaultIp: config.defaultIp == device.ip ? '' : config.defaultIp,
    );
    if (next.hasDefault == false) {
      next = next.copyWith(defaultIp: devices.isNotEmpty ? devices.first.ip : '');
    }
    await _persist(next);
  }

  Future<void> _testPrint(AppStrings s) async {
    final messenger = ScaffoldMessenger.of(context);
    final config = _config;
    final device = config?.defaultDevice;
    if (config == null || !config.enabled || device == null) {
      messenger.showSnackBar(SnackBar(content: Text(s.noPrinterConfigured)));
      return;
    }
    try {
      await _service.printReceipt(
        device: device,
        cols: config.paper.cols,
        copies: 1,
        cut: config.cut,
        fetchBytes: () async => buildTestPrint(
          cols: config.paper.cols,
          cut: config.cut,
        ),
      );
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(s.testPrintSent)));
    } catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(
          content: Text('${s.printFailed}: ${e.toString()}')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final config = _config;
    final device = config?.defaultDevice;
    return Scaffold(
      appBar: AppBar(title: Text(s.printers)),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : config == null
              ? const SizedBox()
              : ListView(
                  padding: const EdgeInsets.all(16),
                  children: [
                    _statusCard(s, config, device),
                    const SizedBox(height: 12),
                    _sectionTitle(s, s.printersSettings),
                    Card(
                      child: Column(
                        children: [
                          SwitchListTile(
                            title: Text(s.printingEnabled),
                            subtitle: Text(s.printingEnabledHint),
                            value: config.enabled,
                            onChanged: _canEdit
                                ? (v) => _persist(config.copyWith(enabled: v))
                                : null,
                          ),
                          if (config.enabled) ...[
                            const Divider(height: 1),
                            SwitchListTile(
                              title: Text(s.autoScan),
                              subtitle: Text(s.autoScanHint),
                              value: config.autoScan,
                              onChanged: _canEdit
                                  ? (v) => _persist(
                                      config.copyWith(autoScan: v))
                                  : null,
                            ),
                            const Divider(height: 1),
                            ListTile(
                              leading: const Icon(Icons.tune),
                              title: Text(s.paperSize),
                              subtitle: Text(s.paperWidthHint),
                              trailing: DropdownButton<PaperSize>(
                                value: config.paper,
                                onChanged: _canEdit
                                    ? (v) => _persist(config.copyWith(
                                        paper: v ?? config.paper))
                                    : null,
                                items: [
                                  DropdownMenuItem(
                                    value: PaperSize.mm80,
                                    child: Text(s.mm80),
                                  ),
                                  DropdownMenuItem(
                                    value: PaperSize.mm58,
                                    child: Text(s.mm58),
                                  ),
                                ],
                              ),
                            ),
                            const Divider(height: 1),
                            ListTile(
                              leading: const Icon(Icons.palette_outlined),
                              title: Text(s.receiptTheme),
                              trailing: DropdownButton<ReceiptTheme>(
                                value: config.theme,
                                onChanged: _canEdit
                                    ? (v) => _persist(config.copyWith(
                                        theme: v ?? config.theme))
                                    : null,
                                items: [
                                  DropdownMenuItem(
                                    value: ReceiptTheme.standard,
                                    child: Text(s.themeStandard),
                                  ),
                                  DropdownMenuItem(
                                    value: ReceiptTheme.compact,
                                    child: Text(s.themeCompact),
                                  ),
                                ],
                              ),
                            ),
                            const Divider(height: 1),
                            ListTile(
                              leading: const Icon(Icons.content_copy),
                              title: Text(s.copies),
                              trailing: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  IconButton(
                                    onPressed: _canEdit &&
                                            config.copies > 1
                                        ? () => _persist(config.copyWith(
                                            copies: config.copies - 1))
                                        : null,
                                    icon: const Icon(Icons.remove_circle_outline),
                                  ),
                                  Text('${config.copies}'),
                                  IconButton(
                                    onPressed: _canEdit &&
                                            config.copies < 3
                                        ? () => _persist(config.copyWith(
                                            copies: config.copies + 1))
                                        : null,
                                    icon: const Icon(Icons.add_circle_outline),
                                  ),
                                ],
                              ),
                            ),
                            const Divider(height: 1),
                            SwitchListTile(
                              title: Text(s.cutAfterPrint),
                              value: config.cut,
                              onChanged: _canEdit
                                  ? (v) => _persist(config.copyWith(cut: v))
                                  : null,
                            ),
                            const Divider(height: 1),
                            SwitchListTile(
                              title: Text(s.cashierPrintAccess),
                              subtitle: Text(s.cashierPrintAccessHint),
                              value: config.cashiersPrint,
                              onChanged: _canEdit
                                  ? (v) => _persist(config.copyWith(
                                      cashiersPrint: v))
                                  : null,
                            ),
                          ],
                        ],
                      ),
                    ),
                    if (config.enabled) ...[
                      const SizedBox(height: 12),
                      _sectionTitle(s, s.devices),
                      Card(
                        child: Column(
                          children: [
                            for (var i = 0; i < config.devices.length; i++)
                              _deviceTile(config, config.devices[i]),
                            if (config.devices.isEmpty)
                              Padding(
                                padding: const EdgeInsets.all(16),
                                child: Text(s.noPrintersFound),
                              ),
                            const Divider(height: 1),
                            Padding(
                              padding: const EdgeInsets.all(12),
                              child: Row(
                                children: [
                                  if (_canEdit)
                                    Expanded(
                                      child: FilledButton.tonalIcon(
                                        onPressed: _scanning ? null : _scan,
                                        icon: _scanning
                                            ? const SizedBox(
                                                width: 16,
                                                height: 16,
                                                child: CircularProgressIndicator(
                                                    strokeWidth: 2),
                                              )
                                            : const Icon(Icons.wifi_tethering),
                                        label: Text(_scanning
                                            ? s.scanning
                                            : s.scanPrinters),
                                      ),
                                    ),
                                  if (_canEdit && _scanResults.isNotEmpty) ...[
                                    const SizedBox(width: 8),
                                    TextButton(
                                      onPressed: _applyScanResults,
                                      child: Text(s.addPrinter),
                                    ),
                                  ],
                                ],
                              ),
                            ),
                            if (_canEdit)
                              Padding(
                                padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
                                child: TextField(
                                  decoration: InputDecoration(
                                    labelText: s.manualIp,
                                    hintText: s.manualIpHint,
                                    errorText: _addError,
                                    border: const OutlineInputBorder(),
                                  ),
                                  keyboardType:
                                      const TextInputType.numberWithOptions(decimal: true),
                                  onSubmitted: (value) =>
                                      _addManual(context, value),
                                ),
                              ),
                          ],
                        ),
                      ),
                      if (_scanResults.isNotEmpty && _canEdit)
                        Padding(
                          padding: const EdgeInsets.only(top: 8),
                          child: Card(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                Padding(
                                  padding: const EdgeInsets.all(12),
                                  child: Text(s.devicesFound,
                                      style: Theme.of(context)
                                          .textTheme
                                          .titleSmall),
                                ),
                                for (final d in _scanResults)
                                  ListTile(
                                    dense: true,
                                    leading: const Icon(Icons.print),
                                    title: Text(d.ip),
                                    subtitle: Text(s.onNetwork9100),
                                    trailing: IconButton(
                                      onPressed: () => _applyScanResults(),
                                      icon: const Icon(Icons.add),
                                    ),
                                  ),
                              ],
                            ),
                          ),
                        ),
                      const SizedBox(height: 12),
                      if (!_canEdit)
                        Padding(
                          padding: const EdgeInsets.only(bottom: 8),
                          child: Text(s.settingsReadOnlyHint,
                              style: TextStyle(
                                  color: Theme.of(context)
                                      .colorScheme
                                      .onSurfaceVariant)),
                        ),
                      FilledButton.icon(
                        onPressed: () => _testPrint(s),
                        icon: const Icon(Icons.print_outlined),
                        label: Text(s.testPrint),
                      ),
                      const SizedBox(height: 12),
                      _sectionTitle(s, s.troubleshoot),
                      Card(
                        child: Padding(
                          padding: const EdgeInsets.all(16),
                          child: Text(s.troubleshootHints),
                        ),
                      ),
                    ],
                  ],
                ),
    );
  }

  Widget _statusCard(
      AppStrings s, PrinterConfig config, PrinterDevice? device) {
    final status = _status ?? (_config?.enabled == true
        ? (device == null ? PrinterStatus.noPrinter : PrinterStatus.offline)
        : PrinterStatus.noPrinter);
    final (icon, color, label) = switch (status) {
      PrinterStatus.online => (
          Icons.check_circle,
          Colors.green,
          s.printerOnline
        ),
      PrinterStatus.offline => (
          Icons.error,
          Theme.of(context).colorScheme.error,
          s.printerOffline
        ),
      PrinterStatus.noPrinter => (
          Icons.info_outline,
          Theme.of(context).colorScheme.onSurfaceVariant,
          s.noPrinterConfigured
        ),
    };
    return Card(
      child: ListTile(
        leading: Icon(icon, color: color),
        title: Text(s.printerStatus),
        subtitle: Text(device == null
            ? s.noPrinterConfigured
            : '$label · ${device.address}'),
        trailing: TextButton.icon(
          onPressed: config.enabled ? () => _refreshStatus() : null,
          icon: const Icon(Icons.refresh),
          label: Text(s.refresh),
        ),
      ),
    );
  }

  Widget _deviceTile(PrinterConfig config, PrinterDevice device) {
    final selected = config.defaultIp == device.ip;
    final s = AppStrings.of(context);
    return ListTile(
      leading: Icon(
        selected ? Icons.print : Icons.print_outlined,
        color: selected ? Theme.of(context).colorScheme.primary : null,
      ),
      title: Text(device.name),
      subtitle: Text(device.address),
      trailing: _canEdit
          ? Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                selected
                    ? Chip(
                        label: Text(s.defaultPrinter),
                        visualDensity: VisualDensity.compact,
                      )
                    : TextButton(
                        onPressed: () => _persist(
                            config.copyWith(defaultIp: device.ip)),
                        child: Text(s.setDefault),
                      ),
                IconButton(
                  onPressed: () => _removeDevice(device),
                  icon: const Icon(Icons.delete_outline),
                ),
              ],
            )
          : (selected ? Text(s.defaultPrinter) : const SizedBox()),
    );
  }

  void _addManual(BuildContext context, String raw) {
    final ip = raw.trim();
    if (ip.isEmpty) return;
    if (!RegExp(
            r'^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$')
        .hasMatch(ip)) {
      setState(() => _addError = AppStrings.of(context).invalidIp);
      return;
    }
    final config = _config;
    if (config == null) return;
    setState(() => _addError = null);
    final device = PrinterDevice(name: ip, ip: ip);
    final devices = [...config.devices, device];
    _persist(config.copyWith(
      devices: devices,
      defaultIp: config.defaultIp.isEmpty ? ip : config.defaultIp,
    ));
  }

  Widget _sectionTitle(AppStrings s, String title) {
    return Padding(
      padding: const EdgeInsets.only(top: 16, bottom: 8),
      child: Text(title, style: Theme.of(context).textTheme.titleMedium),
    );
  }
}