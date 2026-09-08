import 'package:sales_pro/theme/app_theme.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// نافذة منبثقة احترافية لتحديد الكمية في فاتورة المبيعات
/// مشابهة لأنظمة نور والبدر والمنار
class QuantityPickerDialog extends StatefulWidget {
  /// اسم الصنف
  final String itemName;

  /// الكمية الحالية
  final double currentQty;

  /// الرصيد المتوفر في المخزن
  final double availableStock;

  const QuantityPickerDialog({
    required this.itemName,
    required this.currentQty,
    required this.availableStock,
    Key? key,
  }) : super(key: key);

  @override
  State<QuantityPickerDialog> createState() => _QuantityPickerDialogState();

  /// دالة مساعدة لعرض النافذة
  static Future<double?> show({
    required BuildContext context,
    required String itemName,
    required double currentQty,
    required double availableStock,
  }) {
    return showDialog<double>(
      context: context,
      builder: (ctx) => QuantityPickerDialog(
        itemName: itemName,
        currentQty: currentQty,
        availableStock: availableStock,
      ),
    );
  }
}

class _QuantityPickerDialogState extends State<QuantityPickerDialog> {
  late TextEditingController _qtyController;
  late FocusNode _focusNode;
  double _quantity = 1;

  @override
  void initState() {
    super.initState();
    _quantity = widget.currentQty;
    _qtyController = TextEditingController(
      text: _quantity.toStringAsFixed(_quantity % 1 == 0 ? 0 : 2),
    );
    _focusNode = FocusNode();

    // فوكس تلقائي على حقل الكمية
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _focusNode.requestFocus();
      _qtyController.selection = TextSelection(
        baseOffset: 0,
        extentOffset: _qtyController.text.length,
      );
    });
  }

  @override
  void dispose() {
    _qtyController.dispose();
    _focusNode.dispose();
    super.dispose();
  }

  /// تحديث الكمية وعرضها في الحقل
  void _updateQuantity(double newQty) {
    if (newQty < 1) {
      _showWarning('Quantity must be at least 1');
      return;
    }
    if (newQty > widget.availableStock) {
      _showWarning(
        'Requested quantity exceeds available stock (${widget.availableStock.toStringAsFixed(0)})',
      );
      return;
    }

    setState(() {
      _quantity = newQty;
      _qtyController.text = newQty.toStringAsFixed(newQty % 1 == 0 ? 0 : 2);
    });
  }

  /// إضافة كمية إلى القيمة الحالية
  void _addQuantity(double amount) {
    final newQty = _quantity + amount;
    _updateQuantity(newQty);
  }

  /// عرض رسالة تحذير
  void _showWarning(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
          message,
          textAlign: TextAlign.right,
          style: const TextStyle(fontWeight: FontWeight.bold),
        ),
        backgroundColor: Colors.red[700],
        duration: const Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      ),
    );
  }

  /// التأكد من صحة الكمية المدخلة
  bool _validateAndConfirm() {
    final text = _qtyController.text.trim();
    final qty = double.tryParse(text);

    if (qty == null || qty < 1) {
      _showWarning('Quantity must be at least 1');
      return false;
    }

    if (qty > widget.availableStock) {
      _showWarning(
        'Requested quantity exceeds available stock (${widget.availableStock.toStringAsFixed(0)})',
      );
      return false;
    }

    _quantity = qty;
    return true;
  }

  @override
  Widget build(BuildContext context) {
    return Directionality(
      textDirection: TextDirection.ltr,
      child: Dialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
        backgroundColor: Colors.white,
        child: Container(
          constraints: const BoxConstraints(maxWidth: 450, maxHeight: 650),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              // رأس النافذة
              _buildHeader(),

              // الجسم الرئيسي
              Expanded(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 20,
                    vertical: 16,
                  ),
                  child: Column(
                    children: [
                      // عرض الرصيد المتوفر
                      _buildStockDisplay(),
                      const SizedBox(height: 20),

                      // حقل الكمية الكبير
                      _buildQuantityField(),
                      const SizedBox(height: 24),

                      // الأزرار السريعة
                      _buildQuickButtons(),
                    ],
                  ),
                ),
              ),

              // أزرار الإجراء
              _buildActionButtons(),
            ],
          ),
        ),
      ),
    );
  }

  /// رأس النافذة مع اسم الصنف
  Widget _buildHeader() {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.tabActive,
        borderRadius: const BorderRadius.only(
          topLeft: Radius.circular(20),
          topRight: Radius.circular(20),
        ),
      ),
      child: Row(
        children: [
          const Icon(
            Icons.production_quantity_limits,
            color: Colors.white,
            size: 28,
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Text(
                  'Select Quantity',
                  style: TextStyle(
                    color: Colors.white,
                    fontSize: 16,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  widget.itemName,
                  style: const TextStyle(
                    color: Colors.white,
                    fontSize: 20,
                    fontWeight: FontWeight.bold,
                  ),
                  maxLines: 2,
                  overflow: TextOverflow.ellipsis,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }

  /// عرض الرصيد المتوفر
  Widget _buildStockDisplay() {
    final isLowStock = widget.availableStock < 10;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: isLowStock ? Colors.red[50] : Colors.blue[50],
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: isLowStock ? Colors.red[300]! : Colors.blue[300]!,
          width: 2,
        ),
      ),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          if (isLowStock)
            const Icon(
              Icons.warning_amber_rounded,
              color: Colors.red,
              size: 24,
            ),
          if (isLowStock) const SizedBox(width: 8),
          Text(
            'Available Stock: ',
            style: TextStyle(
              fontSize: 18,
              fontWeight: FontWeight.w600,
              color: isLowStock ? Colors.red[900] : Colors.blue[900],
            ),
          ),
          Text(
            widget.availableStock.toStringAsFixed(
              widget.availableStock % 1 == 0 ? 0 : 2,
            ),
            style: TextStyle(
              fontSize: 24,
              fontWeight: FontWeight.bold,
              color: isLowStock ? Colors.red[900] : Colors.blue[900],
            ),
          ),
        ],
      ),
    );
  }

  /// حقل الكمية الكبير
  Widget _buildQuantityField() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
      decoration: BoxDecoration(
        color: Colors.grey[100],
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: AppColors.tabActive, width: 2),
      ),
      child: TextField(
        controller: _qtyController,
        focusNode: _focusNode,
        textAlign: TextAlign.center,
        style: const TextStyle(
          fontSize: 48,
          fontWeight: FontWeight.bold,
          color: AppColors.tabActive,
        ),
        keyboardType: const TextInputType.numberWithOptions(decimal: true),
        inputFormatters: [
          FilteringTextInputFormatter.allow(RegExp(r'^\d*\.?\d*')),
        ],
        decoration: const InputDecoration(
          border: InputBorder.none,
          hintText: '0',
          hintStyle: TextStyle(
            fontSize: 48,
            fontWeight: FontWeight.bold,
            color: Colors.grey,
          ),
        ),
        onChanged: (value) {
          final qty = double.tryParse(value);
          if (qty != null) {
            setState(() => _quantity = qty);
          }
        },
      ),
    );
  }

  /// الأزرار السريعة في شبكة
  Widget _buildQuickButtons() {
    final buttons = [
      _QuickButton(label: '1', value: 1, icon: Icons.looks_one),
      _QuickButton(label: '5', value: 5, icon: Icons.looks_5),
      _QuickButton(label: '10', value: 10, icon: Icons.filter_1),
      _QuickButton(label: '20', value: 20, icon: Icons.filter_2),
      _QuickButton(label: '6', value: 6, icon: Icons.looks_6),
      _QuickButton(label: '12', value: 12, icon: Icons.grid_view),
      _QuickButton(label: '24', value: 24, icon: Icons.view_module),
      _QuickButton(label: '50', value: 50, icon: Icons.filter_5),
      _QuickButton(label: '100', value: 100, icon: Icons.filter_9_plus),
      _QuickButton(
        label: 'Box\n+12',
        value: 12,
        icon: Icons.inbox,
        isSpecial: true,
      ),
      _QuickButton(
        label: 'Carton\n+24',
        value: 24,
        icon: Icons.all_inbox,
        isSpecial: true,
      ),
      _QuickButton(
        label: 'Clear',
        value: 0,
        icon: Icons.backspace,
        isReset: true,
      ),
    ];

    return GridView.builder(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
        crossAxisCount: 3,
        childAspectRatio: 1.2,
        crossAxisSpacing: 10,
        mainAxisSpacing: 10,
      ),
      itemCount: buttons.length,
      itemBuilder: (context, index) {
        final btn = buttons[index];
        return _buildQuickButton(
          label: btn.label,
          value: btn.value,
          icon: btn.icon,
          isSpecial: btn.isSpecial,
          isReset: btn.isReset,
        );
      },
    );
  }

  /// زر سريع واحد
  Widget _buildQuickButton({
    required String label,
    required double value,
    required IconData icon,
    bool isSpecial = false,
    bool isReset = false,
  }) {
    Color bgColor = Colors.white;
    Color textColor = AppColors.tabActive;
    Color borderColor = AppColors.tabActive;

    if (isSpecial) {
      bgColor = Colors.orange[50]!;
      textColor = Colors.orange[900]!;
      borderColor = Colors.orange[700]!;
    } else if (isReset) {
      bgColor = Colors.red[50]!;
      textColor = Colors.red[900]!;
      borderColor = Colors.red[700]!;
    }

    return Material(
      color: bgColor,
      borderRadius: BorderRadius.circular(12),
      child: InkWell(
        onTap: () {
          if (isReset) {
            _updateQuantity(1);
          } else if (isSpecial) {
            _addQuantity(value);
          } else {
            _updateQuantity(value);
          }
        },
        borderRadius: BorderRadius.circular(12),
        child: Container(
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(12),
            border: Border.all(color: borderColor, width: 2),
          ),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(icon, size: 28, color: textColor),
              const SizedBox(height: 4),
              Text(
                label,
                textAlign: TextAlign.center,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.bold,
                  color: textColor,
                  height: 1.2,
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  /// أزرار الإجراء في الأسفل
  Widget _buildActionButtons() {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: Colors.grey[100],
        borderRadius: const BorderRadius.only(
          bottomLeft: Radius.circular(20),
          bottomRight: Radius.circular(20),
        ),
      ),
      child: Row(
        children: [
          // زر الإلغاء
          Expanded(
            child: ElevatedButton(
              onPressed: () => Navigator.pop(context),
              style: ElevatedButton.styleFrom(
                backgroundColor: Colors.grey[400],
                foregroundColor: Colors.white,
                padding: const EdgeInsets.symmetric(vertical: 16),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
                elevation: 0,
              ),
              child: const Text(
                'Cancel',
                style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
              ),
            ),
          ),
          const SizedBox(width: 16),

          // زر التأكيد
          Expanded(
            flex: 2,
            child: ElevatedButton(
              onPressed: () {
                if (_validateAndConfirm()) {
                  Navigator.pop(context, _quantity);
                }
              },
              style: ElevatedButton.styleFrom(
                backgroundColor: AppColors.tabActive,
                foregroundColor: Colors.white,
                padding: const EdgeInsets.symmetric(vertical: 16),
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(12),
                ),
                elevation: 2,
              ),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.center,
                children: const [
                  Icon(Icons.check_circle, size: 24),
                  SizedBox(width: 8),
                  Text(
                    'Confirm',
                    style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

/// نموذج بيانات للزر السريع
class _QuickButton {
  final String label;
  final double value;
  final IconData icon;
  final bool isSpecial;
  final bool isReset;

  _QuickButton({
    required this.label,
    required this.value,
    required this.icon,
    this.isSpecial = false,
    this.isReset = false,
  });
}
