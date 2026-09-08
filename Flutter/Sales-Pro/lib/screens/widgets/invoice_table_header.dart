// lib/features/invoice/widgets/invoice_table_header.dart
import 'package:flutter/material.dart';
import 'package:sales_pro/theme/app_theme.dart';

class InvoiceTableHeader extends StatelessWidget {
  final Map<int, TableColumnWidth> columnWidths;
  final bool isSmallScreen;

  const InvoiceTableHeader({
    Key? key,
    required this.columnWidths,
    required this.isSmallScreen,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final headers = <Widget>[
      _headerCell('', TextAlign.center),
      _headerCell('Name', TextAlign.center),
      _headerCell('Qty', TextAlign.center),
      _headerCell('Price', TextAlign.center),
      _headerCell('Total', TextAlign.center),
      _headerCell('Stock', TextAlign.center),
    ];

    if (!isSmallScreen) {
      headers.insert(6, _headerCell('Code', TextAlign.center));
    }
    headers.add(_headerCell('Note', TextAlign.left));
    headers.add(_headerCell('Del', TextAlign.center));

    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Table(
        border: TableBorder.all(color: Colors.grey.shade300, width: 1),
        defaultVerticalAlignment: TableCellVerticalAlignment.middle,
        columnWidths: columnWidths,
        children: [TableRow(children: headers)],
      ),
    );
  }

  Widget _headerCell(String text, TextAlign align) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 10),
    color: AppColors.tableHeader,
    child: Text(
      text,
      textAlign: align,
      style: const TextStyle(fontWeight: FontWeight.w700),
    ),
  );
}
