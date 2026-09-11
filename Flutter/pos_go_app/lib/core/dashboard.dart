/// Wire models for GET /v1/dashboard/summary (roadmap D1).
library;

class DashboardSummary {
  const DashboardSummary({
    required this.date,
    required this.today,
    required this.topProducts,
    required this.recentSales,
    required this.perCashier,
    required this.paymentMix,
  });

  factory DashboardSummary.fromJson(Map<String, dynamic> json) =>
      DashboardSummary(
        date: json['date'] as String? ?? '',
        today: TodayStats.fromJson(
            json['today'] as Map<String, dynamic>? ?? const {}),
        topProducts: (json['top_products'] as List<dynamic>? ?? const [])
            .map((item) => TopProduct.fromJson(item as Map<String, dynamic>))
            .toList(),
        recentSales: (json['recent_sales'] as List<dynamic>? ?? const [])
            .map((item) => RecentSale.fromJson(item as Map<String, dynamic>))
            .toList(),
        perCashier: (json['per_cashier'] as List<dynamic>? ?? const [])
            .map((item) => CashierStat.fromJson(item as Map<String, dynamic>))
            .toList(),
        paymentMix: (json['payment_mix'] as List<dynamic>? ?? const [])
            .map((item) => PaymentMix.fromJson(item as Map<String, dynamic>))
            .toList(),
      );

  final String date;
  final TodayStats today;
  final List<TopProduct> topProducts;
  final List<RecentSale> recentSales;
  final List<CashierStat> perCashier;
  final List<PaymentMix> paymentMix;
}

class TodayStats {
  const TodayStats({
    this.revenueMinor = 0,
    this.salesCount = 0,
    this.avgSaleMinor = 0,
    this.itemsSold = 0,
  });

  factory TodayStats.fromJson(Map<String, dynamic> json) => TodayStats(
        revenueMinor: (json['revenue_minor'] as num?)?.toInt() ?? 0,
        salesCount: (json['sales_count'] as num?)?.toInt() ?? 0,
        avgSaleMinor: (json['avg_sale_minor'] as num?)?.toInt() ?? 0,
        itemsSold: (json['items_sold'] as num?)?.toInt() ?? 0,
      );

  final int revenueMinor;
  final int salesCount;
  final int avgSaleMinor;
  final int itemsSold;
}

class TopProduct {
  const TopProduct({
    required this.productName,
    required this.sku,
    required this.quantity,
    required this.revenueMinor,
  });

  factory TopProduct.fromJson(Map<String, dynamic> json) => TopProduct(
        productName: json['product_name'] as String? ?? '',
        sku: json['sku'] as String? ?? '',
        quantity: (json['quantity'] as num?)?.toInt() ?? 0,
        revenueMinor: (json['revenue_minor'] as num?)?.toInt() ?? 0,
      );

  final String productName;
  final String sku;
  final int quantity;
  final int revenueMinor;
}

class CashierStat {
  const CashierStat({
    required this.cashier,
    required this.salesCount,
    required this.revenueMinor,
  });

  factory CashierStat.fromJson(Map<String, dynamic> json) => CashierStat(
        cashier: json['cashier'] as String? ?? '',
        salesCount: (json['sales_count'] as num?)?.toInt() ?? 0,
        revenueMinor: (json['revenue_minor'] as num?)?.toInt() ?? 0,
      );

  final String cashier;
  final int salesCount;
  final int revenueMinor;
}

class PaymentMix {
  const PaymentMix({required this.method, required this.amountMinor});

  factory PaymentMix.fromJson(Map<String, dynamic> json) => PaymentMix(
        method: json['method'] as String? ?? '',
        amountMinor: (json['amount_minor'] as num?)?.toInt() ?? 0,
      );

  final String method;
  final int amountMinor;
}

class RecentSale {
  const RecentSale({
    required this.id,
    required this.status,
    required this.totalMinor,
    required this.currency,
    required this.paymentMethod,
    required this.createdAt,
  });

  factory RecentSale.fromJson(Map<String, dynamic> json) => RecentSale(
        id: json['id'] as String? ?? '',
        status: json['status'] as String? ?? '',
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
        currency: json['currency'] as String? ?? 'EGP',
        paymentMethod: json['payment_method'] as String? ?? 'cash',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String status;
  final int totalMinor;
  final String currency;
  final String paymentMethod;
  final String createdAt;
}