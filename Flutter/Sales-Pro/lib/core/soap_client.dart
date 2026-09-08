import 'dart:convert';
import 'package:dio/dio.dart';
import 'package:xml/xml.dart' as xml;

import 'config.dart';
import 'mock_soap_responses.dart';

class SoapClient {
  final Dio _dio;
  SoapClient({Dio? dio})
    : _dio =
          dio ??
          Dio(
            BaseOptions(
              connectTimeout: const Duration(seconds: 30),
              receiveTimeout: const Duration(seconds: 30),
            ),
          );

  /// call() function ready with retries and UTF-8
  Future<xml.XmlDocument> call(String methodName, String innerBody) async {
    // ⚠️ MOCK MODE CHECK - Check BEFORE any network call
    // If we are using the example URL, return mock data immediately
    if (AppConfig.baseUrl.contains('api.example.com') ||
        AppConfig.baseUrl.contains('example.com')) {
      await Future.delayed(
        const Duration(milliseconds: 500),
      ); // Simulate network delay
      final mockXml = MockSoapResponses.getResponse(methodName);
      return xml.XmlDocument.parse(mockXml);
    }

    final envelope = _wrapEnvelope(_wrapMethod(methodName, innerBody));
    // SOAP Envelope prepared (debug info removed for production)

    const maxRetries = 3;
    for (int attempt = 1; attempt <= maxRetries; attempt++) {
      try {
        final response = await _dio.post(
          AppConfig.baseUrl,
          data: utf8.encode(envelope), // Send UTF-8
          options: Options(
            headers: {
              'Content-Type': 'text/xml; charset=utf-8',
              'SOAPAction': '${AppConfig.namespace}$methodName',
            },
          ),
        );

        final data = response.data.toString();
        // SOAP Response received (debug info removed for production)

        return xml.XmlDocument.parse(data);
      } catch (e) {
        // Attempt failed (debug info removed for production)
        if (attempt == maxRetries) rethrow; // Rethrow error after last attempt
        await Future.delayed(
          const Duration(seconds: 2),
        ); // Wait before retrying
      }
    }

    throw Exception('Failed after $maxRetries retries');
  }

  String _wrapMethod(String methodName, String inner) {
    return '<$methodName xmlns="${AppConfig.namespace}">$inner</$methodName>';
  }

  String _wrapEnvelope(String body) {
    return '<?xml version="1.0" encoding="utf-8"?>'
        '<soap:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" '
        'xmlns:xsd="http://www.w3.org/2001/XMLSchema" '
        'xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/">'
        '<soap:Body>'
        '$body'
        '</soap:Body>'
        '</soap:Envelope>';
  }
}
