class SaleReceipt {
  const SaleReceipt({
    required this.tenantName,
    this.tenantAddress = '',
    required this.saleId,
    required this.status,
    required this.createdAt,
    required this.cashier,
    this.device = '',
    this.tableName = '',
    this.floorName = '',
    this.customerName = '',
    required this.currency,
    this.items = const [],
    required this.subtotalMinor,
    this.discountMinor = 0,
    this.tipsMinor = 0,
    required this.totalMinor,
    this.payments = const [],
    this.loyaltyPointsEarned = 0,
    this.idempotencyKey = '',
  });

  factory SaleReceipt.fromJson(Map<String, dynamic> json) => SaleReceipt(
        tenantName: json['tenant_name'] as String? ?? '',
        tenantAddress: json['tenant_address'] as String? ?? '',
        saleId: json['sale_id'] as String? ?? '',
        status: json['status'] as String? ?? 'completed',
        createdAt: json['created_at'] as String? ?? '',
        cashier: json['cashier'] as String? ?? '',
        device: json['device'] as String? ?? '',
        tableName: json['table_name'] as String? ?? '',
        floorName: json['floor_name'] as String? ?? '',
        customerName: json['customer_name'] as String? ?? '',
        currency: json['currency'] as String? ?? 'EGP',
        items: (json['items'] as List<dynamic>? ?? [])
            .map((e) => ReceiptLine.fromJson(e as Map<String, dynamic>))
            .toList(),
        subtotalMinor: (json['subtotal_minor'] as num?)?.toInt() ?? 0,
        discountMinor: (json['discount_minor'] as num?)?.toInt() ?? 0,
        tipsMinor: (json['tips_minor'] as num?)?.toInt() ?? 0,
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
        payments: (json['payments'] as List<dynamic>? ?? [])
            .map((e) => ReceiptPayment.fromJson(e as Map<String, dynamic>))
            .toList(),
        loyaltyPointsEarned:
            (json['loyalty_points_earned'] as num?)?.toInt() ?? 0,
        idempotencyKey: json['idempotency_key'] as String? ?? '',
      );

  final String tenantName;
  final String tenantAddress;
  final String saleId;
  final String status;
  final String createdAt;
  final String cashier;
  final String device;
  final String tableName;
  final String floorName;
  final String customerName;
  final String currency;
  final List<ReceiptLine> items;
  final int subtotalMinor;
  final int discountMinor;
  final int tipsMinor;
  final int totalMinor;
  final List<ReceiptPayment> payments;
  final int loyaltyPointsEarned;
  final String idempotencyKey;
}

class ReceiptLine {
  const ReceiptLine({
    required this.name,
    this.sku = '',
    required this.quantity,
    required this.unitPriceMinor,
    required this.totalMinor,
  });

  factory ReceiptLine.fromJson(Map<String, dynamic> json) => ReceiptLine(
        name: json['name'] as String? ?? '',
        sku: json['sku'] as String? ?? '',
        quantity: (json['quantity'] as num?)?.toInt() ?? 0,
        unitPriceMinor: (json['unit_price_minor'] as num?)?.toInt() ?? 0,
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
      );

  final String name;
  final String sku;
  final int quantity;
  final int unitPriceMinor;
  final int totalMinor;
}

class ReceiptPayment {
  const ReceiptPayment({
    required this.method,
    required this.amountMinor,
    this.tipMinor = 0,
  });

  factory ReceiptPayment.fromJson(Map<String, dynamic> json) =>
      ReceiptPayment(
        method: json['method'] as String? ?? 'cash',
        amountMinor: (json['amount_minor'] as num?)?.toInt() ?? 0,
        tipMinor: (json['tip_minor'] as num?)?.toInt() ?? 0,
      );

  final String method;
  final int amountMinor;
  final int tipMinor;
}