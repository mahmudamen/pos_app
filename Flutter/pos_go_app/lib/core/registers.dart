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
/// A shift that exists only on this device.
///
/// [RegisterSession] is server-owned: it needs `POST /v1/registers/open`, and
/// when the network cuts that call fails outright, so the merchant cannot start
/// a shift and cannot sell. This is the offline counterpart — the cashier's
/// intent (opening float, when the shift started, when it ended) is recorded
/// locally and reconciled into a real server session once the link returns.
///
/// The id is client-generated. It is deliberately never sent as `session_id`:
/// the server would reject an unknown session. Instead a queued sale carries it
/// as `local_session_id`, and replay swaps in the server id once the matching
/// session has been opened.
class LocalRegisterSession {
  const LocalRegisterSession({
    required this.id,
    required this.openingCashMinor,
    required this.openedAt,
    this.closingCashMinor,
    this.closedAt,
  });

  factory LocalRegisterSession.open({
    required String id,
    required int openingCashMinor,
    DateTime? openedAt,
  }) =>
      LocalRegisterSession(
        id: id,
        openingCashMinor: openingCashMinor,
        openedAt: (openedAt ?? DateTime.now()).toIso8601String(),
      );

  factory LocalRegisterSession.fromRow(Map<String, Object?> row) =>
      LocalRegisterSession(
        id: row['id']! as String,
        openingCashMinor: (row['opening_cash_minor']! as num).toInt(),
        openedAt: _utcStamp((row['opened_at']! as num).toInt()),
        closingCashMinor: (row['closing_cash_minor'] as num?)?.toInt(),
        closedAt: (row['closed_at'] as num?) == null
            ? null
            : _utcStamp((row['closed_at']! as num).toInt()),
      );

  /// Reads a stored epoch back as a UTC ISO-8601 string.
  ///
  /// UTC matters for stability, not for display: [toRow] stores the epoch, so
  /// reading it back as a local time would re-emit a different wall clock on
  /// every round trip (17:00Z became 19:00 on a +02:00 device) and each pass
  /// would drift again.
  static String _utcStamp(int epochMillis) =>
      DateTime.fromMillisecondsSinceEpoch(epochMillis, isUtc: true)
          .toIso8601String();

  final String id;
  final int openingCashMinor;
  final String openedAt;
  final int? closingCashMinor;
  final String? closedAt;

  bool get isOpen => closedAt == null;

  /// Closes the shift with the counted cash, returning a new instance.
  LocalRegisterSession closed({int? countedCashMinor, DateTime? at}) =>
      LocalRegisterSession(
        id: id,
        openingCashMinor: openingCashMinor,
        openedAt: openedAt,
        closingCashMinor: countedCashMinor,
        closedAt: (at ?? DateTime.now()).toIso8601String(),
      );

  Map<String, Object?> toRow() => {
        'id': id,
        'opening_cash_minor': openingCashMinor,
        'opened_at':
            DateTime.parse(openedAt).millisecondsSinceEpoch,
        'closing_cash_minor': closingCashMinor,
        'closed_at': closedAt == null
            ? null
            : DateTime.parse(closedAt!).millisecondsSinceEpoch,
        'created_at': DateTime.parse(openedAt).millisecondsSinceEpoch,
        'status': isOpen ? 'open' : 'closed',
      };
}
