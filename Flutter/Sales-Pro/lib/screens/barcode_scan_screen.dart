import 'package:flutter/material.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

class BarcodeScanScreen extends StatefulWidget {
  const BarcodeScanScreen({super.key});
  @override
  State<BarcodeScanScreen> createState() => _BarcodeScanScreenState();
}

class _BarcodeScanScreenState extends State<BarcodeScanScreen> {
  bool _handled = false;
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Barcode Scanner')),
      body: SafeArea(
        child: MobileScanner(
          onDetect: (capture) {
            if (_handled) return;
            final codes = capture.barcodes;
            if (codes.isNotEmpty) {
              _handled = true;
              Navigator.pop(context, codes.first.rawValue ?? '');
            }
          },
        ),
      ),
    );
  }
}
