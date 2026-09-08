import 'dart:convert';
import 'package:flutter/foundation.dart';
import '../repository/api_repository.dart';
import '../features/invoice/dto/table_qu_dto.dart';
import '../db/app_database.dart';
import 'connectivity_service.dart';

class OfflineSyncService {
  final ApiRepository api;
  final AppDatabase db;
  final ConnectivityService connectivity;

  final ValueNotifier<bool> syncing = ValueNotifier(false);

  // This list stores all operation messages
  final ValueNotifier<List<String>> logs = ValueNotifier([]);

  OfflineSyncService({
    required this.api,
    required this.db,
    required this.connectivity,
  });

  Future<void> start() async {
    await connectivity.start();
    connectivity.isOnline.addListener(() {
      if (connectivity.isOnline.value) {
        _syncPending();
      }
    });
  }

  Future<void> _syncPending() async {
    if (syncing.value) return;
    syncing.value = true;
    logs.value = [...logs.value, "[OfflineSync] Starting sync..."];
    try {
      final list = await db.pendingInvoices();
      logs.value = [
        ...logs.value,
        "[OfflineSync] Found ${list.length} pending invoices.",
      ];
      for (final q in list) {
        logs.value = [
          ...logs.value,
          "[OfflineSync] Syncing invoice id=${q.id}...",
        ];
        try {
          final raw = (jsonDecode(q.linesJson) as List)
              .cast<Map<String, dynamic>>();
          final lines = raw.map((e) => TableQuDto.fromJson(e)).toList();
          await api.invoice.setQu(
            lines: lines,
            idCu: q.customerId,
            idUser: '0',
            credit: q.credit,
          );
          await db.markInvoiceSent(q.id);
          logs.value = [
            ...logs.value,
            "[OfflineSync] Invoice id=${q.id} sent successfully.",
          ];
        } catch (e) {
          await db.markInvoiceFailed(q.id, e.toString());
          logs.value = [
            ...logs.value,
            "[OfflineSync] Failed to send invoice id=${q.id}: $e",
          ];
        }
      }
    } finally {
      syncing.value = false;
      logs.value = [...logs.value, "[OfflineSync] Sync completed."];
    }
  }
}
