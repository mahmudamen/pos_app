import 'package:flutter/foundation.dart';

@immutable
class Floor {
  const Floor({required this.id, required this.name, this.sortOrder = 0});

  factory Floor.fromJson(Map<String, dynamic> json) => Floor(
        id: json['id'] as String,
        name: json['name'] as String? ?? '',
        sortOrder: (json['sort_order'] as num?)?.toInt() ?? 0,
      );

  final String id;
  final String name;
  final int sortOrder;
}

@immutable
class RestaurantTable {
  const RestaurantTable({
    required this.id,
    required this.floorId,
    required this.name,
    this.seats = 0,
    this.status = 'free',
    this.floorName = '',
  });

  factory RestaurantTable.fromJson(Map<String, dynamic> json) =>
      RestaurantTable(
        id: json['id'] as String,
        floorId: json['floor_id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        seats: (json['seats'] as num?)?.toInt() ?? 0,
        status: json['status'] as String? ?? 'free',
        floorName: json['floor_name'] as String? ?? '',
      );

  final String id;
  final String floorId;
  final String name;
  final int seats;
  final String status;
  final String floorName;

  bool get isAvailable => status == 'free' || status == 'reserved';
}

class SplitBillLine {
  const SplitBillLine({required this.saleItemId, required this.quantity});

  final String saleItemId;
  final int quantity;

  Map<String, Object> toJson() =>
      {'sale_item_id': saleItemId, 'quantity': quantity};
}

@immutable
class SplitBillResult {
  const SplitBillResult({
    required this.parentSaleId,
    required this.children,
    this.currency = '',
  });

  factory SplitBillResult.fromJson(Map<String, dynamic> json) =>
      SplitBillResult(
        parentSaleId: json['parent_sale_id'] as String? ?? '',
        children: (json['children'] as List<dynamic>? ?? [])
            .map((id) => id as String)
            .toList(),
        currency: json['currency'] as String? ?? '',
      );

  final String parentSaleId;
  final List<String> children;
  final String currency;
}