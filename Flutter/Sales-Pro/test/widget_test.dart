import 'package:flutter/material.dart';
import 'package:xml/xml.dart';
import 'package:http/http.dart' as http;

void main() {
  runApp(const MyApp());
}

/// Invoice Data Model
class BillItem {
  final int id;
  final String? idMat;
  final String? barcode;
  final String? name;
  final String? quantity;
  final String? idStore;
  final String? note;
  final int idList;

  BillItem({
    required this.id,
    this.idMat,
    this.barcode,
    this.name,
    this.quantity,
    this.idStore,
    this.note,
    required this.idList,
  });

  String toXml() {
    return '''
<TableQu>
  <ID_>$id</ID_>
  <id_mat_>${idMat ?? ''}</id_mat_>
  <Barcode_>${barcode ?? ''}</Barcode_>
  <Name_>${name ?? ''}</Name_>
  <Qu_>${quantity ?? ''}</Qu_>
  <ID_Store_>${idStore ?? ''}</ID_Store_>
  <Note_>${note ?? ''}</Note_>
  <IDList_>$idList</IDList_>
</TableQu>
''';
  }
}

/// Function to convert items list to XML
String buildTableQuXml(List<BillItem> items) {
  return items.map((item) => item.toXml()).join();
}

/// Mock object representing SoapClient
class SoapClient {
  static const String url =
      "http://adham1989422-001-site2.qtempurl.com/WebService1.asmx";

  /// Send SOAP request
  static Future<XmlDocument> sendSoapRequest(
    String methodName,
    Map<String, dynamic> parameters,
  ) async {
    // Build XML
    final soapBody = _buildSoapEnvelope(methodName, parameters);

    final response = await http.post(
      Uri.parse(url),
      headers: {
        'Content-Type': 'text/xml; charset=utf-8',
        'SOAPAction': "http://tempuri.org/$methodName",
      },
      body: soapBody,
    );

    if (response.statusCode != 200) {
      throw Exception('SOAP request failed: ${response.statusCode}');
    }

    return XmlDocument.parse(response.body);
  }

  /// Generate SOAP Envelope from Map
  static String _buildSoapEnvelope(
    String methodName,
    Map<String, dynamic> params,
  ) {
    final buffer = StringBuffer();
    buffer.writeln('<?xml version="1.0" encoding="utf-8"?>');
    buffer.writeln(
      '<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" '
      'xmlns:xsd="http://www.w3.org/2001/XMLSchema" '
      'xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">',
    );
    buffer.writeln('<soap:Body>');
    buffer.writeln('<$methodName xmlns="http://tempuri.org/">');

    params.forEach((key, value) {
      buffer.writeln('<$key>$value</$key>');
    });

    buffer.writeln('</$methodName>');
    buffer.writeln('</soap:Body>');
    buffer.writeln('</soap:Envelope>');
    return buffer.toString();
  }

  /// Extract result from XML
  static String? extractSoapResult(XmlDocument doc, String methodName) {
    final resultTag = '${methodName}Result';
    final element = doc.findAllElements(resultTag).firstOrNull;
    return element?.innerText;
  }
}

/// Save Invoice
Future<String?> saveBill({
  required List<BillItem> items,
  required String customerId,
  required String userId,
  required String credit,
}) async {
  try {
    final parameters = {
      'TableQu1': buildTableQuXml(items),
      'IDCu': customerId,
      'IDuser': userId,
      'credit': credit,
    };

    final document = await SoapClient.sendSoapRequest('Set_Qu', parameters);

    return SoapClient.extractSoapResult(document, 'Set_Qu');
  } catch (e) {
    print('Save bill error: $e');
    return 'Error sending invoice: $e';
  }
}

class MyApp extends StatefulWidget {
  const MyApp({super.key});

  @override
  State<MyApp> createState() => _MyAppState();
}

class _MyAppState extends State<MyApp> {
  String? result;

  Future<void> _sendBill() async {
    // Create test invoice data
    final items = [
      BillItem(
        id: 1,
        idMat: "MAT001",
        barcode: "123456789",
        name: "Item 1",
        quantity: "10",
        idStore: "Store01",
        note: "Test item 1",
        idList: 100,
      ),
      BillItem(
        id: 2,
        idMat: "MAT002",
        barcode: "987654321",
        name: "Item 2",
        quantity: "5",
        idStore: "Store02",
        note: "Test item 2",
        idList: 101,
      ),
    ];

    setState(() => result = "Sending...");

    final response = await saveBill(
      items: items,
      customerId: "CUST001",
      userId: "USER001",
      credit: "50",
    );

    setState(() => result = response ?? "No result received");
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: Scaffold(
        appBar: AppBar(title: const Text("SOAP Bill Sender")),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Column(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                ElevatedButton.icon(
                  onPressed: _sendBill,
                  icon: const Icon(Icons.send),
                  label: const Text("Send Invoice"),
                ),
                const SizedBox(height: 20),
                Text(
                  result ?? "Press to send invoice",
                  textAlign: TextAlign.center,
                  style: const TextStyle(fontSize: 16),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
