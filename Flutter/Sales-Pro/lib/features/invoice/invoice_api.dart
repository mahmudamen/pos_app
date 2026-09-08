import 'package:xml/xml.dart' as xml;

import '../../core/soap_client.dart';
import '../../core/xml_utils.dart';
import 'dto/table_qu_dto.dart';

class InvoiceApi {
  final SoapClient _client;
  InvoiceApi(this._client);

  Future<String> setQu({
    required List<TableQuDto> lines,
    required String idCu,
    required String idUser,
    required String credit,
  }) async {
    final buf = StringBuffer();
    buf.write('<TableQu1>');
    for (final l in lines) {
      buf.write(l.toXml());
    }
    buf.write('</TableQu1>');
    buf.write('<IDCu>$idCu</IDCu>');
    buf.write('<IDuser>$idUser</IDuser>');
    buf.write('<credit>$credit</credit>');
       // Buffer output handled silently (debug info removed for production)
    final doc = await _client.call('Set_Qu', buf.toString());
 
    final res = doc.findAllByLocal('Set_QuResult').first.text;
    return res;
  }
}
