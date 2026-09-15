import 'package:flutter_test/flutter_test.dart';
import 'package:pos_go_app/core/security.dart';

void main() {
  group('UserPermissions', () {
    test('parses resolved login payload', () {
      final p = UserPermissions.fromJson({
        'access_level': 'advanced',
        'max_discount_pct': 8,
        'can_discount': true,
        'can_refund': false,
        'can_negative_stock': false,
      });
      expect(p.accessLevel, PosAccessLevel.advanced);
      expect(p.maxDiscountPct, 8);
      expect(p.canDiscount, isTrue);
      expect(p.canRefund, isFalse);
      expect(p.isManagerLevel, isFalse);
    });

    test('level defaults apply when a flag is absent', () {
      final p = UserPermissions.fromJson({'access_level': 'manager'});
      expect(p.canRefund, isTrue);
      expect(p.canChangeQty, isTrue);
      expect(p.canNegativeStock, isFalse);
      expect(p.canDeleteOrder, isTrue);
      expect(p.maxDiscountPct, 50);
    });

    test('admin level is uncapped and manager-level', () {
      final p = UserPermissions.fromJson({'access_level': 'admin'});
      expect(p.isManagerLevel, isTrue);
      expect(p.isAdminLevel, isTrue);
      expect(p.maxDiscountPct, 100);
      expect(p.canNegativeStock, isTrue);
    });

    test('unknown level is neither manager nor capped', () {
      const p = UserPermissions.unknown;
      expect(p.isManagerLevel, isFalse);
      expect(p.isAdminLevel, isFalse);
      expect(p.effectiveDiscountPct(50), 0);
    });

    test('effectiveDiscountPct applies the global cap (min rule)', () {
      // per-user 8 < global 10 -> 8
      final p8 = UserPermissions.fromJson(
          {'access_level': 'advanced', 'max_discount_pct': 8});
      expect(p8.effectiveDiscountPct(10), 8);
      // per-user 50 > global 10 -> 10
      final p50 = UserPermissions.fromJson(
          {'access_level': 'manager', 'max_discount_pct': 50});
      expect(p50.effectiveDiscountPct(10), 10);
      // global 0 => no cap
      expect(p50.effectiveDiscountPct(0), 50);
      // admin ignores the global cap
      expect(
          UserPermissions.fromJson({'access_level': 'admin'})
              .effectiveDiscountPct(5),
          100);
    });
  });

  group('evaluateDiscount', () {
    const subtotal = 2000; // minor units

    DiscountDecision cap({required int discount}) => evaluateDiscount(
          mode: DiscountMode.cap,
          subtotalMinor: subtotal,
          discountMinor: discount,
          effectiveMaxDiscountPct: 5,
          actorIsManagerLevel: false,
          managerOverrideEnabled: true,
        );

    test('within cap is allowed with no clamp', () {
      final d = cap(discount: 90);
      expect(d.kind, DiscountDecisionKind.allowed);
      expect(d.isAllowed, isTrue);
      expect(d.effectiveMinor, 90);
    });

    test('cap mode clamps to the percentage of the subtotal', () {
      // 5% of 2000 = 100
      final d = cap(discount: 150);
      expect(d.kind, DiscountDecisionKind.capped);
      expect(d.clampedMinor, 100);
      expect(d.effectiveMinor, 100);
      expect(d.isAllowed, isFalse);
    });

    test('cap mode uses integer floor like the backend', () {
      final d = evaluateDiscount(
        mode: DiscountMode.cap,
        subtotalMinor: 1000,
        discountMinor: 60,
        effectiveMaxDiscountPct: 5,
        actorIsManagerLevel: false,
        managerOverrideEnabled: true,
      );
      expect(d.kind, DiscountDecisionKind.capped);
      expect(d.clampedMinor, 50);
    });

    test('warn mode allows over-cap with a warning', () {
      final d = evaluateDiscount(
        mode: DiscountMode.warn,
        subtotalMinor: subtotal,
        discountMinor: 150,
        effectiveMaxDiscountPct: 5,
        actorIsManagerLevel: false,
        managerOverrideEnabled: true,
      );
      expect(d.kind, DiscountDecisionKind.warned);
      expect(d.isAllowed, isTrue);
      expect(d.effectiveMinor, 150);
    });

    test('block mode requires a manager PIN for cashiers', () {
      final d = evaluateDiscount(
        mode: DiscountMode.block,
        subtotalMinor: subtotal,
        discountMinor: 150,
        effectiveMaxDiscountPct: 5,
        actorIsManagerLevel: false,
        managerOverrideEnabled: true,
      );
      expect(d.kind, DiscountDecisionKind.needsManagerPin);
    });

    test('block mode without manager override is prohibited', () {
      final d = evaluateDiscount(
        mode: DiscountMode.block,
        subtotalMinor: subtotal,
        discountMinor: 150,
        effectiveMaxDiscountPct: 5,
        actorIsManagerLevel: false,
        managerOverrideEnabled: false,
      );
      expect(d.kind, DiscountDecisionKind.prohibited);
    });

    test('manager actors bypass the block mode cap', () {
      final d = evaluateDiscount(
        mode: DiscountMode.block,
        subtotalMinor: subtotal,
        discountMinor: 150,
        effectiveMaxDiscountPct: 5,
        actorIsManagerLevel: true,
        managerOverrideEnabled: true,
      );
      expect(d.kind, DiscountDecisionKind.allowed);
    });

    test('under-cap discounts are always allowed in block mode', () {
      final d = evaluateDiscount(
        mode: DiscountMode.block,
        subtotalMinor: subtotal,
        discountMinor: 100,
        effectiveMaxDiscountPct: 5,
        actorIsManagerLevel: false,
        managerOverrideEnabled: true,
      );
      expect(d.kind, DiscountDecisionKind.allowed);
    });

    test('zero discount is always allowed', () {
      final d = cap(discount: 0);
      expect(d.kind, DiscountDecisionKind.allowed);
    });

    test('negative and above-total discounts are rejected', () {
      expect(cap(discount: -1).kind, DiscountDecisionKind.invalidNegative);
      expect(
          cap(discount: subtotal + 1).kind, DiscountDecisionKind.invalidAboveTotal);
    });

    test('cap of zero percent means no discount passes', () {
      final d = evaluateDiscount(
        mode: DiscountMode.cap,
        subtotalMinor: subtotal,
        discountMinor: 1,
        effectiveMaxDiscountPct: 0,
        actorIsManagerLevel: false,
        managerOverrideEnabled: true,
      );
      expect(d.kind, DiscountDecisionKind.capped);
      expect(d.clampedMinor, 0);
    });
  });

  group('PinVerifyResult', () {
    test('parses a valid response', () {
      final r = PinVerifyResult.fromJson({'valid': true, 'has_pin': true});
      expect(r.valid, isTrue);
      expect(r.hasPin, isTrue);
      expect(r.locked, isFalse);
    });

    test('parses attempts remaining', () {
      final r = PinVerifyResult.fromJson({'valid': false, 'attempts_left': 2});
      expect(r.attemptsLeft, 2);
      expect(r.valid, isFalse);
    });

    test('locked state has no data', () {
      const r = PinVerifyResult.locked();
      expect(r.locked, isTrue);
      expect(r.valid, isFalse);
      expect(r.attemptsLeft, 0);
    });
  });
}