import 'package:flutter/foundation.dart';

@immutable
class InventoryAdjustment {
  const InventoryAdjustment({
    required this.id,
    required this.productId,
    required this.productName,
    required this.sku,
    required this.reason,
    required this.quantityDelta,
    required this.note,
    required this.createdBy,
    required this.createdAt,
  });

  factory InventoryAdjustment.fromJson(Map<String, dynamic> json) =>
      InventoryAdjustment(
        id: json['id'] as String? ?? '',
        productId: json['product_id'] as String? ?? '',
        productName: json['product_name'] as String? ?? '',
        sku: json['sku'] as String? ?? '',
        reason: json['reason'] as String? ?? '',
        quantityDelta: (json['quantity_delta'] as num?)?.toInt() ?? 0,
        note: json['note'] as String? ?? '',
        createdBy: json['created_by'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String productId;
  final String productName;
  final String sku;
  final String reason;
  final int quantityDelta;
  final String note;
  final String createdBy;
  final String createdAt;

  bool get isCredit => quantityDelta > 0;
}

@immutable
class InventoryAdjustmentPage {
  const InventoryAdjustmentPage({
    required this.items,
    required this.total,
    required this.page,
    required this.limit,
  });

  factory InventoryAdjustmentPage.fromJson(Map<String, dynamic> json) {
    final data = json['data'] as List<dynamic>? ?? const [];
    final meta = json['meta'] as Map<String, dynamic>? ?? const {};
    return InventoryAdjustmentPage(
      items: data
          .map((e) => InventoryAdjustment.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: (meta['total'] as num?)?.toInt() ?? 0,
      page: (meta['page'] as num?)?.toInt() ?? 1,
      limit: (meta['limit'] as num?)?.toInt() ?? 50,
    );
  }

  final List<InventoryAdjustment> items;
  final int total;
  final int page;
  final int limit;
}

@immutable
class AdjustmentResult {
  const AdjustmentResult({required this.adjustment, required this.newStock});

  factory AdjustmentResult.fromJson(Map<String, dynamic> json) {
    final meta = json['meta'] as Map<String, dynamic>? ?? const {};
    return AdjustmentResult(
      adjustment:
          InventoryAdjustment.fromJson(json['data'] as Map<String, dynamic>),
      newStock: (meta['stock_quantity'] as num?)?.toInt() ?? 0,
    );
  }

  final InventoryAdjustment adjustment;
  final int newStock;
}