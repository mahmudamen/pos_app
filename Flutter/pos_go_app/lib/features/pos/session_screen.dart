import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/registers.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

/// Full Z-report for one register session (open or closed).
class SessionReportScreen extends StatelessWidget {
  const SessionReportScreen({
    super.key,
    required this.sessionData,
    required this.apiClient,
  });

  final RegisterSession sessionData;
  final ApiClient apiClient;

  @override
  Widget build(BuildContext context) {
    return _ReportScaffold(sessionData: sessionData);
  }
}

class _ReportScaffold extends StatelessWidget {
  const _ReportScaffold({required this.sessionData});

  final RegisterSession sessionData;

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
                s.formatMoney(sessionData.openingCashMinor, 'EGP')),
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
            isCount ? '$minor' : s.formatMoney(minor, 'EGP'),
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
            _line(s, s.countedCash, s.formatMoney(counted, 'EGP')),
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Text('${s.sessionDifference} ($label)'),
                Text(
                  '${difference > 0 ? '+' : ''}${s.formatMoney(difference.abs(), 'EGP')}',
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

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load({int page = 1}) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final result = await widget.apiClient.sessions(widget.session, page: page);
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

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(s.sessionsHistory)),
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
                    return ListTile(
                      leading: CircleAvatar(
                        child: Icon(
                            item.isOpen ? Icons.play_arrow : Icons.check),
                      ),
                      title: Text(
                        '${s.posSession} · ${item.salesCount} ${s.salesCount} · ${s.formatMoney(item.totalMinor, widget.session.currencyCode)}',
                      ),
                      subtitle: Text('$date · ${item.openedBy}'),
                      trailing: Chip(
                        label: Text(
                            item.isOpen ? s.statusOpen : s.statusClosed),
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