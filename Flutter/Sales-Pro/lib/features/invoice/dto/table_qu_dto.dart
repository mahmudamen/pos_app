class TableQuDto {
  final int id;
  final String idMat;
  final String barcode;
  final String name;
  final String qty;
  final String idStore;
  final String note;
  final int idList;
  final double price_;
  final double stock;
  TableQuDto({
    required this.id,
    required this.idMat,
    required this.barcode,
    required this.name,
    required this.qty,
    required this.idStore,
    required this.note,
    required this.idList,
    required this.price_,
    required this.stock,
  });

  String toXml() {
    return '<TableQu>'
        '<ID_>$id</ID_>'
        '<id_mat_>$idMat</id_mat_>'
        '<Barcode_>$barcode</Barcode_>'
        '<Name_>$name</Name_>'
        '<Qu_>$qty</Qu_>'
        '<ID_Store_>$idStore</ID_Store_>'
        '<Note_>$note</Note_>'
        '<IDList_>$idList</IDList_>'
        '<Price_>$price_</Price_>'
        '</TableQu>';
  }

  // Convert to Map (JSON)
  Map<String, dynamic> toJson() => {
    'id': id,
    'idMat': idMat,
    'barcode': barcode,
    'name': name,
    'qty': qty,
    'idStore': idStore,
    'note': note,
    'idList': idList,
    'price_': price_,
    'stock': stock,
  };

  factory TableQuDto.fromJson(Map<String, dynamic> json) {
    return TableQuDto(
      id: json['id'] as int,
      idMat: json['idMat'] as String,
      barcode: json['barcode'] as String,
      name: json['name'] as String,
      qty: json['qty'] as String,
      idStore: json['idStore'] as String,
      note: json['note'] as String? ?? '',
      idList: json['idList'] as int,
      price_: (json['price_'] != null)
          ? (json['price_'] as num).toDouble()
          : 0.0,
      stock: (json['stock'] != null) ? (json['stock'] as num).toDouble() : 0.0,
    );
  }
}
