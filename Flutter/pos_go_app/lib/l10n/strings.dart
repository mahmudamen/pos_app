import 'package:flutter/material.dart';
import 'package:intl/intl.dart';

class AppStrings {
  AppStrings(Locale locale) : _code = locale.languageCode;

  final String _code;

  static const LocalizationsDelegate<AppStrings> delegate =
      _AppStringsDelegate();

  static const List<Locale> supportedLocales = [Locale('ar'), Locale('en')];

  static AppStrings of(BuildContext context) =>
      Localizations.of<AppStrings>(context, AppStrings)!;

  bool get isArabic => _code == 'ar';

  String get appTitle => isArabic ? 'نقطة البيع' : 'POS Go';
  String get signInSubtitle => isArabic ? 'تسجيل الدخول إلى متجرك' : 'Sign in to your store';
  String get storeId => isArabic ? 'معرف المتجر' : 'Store ID';
  String get email => isArabic ? 'البريد الإلكتروني' : 'Email';
  String get password => isArabic ? 'كلمة المرور' : 'Password';
  String get terminalName => isArabic ? 'اسم الطرفية' : 'Terminal name';
  String get signIn => isArabic ? 'تسجيل الدخول' : 'Sign in';
  String get signingIn => isArabic ? 'جارٍ تسجيل الدخول...' : 'Signing in...';
  String get required => isArabic ? 'مطلوب' : 'Required';
  String get language => isArabic ? 'اللغة' : 'Language';
  String get arabic => isArabic ? 'العربية' : 'Arabic';
  String get english => isArabic ? 'الإنجليزية' : 'English';
  String get country => isArabic ? 'البلد' : 'Country';
  String get currency => isArabic ? 'العملة' : 'Currency';
  String get settings => isArabic ? 'الإعدادات' : 'Settings';
  String get checkout => isArabic ? 'إتمام الدفع' : 'Checkout';
  String get searchProducts => isArabic ? 'ابحث عن المنتجات أو امسح الباركود' : 'Search products or scan barcode';
  String get all => isArabic ? 'الكل' : 'All';
  String get currentSale => isArabic ? 'البيع الحالي' : 'Current sale';
  String get tapToAdd => isArabic ? 'اضغط على منتج لإضافته' : 'Tap a product to add it';
  String get total => isArabic ? 'الإجمالي' : 'Total';
  String get completeSale => isArabic ? 'إتمام البيع' : 'Complete sale';
  String get saleCompleted => isArabic ? 'تم إتمام البيع' : 'Sale completed';
  String get salesHistory => isArabic ? 'سجل المبيعات' : 'Sales History';
  String get retry => isArabic ? 'إعادة المحاولة' : 'Retry';
  String get noSales => isArabic ? 'لا توجد مبيعات' : 'No sales found';
  String get pageOf => isArabic ? 'صفحة' : 'Page';
  String get signOut => isArabic ? 'تسجيل الخروج' : 'Sign out';
  String get controlPanel => isArabic ? 'لوحة التحكم' : 'Control Panel';
  String get platformSummary => isArabic ? 'ملخص المنصة' : 'Platform summary';
  String get tenants => isArabic ? 'الجهات' : 'Tenants';
  String get users => isArabic ? 'المستخدمون' : 'Users';
  String get products => isArabic ? 'المنتجات' : 'Products';
  String get totalSales => isArabic ? 'إجمالي المبيعات' : 'Total sales';
  String get revenue => isArabic ? 'الإيرادات' : 'Revenue';
  String get businessType => isArabic ? 'نوع النشاط' : 'Business type';
  String get loadFailed => isArabic ? 'تعذر تحميل البيانات' : 'Failed to load';
  String get guest => isArabic ? 'ضيف' : 'Guest';
  String get rememberLogins => isArabic ? 'تذكر بيانات تسجيل الدخول' : 'Remember logins';
  String get outOfStock => isArabic ? 'نفد المخزون' : 'Out of stock';
  String get offlineSaved => isArabic ? 'تم حفظ البيع دون اتصال وسيُزامن عند توفر الاتصال' : 'Sale saved offline — will sync when online';
  String get offlineSynced => isArabic ? 'تمت مزامنة المبيعات المحفوظة دون اتصال' : 'Offline sales synced';
  String get payment => isArabic ? 'طريقة الدفع' : 'Payment method';
  String get paymentRequired => isArabic ? 'الدفع لا يغطي قيمة المبلغ' : 'Payment does not cover the total';
  String get cash => isArabic ? 'نقدًا' : 'Cash';
  String get card => isArabic ? 'بطاقة' : 'Card';
  String get mobilePayment => isArabic ? 'محفظة' : 'Mobile wallet';
  String get confirmPayment => isArabic ? 'تأكيد الدفع' : 'Confirm payment';
  String get paymentLine => isArabic ? 'الدفع' : 'Payment';
  String get remainingLabel => isArabic ? 'المتبقي يُدفع نقدًا' : 'Remainder paid in cash';
  String get posSettings => isArabic ? 'إعدادات نقطة البيع' : 'POS settings';
  String get defaultPaymentMethod => isArabic ? 'طريقة الدفع الافتراضية' : 'Default payment method';
  String get defaultPaymentMethodHint => isArabic ? 'الطريقة المحدّدة مسبقًا عند إتمام البيع' : 'Method pre-selected at checkout';
  String get stockBadges => isArabic ? 'شارات المخزون' : 'Stock badges';
  String get stockBadgesHint => isArabic ? 'إظهار الكمية المتاحة على بطاقات المنتجات' : 'Show available quantity on product cards';
  String get receiptFooter => isArabic ? 'تذييل الإيصال' : 'Receipt footer';
  String get receiptFooterHint => isArabic ? 'رسالة تُطبع أسفل الإيصال' : 'Message printed at the bottom of receipts';
  String get account => isArabic ? 'الحساب' : 'Account';
  String get save => isArabic ? 'حفظ' : 'Save';
  String get cancel => isArabic ? 'إلغاء' : 'Cancel';
  String get saved => isArabic ? 'تم الحفظ' : 'Saved';
  String get settingsReadOnlyHint => isArabic ? 'يمكن للمالك أو المدير فقط تعديل الإعدادات' : 'Only owners and managers can edit settings';
  String get roleOwner => isArabic ? 'مالك' : 'Owner';
  String get roleManager => isArabic ? 'مدير' : 'Manager';
  String get roleCashier => isArabic ? 'أمين صندوق' : 'Cashier';
  String get splashTitle => isArabic ? 'نظام نقطة البيع' : 'Point of Sale';
  String get splashTagline => isArabic ? 'تجربة بيع عصرية وأسعار ذكية لتجارتك كل يوم' : 'A modern, smart selling experience for your business every day';
  String get poweredByXamltech => isArabic ? 'بقوة XAMLtech' : 'Powered by XAMLtech';
  String get posSession => isArabic ? 'جلسة البيع' : 'POS session';
  String get openSession => isArabic ? 'فتح جلسة' : 'Open session';
  String get openSessionTitle => isArabic ? 'ابدأ ورديتك' : 'Start your shift';
  String get openSessionHint => isArabic ? 'كمية النقد الموجودة في الدرج الآن' : 'How much cash is in the drawer now';
  String get startingCash => isArabic ? 'رصيد البدء النقدي' : 'Starting cash';
  String get noOpenSession => isArabic ? 'لا توجد جلسة مفتوحة على هذه الطرفية' : 'No open session on this terminal';
  String get resumeSession => isArabic ? 'استئناف الجلسة' : 'Resume session';
  String get finishSession => isArabic ? 'إنهاء الجلسة' : 'Finish session';
  String get finishSessionConfirm => isArabic ? 'سيُغلق درجك وتُطبع خلاصة جلسة هذا الوقت.' : 'This ends your shift and produces the sales summary.';
  String get sessionOpenedAt => isArabic ? 'الافتتاح' : 'Opened';
  String get sessionClosedAt => isArabic ? 'الإغلاق' : 'Closed';
  String get sessionOpenedBy => isArabic ? 'أمين الصندوق' : 'Cashier';
  String get countedCash => isArabic ? 'النقد المعدود' : 'Cash counted';
  String get countedCashHint => isArabic ? 'ما وجدته في الدرج عند الإغلاق (اختياري)' : 'Cash found in the drawer at closing (optional)';
  String get zReport => isArabic ? 'تقرير الجلسة' : 'Session report';
  String get sessionsHistory => isArabic ? 'سجل الجلسات' : 'Session reports';
  String get noSessions => isArabic ? 'لا توجد جلسات بعد' : 'No sessions yet';
  String get salesCount => isArabic ? 'عدد المبيعات' : 'Sales count';
  String get expectedCash => isArabic ? 'النقد المتوقع' : 'Expected cash';
  String get cashOver => isArabic ? 'زيادة' : 'Over';
  String get cashShort => isArabic ? 'عجز' : 'Short';
  String get sessionDifference => isArabic ? 'الفرق' : 'Difference';
  String get subtotalLabel => isArabic ? 'المجموع الفرعي' : 'Subtotal';
  String get discountLabel => isArabic ? 'الخصم' : 'Discount';
  String get taxLabel => isArabic ? 'الضريبة' : 'Tax';
  String get balanceEquals => isArabic ? 'التسوية صحيحة' : 'Reconciled';
  String get closeSessionError => isArabic ? 'تعذر إنهاء الجلسة' : 'Could not finish session';
  String get statusOpen => isArabic ? 'مفتوحة' : 'Open';
  String get statusClosed => isArabic ? 'مغلقة' : 'Closed';
  String get dashboard => isArabic ? 'لوحة التحكم' : 'Dashboard';
  String get revenueToday => isArabic ? 'إيرادات اليوم' : "Today's revenue";
  String get avgSale => isArabic ? 'متوسط البيع' : 'Avg. sale';
  String get itemsSold => isArabic ? 'القطع المباعة' : 'Items sold';
  String get paymentMix => isArabic ? 'مزيج الدفع' : 'Payment mix';
  String get topProducts => isArabic ? 'الأكثر مبيعًا' : 'Top products';
  String get perCashier => isArabic ? 'حسب أمين الصندوق' : 'Per cashier';
  String get recentSales => isArabic ? 'أحدث المبيعات' : 'Recent sales';

  String methodLabel(String method) {
    switch (method) {
      case 'card':
        return card;
      case 'mobile':
        return mobilePayment;
      default:
        return cash;
    }
  }

  String formatMoney(int minor, String currencyCode) {
    final symbol = currencySymbol(currencyCode);
    final formatted = (minor / 100).toStringAsFixed(2);
    if (isArabic) {
      return '$formatted $symbol';
    }
    return '$symbol$formatted';
  }

  static String currencySymbol(String code) {
    switch (code.toUpperCase()) {
      case 'EGP':
        return 'E£';
      default:
        return '\$$code';
    }
  }

  String formatDate(DateTime date) {
    final fmt = isArabic
        ? DateFormat('d MMM yyyy، HH:mm', 'ar')
        : DateFormat('MMM d, yyyy HH:mm');
    return fmt.format(date);
  }
}

class _AppStringsDelegate extends LocalizationsDelegate<AppStrings> {
  const _AppStringsDelegate();

  @override
  bool isSupported(Locale locale) =>
      locale.languageCode == 'en' || locale.languageCode == 'ar';

  @override
  Future<AppStrings> load(Locale locale) async => AppStrings(locale);

  @override
  bool shouldReload(_AppStringsDelegate old) => false;
}