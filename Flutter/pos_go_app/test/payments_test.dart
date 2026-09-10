import 'package:flutter_test/flutter_test.dart';
import 'package:pos_go_app/core/payments.dart';

void main() {
  group('PaymentMethod', () {
    test('fromWire maps wire names', () {
      expect(PaymentMethod.fromWire('cash'), PaymentMethod.cash);
      expect(PaymentMethod.fromWire('card'), PaymentMethod.card);
      expect(PaymentMethod.fromWire('mobile'), PaymentMethod.mobile);
    });

    test('fromWire defaults unknown or blank to cash', () {
      expect(PaymentMethod.fromWire(null), PaymentMethod.cash);
      expect(PaymentMethod.fromWire(''), PaymentMethod.cash);
      expect(PaymentMethod.fromWire('cheque'), PaymentMethod.cash);
      expect(PaymentMethod.fromWire('CARD'), PaymentMethod.card);
    });

    test('wireName round-trips the API contract', () {
      expect(PaymentMethod.cash.wireName, 'cash');
      expect(PaymentMethod.card.wireName, 'card');
      expect(PaymentMethod.mobile.wireName, 'mobile');
    });
  });

  group('PaymentInput', () {
    test('toJson uses the API wire contract', () {
      const input = PaymentInput(method: PaymentMethod.card, amountMinor: 1250);
      expect(input.toJson(), {'method': 'card', 'amount_minor': 1250});
    });

    test('fromJson parses payment lines', () {
      final input = PaymentInput.fromJson({'method': 'mobile', 'amount_minor': 500});
      expect(input.method, PaymentMethod.mobile);
      expect(input.amountMinor, 500);
    });

    test('fromJson defaults missing fields to cash/zero', () {
      final input = PaymentInput.fromJson({});
      expect(input.method, PaymentMethod.cash);
      expect(input.amountMinor, 0);
    });
  });

  group('PaymentSplit', () {
    test('allocate puts the whole total on cash when no parts are given', () {
      final payments = PaymentSplit.allocate(totalMinor: 1250, parts: {});
      expect(payments.length, 1);
      expect(payments.single.method, PaymentMethod.cash);
      expect(payments.single.amountMinor, 1250);
    });

    test('allocate splits across methods and auto-fills the cash remainder',
        () {
      final payments = PaymentSplit.allocate(
        totalMinor: 1250,
        parts: {
          PaymentMethod.card: 400,
          PaymentMethod.mobile: 200,
        },
      );
      expect(payments,
          containsAllInOrder(const [
            PaymentInput(method: PaymentMethod.card, amountMinor: 400),
            PaymentInput(method: PaymentMethod.mobile, amountMinor: 200),
            PaymentInput(method: PaymentMethod.cash, amountMinor: 650),
          ]));
    });

    test('allocate allows a full single-method tender', () {
      final payments = PaymentSplit.allocate(
        totalMinor: 300,
        parts: {PaymentMethod.card: 300},
      );
      expect(payments.length, 1);
      expect(payments.single.method, PaymentMethod.card);
    });

    test('allocate drops zero-amount parts', () {
      final payments = PaymentSplit.allocate(
        totalMinor: 300,
        parts: {
          PaymentMethod.card: 300,
          PaymentMethod.mobile: 0,
        },
      );
      expect(payments.length, 1);
      expect(payments.single.amountMinor, 300);
    });

    test('allocate throws when parts exceed the total', () {
      expect(
        () => PaymentSplit.allocate(
          totalMinor: 100,
          parts: {PaymentMethod.card: 150},
        ),
        throwsArgumentError,
      );
    });

    test('allocate throws on negative parts and totals', () {
      expect(
        () => PaymentSplit.allocate(
            totalMinor: 100, parts: {PaymentMethod.card: -5}),
        throwsArgumentError,
      );
      expect(
        () => PaymentSplit.allocate(totalMinor: -1, parts: {}),
        throwsArgumentError,
      );
    });

    test('single always sums to the total', () {
      final cash = PaymentSplit.single(PaymentMethod.cash, 1250);
      expect(cash.single.amountMinor, 1250);
    });

    test('single returns empty for a non-positive total', () {
      expect(PaymentSplit.single(PaymentMethod.cash, 0), isEmpty);
      expect(PaymentSplit.single(PaymentMethod.cash, -10), isEmpty);
    });
  });

  group('minorFromInput', () {
    test('parses decimals into minor units', () {
      expect(minorFromInput('12.50'), 1250);
      expect(minorFromInput('12.5'), 1250);
      expect(minorFromInput('12'), 1200);
      expect(minorFromInput('0'), 0);
      expect(minorFromInput('3'), 300);
    });

    test('empty input means zero', () {
      expect(minorFromInput(''), 0);
      expect(minorFromInput('   '), 0);
    });

    test('rejects malformed or negative input', () {
      expect(minorFromInput('abc'), isNull);
      expect(minorFromInput('-5'), isNull);
      expect(minorFromInput('12.5.5'), isNull);
    });
  });
}