import 'dart:convert';

import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/session_store.dart';
import '../../core/storage/local_database.dart';
import '../../l10n/strings.dart';

/// Lists the operations the server refused, split into recoverable conflicts
/// (state changed elsewhere — usually another device) and plain rejections,
/// with the reason and the attempt count. Each can be retried or discarded;
/// a conflict can additionally refresh the catalog first, so a stale local
/// stock count is re-read before the retry. Reached from the "needs
/// attention" snackbar after a replay.
class PendingSyncScreen extends StatefulWidget {
  const PendingSyncScreen({
    super.key,
    required this.apiClient,
    required this.session,
    required this.localDatabase,
    this.onReplay,
    this.onRefresh,
  });

  final ApiClient apiClient;
  final Session session;
  final LocalDatabase localDatabase;
  final Future<void> Function()? onReplay;

  /// Reloads the catalog (and replays the queue). Used by a conflict's
  /// "Refresh & retry" so the retry runs against fresh server state.
  final Future<void> Function()? onRefresh;

  @override
  State<PendingSyncScreen> createState() => _PendingSyncScreenState();
}

class _PendingSyncScreenState extends State<PendingSyncScreen> {
  List<PendingCommand>? _failed;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _error = null;
      _failed = null;
    });
    try {
      final failed = await widget.localDatabase.failedCommands();
      if (!mounted) return;
      setState(() => _failed = failed);
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = e.toString());
    }
  }

  int _itemCount(PendingCommand command) {
    try {
      final payload = jsonDecode(command.payload) as Map<String, dynamic>;
      final items = payload['items'];
      return items is List ? items.length : 0;
    } catch (_) {
      return 0;
    }
  }

  Future<void> _retry(PendingCommand command, {bool refresh = false}) async {
    await widget.localDatabase.retryCommand(command.id);
    if (refresh && widget.onRefresh != null) {
      await widget.onRefresh!();
    } else {
      await widget.onReplay?.call();
    }
    await _load();
  }

  Future<void> _discard(PendingCommand command) async {
    final s = AppStrings.of(context);
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: Text(s.discardQueued),
        content: Text(s.discardQueuedBody),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: Text(s.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: Text(s.discardQueued),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    await widget.localDatabase.dropCommand(command.id);
    await _load();
  }

  Widget _sectionHeader(String title, int count) {
    final scheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
      child: Text(
        '$title ($count)',
        style: Theme.of(context)
            .textTheme
            .labelLarge
            ?.copyWith(color: scheme.primary, letterSpacing: 0.4),
      ),
    );
  }

  Widget _tile(PendingCommand command, {required bool conflict}) {
    final s = AppStrings.of(context);
    final scheme = Theme.of(context).colorScheme;
    final reason = conflict
        ? s.syncConflictReason(command.lastError)
        : (command.lastError ?? '');
    final subtitle = <String>[
      if (reason.isNotEmpty) reason,
      s.syncAttempts(command.attempts),
    ].join(' · ');
    final detail = command.errorDetail;
    final hasDetail = conflict && detail != null && detail.isNotEmpty;
    return ListTile(
      leading: Icon(
        conflict ? Icons.sync_problem : Icons.error_outline,
        color: conflict ? scheme.tertiary : scheme.error,
      ),
      title: Text('${s.queuedSale} · ${_itemCount(command)} ${s.items}'),
      subtitle: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(subtitle),
          if (hasDetail)
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text(
                detail,
                style: Theme.of(context)
                    .textTheme
                    .bodySmall
                    ?.copyWith(color: scheme.onSurfaceVariant),
              ),
            ),
        ],
      ),
      isThreeLine: hasDetail,
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            tooltip: conflict ? s.refreshAndRetry : s.retry,
            icon: const Icon(Icons.refresh),
            onPressed: () => _retry(command, refresh: conflict),
          ),
          IconButton(
            tooltip: s.discardQueued,
            icon: const Icon(Icons.delete_outline),
            onPressed: () => _discard(command),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final failed = _failed;
    return Scaffold(
      appBar: AppBar(title: Text(s.syncIssues)),
      body: _error != null
          ? Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(_error!,
                        textAlign: TextAlign.center,
                        style: TextStyle(
                            color: Theme.of(context).colorScheme.error)),
                    const SizedBox(height: 12),
                    FilledButton(onPressed: _load, child: Text(s.retry)),
                  ],
                ),
              ),
            )
          : failed == null
              ? const Center(child: CircularProgressIndicator())
              : failed.isEmpty
                  ? Center(child: Text(s.offlineSynced))
                  : _list(s, failed),
    );
  }

  Widget _list(AppStrings s, List<PendingCommand> failed) {
    final conflicts = failed.where((c) => c.isConflict).toList();
    final rejected = failed.where((c) => !c.isConflict).toList();
    final tiles = <Widget>[
      if (conflicts.isNotEmpty)
        _sectionHeader(s.syncConflictsTitle, conflicts.length),
      ...conflicts.map((c) => _tile(c, conflict: true)),
      if (rejected.isNotEmpty)
        _sectionHeader(s.failedSyncTitle, rejected.length),
      ...rejected.map((c) => _tile(c, conflict: false)),
    ];
    return ListView(children: tiles);
  }
}
