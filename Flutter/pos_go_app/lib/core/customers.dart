import 'package:flutter/foundation.dart';

@immutable
class Customer {
  const Customer({
    required this.id,
    required this.name,
    this.email = '',
    this.phone = '',
    this.loyaltyPoints = 0,
    this.loyaltyPointsTotal = 0,
    this.createdAt = '',
  });

  factory Customer.fromJson(Map<String, dynamic> json) => Customer(
        id: json['id'] as String? ?? '',
        name: json['name'] as String? ?? '',
        email: json['email'] as String? ?? '',
        phone: json['phone'] as String? ?? '',
        loyaltyPoints: _toInt(json['loyalty_points']),
        loyaltyPointsTotal: _toInt(json['loyalty_points_total']),
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String name;
  final String email;
  final String phone;
  final int loyaltyPoints;
  final int loyaltyPointsTotal;
  final String createdAt;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Customer &&
          runtimeType == other.runtimeType &&
          id == other.id &&
          name == other.name &&
          email == other.email &&
          phone == other.phone &&
          loyaltyPoints == other.loyaltyPoints &&
          loyaltyPointsTotal == other.loyaltyPointsTotal;

  @override
  int get hashCode => Object.hash(id, name, email, phone, loyaltyPoints, loyaltyPointsTotal);
}

int _toInt(dynamic v) {
  if (v is int) return v;
  if (v is String) return int.tryParse(v) ?? 0;
  return 0;
}
