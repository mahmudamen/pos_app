class TableBillLineDto {
  final String id;
  final String idBill;
  final String idMat;
  final String barcode;
  final String name;
  final String quantity;
  final String dateEnd;
  final String idCustomer;
  final String note;
  final String idStore;
  final double price;
  final double stock;
  const TableBillLineDto({
    required this.id,
    required this.idBill,
    required this.idMat,
    required this.barcode,
    required this.name,
    required this.quantity,
    required this.dateEnd,
    required this.idCustomer,
    required this.note,
    required this.idStore,
    required this.price,
    required this.stock,
  });
}
