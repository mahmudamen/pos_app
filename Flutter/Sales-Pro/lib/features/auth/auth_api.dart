import 'package:xml/xml.dart' as xml;

import '../../core/soap_client.dart';
import '../../core/xml_utils.dart';
import 'dto/user_dto.dart';

class AuthApi {
  final SoapClient _client;
  AuthApi(this._client);

  Future<ClsUserDto> selectUser({
    required String name,
    required String password,
  }) async {
    final inner =
        '<Name>${_escape(name)}</Name><Password>${_escape(password)}</Password>';
    final doc = await _client.call('Select_User', inner);
    final res = doc.findAllByLocal('Select_UserResult').first;
    return ClsUserDto(
      idUser: res.textOf('IDUser_'),
      name: name,
      saleMan: int.tryParse(res.textOf('SaleMan_') ?? '0') ?? 0,
      idTStore: res.textOf('IDTStore_'),
      nameStore: res.textOf('Name_Store_'),
    );
  }

  Future<bool> updateUserStore({
    required int idUser,
    required int newIdTStore,
  }) async {
    final body =
        '''
      <IDUser>$idUser</IDUser>
      <NewIDTStore>$newIdTStore</NewIDTStore>
    ''';
    final xml.XmlDocument doc = await _client.call('Update_User_Store', body);
    final text = doc.descendants
        .whereType<xml.XmlElement>()
        .firstWhere(
          (e) => e.name.local == 'Update_User_StoreResult',
          orElse: () => xml.XmlElement(xml.XmlName('none')),
        )
        .text
        .trim()
        .toLowerCase();
    return text == 'true' || text == '1';
  }
}

String _escape(String s) => xml.XmlText(s).toString();
