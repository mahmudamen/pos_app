import 'dart:convert';

class ClsUserDto {
  final String idUser;
  final int saleMan;
  final String idTStore;
  final String nameStore;
  final String name;

  ClsUserDto({
    required this.idUser,
    required this.saleMan,
    required this.idTStore,
    required this.nameStore,
    required this.name,
  });

  // حالة فارغة
  factory ClsUserDto.empty() {
    return ClsUserDto(
      idUser: '',
      saleMan: 0,
      idTStore: '',
      nameStore: '',
      name: '',
    );
  }

  bool get isEmpty =>
      idUser.isEmpty &&
      saleMan == 0 &&
      idTStore.isEmpty &&
      nameStore.isEmpty &&
      name.isEmpty;

  // تحويل إلى JSON
  Map<String, dynamic> toJson() {
    return {
      'idUser': idUser,
      'saleMan': saleMan,
      'idTStore': idTStore,
      'nameStore': nameStore,
      'name': name,
    };
  }

  // تحويل من JSON
  factory ClsUserDto.fromJson(Map<String, dynamic> json) {
    return ClsUserDto(
      idUser: json['idUser']?.toString() ?? '',
      saleMan: int.tryParse(json['saleMan']?.toString() ?? '0') ?? 0,
      idTStore: json['idTStore']?.toString() ?? '',
      nameStore: json['nameStore']?.toString() ?? '',
      name: json['name']?.toString() ?? '',
    );
  }

  // نسخة مع تحديث حقل واحد
  ClsUserDto copyWith({
    String? idUser,
    int? saleMan,
    String? idTStore,
    String? nameStore,
    String? name,
  }) {
    return ClsUserDto(
      idUser: idUser ?? this.idUser,
      saleMan: saleMan ?? this.saleMan,
      idTStore: idTStore ?? this.idTStore,
      nameStore: nameStore ?? this.nameStore,
      name: name ?? this.name,
    );
  }

  @override
  String toString() {
    return 'ClsUserDto(idUser: $idUser, saleMan: $saleMan, idTStore: $idTStore, nameStore: $nameStore, name: $name)';
  }

  @override
  bool operator ==(Object other) {
    if (identical(this, other)) return true;
    return other is ClsUserDto &&
        other.idUser == idUser &&
        other.saleMan == saleMan &&
        other.idTStore == idTStore &&
        other.nameStore == nameStore &&
        other.name == name;
  }

  @override
  int get hashCode =>
      idUser.hashCode ^
      saleMan.hashCode ^
      idTStore.hashCode ^
      nameStore.hashCode ^
      name.hashCode;

  String toRawJson() => json.encode(toJson());
  factory ClsUserDto.fromRawJson(String str) =>
      ClsUserDto.fromJson(json.decode(str));
}
