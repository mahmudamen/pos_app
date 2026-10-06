import 'dart:convert';

import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/session_store.dart';
import '../../core/storage/local_database.dart';
import '../../l10n/strings.dart';

/// Lists the queued sales the server refused, with the reason and the attempt
/// count, and lets the cashier retry or discard each one. Reached from the
/// "needs attention" snackbar after a replay.
class PendingSyncScreen extends StatefulWidget {
  const PendingSyncScreen({
    super.key,
    required this.apiClient,
    required this.session,
    required this.localDatabase,
    this.onReplay,
  });

  final ApiClient apiClient;
  final Session session;
  final LocalDatabase localDatabase;
  final Future<void> Function()? onReplay;

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

  Future<void> _retry(PendingCommand command) async {
    await widget.localDatabase.retryCommand(command.id);
    await widget.onReplay?.call();
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

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final failed = _failed;
    return Scaffold(
      appBar: AppBar(title: Text(s.failedSyncTitle)),
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
                  : ListView.separated(
                      itemCount: failed.length,
                      separatorBuilder: (_, __) => const Divider(height: 1),
                      itemBuilder: (context, index) {
                        final command = failed[index];
                        final subtitle = <String>[
                          if (command.lastError != null &&
                              command.lastError!.isNotEmpty)
                            command.lastError!,
                          s.syncAttempts(command.attempts),
                        ].join(' · ');
                        return ListTile(
                          leading: const Icon(Icons.sync_problem),
                          title: Text(
                              '${s.queuedSale} · ${_itemCount(command)} ${s.items}'),
                          subtitle: Text(subtitle),
                          trailing: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              IconButton(
                                tooltip: s.retry,
                                icon: const Icon(Icons.refresh),
                                onPressed: () => _retry(command),
                              ),
                              IconButton(
                                tooltip: s.discardQueued,
                                icon: const Icon(Icons.delete_outline),
                                onPressed: () => _discard(command),
                              ),
                            ],
                          ),
                        );
                      },
                    ),
    );
  }
}
