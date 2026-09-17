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
  String get onboardingWelcomeTitle => isArabic ? 'مرحبًا بك في نقطة البيع' : 'Welcome to POS Go';
  String get onboardingWelcomeSubtitle => isArabic ? 'أنشئ متجرك وابدأ البيع خلال دقيقة' : 'Create your store and start selling in under a minute';
  String get onboardingFreeTrial => isArabic ? 'جرّب جميع الميزات مجانًا لمدة 15 يومًا' : 'Try every feature free for 15 days';
  String get pickBusinessType => isArabic ? 'اختر نوع متجرك لنكتب لك ديمو حقيقي' : 'Pick your store type and get a real demo catalog';
  String get coffeeShop => isArabic ? 'مقهى' : 'Coffee shop';
  String get restaurant => isArabic ? 'مطعم' : 'Restaurant';
  String get retail => isArabic ? 'متجر تجزئة' : 'Retail store';
  String get bookStore => isArabic ? 'مكتبة' : 'Bookstore';
  String get mobileShop => isArabic ? 'محل موبايل' : 'Mobile shop';
  String get computerShop => isArabic ? 'محل كمبيوتر' : 'Computer shop';
  String get grocery => isArabic ? 'بقالة وسوبر ماركت' : 'Grocery';
  String get bakery => isArabic ? 'مخبز' : 'Bakery';
  String get shawerma => isArabic ? 'شاورما' : 'Shawerma';
  String get falafel => isArabic ? 'فول وفلافل' : 'Falafel';
  String get pharmacy => isArabic ? 'صيدلية' : 'Pharmacy';
  String get butcher => isArabic ? 'جزارة' : 'Butcher';
  String get fruitsVeg => isArabic ? 'خضار وفاكهة' : 'Fruits & vegetables';
  String get clothing => isArabic ? 'ملابس' : 'Clothing';
  String get sweets => isArabic ? 'حلويات' : 'Sweets';
  String get jewelry => isArabic ? 'ذهب ومجوهرات' : 'Jewelry';
  String get hardware => isArabic ? 'عدد ومعدات' : 'Hardware';
  String get unit => isArabic ? 'الوحدة' : 'Unit';
  String unitLabel(String code) {
    if (isArabic) {
      switch (code) {
        case 'piece':
          return 'قطعة';
        case 'dozen':
          return 'درزن';
        case 'box':
          return 'علبة';
        case 'pack':
          return 'عبوة';
        case 'kg':
          return 'كجم';
        case 'g':
          return 'جرام';
        case 'liter':
          return 'لتر';
        case 'ml':
          return 'مل';
        case 'm':
          return 'متر';
        case 'qm':
          return 'م²';
        default:
          return code;
      }
    }
    switch (code) {
      case 'piece':
        return 'piece';
      case 'dozen':
        return 'dozen';
      case 'box':
        return 'box';
      case 'pack':
        return 'pack';
      case 'kg':
        return 'kg';
      case 'g':
        return 'g';
      case 'liter':
        return 'liter';
      case 'ml':
        return 'ml';
      case 'm':
        return 'm';
      case 'qm':
        return 'm²';
      default:
        return code;
    }
  }
  String get createStore => isArabic ? 'إنشاء المتجر' : 'Create store';
  String get storeName => isArabic ? 'اسم المتجر' : 'Store name';
  String get ownerName => isArabic ? 'اسم المالك' : 'Owner name';
  String get signUp => isArabic ? 'إنشاء الحساب وبدء البيع' : 'Sign up & start selling';
  String get signingUp => isArabic ? 'جارٍ إنشاء متجرك...' : 'Creating your store...';
  String get alreadyHaveStore => isArabic ? 'لديك متجر بالفعل؟ سجّل الدخول' : 'Already have a store? Sign in';
  String get passwordMin => isArabic ? 'كلمة المرور 8 أحرف على الأقل' : 'Use at least 8 characters';
  String get storeCreated => isArabic ? 'متجرك جاهز للبيع!' : 'Your store is ready to sell!';
  String get trialExpired => isArabic ? 'انتهت الفترة التجريبية لمتجرك. جدد الاشتراك للمتابعة.' : 'Your store trial has ended. Renew to continue.';
  String get continueLabel => isArabic ? 'متابعة' : 'Continue';
  String get skipLabel => isArabic ? 'لاحقًا' : 'Skip';
  String get interestTitle => isArabic ? 'ما الذي يهمك في متجرك؟' : 'What matters most for your store?';
  String get interestSubtitle => isArabic ? 'اختر الميزات التي تهمك — كلها مشمولة في التجربة المجانية' : 'Pick the features you care about — they are all included in the free trial';
  String get interestsHint => isArabic ? 'يمكنك التخطي وتحديد الخيارات لاحقًا' : 'You can skip and refine your choices later';
  String get featureInventory => isArabic ? 'إدارة المخزون' : 'Inventory control';
  String get featureLoyalty => isArabic ? 'العملاء ونقاط الولاء' : 'Customers & loyalty points';
  String get featureTables => isArabic ? 'طاولات وتقسيم الفواتير' : 'Tables & split bills';
  String get featureAnalytics => isArabic ? 'تقارير ولوحة معلومات' : 'Reports & dashboard';
  String get featureReceipts => isArabic ? 'إيصالات حرارية' : 'Thermal receipts';
  String get featureDiscounts => isArabic ? 'خصومات ورمز المدير' : 'Discounts & manager PIN';
  String get featureSync => isArabic ? 'مزامنة متعددة الأجهزة' : 'Multi-device sync';
  String get featureRefunds => isArabic ? 'الاستردادات' : 'Refunds';
  String get plansTitle => isArabic ? 'اختر باقتك' : 'Choose your plan';
  String get plansSubtitle => isArabic ? 'ابدأ مجانًا ورقِّم عندما تكبر' : 'Start free, upgrade when you grow';
  String get planTrial => isArabic ? 'تجربة مجانية' : 'Free trial';
  String get planStandard => isArabic ? 'قياسي' : 'Standard';
  String get planPremium => isArabic ? 'بريميوم' : 'Premium';
  String get planEnterprise => isArabic ? 'مؤسسات' : 'Enterprise';
  String get planPerMonth => isArabic ? '/شهر' : '/month';
  String get planPopular => isArabic ? 'الأكثر شيوعًا' : 'Most popular';
  String get startTrial => isArabic ? 'ابدأ التجربة المجانية لمدة 15 يومًا' : 'Start free 15-day trial';
  String get trialBadge => isArabic ? '15 يومًا مجانًا — كل الميزات مفعّلة' : '15 days free — every feature unlocked';
  String get trialLimitsTitle => isArabic ? 'حدود التجربة المجانية' : 'Free trial limits';
  String get trialLimitDays => isArabic ? '15 يومًا وصول كامل لجميع الميزات' : 'Full access to every feature for 15 days';
  String get trialLimitUsers => isArabic ? 'حتى 2 مستخدم' : 'Up to 2 users';
  String get trialLimitProducts => isArabic ? 'حتى 150 منتج' : 'Up to 150 products';
  String get trialLimitNoCard => isArabic ? 'بدون بطاقة — اختر باقتك لاحقًا من الإعدادات' : 'No card required — choose your plan later in Settings';
  String get planStandardDesc => isArabic ? 'لمتجرك الصغير الناشئ' : 'For your small, growing store';
  String get planPremiumDesc => isArabic ? 'للأعمال التي تنمو بسرعة' : 'For fast-growing businesses';
  String get planEnterpriseDesc => isArabic ? 'لتعدد الفروع والمؤسسات' : 'For multi-branch & enterprise';
  String get planUnlimitedUsers => isArabic ? 'مستخدمون غير محدودين' : 'Unlimited users';
  String get planUnlimitedProducts => isArabic ? 'منتجات غير محدودة' : 'Unlimited products';
  String get planExtraSupport => isArabic ? 'دعم أولوي' : 'Priority support';
  String get planAnyFeat => isArabic ? 'كل ميزات الخطة السابقة' : 'Everything in the previous plan';
  String planAfterTrialLabel(String plan) => isArabic ? 'بعد التجربة يمكنك الترقية إلى خطة $plan' : 'After the trial you can upgrade to the $plan plan';
  String planLimitsLabel(int users, int products) =>
      isArabic ? 'حتى $users مستخدمَين و$products منتج' : 'Up to $users users & $products products';
  String get trialPlanNotice => isArabic ? 'ستبدأ بتجربة مجانية محدودة: كل الميزات لمدة 15 يومًا، ثم اختر باقتك للمتابعة' : 'You will start on the free limited trial: every feature for 15 days, then choose a plan to continue';
  String get interestsShort => isArabic ? 'اهتمام' : 'interests';
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
  String get platformOverview => isArabic ? 'نظرة عامة' : 'Overview';
  String get platformTenantsTab => isArabic ? 'الجهات' : 'Tenants';
  String get platformTrialsTab => isArabic ? 'التجارب' : 'Trials';
  String get platformAuditTab => isArabic ? 'السجل' : 'Audit';
  String get searchTenants => isArabic ? 'ابحث عن جهة...' : 'Search tenants...';
  String get planLabel => isArabic ? 'خطة' : 'Plan';
  String get suspendedLabel => isArabic ? 'موقوفة' : 'Suspended';
  String get activeLabel => isArabic ? 'نشطة' : 'Active';
  String get tenantDetails => isArabic ? 'تفاصيل الجهة' : 'Tenant details';
  String get tenantAnalytics => isArabic ? 'تحليلات الجهة' : 'Tenant analytics';
  String get todayStats => isArabic ? 'إحصاءات اليوم' : "Today's stats";
  String get revenueTrend => isArabic ? 'الإيرادات (آخر 7 أيام)' : 'Revenue (7 days)';
  String get trialStatus => isArabic ? 'حالة التجربة' : 'Trial status';
  String get trialStartedAt => isArabic ? 'بداية التجربة' : 'Trial started';
  String get trialExpiresAt => isArabic ? 'نهاية التجربة' : 'Trial expires';
  String get trialSource => isArabic ? 'المصدر' : 'Source';
  String get trialType => isArabic ? 'النوع' : 'Type';
  String get extendTrial => isArabic ? 'تمديد' : 'Extend';
  String get revokeTrial => isArabic ? 'إلغاء' : 'Revoke';
  String get convertTrial => isArabic ? 'تحويل لاشتراك' : 'Convert to subscription';
  String get extendDays => isArabic ? 'عدد الأيام' : 'Extra days';
  String get extendDaysHint => isArabic ? 'أضف أيامًا للتجربة' : 'Add days to the trial';
  String get trialExtended => isArabic ? 'تم تمديد التجربة' : 'Trial extended';
  String get trialRevoked => isArabic ? 'أُلغيت التجربة' : 'Trial revoked';
  String get trialConverted => isArabic ? 'حُوِّلت التجربة إلى اشتراك' : 'Trial converted to subscription';
  String get actionFailed => isArabic ? 'تعذر تنفيذ العملية' : 'Action failed';
  String get auditLog => isArabic ? 'سجل التدقيق' : 'Audit log';
  String get noAuditEntries => isArabic ? 'لا توجد أحداث بعد' : 'No audit events yet';
  String get actionLabel => isArabic ? 'الإجراء' : 'Action';
  String get actorLabel => isArabic ? 'المُنفِّذ' : 'Actor';
  String get reasonLabel => isArabic ? 'السبب' : 'Reason';
  String get statusPending => isArabic ? 'قيد الانتظار' : 'Pending';
  String get statusActive => isArabic ? 'نشطة' : 'Active';
  String get statusExpired => isArabic ? 'منتهية' : 'Expired';
  String get statusConsumed => isArabic ? 'مُستهلَكة' : 'Consumed';
  String get statusRevoked => isArabic ? 'مُلغاة' : 'Revoked';
  String get statusConverted => isArabic ? 'مُحوَّلة' : 'Converted';
  String get trialSettings => isArabic ? 'إعدادات التجربة' : 'Trial policy';
  String get durationDaysLabel => isArabic ? 'مدة التجربة (أيام)' : 'Trial duration (days)';
  String get maxOrgsLabel => isArabic ? 'أقصى جهات للحساب' : 'Max orgs per account';
  String get maxInstallationsLabel => isArabic ? 'أقصى تثبيتات' : 'Max active installations';
  String get rateLimitLabel => isArabic ? 'تسجيلات لكل IP/ساعة' : 'Registrations per IP/hour';
  String get requireEmailVerificationLabel => isArabic ? 'اشتراط تفعيل البريد' : 'Require email verification';
  String get requirePhoneVerificationLabel => isArabic ? 'اشتراط تفعيل الهاتف' : 'Require phone verification';
  String get requireDeviceIntegrityLabel => isArabic ? 'اشتراط سلامة الجهاز' : 'Require device integrity';
  String get promoEnabledLabel => isArabic ? 'تفعيل العروض الترويجية' : 'Enable promo trials';
  String get policySaveFailed => isArabic ? 'تعذر حفظ الإعدادات' : 'Could not save settings';
  String get noTrialsYet => isArabic ? 'لا توجد تجارب بعد' : 'No trial entitlements yet';
  String get noTenantsFound => isArabic ? 'لا توجد جهات مطابقة' : 'No tenants found';
  String get plans => isArabic ? 'الخطط' : 'Plans';
  String get markets => isArabic ? 'الأسواق' : 'Markets';
  String get tenantSelectHint => isArabic ? 'اضغط لعرض التفاصيل والتحليلات' : 'Tap for details and analytics';
  String get suspendTenant => isArabic ? 'إيقاف الجهة' : 'Suspend tenant';
  String get suspendTenantHint => isArabic ? 'سبب الإيقاف (اختياري)' : 'Suspension reason (optional)';
  String get tenantSuspended => isArabic ? 'تم إيقاف الجهة' : 'Tenant suspended';
  String get loadFailed => isArabic ? 'تعذر تحميل البيانات' : 'Failed to load';
  String get guest => isArabic ? 'ضيف' : 'Guest';
  String get rememberLogins => isArabic ? 'تذكر بيانات تسجيل الدخول' : 'Remember logins';
  String get outOfStock => isArabic ? 'نفد المخزون' : 'Out of stock';
  String get offlineSaved => isArabic ? 'تم حفظ البيع دون اتصال وسيُزامن عند توفر الاتصال' : 'Sale saved offline — will sync when online';
  String get offlineSynced => isArabic ? 'تمت مزامنة المبيعات المحفوظة دون اتصال' : 'Offline sales synced';
  String get offlineNeedsAttention => isArabic ? 'تعذرت مزامنة بعض العمليات؛ راجع سجل المبيعات' : 'Some offline operations were not synced — check sales history';
  String get payment => isArabic ? 'طريقة الدفع' : 'Payment method';
  String get paymentRequired => isArabic ? 'الدفع لا يغطي قيمة المبلغ' : 'Payment does not cover the total';
  String get openOrders => isArabic ? 'الطلبات المفتوحة' : 'Open orders';
  String get newOrder => isArabic ? 'طلب جديد' : 'New order';
  String orderLabel(int number) => isArabic ? 'طلب $number' : 'Order $number';
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
  String get advancedInventory => isArabic ? 'خيارات المخزون المتقدمة' : 'Advanced inventory';
  String get allowNegativeStock => isArabic ? 'البيع بالمخزون السالب' : 'Allow negative stock';
  String get allowNegativeStockHint => isArabic ? 'السماح بإتمام البيع عند نفاد الكمية بحيث يصبح الرصيد سالبًا (طلبيات مُعلّقة) ويُعاد تغطيته عند التوريد' : 'Allow checkout past zero on-hand (backorders); balance goes negative until a refill covers it';
  String get backorder => isArabic ? 'طلبية مُعلّقة' : 'Backorder';
  String get backorderHint => isArabic ? 'المخزون سالب؛ حدّث التوريد لتغطية الطلبيات المعلّقة' : 'Negative balance — receive stock to cover backorders';
  String get negativeStockAllowed => isArabic ? 'الرصيد قد يصبح سالبًا لتغطية التالف (المخزون السالب مفعّل)' : 'Balance may go negative (negative stock is enabled)';
  String get receipt => isArabic ? 'الإيصال' : 'Receipt';
  String get printReceipt => isArabic ? 'طباعة' : 'Print';
  String get receiptPrintHint => isArabic ? 'فعِّل الطباعة من إعدادات الطابعة ثم أعد المحاولة' : 'Enable printing from the printer settings, then try again';
  String get receiptSentToPrinter => isArabic ? 'أُرسل الإيصال إلى الطابعة' : 'Receipt sent to printer';
  String get printers => isArabic ? 'الطابعات' : 'Printers';
  String get printersSettings => isArabic ? 'إعدادات الطباعة' : 'Printing preferences';
  String get printerStatus => isArabic ? 'حالة الطابعة' : 'Printer status';
  String get printingEnabled => isArabic ? 'الطباعة مفعّلة' : 'Printing enabled';
  String get printingEnabledHint => isArabic ? 'طباعة الإيصالات تلقائيًا بعد إتمام البيع (طابعة حرارية عبر الشبكة)' : 'Automatically print receipts after checkout (network thermal printer)';
  String get autoScan => isArabic ? 'فحص تلقائي' : 'Auto-scan';
  String get autoScanHint => isArabic ? 'مسح الشبكة المحلية تلقائيًا عند فتح الإعدادات (منفذ 9100)' : 'Scan the local network when opening settings (port 9100)';
  String get scanPrinters => isArabic ? 'فحص الطابعات' : 'Scan printers';
  String get scanning => isArabic ? 'جارٍ الفحص...' : 'Scanning...';
  String get noPrintersFound => isArabic ? 'لا توجد طابعات مُعرّفة بعد' : 'No printers configured yet';
  String get devices => isArabic ? 'الطابعات المُعرّفة' : 'Devices';
  String get devicesFound => isArabic ? 'طابعات مكتشفة' : 'Discovered printers';
  String get addPrinter => isArabic ? 'إضافة المُكتشَف' : 'Add found';
  String get manualIp => isArabic ? 'إضافة يدويًا (IP)' : 'Add manually (IP)';
  String get manualIpHint => isArabic ? 'مثال: 192.168.1.50' : 'e.g. 192.168.1.50';
  String get invalidIp => isArabic ? 'عنوان IP غير صالح' : 'Invalid IP address';
  String get onNetwork9100 => isArabic ? 'على الشبكة عبر المنفذ 9100' : 'Network printer (port 9100)';
  String get setDefault => isArabic ? 'تعيين' : 'Set';
  String get defaultPrinter => isArabic ? 'الافتراضي' : 'Default';
  String get paperSize => isArabic ? 'مقاس الورق' : 'Paper size';
  String get paperWidthHint => isArabic ? 'عرض أعمدة إيصال ESC/POS المطلوب من الخادم' : 'ESC/POS column width requested from the server';
  String get mm80 => isArabic ? '80 مم (32 عمودًا)' : '80 mm (32 columns)';
  String get mm58 => isArabic ? '58 مم (24 عمودًا)' : '58 mm (24 columns)';
  String get receiptTheme => isArabic ? 'شكل الإيصال' : 'Receipt theme';
  String get themeStandard => isArabic ? 'قياسي' : 'Standard';
  String get themeCompact => isArabic ? 'مضغوط' : 'Compact';
  String get copies => isArabic ? 'عدد النسخ' : 'Copies';
  String get cutAfterPrint => isArabic ? 'قصّ الورق بعد الطباعة' : 'Cut paper after printing';
  String get cashierPrintAccess => isArabic ? 'سماح أمين الصندوق بالطباعة' : 'Cashiers may print';
  String get cashierPrintAccessHint => isArabic ? 'عند تفعيلها يمكن لأمناء الصناديق طباعة الإيصالات أيضًا' : 'When on, cashiers can also print receipts';
  String get testPrint => isArabic ? 'طباعة تجريبية' : 'Test print';
  String get testPrintSent => isArabic ? 'أُرسلت الطباعة التجريبية' : 'Test page sent';
  String get printFailed => isArabic ? 'فشلت الطباعة' : 'Printing failed';
  String get printerOnline => isArabic ? 'متصل' : 'Online';
  String get printerOffline => isArabic ? 'غير متصل' : 'Offline';
  String get noPrinterConfigured => isArabic ? 'لا توجد طابعة افتراضية' : 'No default printer configured';
  String get troubleshoot => isArabic ? 'استكشاف الأخطاء' : 'Troubleshooting';
  String get troubleshootHints => isArabic ? 'تحقق من توصيل الطابعة بنفس الشبكة، ومنفذ 9100 مفتوحًا. استخدم الطباعة التجريبية للتحقق، وعدد النسخ للرجوع لطابعة الشبكة إن تعذّر الاتصال.' : 'Make sure the printer is on the same network and port 9100 is reachable. Use the test print to verify; keep copies low when testing.';
  String get account => isArabic ? 'الحساب' : 'Account';
  String get save => isArabic ? 'حفظ' : 'Save';
  String get saving => isArabic ? 'جارٍ الحفظ...' : 'Saving...';
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
  String get discountAmount => isArabic ? 'الخصم الإضافي' : 'Discount';
  String get discountAmountHint => isArabic ? 'مثال: 25.00' : 'e.g. 25.00';
  String get discountCapped => isArabic ? 'خُفِض الخصم إلى الحد المسموح لك' : 'Discount reduced to your limit';
  String get discountWarning => isArabic ? 'الخصم يتجاوز حدك المسموح وسيُعتمد مع تنبيه' : 'Discount exceeds your limit and will apply with a warning';
  String get discountNeedsPin => isArabic ? 'الخصم يتجاوز حدك؛ أدخل رمز المدير للموافقة' : 'Discount exceeds your limit — enter the manager PIN to approve';
  String get discountProhibited => isArabic ? 'لا يمكنك تطبيق خصم يتجاوز حدك المسموح' : 'Discount above your limit is not allowed';
  String get managerPin => isArabic ? 'رمز المدير' : 'Manager PIN';
  String get managerPinHint => isArabic ? 'رمز القبول المكوّن من 4-8 أرقام' : 'The 4-8 digit manager PIN';
  String get managerPinRequired => isArabic ? 'هذه العملية تتطلب رمز المدير' : 'This action requires the manager PIN';
  String get pinWrong => isArabic ? 'رمز غير صحيح' : 'Incorrect PIN';
  String get pinLocked => isArabic ? 'الرمز مؤقّت بسبب المحاولات المتكررة؛ حاول لاحقًا' : 'Locked after repeated attempts — try again later';
  String attemptsLeft(int n) => isArabic ? 'المحاولات المتبقية: $n' : '$n attempts left';
  String get lowStock => isArabic ? 'مخزون منخفض' : 'Low stock';
  String get lowStockOnly => isArabic ? 'المنخفض فقط' : 'Low stock only';
  String get blockedOutOfStock => isArabic ? 'هذا المنتج غير متوفر (نفد المخزون)' : 'Out of stock — cannot add';
  String get stockExceeded => isArabic ? 'الكمية تتجاوز المتاح في المخزون' : 'Quantity exceeds available stock';
  String get maxStockReached => isArabic ? 'بلغت الكمية المتاحة' : 'You reached the available quantity';
  String get refresh => isArabic ? 'تحديث' : 'Refresh';
  String get closeNeedsManagerPin => isArabic ? 'إنهاء الجلسة يتطلب رمز المدير' : 'Closing the session requires the manager PIN';
  String get discountNet => isArabic ? 'الخصم' : 'Discount';
  String get netTotal => isArabic ? 'الإجمالي بعد الخصم' : 'Total after discount';
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
  String get customers => isArabic ? 'العملاء' : 'Customers';
  String get customerName => isArabic ? 'اسم العميل' : 'Customer name';
  String get customerEmail => isArabic ? 'البريد الإلكتروني' : 'Email';
  String get customerPhone => isArabic ? 'الهاتف' : 'Phone';
  String get addCustomer => isArabic ? 'إضافة عميل' : 'Add customer';
  String get noCustomers => isArabic ? 'لا يوجد عملاء' : 'No customers found';
  String get searchCustomers => isArabic ? 'ابحث عن عميل...' : 'Search customers...';
  String get loyaltyPoints => isArabic ? 'نقاط الولاء' : 'Loyalty points';
  String get refund => isArabic ? 'استرداد' : 'Refund';
  String get refundSale => isArabic ? 'استرداد البيع' : 'Refund sale';
  String get refundFromHistory => isArabic ? 'استرداد' : 'Refund';
  String get refundReason => isArabic ? 'سبب الاسترداد (اختياري)' : 'Reason (optional)';
  String get refundReasonHint => isArabic ? 'مثال: إرجاع منتج معيب' : 'e.g. defective item returned';
  String get refundPin => isArabic ? 'رمز المدير (اختياري)' : 'Manager PIN (optional)';
  String get refundPinHint => isArabic ? 'مطلوب فقط إذا كان حسابك يملك رمز مدير' : 'Only needed if your account has a manager PIN';
  String get refundProcessing => isArabic ? 'جارٍ الاسترداد...' : 'Refunding...';
  String get refundSuccess => isArabic ? 'تم استرداد البيع وإعادة الكمية للمخزون' : 'Sale refunded and stock restored';
  String get refundFailed => isArabic ? 'تعذر استرداد البيع' : 'Could not refund sale';
  String get refundNotAllowed => isArabic ? 'الاسترداد متاح للمدير أو المالك فقط' : 'Only managers and owners can refund';
  String get onlyCompletedRefundable => isArabic ? 'المبيعات المكتملة فقط' : 'Only completed sales';
  String get refundedStatus => isArabic ? 'مُسترجع' : 'Refunded';
  String get table => isArabic ? 'طاولة' : 'Table';
  String get tables => isArabic ? 'الطاولات' : 'Tables';
  String get pickTable => isArabic ? 'اختر طاولة' : 'Pick a table';
  String get noTable => isArabic ? 'بدون طاولة' : 'No table';
  String get clearTable => isArabic ? 'إلغاء الطاولة' : 'Clear table';
  String get tablesLoadFailed => isArabic ? 'تعذر تحميل الطاولات' : 'Could not load tables';
  String get tableFreeStatus => isArabic ? 'فارغة' : 'Free';
  String get tableOccupiedStatus => isArabic ? 'مشغولة' : 'Occupied';
  String get tableReservedStatus => isArabic ? 'محجوزة' : 'Reserved';
  String get tableClosedStatus => isArabic ? 'مغلقة' : 'Closed';
  String get tip => isArabic ? 'الإكرامية' : 'Tip';
  String get tipHint => isArabic ? 'مثال: 10.00' : 'e.g. 10.00';
  String get grandTotal => isArabic ? 'الإجمالي مع الإكرامية' : 'Grand total';
  String get splitBill => isArabic ? 'تقسيم الفاتورة' : 'Split bill';
  String get covers => isArabic ? 'عدد القسائم' : 'Covers';
  String get coverLabel => isArabic ? 'قسيمة' : 'Cover';
  String get splitQuantityCheck => isArabic ? 'لا تتطابق الكميات مع فاتورة الأصل' : 'Quantities do not match the original sale';
  String get splitEmptyChild => isArabic ? 'كل قسيمة يجب أن تحتوي على صنف واحد على الأقل' : 'Every cover needs at least one item';
  String get splitSuccess => isArabic ? 'تم تقسيم الفاتورة ودفع كل قسيمة نقدًا' : 'Bill split and each cover paid in cash';
  String get splitFailed => isArabic ? 'تعذر تقسيم الفاتورة' : 'Could not split the bill';
  String get splitHint => isArabic ? 'وزّع العناصر على القسائم، ثم قسّم' : 'Split the items across covers, then divide';
  String get splitSaleAction => isArabic ? 'تقسيم' : 'Split';
  String get splitSavedAs => isArabic ? 'قُسّمت إلى' : 'Split into';
  String get reloadTables => isArabic ? 'إعادة المحاولة' : 'Retry';
  String get ok => isArabic ? 'حسنًا' : 'OK';
  String get focusMode => isArabic ? 'وضع التركيز' : 'Focus mode';
  String get focusModeOn => isArabic ? 'وضع التركيز مفعّل: يخفي الإشعارات ويمنع إيقاف الشاشة' : 'Focus mode on: hides notifications and keeps the screen awake';
  String get focusModeOff => isArabic ? 'تم إيقاف وضع التركيز' : 'Focus mode off';
  String get focusModeDndHint => isArabic ? 'لمنح التطبيق إذن كتم المكالمات والإشعارات عبر عدم الإزعاج' : 'Grant Do-Not-Disturb access to silence calls and notifications';
  String get focusModeGrant => isArabic ? 'منح الإذن' : 'Grant access';
  String get inventory => isArabic ? 'المخزون' : 'Inventory';
  String get onHand => isArabic ? 'المتوفر' : 'On hand';
  String get adjustStock => isArabic ? 'تعديل المخزون' : 'Adjust stock';
  String get refillPurchase => isArabic ? 'توريد / شراء' : 'Refill (purchase)';
  String get removeStock => isArabic ? 'خصم (تالف / خسارة)' : 'Remove (damage / loss)';
  String get setOnHandNow => isArabic ? 'الرصيد الفعلي الآن' : 'Set on-hand now';
  String get quantityToAdd => isArabic ? 'الكمية المضافة' : 'Quantity to add';
  String get quantityToRemove => isArabic ? 'الكمية المخصومة' : 'Quantity to remove';
  String get newOnHand => isArabic ? 'الرصيد الجديد' : 'New on-hand count';
  String get currentStock => isArabic ? 'الرصيد الحالي' : 'Current stock';
  String get stockUpdated => isArabic ? 'تم تحديث المخزون' : 'Stock updated';
  String get stockNow => isArabic ? 'الرصيد الآن' : 'On hand now';
  String get adjustmentsHistory => isArabic ? 'سجل التعديلات' : 'Adjustment log';
  String get adjustmentNote => isArabic ? 'ملاحظة (اختياري)' : 'Note (optional)';
  String get noAdjustments => isArabic ? 'لا توجد تعديلات بعد' : 'No adjustments yet';
  String get requestNoteHint => isArabic ? 'مثال: فاتورة توريد رقم ١٢' : 'e.g. supplier invoice #12';
  String get reasonRestock => isArabic ? 'توريد' : 'Restock';
  String get reasonDamaged => isArabic ? 'تالف' : 'Damaged';
  String get reasonCount => isArabic ? 'جرد' : 'Count';
  String get reasonSale => isArabic ? 'بيع' : 'Sale';
  String get cannotGoNegative => isArabic ? 'الرصيد لا يسمح بهذا الخصم' : 'Stock cannot go below zero';
  String get enteringValue => isArabic ? 'أدخل الكمية' : 'Enter a quantity';
  String get invalidQuantity => isArabic ? 'الكمية يجب أن تكون أكبر من صفر' : 'Quantity must be greater than zero';
  String get adjustConfirmation => isArabic ? 'سيتم تحديث الرصيد مباشرة' : 'Stock will be updated immediately';

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