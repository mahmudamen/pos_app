import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/registers.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

/// Aggregated register activity for a chosen window
/// (`GET /v1/registers/summary`): how the till traded over a day, a week or a
/// month without stepping through individual Z-reports.
class SessionsSummaryScreen extends StatefulWidget {
  const SessionsSummaryScreen({
    super.key,
    required this.session,
    required this.apiClient,
  });

  final Session session;
  final ApiClient apiClient;

  @override
  State<SessionsSummaryScreen> createState() => _SessionsSummaryScreenState();
}

enum _SummaryRange { today, last7, last30 }

class _SessionsSummaryScreenState extends State<SessionsSummaryScreen> {
  _SummaryRange _range = _SummaryRange.today;
  RangeSummary? _summary;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  String get _from {
    final now = DateTime.now();
    switch (_range) {
      case _SummaryRange.today:
        return DateTime(now.year, now.month, now.day).toUtc().toIso8601String();
      case _SummaryRange.last7:
        return now.subtract(const Duration(days: 7)).toUtc().toIso8601String();
      case _SummaryRange.last30:
        return now.subtract(const Duration(days: 30)).toUtc().toIso8601String();
    }
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final summary =
          await widget.apiClient.sessionSummary(widget.session, from: _from);
      if (!mounted) return;
      setState(() {
        _summary = summary;
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

  Future<void> _setRange(_SummaryRange range) async {
    if (range == _range) return;
    setState(() => _range = range);
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(s.sessionsSummary),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: s.refresh,
            onPressed: _loading ? null : _load,
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
                      FilledButton(onPressed: _load, child: Text(s.retry)),
                    ],
                  ),
                )
              : _buildBody(context, s),
    );
  }

  Widget _buildBody(BuildContext context, AppStrings s) {
    final summary = _summary;
    if (summary == null) return const SizedBox.shrink();
    final currency = widget.session.currencyCode;
    final difference = summary.cashDifferenceMinor;
    final diffColor = difference == 0
        ? Colors.green.shade700
        : difference > 0
            ? Colors.orange.shade800
            : Theme.of(context).colorScheme.error;
    final diffText = difference == 0
        ? s.formatMoney(0, currency)
        : '${difference > 0 ? '+' : '-'}'
            '${s.formatMoney(difference.abs(), currency)}';

    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        Row(
          children: [
            FilterChip(
              label: Text(s.today),
              selected: _range == _SummaryRange.today,
              onSelected: (_) => _setRange(_SummaryRange.today),
            ),
            const SizedBox(width: 8),
            FilterChip(
              label: Text(s.last7Days),
              selected: _range == _SummaryRange.last7,
              onSelected: (_) => _setRange(_SummaryRange.last7),
            ),
            const SizedBox(width: 8),
            FilterChip(
              label: Text(s.last30Days),
              selected: _range == _SummaryRange.last30,
              onSelected: (_) => _setRange(_SummaryRange.last30),
            ),
          ],
        ),
        const SizedBox(height: 16),
        GridView.count(
          crossAxisCount: 2,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          mainAxisSpacing: 12,
          crossAxisSpacing: 12,
          childAspectRatio: 1.9,
          children: [
            _StatCard(
                label: s.revenue,
                value: s.formatMoney(summary.totalMinor, currency)),
            _StatCard(label: s.salesCount, value: '${summary.salesCount}'),
            _StatCard(
                label: s.avgSale,
                value: s.formatMoney(summary.averageSaleMinor, currency)),
            _StatCard(
                label: s.cash,
                value: s.formatMoney(summary.cashMinor, currency)),
            _StatCard(
                label: s.card,
                value: s.formatMoney(summary.cardMinor, currency)),
            _StatCard(
                label: s.mobilePayment,
                value: s.formatMoney(summary.mobileMinor, currency)),
            _StatCard(
                label: s.sessionsOpened, value: '${summary.sessionsOpened}'),
            _StatCard(
                label: s.sessionsClosed, value: '${summary.sessionsClosed}'),
            _StatCard(
                label: s.sessionDifference, value: diffText, accent: diffColor),
          ],
        ),
        const SizedBox(height: 16),
        Text(
          '${s.subtotalLabel} ${s.formatMoney(summary.subtotalMinor, currency)}'
          ' · ${s.discountLabel} ${s.formatMoney(summary.discountMinor, currency)}'
          ' · ${s.taxLabel} ${s.formatMoney(summary.taxMinor, currency)}',
          style: Theme.of(context).textTheme.bodySmall,
          textAlign: TextAlign.center,
        ),
      ],
    );
  }
}

class _StatCard extends StatelessWidget {
  const _StatCard({required this.label, required this.value, this.accent});

  final String label;
  final String value;
  final Color? accent;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            Text(
              label,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  color: Theme.of(context).colorScheme.onSurfaceVariant),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            const SizedBox(height: 4),
            Text(
              value,
              style: Theme.of(context)
                  .textTheme
                  .titleMedium
                  ?.copyWith(color: accent),
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
          ],
        ),
      ),
    );
  }
}
