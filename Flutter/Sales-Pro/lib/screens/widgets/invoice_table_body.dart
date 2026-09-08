// lib/features/invoice/widgets/invoice_table_body.dart
import 'package:sales_pro/screens/widgets/invoice_row.dart';
import 'package:sales_pro/screens/widgets/quantity_picker_dialog.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'package:sales_pro/theme/app_theme.dart';

class InvoiceTableBody extends StatelessWidget {
  final List<InvoiceRow> rows;
  final int? selectedIndex;
  final Map<int, TableColumnWidth> columnWidths;
  final bool isSmallScreen;
  final Color totalDarkGreen;
  final Color warnAmber;
  final Function(int) onDelete;
  final Function(InvoiceRow, double) onQtyChange;
  final Function(InvoiceRow, double) onPriceChange;
  final Function(InvoiceRow, String) onNoteChange;

  const InvoiceTableBody({
    Key? key,
    required this.rows,
    required this.selectedIndex,
    required this.columnWidths,
    required this.isSmallScreen,
    required this.totalDarkGreen,
    required this.warnAmber,
    required this.onDelete,
    required this.onQtyChange,
    required this.onPriceChange,
    required this.onNoteChange,
  }) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final rowsWidgets = <TableRow>[];

    for (int i = 0; i < rows.length; i++) {
      final r = rows[i];
      final isSel = selectedIndex == i;
      final bg = isSel
          ? AppColors.rowHighlight
          : (i % 2 == 0 ? Colors.white : Colors.grey.shade50);
      final qtyExceedsStock = r.qty > r.stock && r.stock >= 0;
      final outOfStock = r.stock <= 0;
      final hasNote = r.noteCtrl.text.trim().isNotEmpty;
      final nameCellBg = hasNote ? Colors.blue.shade50.withOpacity(0.3) : bg;

      final rowCells = <Widget>[];
      // Row number
      rowCells.add(_cell(Text('${i + 1}', textAlign: TextAlign.center), bg));
      // Item name cell with long press to open quick note window
      rowCells.add(
        _cell(
          GestureDetector(
            onLongPress: () async {
              HapticFeedback.selectionClick();
              await _showQuickNoteOverlay(context, r);
            },
            behavior: HitTestBehavior.opaque,
            child: Align(
              alignment: Alignment.centerLeft,
              child: Text(
                r.name.isEmpty ? '—' : r.name,
                textAlign: TextAlign.left,
                style: const TextStyle(
                  fontSize: 13,
                  height: 1.25,
                  fontWeight: FontWeight.w500,
                ),
                maxLines: 3,
                overflow: TextOverflow.ellipsis,
                softWrap: true,
              ),
            ),
          ),
          nameCellBg,
        ),
      );
      // الكمية
      rowCells.add(
        _cell(_qtyEditor(r, onQtyChange), qtyExceedsStock ? warnAmber : bg),
      );
      // السعر للوحدة
      rowCells.add(
        _cell(
          SizedBox(
            width: 90,
            child: TextField(
              controller: r.priceCtrl,
              keyboardType: TextInputType.number,
              textAlign: TextAlign.left,
              readOnly: true,
              style: const TextStyle(fontSize: 13),
              decoration: const InputDecoration(
                isDense: true,
                contentPadding: EdgeInsets.symmetric(
                  horizontal: 8,
                  vertical: 8,
                ),
                border: InputBorder.none,
              ),
              onChanged: (v) => onPriceChange(r, double.tryParse(v) ?? r.price),
            ),
          ),
          bg,
        ),
      );
      // الإجمالي
      rowCells.add(
        _cell(
          Text(
            (r.price * r.qty).toStringAsFixed(2),
            style: TextStyle(
              fontSize: 13,
              color: totalDarkGreen,
              fontWeight: FontWeight.bold,
            ),
            textAlign: TextAlign.left,
          ),
          bg,
        ),
      );
      // المخزون
      rowCells.add(
        _cell(
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              if (outOfStock)
                const Icon(
                  Icons.warning_amber_rounded,
                  size: 16,
                  color: Colors.red,
                ),
              if (outOfStock) const SizedBox(width: 4),
              Text(
                r.stock.toStringAsFixed(2),
                style: TextStyle(
                  fontSize: 13,
                  color: outOfStock ? Colors.red : null,
                ),
              ),
            ],
          ),
          outOfStock ? warnAmber : bg,
        ),
      );

      if (!isSmallScreen) {
        rowCells.insert(
          6,
          _cell(Text(r.itemId, textAlign: TextAlign.center), bg),
        );
      }

      rowCells.add(
        _cell(
          SizedBox(
            width: 180,
            child: TextField(
              controller: r.noteCtrl,
              textAlign: TextAlign.left,
              style: const TextStyle(fontSize: 13),
              decoration: const InputDecoration(
                isDense: true,
                contentPadding: EdgeInsets.symmetric(
                  horizontal: 8,
                  vertical: 8,
                ),
                border: InputBorder.none,
                hintText: 'Note...',
                hintStyle: TextStyle(color: Colors.grey),
              ),
              onChanged: (v) => onNoteChange(r, v),
            ),
          ),
          bg,
        ),
      );

      rowCells.add(
        _cell(
          Center(
            child: IconButton(
              icon: const Icon(Icons.delete, color: Colors.redAccent, size: 20),
              tooltip: 'Delete',
              onPressed: () => onDelete(i),
            ),
          ),
          bg,
        ),
      );

      rowsWidgets.add(TableRow(children: rowCells));
    }

    return Table(
      border: TableBorder.all(color: Colors.grey.shade300, width: 1),
      defaultVerticalAlignment: TableCellVerticalAlignment.middle,
      columnWidths: columnWidths,
      children: rowsWidgets,
    );
  }

  Widget _cell(Widget child, Color bg) => Container(
    height: 50,
    alignment: Alignment.center,
    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
    decoration: BoxDecoration(
      color: bg,
      borderRadius: BorderRadius.circular(6),
    ),
    child: child,
  );

  Future<void> _showQuickNoteOverlay(
    BuildContext context,
    InvoiceRow row,
  ) async {
    final existing = row.noteCtrl.text.trim();
    final firstWord = row.name.trim().isEmpty
        ? ''
        : row.name.trim().split(RegExp(r'\s+'))[0];
    final titleText = existing.isEmpty ? 'Add Note' : 'Note: $firstWord';
    final controller = TextEditingController(text: existing);

    await showDialog<String>(
      context: context,
      barrierDismissible: true,
      useSafeArea: false,
      builder: (dialogContext) {
        return Center(
          child: Material(
            color: Colors.white,
            elevation: 18,
            shadowColor: Colors.black.withOpacity(0.35),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(18),
            ),
            child: ConstrainedBox(
              constraints: BoxConstraints(
                maxWidth: 420,
                minWidth: 320,
                maxHeight: MediaQuery.of(dialogContext).size.height * 0.7,
              ),
              child: Directionality(
                textDirection: TextDirection.ltr,
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 20,
                    vertical: 18,
                  ),
                  child: SingleChildScrollView(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Align(
                          alignment: Alignment.centerLeft,
                          child: Text(
                            titleText,
                            style: const TextStyle(
                              fontSize: 16,
                              fontWeight: FontWeight.bold,
                            ),
                          ),
                        ),
                        const SizedBox(height: 14),
                        TextField(
                          controller: controller,
                          autofocus: true,
                          textDirection: TextDirection.ltr,
                          maxLines: 6,
                          decoration: InputDecoration(
                            hintText: 'Write note here...',
                            hintStyle: TextStyle(color: Colors.grey.shade500),
                            filled: true,
                            fillColor: Colors.grey.shade100,
                            border: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                              borderSide: BorderSide(
                                color: Colors.grey.shade300,
                              ),
                            ),
                            enabledBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                              borderSide: BorderSide(
                                color: Colors.grey.shade300,
                              ),
                            ),
                            focusedBorder: OutlineInputBorder(
                              borderRadius: BorderRadius.circular(12),
                              borderSide: BorderSide(
                                color: Colors.blue.shade400,
                                width: 1.4,
                              ),
                            ),
                            contentPadding: const EdgeInsets.symmetric(
                              horizontal: 14,
                              vertical: 12,
                            ),
                          ),
                          style: const TextStyle(fontSize: 14.5, height: 1.4),
                          textAlign: TextAlign.left,
                        ),
                        const SizedBox(height: 20),
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            TextButton(
                              onPressed: () =>
                                  Navigator.of(dialogContext).pop(),
                              child: const Text(
                                'Cancel',
                                style: TextStyle(
                                  fontSize: 14,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                            ),
                            ElevatedButton(
                              style: ElevatedButton.styleFrom(
                                backgroundColor: Colors.blue.shade600,
                                foregroundColor: Colors.white,
                                padding: const EdgeInsets.symmetric(
                                  horizontal: 26,
                                  vertical: 12,
                                ),
                                shape: RoundedRectangleBorder(
                                  borderRadius: BorderRadius.circular(12),
                                ),
                                elevation: 4,
                              ),
                              onPressed: () {
                                final value = controller.text.trim();
                                row.updateNote(value); // تحديث مباشر
                                onNoteChange(row, value);
                                Navigator.of(dialogContext).pop(value);
                              },
                              child: const Text(
                                'Save',
                                style: TextStyle(
                                  fontSize: 14,
                                  fontWeight: FontWeight.bold,
                                ),
                              ),
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
              ),
            ),
          ),
        );
      },
    );

    // ترك الـ controller دون التخلص الفوري لتجنب أي إعادة بناء متأخرة أثناء إغلاق الـ Dialog.
    // controller.dispose(); // يمكن تفعيله إذا ظهرت حاجة واضحة.
  }

  Widget _qtyEditor(InvoiceRow r, Function(InvoiceRow, double) onChange) {
    return Builder(
      builder: (context) {
        final qtyExceedsStock = r.qty > r.stock && r.stock >= 0;

        return InkWell(
          onTap: () async {
            final newQty = await QuantityPickerDialog.show(
              context: context,
              itemName: r.name.isEmpty ? 'Item (${r.itemId})' : r.name,
              currentQty: r.qty,
              availableStock: r.stock,
            );

            if (newQty != null && newQty > 0) {
              onChange(r, newQty);
            }
          },
          borderRadius: BorderRadius.circular(8),
          child: Container(
            width: 130,
            padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 8),
            decoration: BoxDecoration(
              color: qtyExceedsStock ? Colors.orange[50] : Colors.blue[50],
              borderRadius: BorderRadius.circular(8),
              border: Border.all(
                color: qtyExceedsStock
                    ? Colors.orange[300]!
                    : Colors.blue[300]!,
                width: 2,
              ),
            ),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.center,
              mainAxisSize: MainAxisSize.min,
              children: [
                Icon(
                  Icons.touch_app,
                  size: 16,
                  color: qtyExceedsStock
                      ? Colors.orange[900]
                      : Colors.blue[900],
                ),
                const SizedBox(width: 4),
                Flexible(
                  child: Text(
                    r.qty.toStringAsFixed(r.qty % 1 == 0 ? 0 : 2),
                    style: TextStyle(
                      fontSize: 15,
                      fontWeight: FontWeight.bold,
                      color: qtyExceedsStock
                          ? Colors.orange[900]
                          : Colors.blue[900],
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (qtyExceedsStock) ...[
                  const SizedBox(width: 4),
                  Icon(
                    Icons.warning_amber_rounded,
                    size: 14,
                    color: Colors.orange[900],
                  ),
                ],
              ],
            ),
          ),
        );
      },
    );
  }
}
