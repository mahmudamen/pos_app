import '../../core/soap_client.dart';
import '../../core/xml_utils.dart';

class BalanceApi {
  final SoapClient _client;
  BalanceApi(this._client);

  Future<double> getBalance(String saleMan) async {
    if (saleMan == "0" || saleMan.isEmpty) {
      return 0.0;
    }
    final doc = await _client.call(
      'GetBalance',
      '<SaleMan_>$saleMan</SaleMan_>',
    );
    final nodes = doc.findAllByLocal('GetBalanceResult');
    if (nodes.isEmpty) return 0.0;
    final v = nodes.first.text;
    return double.tryParse(v) ?? 0.0;
  }
}
