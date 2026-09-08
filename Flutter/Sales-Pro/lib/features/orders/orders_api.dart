import 'package:xml/xml.dart' as xml;

import '../../core/soap_client.dart';
import '../../core/xml_utils.dart';
import 'dto/t_bill_dto.dart';
import 'dto/table_bill_line_dto.dart';

double _parseD(String v) {
  final t = v.trim().replaceAll(',', '.');
  return double.tryParse(t) ?? 0.0;
}

class OrdersApi {
  final SoapClient _client;
  OrdersApi(this._client);

  Future<List<TBillDto>> getTBills({
    required int idUser,
    DateTime? date1,
  }) async {
    final formattedDate = (date1 ?? DateTime.now()).toIso8601String();

    final body =
        '''
      <IDUser>$idUser</IDUser>
      <date1>$formattedDate</date1>
    ''';

    try {
      final doc = await _client.call('GetBillCustomersByUserAndDate', body);
      final list = _parseBills(doc);

      return list..sort((a, b) => b.date.compareTo(a.date));
    } catch (e) {
      // SOAP error handled silently (debug info removed for production)
      return const <TBillDto>[];
    }
  }

  List<TBillDto> _parseBills(xml.XmlDocument doc) {
    final parents = <xml.XmlElement>{};
    final idNodes = doc.findAllByLocal('ID_');
    for (final n in idNodes) {
      if (n.parent is xml.XmlElement) parents.add(n.parent as xml.XmlElement);
    }
    if (parents.isEmpty) {
      parents.addAll(doc.findAllByLocal('TableTBill'));
      parents.addAll(doc.findAllByLocal('TBill'));
      parents.addAll(doc.findAllByLocal('Table'));
      parents.addAll(doc.findAllByLocal('NewDataSet'));
    }

    final out = <TBillDto>[];
    for (final e in parents) {
      // Error handled silently (debug info removed for production)
      var id = e.textOf('ID_');
      var date = e.textOf('Date1_');
      var total = _parseD(e.textOf('Amount_'));
      var customer = e.textOf('Customer');
      if (id.isEmpty) id = e.textOf('ID');
      if (date.isEmpty) date = e.textOf('Date');
      if (total == 0) total = _parseD(e.textOf('Total'));
      if (customer.isEmpty) customer = e.textOf('Customer');
      if (id.isEmpty) continue;
      out.add(TBillDto(id: id, date: date, total: total, customer: customer));
    }
    return out;
  }

  Future<List<TableBillLineDto>> getTBillByIdBC({required int idBC}) async {
    final body =
        '''
      <ID_BC>$idBC</ID_BC>
    ''';
    try {
      final doc = await _client.call('Get_TBill_By_IDBC', body);
      return _parseBillLines(doc);
    } catch (e) {
      // SOAP error handled silently (debug info removed for production)
      return const <TableBillLineDto>[];
    }
  }

  List<TableBillLineDto> _parseBillLines(xml.XmlDocument doc) {
    final items = <TableBillLineDto>[];
    final parents = <xml.XmlElement>{};
    parents.addAll(doc.findAllByLocal('TableBill'));
    if (parents.isEmpty) {
      parents.addAll(doc.findAllByLocal('Table'));
      parents.addAll(doc.findAllByLocal('NewDataSet'));
    }
    for (final e in parents) {
      final id = e.textOf('ID_');
      final idBill = e.textOf('ID_Bill_');
      final idMat = e.textOf('ID_Mat_');
      final barcode = e.textOf('Barcode_');
      final name = e.textOf('Name_');
      final qu = e.textOf('Qu_');
      final dataEnd = e.textOf('Data_End_');
      final idCu = e.textOf('ID_Cu_');
      final note = e.textOf('Note_');
      final idStore = e.textOf('ID_Store_');
      final price = e.textOf('Price');
      if (id.isEmpty && idBill.isEmpty && barcode.isEmpty) continue;
      items.add(
        TableBillLineDto(
          id: id,
          idBill: idBill,
          idMat: idMat,
          barcode: barcode,
          name: name,
          quantity: qu,
          dateEnd: dataEnd,
          idCustomer: idCu,
          note: note,
          idStore: idStore,
          price: double.tryParse(price) ?? 0.0,
          stock: double.tryParse(e.textOf('Stock')) ?? 0.0,
        ),
      );
    }
    return items;
  }

  Future<void> setQuXml(
    String linesXml,
    String idCu,
    String s,
    String credit,
  ) async {}
  Future<void> sendInvoiceSOAP({
    required String customerId,
    required String credit,
    required int storeId,
    required List<TableBillLineDto> lines,
  }) async {
    // Generate XML for invoices (items)
    final linesXml = StringBuffer('<TableQu1>');
    for (final line in lines) {
      linesXml.write('''
      <TableQu>
        <ID_>0</ID_>
        <id_mat_>${_escape(line.idMat)}</id_mat_>
        <Barcode_>${_escape(line.barcode)}</Barcode_>
        <Name_>${_escape(line.name)}</Name_>
        <Qu_>${_escape(line.quantity)}</Qu_>
        <ID_Store_>${_escape(line.idStore)}</ID_Store_>
        <Note_>${_escape(line.note)}</Note_>
        <IDList_>${_escape(line.id)}</IDList_>
        <Price_>${line.price}</Price_>
      </TableQu>
    ''');
    }
    linesXml.write('</TableQu1>');

    // جسم الطلب الكامل
    final body =
        '''
    $linesXml
    <IDCu>$customerId</IDCu>
    <IDuser>1</IDuser>
    <credit>$credit</credit>
  ''';

    try {
      await _client.call('Set_Qu', body);
      // SOAP invoice sent successfully (debug info removed for production)
    } catch (e) {
      // SOAP error handled silently (debug info removed for production)
      rethrow; // ليقوم AppProvider بمعالجة الخطأ
    }

    Future<List<TBillDto>> getTBills({
      required int idUser,
      DateTime? date1,
    }) async {
      final formattedDate = (date1 ?? DateTime.now()).toIso8601String();

      final body =
          '''
      <IDUser>$idUser</IDUser>
      <date1>$formattedDate</date1>
    ''';

      final doc = await _client.call('GetBillCustomersByUserAndDate', body);
      final list = _parseBills(doc);
      return list..sort((a, b) => b.date.compareTo(a.date));
    }

    // ✅ alias اختياري لسهولة الاستخدام
    Future<List<TBillDto>> getBills({required int idUser, DateTime? date1}) {
      return getTBills(idUser: idUser, date1: date1);
    }
  }
}

String _escape(String s) => xml.XmlText(s).toString();
