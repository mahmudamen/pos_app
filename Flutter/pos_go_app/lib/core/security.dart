/// POS security model (ma_pos_base parity): the resolved per-user permission
/// profile from `/v1/auth/login` + the pure discount-policy evaluator shared by
/// the checkout and session-close flows. Kept free of Flutter/widget types so
/// it stays trivially unit-testable.
library;

enum PosAccessLevel { unknown, none, cashier, advanced, manager, admin }

PosAccessLevel posAccessLevelFromWire(String? value) {
  switch (value) {
    case 'none':
      return PosAccessLevel.none;
    case 'cashier':
      return PosAccessLevel.cashier;
    case 'advanced':
      return PosAccessLevel.advanced;
    case 'manager':
      return PosAccessLevel.manager;
    case 'admin':
      return PosAccessLevel.admin;
    default:
      return PosAccessLevel.unknown;
  }
}

String _levelName(PosAccessLevel level) {
  switch (level) {
    case PosAccessLevel.none:
      return 'none';
    case PosAccessLevel.cashier:
      return 'cashier';
    case PosAccessLevel.advanced:
      return 'advanced';
    case PosAccessLevel.manager:
      return 'manager';
    case PosAccessLevel.admin:
      return 'admin';
    case PosAccessLevel.unknown:
      return '';
  }
}

/// The resolved operation flags for a user, as returned by the backend after
/// applying level defaults + custom overrides. `maxDiscountPct` mirrors the
/// backend's `EffectiveDiscountPct` baseline (per-user cap) and is combined
/// with the tenant-wide `pos.max_discount_pct` by [effectiveDiscountPct].
class UserPermissions {
  const UserPermissions({
    this.accessLevel = PosAccessLevel.unknown,
    this.maxDiscountPct = 0,
    this.canDeleteOrder = false,
    this.canDeleteLine = false,
    this.canChangeQty = false,
    this.canNegativeQty = false,
    this.canPriceChange = false,
    this.canDiscount = false,
    this.canOpenSession = false,
    this.canCloseSession = false,
    this.canPaymentModification = false,
    this.canRefund = false,
    this.canNegativeStock = false,
  });

  factory UserPermissions.fromJson(Map<String, dynamic> json) {
    final base = _levelDefaults(
        posAccessLevelFromWire(json['access_level'] as String?));
    return UserPermissions(
      accessLevel: posAccessLevelFromWire(json['access_level'] as String?),
      maxDiscountPct: (json['max_discount_pct'] as num?)?.toInt() ?? base.maxDiscountPct,
      canDeleteOrder: json['can_delete_order'] as bool? ?? base.canDeleteOrder,
      canDeleteLine: json['can_delete_line'] as bool? ?? base.canDeleteLine,
      canChangeQty: json['can_change_qty'] as bool? ?? base.canChangeQty,
      canNegativeQty: json['can_negative_qty'] as bool? ?? base.canNegativeQty,
      canPriceChange: json['can_price_change'] as bool? ?? base.canPriceChange,
      canDiscount: json['can_discount'] as bool? ?? base.canDiscount,
      canOpenSession: json['can_open_session'] as bool? ?? base.canOpenSession,
      canCloseSession: json['can_close_session'] as bool? ?? base.canCloseSession,
      canPaymentModification:
          json['can_payment_modification'] as bool? ?? base.canPaymentModification,
      canRefund: json['can_refund'] as bool? ?? base.canRefund,
      canNegativeStock: json['can_negative_stock'] as bool? ?? base.canNegativeStock,
    );
  }

  static const UserPermissions unknown = UserPermissions();

  final PosAccessLevel accessLevel;
  final int maxDiscountPct;
  final bool canDeleteOrder;
  final bool canDeleteLine;
  final bool canChangeQty;
  final bool canNegativeQty;
  final bool canPriceChange;
  final bool canDiscount;
  final bool canOpenSession;
  final bool canCloseSession;
  final bool canPaymentModification;
  final bool canRefund;
  final bool canNegativeStock;

  bool get isManagerLevel =>
      accessLevel == PosAccessLevel.manager || accessLevel == PosAccessLevel.admin;

  bool get isAdminLevel => accessLevel == PosAccessLevel.admin;

  Map<String, dynamic> toJson() => {
        'access_level': _levelName(accessLevel),
        'max_discount_pct': maxDiscountPct,
        'can_delete_order': canDeleteOrder,
        'can_delete_line': canDeleteLine,
        'can_change_qty': canChangeQty,
        'can_negative_qty': canNegativeQty,
        'can_price_change': canPriceChange,
        'can_discount': canDiscount,
        'can_open_session': canOpenSession,
        'can_close_session': canCloseSession,
        'can_payment_modification': canPaymentModification,
        'can_refund': canRefund,
        'can_negative_stock': canNegativeStock,
      };

  /// Mirrors the backend's `access.EffectiveDiscountPct`: the tenant-wide
  /// `pos.max_discount_pct` caps the per-user limit, admin is uncapped, and a
  /// zero global limit means "no global cap".
  int effectiveDiscountPct(int globalMaxPct) {
    if (isAdminLevel) return 100;
    if (globalMaxPct <= 0) return maxDiscountPct;
    if (maxDiscountPct <= 0) return 0;
    return maxDiscountPct < globalMaxPct ? maxDiscountPct : globalMaxPct;
  }
}

UserPermissions _levelDefaults(PosAccessLevel level) {
  switch (level) {
    case PosAccessLevel.manager:
      return const UserPermissions(
        accessLevel: PosAccessLevel.manager,
        maxDiscountPct: 50,
        canDeleteOrder: true,
        canDeleteLine: true,
        canChangeQty: true,
        canNegativeQty: true,
        canPriceChange: true,
        canDiscount: true,
        canOpenSession: true,
        canCloseSession: true,
        canPaymentModification: true,
        canRefund: true,
      );
    case PosAccessLevel.admin:
      return const UserPermissions(
        accessLevel: PosAccessLevel.admin,
        maxDiscountPct: 100,
        canDeleteOrder: true,
        canDeleteLine: true,
        canChangeQty: true,
        canNegativeQty: true,
        canPriceChange: true,
        canDiscount: true,
        canOpenSession: true,
        canCloseSession: true,
        canPaymentModification: true,
        canRefund: true,
        canNegativeStock: true,
      );
    case PosAccessLevel.advanced:
      return const UserPermissions(
        accessLevel: PosAccessLevel.advanced,
        maxDiscountPct: 10,
        canDeleteLine: true,
        canChangeQty: true,
        canDiscount: true,
        canOpenSession: true,
        canCloseSession: true,
        canPaymentModification: true,
      );
    case PosAccessLevel.cashier:
      return const UserPermissions(
        accessLevel: PosAccessLevel.cashier,
        maxDiscountPct: 5,
        canChangeQty: true,
        canDiscount: true,
        canOpenSession: true,
        canCloseSession: true,
      );
    case PosAccessLevel.none:
    case PosAccessLevel.unknown:
      return UserPermissions.unknown;
  }
}

/// Discount enforcement strategy (`pos.discount_mode`).
enum DiscountMode {
  cap,
  block,
  warn,
  unknown;

  static DiscountMode fromSetting(String? value) {
    switch (value) {
      case 'cap':
        return DiscountMode.cap;
      case 'block':
        return DiscountMode.block;
      case 'warn':
        return DiscountMode.warn;
      default:
        return DiscountMode.unknown;
    }
  }
}

enum DiscountDecisionKind {
  /// Discount within limits — proceed as entered.
  allowed,

  /// Cap mode: exceeds the actor's cap; the sale must use [clampedMinor].
  capped,

  /// Warn mode: exceeds the cap but allowed — proceed with a heads-up.
  warned,

  /// Block mode: exceeds the cap and the actor may present a manager PIN.
  needsManagerPin,

  /// Block mode: exceeds the cap and no manager override is possible.
  prohibited,

  invalidNegative,
  invalidAboveTotal,
}

class DiscountDecision {
  const DiscountDecision._(this.kind, this.discountMinor, this.clampedMinor);

  final DiscountDecisionKind kind;
  final int discountMinor;
  final int? clampedMinor;

  bool get isAllowed =>
      kind == DiscountDecisionKind.allowed ||
      kind == DiscountDecisionKind.warned;

  /// The discount to actually send, honoring a cap clamp.
  int get effectiveMinor => clampedMinor ?? discountMinor;

  static const _invalidNegative =
      DiscountDecision._(DiscountDecisionKind.invalidNegative, 0, null);
  static const _invalidAboveTotal =
      DiscountDecision._(DiscountDecisionKind.invalidAboveTotal, 0, null);
}

/// Pure discount policy, mirroring the Go `sales.EvaluateDiscount`. Callers use
/// [DiscountDecision.effectiveMinor] for the amount to submit in cap mode and
/// [DiscountDecision.kind] to drive the manager-PIN modal / warnings.
DiscountDecision evaluateDiscount({
  required DiscountMode mode,
  required int subtotalMinor,
  required int discountMinor,
  required int effectiveMaxDiscountPct,
  required bool actorIsManagerLevel,
  required bool managerOverrideEnabled,
}) {
  if (discountMinor < 0) return DiscountDecision._invalidNegative;
  if (discountMinor > subtotalMinor) return DiscountDecision._invalidAboveTotal;
  if (discountMinor == 0) {
    return DiscountDecision._(
        DiscountDecisionKind.allowed, discountMinor, null);
  }

  final capMinor = effectiveMaxDiscountPct <= 0
      ? 0
      : subtotalMinor * effectiveMaxDiscountPct ~/ 100;
  final overCap = discountMinor > capMinor;
  if (!overCap) {
    return DiscountDecision._(DiscountDecisionKind.allowed, discountMinor, null);
  }

  switch (mode) {
    case DiscountMode.cap:
      return DiscountDecision._(
          DiscountDecisionKind.capped, discountMinor, capMinor);
    case DiscountMode.warn:
      return DiscountDecision._(DiscountDecisionKind.warned, discountMinor, null);
    case DiscountMode.block:
    case DiscountMode.unknown:
      if (actorIsManagerLevel) {
        return DiscountDecision._(
            DiscountDecisionKind.allowed, discountMinor, null);
      }
      if (managerOverrideEnabled) {
        return DiscountDecision._(
            DiscountDecisionKind.needsManagerPin, discountMinor, null);
      }
      return DiscountDecision._(
          DiscountDecisionKind.prohibited, discountMinor, null);
  }
}

/// Result of `POST /v1/auth/verify-pin`.
class PinVerifyResult {
  const PinVerifyResult({
    required this.valid,
    required this.hasPin,
    required this.attemptsLeft,
    required this.locked,
  });

  factory PinVerifyResult.fromJson(Map<String, dynamic> data) => PinVerifyResult(
        valid: data['valid'] as bool? ?? false,
        hasPin: data['has_pin'] as bool? ?? false,
        attemptsLeft: (data['attempts_left'] as num?)?.toInt() ?? 0,
        locked: data['locked'] as bool? ?? false,
      );

  /// A 423 `pin_locked` response carries no data body.
  const PinVerifyResult.locked()
      : valid = false,
        hasPin = false,
        attemptsLeft = 0,
        locked = true;

  final bool valid;
  final bool hasPin;
  final int attemptsLeft;
  final bool locked;
}