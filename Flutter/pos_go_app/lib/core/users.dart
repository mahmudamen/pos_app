import 'security.dart';

/// A tenant staff member as returned by `GET /v1/users` and created by
/// `POST /v1/users`.
class TenantUser {
  const TenantUser({
    required this.id,
    required this.email,
    required this.displayName,
    required this.role,
    this.accountType = 'standard',
    this.isActive = true,
    this.permissions,
    this.security,
  });

  factory TenantUser.fromJson(Map<String, dynamic> json) => TenantUser(
        id: json['id'] as String? ?? '',
        email: json['email'] as String? ?? '',
        displayName: json['display_name'] as String? ?? '',
        role: json['role'] as String? ?? '',
        accountType: json['account_type'] as String? ?? 'standard',
        isActive: json['is_active'] as bool? ?? true,
        permissions: json['permissions'] is Map<String, dynamic>
            ? UserPermissions.fromJson(
                json['permissions'] as Map<String, dynamic>)
            : null,
        security: json['security'] is Map<String, dynamic>
            ? UserSecurityProfile.fromJson(
                json['security'] as Map<String, dynamic>)
            : null,
      );

  final String id;
  final String email;
  final String displayName;
  final String role;
  final String accountType;
  final bool isActive;
  final UserPermissions? permissions;

  /// The RAW stored profile row (absent for viewers without edit rights, or
  /// when the user predates the security table). Null-safe fallback for the
  /// editor: seed from [permissions] with `useCustomPermissions == false`.
  final UserSecurityProfile? security;
}

/// The stored `users_pos_security` row, verbatim — unlike [UserPermissions]
/// it is not resolved against the level defaults, so it round-trips through
/// `PUT /users/:id/security` without inventing values. `maxDiscountPct == null`
/// means "no per-user cap stored" (level default applies).
class UserSecurityProfile {
  const UserSecurityProfile({
    required this.accessLevel,
    this.maxDiscountPct,
    this.useCustomPermissions = false,
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

  /// Seeds an editor from the RESOLVED permissions when no raw row exists
  /// (pre-security-table users): level + cap come from the effective profile,
  /// every custom flag stays off until the user opts in.
  factory UserSecurityProfile.fromPermissions(UserPermissions permissions) =>
      UserSecurityProfile(
        accessLevel: permissions.accessLevel,
        maxDiscountPct: permissions.maxDiscountPct,
      );

  factory UserSecurityProfile.fromJson(Map<String, dynamic> json) =>
      UserSecurityProfile(
        accessLevel: posAccessLevelFromWire(json['access_level'] as String?),
        maxDiscountPct: (json['max_discount_pct'] as num?)?.toInt(),
        useCustomPermissions: json['use_custom_permissions'] as bool? ?? false,
        canDeleteOrder: json['can_delete_order'] as bool? ?? false,
        canDeleteLine: json['can_delete_line'] as bool? ?? false,
        canChangeQty: json['can_change_qty'] as bool? ?? false,
        canNegativeQty: json['can_negative_qty'] as bool? ?? false,
        canPriceChange: json['can_price_change'] as bool? ?? false,
        canDiscount: json['can_discount'] as bool? ?? false,
        canOpenSession: json['can_open_session'] as bool? ?? false,
        canCloseSession: json['can_close_session'] as bool? ?? false,
        canPaymentModification:
            json['can_payment_modification'] as bool? ?? false,
        canRefund: json['can_refund'] as bool? ?? false,
        canNegativeStock: json['can_negative_stock'] as bool? ?? false,
      );

  final PosAccessLevel accessLevel;
  final int? maxDiscountPct;
  final bool useCustomPermissions;
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

  Map<String, dynamic> toJson() => {
        'access_level': posAccessLevelToWire(accessLevel),
        if (maxDiscountPct != null) 'max_discount_pct': maxDiscountPct,
        'use_custom_permissions': useCustomPermissions,
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

  UserSecurityProfile copyWith({
    PosAccessLevel? accessLevel,
    int? maxDiscountPct,
    bool clearMaxDiscountPct = false,
    bool? useCustomPermissions,
    bool? canDeleteOrder,
    bool? canDeleteLine,
    bool? canChangeQty,
    bool? canNegativeQty,
    bool? canPriceChange,
    bool? canDiscount,
    bool? canOpenSession,
    bool? canCloseSession,
    bool? canPaymentModification,
    bool? canRefund,
    bool? canNegativeStock,
  }) =>
      UserSecurityProfile(
        accessLevel: accessLevel ?? this.accessLevel,
        maxDiscountPct:
            clearMaxDiscountPct ? null : (maxDiscountPct ?? this.maxDiscountPct),
        useCustomPermissions: useCustomPermissions ?? this.useCustomPermissions,
        canDeleteOrder: canDeleteOrder ?? this.canDeleteOrder,
        canDeleteLine: canDeleteLine ?? this.canDeleteLine,
        canChangeQty: canChangeQty ?? this.canChangeQty,
        canNegativeQty: canNegativeQty ?? this.canNegativeQty,
        canPriceChange: canPriceChange ?? this.canPriceChange,
        canDiscount: canDiscount ?? this.canDiscount,
        canOpenSession: canOpenSession ?? this.canOpenSession,
        canCloseSession: canCloseSession ?? this.canCloseSession,
        canPaymentModification:
            canPaymentModification ?? this.canPaymentModification,
        canRefund: canRefund ?? this.canRefund,
        canNegativeStock: canNegativeStock ?? this.canNegativeStock,
      );
}

/// One page of `GET /v1/users` (meta carries page/limit/total).
class TenantUsersPage {
  const TenantUsersPage({
    required this.users,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<TenantUser> users;
  final int total;
  final int page;
  final int limit;
}
