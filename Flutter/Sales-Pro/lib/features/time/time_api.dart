import 'package:xml/xml.dart' as xml;
import '../../core/soap_client.dart';

class TimeApi {
  final SoapClient _client;
  TimeApi(this._client);

  Future<String?> getTimeZoneName() async {
    final xml.XmlDocument doc = await _client.call('GetTimeZoneName', '');
    final text = doc
        .descendants
        .whereType<xml.XmlElement>()
        .firstWhere(
          (e) => e.name.local == 'GetTimeZoneNameResult',
          orElse: () => xml.XmlElement(xml.XmlName('none')),
        )
        .text
        .trim();
    return text.isEmpty ? null : text;
  }
}
