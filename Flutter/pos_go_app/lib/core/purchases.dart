import 'package:flutter/foundation.dart';

/// Models for the supplier-invoice OCR + purchases feature. Mirrors
/// internal/transport/purchases on the backend: an OCR scan proposes parsed
/// lines (no stock is touched), the merchant reviews/edits them, and applying
/// a purchase deterministically updates products (stock += qty, cost = invoice
/// unit price, unit = invoice unit when it differs).

@immutable
class OcrLine {
  const OcrLine({
    required this.name,
    required this.quantity,
    required this.unit,
    required this.unitPriceMinor,
    required this.totalMinor,
    required this.score,
    this.productId = '',
    this.productName = '',
    this.matchScore = 0,
  });

  factory OcrLine.fromJson(Map<String, dynamic> json) => OcrLine(
        name: json['name'] as String? ?? '',
        quantity: (json['quantity'] as num?)?.toDouble() ?? 0,
        unit: json['unit'] as String? ?? 'piece',
        unitPriceMinor: (json['unit_price_minor'] as num?)?.toInt() ?? 0,
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
        score: (json['score'] as num?)?.toInt() ?? 0,
        productId: json['product_id'] as String? ?? '',
        productName: json['product_name'] as String? ?? '',
        matchScore: (json['match_score'] as num?)?.toInt() ?? 0,
      );

  final String name;
  final double quantity;
  final String unit;
  final int unitPriceMinor;
  final int totalMinor;
  final int score;
  final String productId;
  final String productName;
  final int matchScore;
}

@immutable
class OcrScanResult {
  const OcrScanResult({
    required this.rawText,
    required this.lines,
    required this.discarded,
  });

  factory OcrScanResult.fromJson(Map<String, dynamic> json) => OcrScanResult(
        rawText: json['raw_text'] as String? ?? '',
        lines: (json['lines'] as List<dynamic>? ?? [])
            .map((e) => OcrLine.fromJson(e as Map<String, dynamic>))
            .toList(),
        discarded: (json['discarded'] as List<dynamic>? ?? [])
            .map((e) => e as String)
            .toList(),
      );

  final String rawText;
  final List<OcrLine> lines;
  final List<String> discarded;
}

/// One line the merchant sends back to POST /v1/purchases: an existing
/// product (product_id set; stock/cost/unit updated) or a new one (back-end
/// creates it with price = cost, then the client patches a desired sale price).
@immutable
class PurchaseLineInput {
  const PurchaseLineInput({
    this.productId,
    required this.productName,
    required this.quantity,
    required this.unit,
    required this.unitPriceMinor,
    this.matchScore = 0,
    this.rowText = '',
  });

  final String? productId;
  final String productName;
  final double quantity;
  final String unit;
  final int unitPriceMinor;
  final int matchScore;
  final String rowText;

  Map<String, dynamic> toJson() => {
        if (productId != null) 'product_id': productId,
        'product_name': productName,
        'quantity': quantity,
        'unit': unit,
        'unit_price_minor': unitPriceMinor,
        'match_score': matchScore,
        'row_text': rowText,
      };
}

@immutable
class AppliedLine {
  const AppliedLine({
    required this.productId,
    required this.productName,
    required this.unit,
    required this.stockDelta,
    required this.newStock,
    required this.newCostMinor,
    required this.created,
  });

  factory AppliedLine.fromJson(Map<String, dynamic> json) => AppliedLine(
        productId: json['product_id'] as String? ?? '',
        productName: json['product_name'] as String? ?? '',
        unit: json['unit'] as String? ?? 'piece',
        stockDelta: (json['stock_delta'] as num?)?.toInt() ?? 0,
        newStock: (json['new_stock'] as num?)?.toInt() ?? 0,
        newCostMinor: (json['new_cost_minor'] as num?)?.toInt() ?? 0,
        created: json['created'] as bool? ?? false,
      );

  final String productId;
  final String productName;
  final String unit;
  final int stockDelta;
  final int newStock;
  final int newCostMinor;
  final bool created;
}

@immutable
class ApplyPurchaseResult {
  const ApplyPurchaseResult({
    required this.purchaseId,
    required this.subtotalMinor,
    required this.taxMinor,
    required this.totalMinor,
    required this.currency,
    required this.createdAt,
    required this.lines,
    this.supplier = '',
    this.invoiceNo = '',
  });

  factory ApplyPurchaseResult.fromJson(Map<String, dynamic> json) =>
      ApplyPurchaseResult(
        purchaseId: json['purchase_id'] as String? ?? '',
        supplier: json['supplier'] as String? ?? '',
        invoiceNo: json['invoice_no'] as String? ?? '',
        subtotalMinor: (json['subtotal_minor'] as num?)?.toInt() ?? 0,
        taxMinor: (json['tax_minor'] as num?)?.toInt() ?? 0,
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
        currency: json['currency'] as String? ?? 'EGP',
        createdAt: json['created_at'] as String? ?? '',
        lines: (json['lines'] as List<dynamic>? ?? [])
            .map((e) => AppliedLine.fromJson(e as Map<String, dynamic>))
            .toList(),
      );

  final String purchaseId;
  final String supplier;
  final String invoiceNo;
  final int subtotalMinor;
  final int taxMinor;
  final int totalMinor;
  final String currency;
  final String createdAt;
  final List<AppliedLine> lines;
}

/// Rolling-window OCR usage: how many scans were consumed on the day/week/month
/// window, the configured caps (0 = unlimited), and the tenant's scan-borrowing
/// credit balance. See internal/transport/purchases/metering.go.
@immutable
class OcrUsageWindow {
  const OcrUsageWindow({
    this.day = 0,
    this.week = 0,
    this.month = 0,
  });

  factory OcrUsageWindow.fromJson(Map<String, dynamic> json) => OcrUsageWindow(
        day: (json['day'] as num?)?.toInt() ?? 0,
        week: (json['week'] as num?)?.toInt() ?? 0,
        month: (json['month'] as num?)?.toInt() ?? 0,
      );

  final int day;
  final int week;
  final int month;
}

@immutable
class OcrMeterState {
  const OcrMeterState({
    this.used = const OcrUsageWindow(),
    this.limits = const OcrUsageWindow(),
    this.creditsRemaining = 0,
  });

  factory OcrMeterState.fromJson(Map<String, dynamic> json) => OcrMeterState(
        used: OcrUsageWindow.fromJson(
            json['used'] as Map<String, dynamic>? ?? const {}),
        limits: OcrUsageWindow.fromJson(
            json['limits'] as Map<String, dynamic>? ?? const {}),
        creditsRemaining: (json['credits_remaining'] as num?)?.toInt() ?? 0,
      );

  final OcrUsageWindow used;
  final OcrUsageWindow limits;
  final int creditsRemaining;

  /// OCR is metered only when some window has a cap; unlimited deployments keep
  /// the meter off.
  bool get metered =>
      limits.day > 0 || limits.week > 0 || limits.month > 0;
}

@immutable
class PurchaseSummary {
  const PurchaseSummary({
    required this.id,
    required this.supplier,
    required this.invoiceNo,
    required this.currency,
    required this.itemCount,
    required this.subtotalMinor,
    required this.totalMinor,
    required this.createdBy,
    required this.createdAt,
    this.taxMinor = 0,
  });

  factory PurchaseSummary.fromJson(Map<String, dynamic> json) => PurchaseSummary(
        id: json['id'] as String,
        supplier: json['supplier'] as String? ?? '',
        invoiceNo: json['invoice_no'] as String? ?? '',
        currency: json['currency'] as String? ?? 'EGP',
        itemCount: (json['item_count'] as num?)?.toInt() ?? 0,
        subtotalMinor: (json['subtotal_minor'] as num?)?.toInt() ?? 0,
        taxMinor: (json['tax_minor'] as num?)?.toInt() ?? 0,
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
        createdBy: json['created_by'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String supplier;
  final String invoiceNo;
  final String currency;
  final int itemCount;
  final int subtotalMinor;
  final int taxMinor;
  final int totalMinor;
  final String createdBy;
  final String createdAt;
}

@immutable
class PurchasesPage {
  const PurchasesPage({
    required this.purchases,
    required this.total,
    required this.page,
    required this.limit,
  });

  factory PurchasesPage.fromJson(Map<String, dynamic> json) {
    final meta = json['meta'] as Map<String, dynamic>? ?? const {};
    return PurchasesPage(
      purchases: (json['data'] as List<dynamic>? ?? [])
          .map((e) => PurchaseSummary.fromJson(e as Map<String, dynamic>))
          .toList(),
      total: (meta['total'] as num?)?.toInt() ?? 0,
      page: (meta['page'] as num?)?.toInt() ?? 1,
      limit: (meta['limit'] as num?)?.toInt() ?? 50,
    );
  }

  final List<PurchaseSummary> purchases;
  final int total;
  final int page;
  final int limit;

  int get totalPages => limit < 1 ? 1 : (total / limit).ceil();
}