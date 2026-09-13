import 'package:flutter/material.dart';

import '../../core/api_client.dart';
import '../../core/restaurants.dart';
import '../../core/session_store.dart';
import '../../l10n/strings.dart';

class TablePickResult {
  const TablePickResult({this.table, this.cleared = false});
  final RestaurantTable? table;
  final bool cleared;
}

class TablePickerSheet extends StatefulWidget {
  const TablePickerSheet({
    super.key,
    required this.apiClient,
    required this.session,
    required this.currentTableId,
  });

  final ApiClient apiClient;
  final Session session;
  final String? currentTableId;

  static Future<TablePickResult?> pick({
    required BuildContext context,
    required ApiClient apiClient,
    required Session session,
    String? currentTableId,
  }) {
    return showModalBottomSheet<TablePickResult>(
      context: context,
      isScrollControlled: true,
      builder: (_) => TablePickerSheet(
        apiClient: apiClient,
        session: session,
        currentTableId: currentTableId,
      ),
    );
  }

  @override
  State<TablePickerSheet> createState() => _TablePickerSheetState();
}

class _TablePickerSheetState extends State<TablePickerSheet> {
  List<RestaurantTable>? _tables;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _error = null;
      _tables = null;
    });
    try {
      final tables = await widget.apiClient.tables(widget.session);
      if (!mounted) return;
      setState(() => _tables = tables);
    } catch (error) {
      if (!mounted) return;
      setState(() => _error = error.toString());
    }
  }

  String _statusLabel(AppStrings s, String status) {
    switch (status) {
      case 'occupied':
        return s.tableOccupiedStatus;
      case 'reserved':
        return s.tableReservedStatus;
      case 'closed':
        return s.tableClosedStatus;
      default:
        return s.tableFreeStatus;
    }
  }

  IconData _statusIcon(String status) {
    switch (status) {
      case 'occupied':
        return Icons.table_bar;
      case 'reserved':
        return Icons.event_seat;
      case 'closed':
        return Icons.no_meals_outlined;
      default:
        return Icons.event_seat;
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = AppStrings.of(context);
    final tables = _tables;
    return SafeArea(
      child: Container(
        height: MediaQuery.of(context).size.height * 0.65,
        padding: const EdgeInsets.fromLTRB(20, 20, 20, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(s.pickTable,
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 12),
            if (widget.currentTableId != null)
              TextButton.icon(
                onPressed: () =>
                    Navigator.of(context).pop(const TablePickResult(cleared: true)),
                icon: const Icon(Icons.close),
                label: Text(s.clearTable),
              ),
            const SizedBox(height: 4),
            Expanded(
              child: _error != null
                  ? Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Text(_error!,
                              style: TextStyle(
                                  color: Theme.of(context).colorScheme.error)),
                          const SizedBox(height: 12),
                          FilledButton(
                            onPressed: _load,
                            child: Text(s.reloadTables),
                          ),
                        ],
                      ),
                    )
                  : tables == null
                      ? const Center(child: CircularProgressIndicator())
                      : tables.isEmpty
                          ? Center(child: Text(s.noTable))
                          : ListView.builder(
                              itemCount: tables.length,
                              itemBuilder: (context, index) {
                                final table = tables[index];
                                final isCurrent =
                                    table.id == widget.currentTableId;
                                final selectable = table.isAvailable || isCurrent;
                                return ListTile(
                                  leading: Icon(_statusIcon(table.status)),
                                  title: Text(table.name),
                                  subtitle: Text(table.floorName),
                                  trailing: Text(
                                    _statusLabel(s, table.status),
                                    style: TextStyle(
                                      color: selectable
                                          ? Theme.of(context).colorScheme.primary
                                          : Theme.of(context)
                                              .colorScheme
                                              .onSurfaceVariant,
                                      fontWeight: FontWeight.w600,
                                    ),
                                  ),
                                  selected: isCurrent,
                                  enabled: selectable,
                                  onTap: selectable
                                      ? () => Navigator.of(context)
                                          .pop(TablePickResult(table: table))
                                      : null,
                                );
                              },
                            ),
            ),
          ],
        ),
      ),
    );
  }
}