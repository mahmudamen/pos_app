import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import 'sessions_summary_screen.dart';
import '../../core/payments.dart';
import '../../core/registers.dart';
import '../../core/security.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

/// Full Z-report for one register session (open or closed).
class SessionReportScreen extends StatelessWidget {
  const SessionReportScreen({
    super.key,
    required this.sessionData,
    required this.apiClient,
    this.currency = 'EGP',
  });

  final RegisterSession sessionData;
  final ApiClient apiClient;
  final String currency;

  @override
  Widget build(BuildContext context) {
    return _ReportScaffold(sessionData: sessionData, currency: currency);
  }
}

class _ReportScaffold extends StatelessWidget {
  const _ReportScaffold({required this.sessionData, this.currency = 'EGP'});

  final RegisterSession sessionData;
  final String currency;

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final summary = sessionData.summary;
    return Scaffold(
      appBar: AppBar(title: Text(s.zReport)),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          _header(context, s),
          const Divider(height: 24),
          Text(s.subtotalLabel,
              style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 8),
          _moneyRow(context, s, s.subtotalLabel, summary.subtotalMinor),
          _moneyRow(context, s, s.discountLabel, summary.discountMinor,
              muted: true),
          _moneyRow(context, s, s.taxLabel, summary.taxMinor),
          _moneyRow(context, s, s.total, summary.totalMinor,
              emphasized: true),
          _moneyRow(context, s, s.salesCount, summary.salesCount,
              isCount: true),
          const Divider(height: 24),
          Text(s.payment, style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 8),
          _moneyRow(context, s, s.cash, summary.cashMinor),
          _moneyRow(context, s, s.card, summary.cardMinor),
          _moneyRow(context, s, s.mobilePayment, summary.mobileMinor),
          _moneyRow(context, s, s.expectedCash, summary.expectedCashMinor,
              emphasized: true),
          if (!sessionData.isOpen &&
              sessionData.expectedCashMinor != null) ...[
            const SizedBox(height: 8),
            _diffCard(context, s),
          ],
          const SizedBox(height: 16),
        ],
      ),
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Text(
            s.poweredByXamltech,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ),
      ),
    );
  }

  Widget _header(BuildContext context, AppStrings s) {
    final statusColor = sessionData.isOpen
        ? Colors.green.shade700
        : Theme.of(context).colorScheme.onSurfaceVariant;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text('${s.posSession}: ${_shortId(sessionData.id)}',
                    style: Theme.of(context).textTheme.titleMedium),
                const Spacer(),
                Chip(
                  label: Text(
                    sessionData.isOpen ? s.statusOpen : s.statusClosed,
                  ),
                  side: BorderSide(color: statusColor),
                ),
              ],
            ),
            const SizedBox(height: 12),
            _line(s, s.sessionOpenedBy, sessionData.openedBy),
            if (sessionData.openedAt.isNotEmpty)
              _line(s, s.sessionOpenedAt,
                  s.formatDate(DateTime.parse(sessionData.openedAt).toLocal())),
            if (sessionData.closedAt != null)
              _line(s, s.sessionClosedAt, s
                  .formatDate(DateTime.parse(sessionData.closedAt!).toLocal())),
            _line(s, s.startingCash,
                s.formatMoney(sessionData.openingCashMinor, currency)),
          ],
        ),
      ),
    );
  }

  Widget _line(AppStrings s, String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [Text(label), Text(value)],
      ),
    );
  }

  Widget _moneyRow(BuildContext context, AppStrings s, String label, int minor,
      {bool emphasized = false, bool muted = false, bool isCount = false}) {
    final base = Theme.of(context).textTheme.bodyMedium;
    final valueStyle = emphasized
        ? Theme.of(context).textTheme.titleMedium
        : base;
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 4),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(
            label,
            style: muted
                ? base?.copyWith(
                    color: Theme.of(context).colorScheme.onSurfaceVariant)
                : base,
          ),
          Text(
            isCount ? '$minor' : s.formatMoney(minor, currency),
            style: valueStyle,
          ),
        ],
      ),
    );
  }

  Widget _diffCard(BuildContext context, AppStrings s) {
    final difference = sessionData.cashDifferenceMinor;
    if (difference == null) return const SizedBox.shrink();
    final counted = sessionData.closingCashMinor ?? 0;
    final color = difference == 0
        ? Colors.green.shade700
        : difference > 0
            ? Colors.orange.shade800
            : Theme.of(context).colorScheme.error;
    final label = difference == 0
        ? s.balanceEquals
        : difference > 0
            ? s.cashOver
            : s.cashShort;
    return Card(
      color: color.withValues(alpha: 0.08),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _line(s, s.countedCash, s.formatMoney(counted, currency)),
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text('${s.sessionDifference} ($label)'),
                Text(
                  '${difference > 0 ? '+' : ''}${s.formatMoney(difference.abs(), currency)}',
                  style: Theme.of(context)
                      .textTheme
                      .titleMedium
                      ?.copyWith(color: color),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  String _shortId(String id) => id.isEmpty
      ? '—'
      : id.split('-').first.substring(0, 4).toUpperCase();
}

/// History of register sessions; tapping one opens its Z-report.
class SessionHistoryScreen extends StatefulWidget {
  const SessionHistoryScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<SessionHistoryScreen> createState() => _SessionHistoryScreenState();
}

class _SessionHistoryScreenState extends State<SessionHistoryScreen> {
  SessionsPage? _page;
  bool _loading = true;
  String? _error;
  int _currentPage = 1;
  String _statusFilter = '';
  TenantSettings? _settings;

  @override
  void initState() {
    super.initState();
    _load();
    _loadSettings();
  }

  Future<void> _loadSettings() async {
    try {
      final settings = await widget.apiClient.settings(widget.session);
      if (mounted) setState(() => _settings = settings);
    } catch (_) {
      // Unreachable: the close flow then skips the PIN gate, matching a
      // server whose pos.manager.close key is unset (default false).
    }
  }

  Future<void> _setFilter(String filter) {
    if (_statusFilter == filter) return Future.value();
    setState(() => _statusFilter = filter);
    return _load(page: 1);
  }

  Future<void> _load({int page = 1}) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await widget.apiClient.sessions(
        widget.session,
        page: page,
        status: _statusFilter.isEmpty ? null : _statusFilter,
      );
      if (!mounted) return;
      setState(() {
        _page = result;
        _currentPage = page;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  Future<void> _openReport(SessionListItem item) async {
    try {
      final detail =
          await widget.apiClient.sessionDetail(widget.session, item.id);
      if (!mounted) return;
      await Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) => SessionReportScreen(
            sessionData: detail,
            apiClient: widget.apiClient,
            currency: widget.session.currencyCode,
          ),
        ),
      );
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(AppStrings.of(context).loadFailed)),
      );
    }
  }

  bool _canClose(SessionListItem item) =>
      item.isOpen &&
      (item.deviceId == widget.session.deviceId ||
          widget.session.isManager ||
          widget.session.isPlatformAdmin);

  bool get _needsManagerPin =>
      (_settings?.managerClosePin ?? false) &&
      widget.session.role != 'owner' &&
      !widget.session.isPlatformAdmin;

  Future<void> _closeSession(SessionListItem item) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    final controller = TextEditingController();
    final raw = await showDialog<String>(
      context: context,
      builder: (_) => AlertDialog(
        title: Text(s.finishSession),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(s.finishSessionConfirm),
            const SizedBox(height: 16),
            TextField(
              controller: controller,
              autofocus: true,
              keyboardType:
                  const TextInputType.numberWithOptions(decimal: true),
              decoration: InputDecoration(
                labelText: s.countedCash,
                hintText: s.countedCashHint,
                border: const OutlineInputBorder(),
              ),
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(),
            child: Text(s.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(controller.text),
            child: Text(s.finishSession),
          ),
        ],
      ),
    );
    if (raw == null || !mounted) return;
    final countedMinor = minorFromInput(raw.trim());
    String managerPin = '';
    if (_needsManagerPin) {
      final pin = await _promptManagerPin(s.closeNeedsManagerPin);
      if (pin == null || !mounted) return;
      managerPin = pin;
    }
    try {
      final closed = await widget.apiClient.closeSession(
        widget.session,
        item.id,
        closingCashMinor: countedMinor,
        managerPin: managerPin,
      );
      if (!mounted) return;
      messenger
          .showSnackBar(SnackBar(content: Text('${s.zReport} ${s.statusClosed}')));
      await _load(page: _currentPage);
      if (!mounted) return;
      await Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) => SessionReportScreen(
            sessionData: closed,
            apiClient: widget.apiClient,
            currency: widget.session.currencyCode,
          ),
        ),
      );
    } catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  Future<String?> _promptManagerPin(String message) async {
    final s = AppStrings.of(context);
    final messenger = ScaffoldMessenger.of(context);
    String? error;
    while (true) {
      if (!mounted) return null;
      final pin = await showDialog<String>(
        context: context,
        builder: (_) => _PinPrompt(message: message, strings: s, error: error),
      );
      if (pin == null || pin.isEmpty || !mounted) return null;
      final PinVerifyResult result;
      try {
        result = await widget.apiClient.verifyPin(widget.session, pin: pin);
      } catch (e) {
        if (!mounted) return null;
        messenger.showSnackBar(SnackBar(content: Text(e.toString())));
        return null;
      }
      if (result.locked) {
        if (!mounted) return null;
        messenger.showSnackBar(SnackBar(content: Text(s.pinLocked)));
        return null;
      }
      if (result.valid) return pin;
      error = result.attemptsLeft > 0
          ? s.attemptsLeft(result.attemptsLeft)
          : s.pinWrong;
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(s.sessionsHistory),
        actions: [
          IconButton(
            icon: const Icon(Icons.insights_outlined),
            tooltip: s.sessionsSummary,
            onPressed: () => Navigator.of(context).push(
              MaterialPageRoute<void>(
                builder: (_) => SessionsSummaryScreen(
                  session: widget.session,
                  apiClient: widget.apiClient,
                ),
              ),
            ),
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(_error!,
                          style: TextStyle(
                              color: Theme.of(context).colorScheme.error)),
                      const SizedBox(height: 12),
                      FilledButton(
                          onPressed: () => _load(page: _currentPage),
                          child: Text(s.retry)),
                    ],
                  ),
                )
              : _buildList(context),
    );
  }

  Widget _buildList(BuildContext context) {
    final s = AppStrings.of(context);
    final sessions = _page?.sessions ?? [];
    final total = _page?.total ?? 0;
    final totalPages = (_page?.limit ?? 50) > 0
        ? (total / (_page?.limit ?? 50)).ceil()
        : 1;

    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
          child: Row(
            children: [
              FilterChip(
                label: Text(s.all),
                selected: _statusFilter.isEmpty,
                onSelected: (_) => _setFilter(''),
              ),
              const SizedBox(width: 8),
              FilterChip(
                label: Text(s.statusOpen),
                selected: _statusFilter == 'open',
                onSelected: (_) => _setFilter('open'),
              ),
              const SizedBox(width: 8),
              FilterChip(
                label: Text(s.statusClosed),
                selected: _statusFilter == 'closed',
                onSelected: (_) => _setFilter('closed'),
              ),
            ],
          ),
        ),
        Expanded(
          child: sessions.isEmpty
              ? Center(child: Text(s.noSessions))
              : ListView.separated(
                  itemCount: sessions.length,
                  separatorBuilder: (_, __) => const Divider(height: 1),
                  itemBuilder: (context, index) {
                    final item = sessions[index];
                    final date = item.openedAt.isNotEmpty
                        ? s.formatDate(
                            DateTime.parse(item.openedAt).toLocal())
                        : '';
                    final trail = item.deviceName.isEmpty
                        ? '$date · ${item.openedBy}'
                        : '$date · ${item.openedBy} · ${item.deviceName}';
                    return ListTile(
                      leading: CircleAvatar(
                        child: Icon(
                            item.isOpen ? Icons.play_arrow : Icons.check),
                      ),
                      title: Text(
                        '${s.posSession} · ${item.salesCount} ${s.salesCount} · ${s.formatMoney(item.totalMinor, widget.session.currencyCode)}',
                      ),
                      subtitle: Text(trail),
                      trailing: _canClose(item)
                          ? OutlinedButton(
                              onPressed: () => _closeSession(item),
                              child: Text(s.finishSession),
                            )
                          : Chip(
                              label: Text(item.isOpen
                                  ? s.statusOpen
                                  : s.statusClosed),
                              visualDensity: VisualDensity.compact,
                            ),
                      onTap: () => _openReport(item),
                    );
                  },
                ),
        ),
        if (totalPages > 1)
          Padding(
            padding: const EdgeInsets.all(8.0),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                IconButton(
                  onPressed: _currentPage > 1
                      ? () => _load(page: _currentPage - 1)
                      : null,
                  icon: const Icon(Icons.chevron_left),
                ),
                Text('${s.pageOf} $_currentPage / $totalPages'),
                IconButton(
                  onPressed: _currentPage < totalPages
                      ? () => _load(page: _currentPage + 1)
                      : null,
                  icon: const Icon(Icons.chevron_right),
                ),
              ],
            ),
          ),
      ],
    );
  }
}
class _PinPrompt extends StatefulWidget {
  const _PinPrompt({
    required this.message,
    required this.strings,
    this.error,
  });

  final String message;
  final AppStrings strings;
  final String? error;

  @override
  State<_PinPrompt> createState() => _PinPromptState();
}

class _PinPromptState extends State<_PinPrompt> {
  late final TextEditingController _controller;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController();
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final s = widget.strings;
    return AlertDialog(
      title: Text(s.managerPin),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(widget.message,
              style: Theme.of(context).textTheme.bodyMedium),
          const SizedBox(height: 16),
          TextField(
            controller: _controller,
            autofocus: true,
            obscureText: true,
            keyboardType: TextInputType.number,
            onSubmitted: (value) => Navigator.of(context).pop(value.trim()),
            decoration: InputDecoration(
              labelText: s.managerPin,
              hintText: s.managerPinHint,
              errorText: widget.error,
              border: const OutlineInputBorder(),
            ),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(s.cancel),
        ),
        FilledButton(
          onPressed: () => Navigator.of(context).pop(_controller.text.trim()),
          child: Text(s.ok),
        ),
      ],
    );
  }
}
