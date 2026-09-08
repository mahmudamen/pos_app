// lib/features/invoice/models/invoice_row.dart
import 'package:flutter/material.dart';

class InvoiceRow {
  String itemId;
  String barcode;
  String name;
  double qty;
  double stock;
  double price;
  String note;

  // Controllers for TextFields
  late final TextEditingController qtyCtrl;
  late final TextEditingController priceCtrl;
  late final TextEditingController noteCtrl;

  InvoiceRow({
    required this.itemId,
    required this.barcode,
    required this.name,
    required this.qty,
    required this.stock,
    required this.price,
    required this.note,
  }) {
    _initializeControllers();
    _setupListeners();
  }

  void _initializeControllers() {
    qtyCtrl = TextEditingController(text: qty.toStringAsFixed(0));
    priceCtrl = TextEditingController(text: price.toStringAsFixed(2));
    noteCtrl = TextEditingController(text: note);
  }

  void _setupListeners() {
    qtyCtrl.addListener(() {
      final val = double.tryParse(qtyCtrl.text.replaceAll(',', '.')) ?? 1;
      if (val > 0 && val != qty) {
        qty = val;
      }
    });

    priceCtrl.addListener(() {
      final val = double.tryParse(priceCtrl.text.replaceAll(',', '.')) ?? 0;
      if (val != price) {
        price = val;
      }
    });

    noteCtrl.addListener(() {
      note = noteCtrl.text;
    });
  }

  /// Update quantity with Controller update
  void updateQty(double newQty) {
    qty = newQty > 0 ? newQty : 1;
    qtyCtrl.text = qty.toStringAsFixed(0);
  }

  /// تحديث السعر مع تحديث الـ Controller
  void updatePrice(double newPrice) {
    price = newPrice;
    priceCtrl.text = price.toStringAsFixed(2);
  }

  /// تحديث الملاحظة
  void updateNote(String newNote) {
    note = newNote;
    noteCtrl.text = newNote;
  }

  /// تنظيف الـ Controllers
  void dispose() {
    qtyCtrl.dispose();
    priceCtrl.dispose();
    noteCtrl.dispose();
  }

  /// نسخة من الصف (مفيدة للـ Draft)
  InvoiceRow copy() {
    return InvoiceRow(
      itemId: itemId,
      barcode: barcode,
      name: name,
      qty: qty,
      stock: stock,
      price: price,
      note: note,
    );
  }
}
