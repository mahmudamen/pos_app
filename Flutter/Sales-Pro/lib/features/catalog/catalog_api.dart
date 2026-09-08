import '../../core/soap_client.dart';
import '../../core/xml_utils.dart';
import 'dto/customer_dto.dart';
import 'dto/store_dto.dart';
import 'dto/price_dto.dart';

double _parseD(String v) {
  final t = v.trim().replaceAll(',', '.');
  return double.tryParse(t) ?? 0.0;
}

class CatalogApi {
  final SoapClient _client;
  CatalogApi(this._client);

  Future<List<TCustomerDto>> getCustomers(int manSale) async {
    final doc = await _client.call('Get_customers', '<ManSale>$manSale</ManSale>');
    final arr = doc.findAllByLocal('TCustomer');
    return [
      for (final e in arr)
        TCustomerDto(
          id: int.tryParse(e.textOf('ID_')) ?? 0,
          barcode: e.textOf('Barcode_'),
          name: e.textOf('Name_'),
          credit: _parseD(e.textOf('credit_')),
        )
    ];
  }

  Future<List<TableStoreDto>> getStores() async {
    final doc = await _client.call('Get_Store', '');
    final arr = doc.findAllByLocal('TableStore');
    return [
      for (final e in arr)
        TableStoreDto(
          id: int.tryParse(e.textOf('ID_')) ?? 0,
          name: e.textOf('Name_'),
        )
    ];
  }

  Future<List<TablePriceDto>> getPrice(int idStore) async {
    final doc = await _client.call('Get_Price', '<IDStore>$idStore</IDStore>');
    final arr = doc.findAllByLocal('TablePrice');
    return [
      for (final e in arr)
        TablePriceDto(
          id: e.textOf('ID_'),
          barcode: e.textOf('Barcode_'),
          name: e.textOf('Name_'),
          price: _parseD(e.textOf('Price_')),
          availableQty: _parseD(e.textOf('QU_')),
        )
    ];
  }
}
