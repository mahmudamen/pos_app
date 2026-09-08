import 'package:xml/xml.dart' as xml;

extension XmlNodeFind on xml.XmlNode {
  Iterable<xml.XmlElement> findAllByLocal(String name) {
    return descendants.whereType<xml.XmlElement>().where((e) => e.name.local == name);
  }

  xml.XmlElement? firstByLocal(String name) {
    return findAllByLocal(name).cast<xml.XmlElement?>().firstWhere((e) => e != null, orElse: () => null);
  }

  String textOf(String name, {String fallback = ''}) {
    final el = firstByLocal(name);
    return el?.text ?? fallback;
  }
}
