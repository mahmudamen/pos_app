enum PaymentMethod {
  cash('cash'),
  card('card'),
  mobile('mobile');

  const PaymentMethod(this.wireName);
  final String wireName;

  static PaymentMethod fromWire(String? value) {
    switch (value?.trim().toLowerCase()) {
      case 'card':
        return PaymentMethod.card;
      case 'mobile':
        return PaymentMethod.mobile;
      default:
        return PaymentMethod.cash;
    }
  }
}

class PaymentInput {
  const PaymentInput({
    required this.method,
    required this.amountMinor,
    this.tipMinor = 0,
  });

  factory PaymentInput.fromJson(Map<String, dynamic> json) => PaymentInput(
        method: PaymentMethod.fromWire(json['method'] as String?),
        amountMinor: (json['amount_minor'] as num?)?.toInt() ?? 0,
        tipMinor: (json['tip_minor'] as num?)?.toInt() ?? 0,
      );

  final PaymentMethod method;
  final int amountMinor;
  final int tipMinor;

  Map<String, Object> toJson() => {
        'method': method.wireName,
        'amount_minor': amountMinor,
        if (tipMinor > 0) 'tip_minor': tipMinor,
      };

  @override
  bool operator ==(Object other) =>
      other is PaymentInput &&
      other.method == method &&
      other.amountMinor == amountMinor &&
      other.tipMinor == tipMinor;

  @override
  int get hashCode => Object.hash(method, amountMinor, tipMinor);
}

class PaymentSplit {
  /// Allocates [totalMinor] across explicit [parts]; the remainder goes to
  /// [tender] (default cash). Pure & unit-testable:
  ///
  ///   * negative totals or parts are rejected,
  ///   * parts exceeding the total are rejected,
  ///   * zero-amount parts are dropped,
  ///   * when totalMinor == 0 an empty list is returned.
  static List<PaymentInput> allocate({
    required int totalMinor,
    required Map<PaymentMethod, int> parts,
    PaymentMethod tender = PaymentMethod.cash,
  }) {
    if (totalMinor < 0) {
      throw ArgumentError.value(totalMinor, 'totalMinor', 'must be >= 0');
    }
    var committed = 0;
    for (final entry in parts.entries) {
      if (entry.value < 0) {
        throw ArgumentError.value(
            entry.value, 'parts.$entry.key.wireName', 'must be >= 0');
      }
      committed += entry.value;
    }
    if (committed > totalMinor) {
      throw ArgumentError('parts ($committed) exceed total ($totalMinor)');
    }
    final result = <PaymentInput>[
      for (final entry in parts.entries)
        if (entry.value > 0)
          PaymentInput(method: entry.key, amountMinor: entry.value),
      if (totalMinor - committed > 0)
        PaymentInput(method: tender, amountMinor: totalMinor - committed),
    ];
    return result;
  }

  /// A single-method tender (empty when the amount is not positive).
  static List<PaymentInput> single(PaymentMethod method, int totalMinor) =>
      totalMinor <= 0
          ? const <PaymentInput>[]
          : <PaymentInput>[PaymentInput(method: method, amountMinor: totalMinor)];
}

/// Parses a user-entered decimal (e.g. "12.50") into minor units.
/// Returns 0 on empty/blank input and null on malformed or negative input.
int? minorFromInput(String input) {
  final trimmed = input.trim().replaceAll(',', '');
  if (trimmed.isEmpty) return 0;
  final parsed = double.tryParse(trimmed);
  if (parsed == null || parsed < 0) return null;
  return (parsed * 100).round();
}