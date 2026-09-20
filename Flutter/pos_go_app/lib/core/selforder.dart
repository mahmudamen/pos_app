/// Wire models for the self-order (QR) vertical — see api_client.dart for the
/// matching network methods and selforder_screens for the UI.
library;

/// A single line on a self-order (server-computed totals).
class SelfOrderItem {
  const SelfOrderItem({
    required this.productId,
    required this.name,
    required this.sku,
    required this.quantity,
    required this.unitPriceMinor,
    required this.totalMinor,
  });

  factory SelfOrderItem.fromJson(Map<String, dynamic> json) => SelfOrderItem(
        productId: json['product_id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        sku: json['sku'] as String? ?? '',
        quantity: (json['quantity'] as num?)?.toInt() ?? 0,
        unitPriceMinor: (json['unit_price_minor'] as num?)?.toInt() ?? 0,
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
      );

  final String productId;
  final String name;
  final String sku;
  final int quantity;
  final int unitPriceMinor;
  final int totalMinor;
}

/// A customer self-order awaiting cashier approval.
class SelfOrder {
  const SelfOrder({
    required this.id,
    required this.reference,
    required this.customerName,
    required this.tableName,
    required this.status,
    required this.subtotalMinor,
    required this.totalMinor,
    required this.currency,
    required this.items,
    required this.createdAt,
    this.approvedSaleId,
  });

  factory SelfOrder.fromJson(Map<String, dynamic> json) => SelfOrder(
        id: json['id'] as String,
        reference: json['reference'] as String? ?? '',
        customerName: json['customer_name'] as String? ?? '',
        tableName: json['table_name'] as String? ?? '',
        status: json['status'] as String? ?? 'pending',
        subtotalMinor: (json['subtotal_minor'] as num?)?.toInt() ?? 0,
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
        currency: json['currency'] as String? ?? 'EGP',
        items: (json['items'] as List<dynamic>? ?? [])
            .map((e) => SelfOrderItem.fromJson(e as Map<String, dynamic>))
            .toList(),
        approvedSaleId: json['approved_sale_id'] as String?,
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String reference;
  final String customerName;
  final String tableName;
  final String status;
  final int subtotalMinor;
  final int totalMinor;
  final String currency;
  final List<SelfOrderItem> items;
  final String? approvedSaleId;
  final String createdAt;

  bool get isPending => status == 'pending';
  bool get isApproved => status == 'approved';
  bool get isCancelled => status == 'cancelled';
}

class SelfOrdersPage {
  const SelfOrdersPage({
    required this.orders,
    required this.total,
    required this.page,
    required this.limit,
  });

  factory SelfOrdersPage.fromJson(Map<String, dynamic> json) => SelfOrdersPage(
        orders: (json['items'] as List<dynamic>? ?? [])
            .map((e) => SelfOrder.fromJson(e as Map<String, dynamic>))
            .toList(),
        total: (json['total'] as num?)?.toInt() ?? 0,
        page: (json['page'] as num?)?.toInt() ?? 1,
        limit: (json['limit'] as num?)?.toInt() ?? 50,
      );

  final List<SelfOrder> orders;
  final int total;
  final int page;
  final int limit;

  bool get hasMore => page * limit < total;
}

/// Result of approving a self-order — it becomes a real sale.
class SelfOrderApproval {
  const SelfOrderApproval({
    required this.orderId,
    required this.status,
    required this.saleId,
  });

  factory SelfOrderApproval.fromJson(Map<String, dynamic> json) =>
      SelfOrderApproval(
        orderId: json['order_id'] as String? ?? '',
        status: json['status'] as String? ?? 'approved',
        saleId: (json['sale'] as Map<String, dynamic>?)?['id'] as String? ?? '',
      );

  final String orderId;
  final String status;
  final String saleId;
}

/// A customer feedback / request message (optionally with a web-push
/// subscription for back-in-stock notifications).
class ProductRequest {
  const ProductRequest({
    required this.id,
    required this.status,
    required this.createdAt,
    this.productId,
    required this.productName,
    this.note = '',
    this.contact = '',
    this.hasWebPush = false,
    this.notifiedAt,
  });

  factory ProductRequest.fromJson(Map<String, dynamic> json) => ProductRequest(
        id: json['id'] as String,
        status: json['status'] as String? ?? 'open',
        createdAt: json['created_at'] as String? ?? '',
        productId: json['product_id'] as String?,
        productName: json['product_name'] as String? ?? '',
        note: json['note'] as String? ?? '',
        contact: json['contact'] as String? ?? '',
        hasWebPush: json['has_webpush'] as bool? ?? false,
        notifiedAt: json['notified_at'] as String?,
      );

  final String id;
  final String status;
  final String createdAt;
  final String? productId;
  final String productName;
  final String note;
  final String contact;
  final bool hasWebPush;
  final String? notifiedAt;

  bool get isOpen => status == 'open';
  bool get isFulfilled => status == 'fulfilled';
}

class ProductRequestsPage {
  const ProductRequestsPage({
    required this.requests,
    required this.total,
    required this.page,
    required this.limit,
  });

  factory ProductRequestsPage.fromJson(Map<String, dynamic> json) =>
      ProductRequestsPage(
        requests: (json['items'] as List<dynamic>? ?? [])
            .map((e) => ProductRequest.fromJson(e as Map<String, dynamic>))
            .toList(),
        total: (json['total'] as num?)?.toInt() ?? 0,
        page: (json['page'] as num?)?.toInt() ?? 1,
        limit: (json['limit'] as num?)?.toInt() ?? 50,
      );

  final List<ProductRequest> requests;
  final int total;
  final int page;
  final int limit;

  bool get hasMore => page * limit < total;
}