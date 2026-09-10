/// Wire models for the register (POS) session feature — see api_client.dart
/// for the matching network methods.
library;

class SessionSummary {
  const SessionSummary({
    required this.salesCount,
    required this.subtotalMinor,
    required this.discountMinor,
    required this.taxMinor,
    required this.totalMinor,
    required this.cashMinor,
    required this.cardMinor,
    required this.mobileMinor,
    required this.expectedCashMinor,
  });

  factory SessionSummary.fromJson(Map<String, dynamic> json) => SessionSummary(
        salesCount: (json['sales_count'] as num?)?.toInt() ?? 0,
        subtotalMinor: (json['subtotal_minor'] as num?)?.toInt() ?? 0,
        discountMinor: (json['discount_minor'] as num?)?.toInt() ?? 0,
        taxMinor: (json['tax_minor'] as num?)?.toInt() ?? 0,
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
        cashMinor: (json['cash_minor'] as num?)?.toInt() ?? 0,
        cardMinor: (json['card_minor'] as num?)?.toInt() ?? 0,
        mobileMinor: (json['mobile_minor'] as num?)?.toInt() ?? 0,
        expectedCashMinor: (json['expected_cash_minor'] as num?)?.toInt() ?? 0,
      );

  final int salesCount;
  final int subtotalMinor;
  final int discountMinor;
  final int taxMinor;
  final int totalMinor;
  final int cashMinor;
  final int cardMinor;
  final int mobileMinor;
  final int expectedCashMinor;
}

/// A register session: one cashier shift on one terminal.
class RegisterSession {
  const RegisterSession({
    required this.id,
    required this.status,
    required this.openingCashMinor,
    this.closingCashMinor,
    this.expectedCashMinor,
    this.cashDifferenceMinor,
    required this.openedAt,
    this.closedAt,
    required this.openedBy,
    required this.summary,
  });

  factory RegisterSession.fromJson(Map<String, dynamic> json) =>
      RegisterSession(
        id: json['id'] as String,
        status: json['status'] as String? ?? 'open',
        openingCashMinor: (json['opening_cash_minor'] as num?)?.toInt() ?? 0,
        closingCashMinor: (json['closing_cash_minor'] as num?)?.toInt(),
        expectedCashMinor: (json['expected_cash_minor'] as num?)?.toInt(),
        cashDifferenceMinor: (json['cash_difference_minor'] as num?)?.toInt(),
        openedAt: json['opened_at'] as String? ?? '',
        closedAt: json['closed_at'] as String?,
        openedBy: json['opened_by'] as String? ?? '',
        summary: SessionSummary.fromJson(
            json['summary'] as Map<String, dynamic>? ?? const {}),
      );

  final String id;
  final String status;
  final int openingCashMinor;
  final int? closingCashMinor;
  final int? expectedCashMinor;
  final int? cashDifferenceMinor;
  final String openedAt;
  final String? closedAt;
  final String openedBy;
  final SessionSummary summary;

  bool get isOpen => status == 'open';
}

/// A lean row from GET /v1/registers (list) — no payment breakdown.
class SessionListItem {
  const SessionListItem({
    required this.id,
    required this.status,
    required this.openingCashMinor,
    this.closingCashMinor,
    this.expectedCashMinor,
    this.cashDifferenceMinor,
    required this.openedAt,
    this.closedAt,
    required this.openedBy,
    required this.salesCount,
    required this.totalMinor,
  });

  factory SessionListItem.fromJson(Map<String, dynamic> json) =>
      SessionListItem(
        id: json['id'] as String,
        status: json['status'] as String? ?? 'closed',
        openingCashMinor: (json['opening_cash_minor'] as num?)?.toInt() ?? 0,
        closingCashMinor: (json['closing_cash_minor'] as num?)?.toInt(),
        expectedCashMinor: (json['expected_cash_minor'] as num?)?.toInt(),
        cashDifferenceMinor: (json['cash_difference_minor'] as num?)?.toInt(),
        openedAt: json['opened_at'] as String? ?? '',
        closedAt: json['closed_at'] as String?,
        openedBy: json['opened_by'] as String? ?? '',
        salesCount: (json['sales_count'] as num?)?.toInt() ?? 0,
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
      );

  final String id;
  final String status;
  final int openingCashMinor;
  final int? closingCashMinor;
  final int? expectedCashMinor;
  final int? cashDifferenceMinor;
  final String openedAt;
  final String? closedAt;
  final String openedBy;
  final int salesCount;
  final int totalMinor;

  bool get isOpen => status == 'open';
}

class SessionsPage {
  const SessionsPage({
    required this.sessions,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<SessionListItem> sessions;
  final int total;
  final int page;
  final int limit;
}