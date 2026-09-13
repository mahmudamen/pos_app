import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/customers.dart';

void main() {
  test('customer parses numeric loyalty fields from ints', () {
    final customer = Customer.fromJson(const {
      'id': 'c-1',
      'name': 'Ahmed',
      'email': 'ahmed@example.com',
      'phone': '+201200000000',
      'loyalty_points': 12,
      'loyalty_points_total': 34,
      'created_at': '2026-01-01T00:00:00Z',
    });

    expect(customer.id, 'c-1');
    expect(customer.name, 'Ahmed');
    expect(customer.email, 'ahmed@example.com');
    expect(customer.phone, '+201200000000');
    expect(customer.loyaltyPoints, 12);
    expect(customer.loyaltyPointsTotal, 34);
    expect(customer.createdAt, '2026-01-01T00:00:00Z');
  });

  test('customer tolerates missing and string loyalty values', () {
    final customer = Customer.fromJson(const {'id': 'c-2', 'name': 'Mohamed'});

    expect(customer.loyaltyPoints, 0);
    expect(customer.loyaltyPointsTotal, 0);
    expect(customer.email, '');
    expect(customer.phone, '');
    expect(customer.createdAt, '');

    final fromStrings = Customer.fromJson(const {
      'id': 'c-3',
      'name': 'Sara',
      'loyalty_points': '7',
      'loyalty_points_total': '99',
    });
    expect(fromStrings.loyaltyPoints, 7);
    expect(fromStrings.loyaltyPointsTotal, 99);

    final withGarbage = Customer.fromJson(const {
      'id': 'c-4',
      'name': 'Omar',
      'loyalty_points': 'nope',
    });
    expect(withGarbage.loyaltyPoints, 0);
  });

  test('customer equality and hashCode track every loyalty field', () {
    const a = Customer(id: 'c-1', name: 'Ahmed', email: 'a', phone: 'p',
        loyaltyPoints: 1, loyaltyPointsTotal: 2, createdAt: 't');
    const same = Customer(id: 'c-1', name: 'Ahmed', email: 'a', phone: 'p',
        loyaltyPoints: 1, loyaltyPointsTotal: 2);

    expect(a == same, isTrue);
    expect(a.hashCode, same.hashCode);

    const differentPoints = Customer(id: 'c-1', name: 'Ahmed', email: 'a',
        phone: 'p', loyaltyPoints: 2, loyaltyPointsTotal: 2);
    const differentName = Customer(id: 'c-1', name: 'Omar', email: 'a',
        phone: 'p', loyaltyPoints: 1, loyaltyPointsTotal: 2);

    expect(a == differentPoints, isFalse);
    expect(a == differentName, isFalse);
  });
}