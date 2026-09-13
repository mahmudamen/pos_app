import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/l10n/strings.dart';

void main() {
  test('default locale is Arabic', () {
    const all = AppStrings.supportedLocales;
    expect(all.first, const Locale('ar'));
    expect(all, contains(const Locale('en')));
  });

  test('Arabic strings are provided for core labels', () {
    final ar = AppStrings(const Locale('ar'));
    expect(ar.isArabic, isTrue);
    expect(ar.signIn, 'تسجيل الدخول');
    expect(ar.checkout, 'إتمام الدفع');
    expect(ar.total, 'الإجمالي');
    expect(ar.rememberLogins, 'تذكر بيانات تسجيل الدخول');
    expect(ar.outOfStock, 'نفد المخزون');
    expect(ar.offlineSaved, isNotEmpty);
    expect(ar.payment, 'طريقة الدفع');
    expect(ar.cash, 'نقدًا');
    expect(ar.card, 'بطاقة');
    expect(ar.mobilePayment, 'محفظة');
    expect(ar.remainingLabel, isNotEmpty);
    expect(ar.posSettings, 'إعدادات نقطة البيع');
    expect(ar.defaultPaymentMethod, 'طريقة الدفع الافتراضية');
    expect(ar.stockBadges, 'شارات المخزون');
    expect(ar.receiptFooter, 'تذييل الإيصال');
    expect(ar.save, 'حفظ');
    expect(ar.roleOwner, 'مالك');
    expect(ar.roleManager, 'مدير');
    expect(ar.roleCashier, 'أمين صندوق');
    expect(ar.splashTitle, 'نظام نقطة البيع');
    expect(ar.splashTagline, isNotEmpty);
    expect(ar.poweredByXamltech, contains('XAMLtech'));
    expect(ar.posSession, isNotEmpty);
    expect(ar.openSession, 'فتح جلسة');
    expect(ar.startingCash, 'رصيد البدء النقدي');
    expect(ar.finishSession, 'إنهاء الجلسة');
    expect(ar.countedCash, isNotEmpty);
    expect(ar.zReport, 'تقرير الجلسة');
    expect(ar.expectedCash, 'النقد المتوقع');
    expect(ar.cashOver, 'زيادة');
    expect(ar.cashShort, 'عجز');
    expect(ar.statusOpen, 'مفتوحة');
    expect(ar.statusClosed, 'مغلقة');
    expect(ar.dashboard, 'لوحة التحكم');
    expect(ar.revenueToday, 'إيرادات اليوم');
    expect(ar.topProducts, 'الأكثر مبيعًا');
    expect(ar.perCashier, 'حسب أمين الصندوق');
    expect(ar.paymentMix, 'مزيج الدفع');
    expect(ar.methodLabel('card'), 'بطاقة');
    expect(ar.methodLabel('mobile'), 'محفظة');
    expect(ar.methodLabel('cash'), 'نقدًا');
    expect(ar.methodLabel('unknown'), 'نقدًا');
    expect(ar.customers, 'العملاء');
    expect(ar.addCustomer, 'إضافة عميل');
    expect(ar.loyaltyPoints, 'نقاط الولاء');
    expect(ar.refund, 'استرداد');
    expect(ar.refundSale, 'استرداد البيع');
    expect(ar.refundReason, isNotEmpty);
    expect(ar.refundPin, isNotEmpty);
    expect(ar.refundProcessing, isNotEmpty);
    expect(ar.refundSuccess, isNotEmpty);
    expect(ar.refundedStatus, 'مُسترجع');
  });

  test('English strings are provided for core labels', () {
    final en = AppStrings(const Locale('en'));
    expect(en.isArabic, isFalse);
    expect(en.signIn, 'Sign in');
    expect(en.checkout, 'Checkout');
    expect(en.total, 'Total');
    expect(en.rememberLogins, 'Remember logins');
    expect(en.outOfStock, 'Out of stock');
    expect(en.offlineSynced, 'Offline sales synced');
    expect(en.payment, 'Payment method');
    expect(en.cash, 'Cash');
    expect(en.card, 'Card');
    expect(en.mobilePayment, 'Mobile wallet');
    expect(en.posSettings, 'POS settings');
    expect(en.defaultPaymentMethod, 'Default payment method');
    expect(en.stockBadges, 'Stock badges');
    expect(en.receiptFooter, 'Receipt footer');
    expect(en.save, 'Save');
    expect(en.roleOwner, 'Owner');
    expect(en.roleManager, 'Manager');
    expect(en.roleCashier, 'Cashier');
    expect(en.splashTitle, 'Point of Sale');
    expect(en.splashTagline, isNotEmpty);
    expect(en.poweredByXamltech, contains('XAMLtech'));
    expect(en.posSession, 'POS session');
    expect(en.openSession, 'Open session');
    expect(en.startingCash, 'Starting cash');
    expect(en.finishSession, 'Finish session');
    expect(en.countedCash, isNotEmpty);
    expect(en.zReport, 'Session report');
    expect(en.expectedCash, 'Expected cash');
    expect(en.cashOver, 'Over');
    expect(en.cashShort, 'Short');
    expect(en.statusOpen, 'Open');
    expect(en.statusClosed, 'Closed');
    expect(en.dashboard, 'Dashboard');
    expect(en.revenueToday, "Today's revenue");
    expect(en.topProducts, 'Top products');
    expect(en.perCashier, 'Per cashier');
    expect(en.paymentMix, 'Payment mix');
    expect(en.methodLabel('card'), 'Card');
    expect(en.methodLabel('mobile'), 'Mobile wallet');
    expect(en.methodLabel('cash'), 'Cash');
    expect(en.methodLabel('unknown'), 'Cash');
    expect(en.customers, 'Customers');
    expect(en.addCustomer, 'Add customer');
    expect(en.loyaltyPoints, 'Loyalty points');
    expect(en.refund, 'Refund');
    expect(en.refundSale, 'Refund sale');
    expect(en.refundReason, isNotEmpty);
    expect(en.refundPin, isNotEmpty);
    expect(en.offlineNeedsAttention, isNotEmpty);
    expect(en.refundedStatus, 'Refunded');
  });

  test('money formatted with EGP symbol', () {
    expect(AppStrings(const Locale('en')).formatMoney(1250, 'EGP'), 'E£12.50');
    expect(AppStrings(const Locale('ar')).formatMoney(1250, 'EGP'), '12.50 E£');
  });

  test('unsupported locales fall back to English', () {
    final fr = AppStrings(const Locale('fr'));
    expect(fr.isArabic, isFalse);
    expect(fr.signIn, 'Sign in');
  });
}