import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/billing.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/saas/billing_screen.dart';
import 'package:pos_go_app/l10n/strings.dart';

const _saasAdmin = Session(
  accessToken: 'access-token',
  refreshToken: 'refresh-token',
  userId: 'user-admin',
  displayName: 'Platform Admin',
  tenantId: 'tenant-saas',
  deviceId: 'device-1',
  role: 'saas_admin',
  currencyCode: 'EGP',
);

const _tenant = SaasTenant(
  id: 't-1',
  name: 'Nile Cafe',
  slug: 'nile-cafe',
  businessType: 'restaurant',
  countryCode: 'EG',
  currencyCode: 'EGP',
  defaultLanguage: 'ar',
  users: 2,
  products: 24,
  plan: 'trial',
  status: 'active',
  revenueMinor: 8000,
  totalSales: 4,
  createdAt: '2026-09-01T09:00:00Z',
);

const _plan = BillingPlan(
  id: 'p-1',
  code: 'premium',
  name: 'Premium',
  priceMinor: 29900,
  currency: 'EGP',
  billingPeriod: 'monthly',
  isActive: true,
);

const _subscription = BillingSubscription(
  id: 'sub-1',
  tenantId: 't-1',
  tenantName: 'Nile Cafe',
  planCode: 'premium',
  planName: 'Premium',
  priceMinor: 29900,
  currency: 'EGP',
  billingPeriod: 'monthly',
  status: 'active',
  provider: 'manual',
  currentPeriodEnd: '2026-10-01T00:00:00Z',
  createdAt: '2026-09-01T09:00:00Z',
);

const _openInvoice = BillingInvoice(
  id: 'inv-1',
  tenantId: 't-1',
  tenantName: 'Nile Cafe',
  amountMinor: 30000,
  currency: 'EGP',
  status: 'open',
  provider: 'manual',
  description: 'September plan',
  createdAt: '2026-09-01T09:00:00Z',
);

const _paidInvoice = BillingInvoice(
  id: 'inv-2',
  tenantId: 't-1',
  tenantName: 'Nile Cafe',
  amountMinor: 29900,
  currency: 'EGP',
  status: 'paid',
  provider: 'manual',
  description: 'August plan',
  createdAt: '2026-08-01T09:00:00Z',
);

class _FakeBillingApi extends ApiClient {
  _FakeBillingApi();

  int payCalls = 0;
  int createUserCalls = 0;
  String? assignStatus;

  @override
  Future<BillingSubscriptionsPage> billingSubscriptions(
    Session session, {
    String? status,
    int page = 1,
    int limit = 50,
  }) async {
    return const BillingSubscriptionsPage(
      subscriptions: [_subscription],
      total: 1,
      page: 1,
      limit: 50,
    );
  }

  @override
  Future<List<BillingPlan>> billingPlans(Session session) async => const [_plan];

  @override
  Future<List<PaymentProvider>> paymentProviders(Session session) async =>
      const [PaymentProvider(name: 'manual')];

  @override
  Future<BillingSummaryData> billingSummary(Session session) async =>
      const BillingSummaryData(
        activePlans: 1,
        subscriptionCounts: {'active': 1},
        mrrMinor: 29900,
        currency: 'EGP',
        outstandingMinor: 30000,
        providers: ['manual'],
        recentInvoices: [
          BillingRecentInvoice(
            id: 'inv-1',
            tenantName: 'Nile Cafe',
            amountMinor: 30000,
            currency: 'EGP',
            status: 'open',
            createdAt: '2026-09-01T09:00:00Z',
          )
        ],
      );

  @override
  Future<SaasTenantsPage> saasTenants(
    Session session, {
    int page = 1,
    int limit = 50,
    String? q,
    String? businessType,
    String? plan,
    String? status,
    bool includeInternal = false,
  }) async {
    return SaasTenantsPage(
      tenants: const [_tenant],
      total: 1,
      page: page,
      limit: limit,
    );
  }

  @override
  Future<BillingInvoicesPage> billingInvoices(
    Session session, {
    String? status,
    int page = 1,
    int limit = 50,
  }) async {
    const all = [_openInvoice, _paidInvoice];
    final filtered = (status == null || status.isEmpty)
        ? const <BillingInvoice>[] + all
        : all
            .where((i) => i.status == status)
            .toList();
    return BillingInvoicesPage(
      invoices: filtered,
      total: filtered.length,
      page: page,
      limit: limit,
    );
  }

  @override
  Future<SaasUsersPage> saasUsers(
    Session session, {
    String? tenantId,
    int page = 1,
    int limit = 50,
  }) async {
    return const SaasUsersPage(users: [
      SaasUser(
        id: 'u-1',
        tenantId: 't-1',
        tenantName: 'Nile Cafe',
        email: 'cashier@nile-cafe.com',
        displayName: 'Sara',
        role: 'cashier',
        accountType: 'standard',
        isActive: true,
        accessLevel: 'cashier',
      ),
    ], total: 1, page: 1, limit: 50);
  }

  @override
  Future<BillingInvoice> payInvoice(
    Session session,
    String invoiceId, {
    String? provider,
  }) async {
    payCalls++;
    return _openInvoice;
  }

  @override
  Future<SaasUser> createSaasUser(
    Session session,
    String tenantId, {
    required String email,
    required String displayName,
    required String password,
    required String role,
    String? accountType,
  }) async {
    createUserCalls++;
    return const SaasUser(
      id: 'u-2',
      tenantId: 't-1',
      tenantName: 'Nile Cafe',
      email: '',
      displayName: '',
      role: 'manager',
      isActive: true,
    );
  }
}

Widget _wrap(Widget child) {
  return MaterialApp(
    locale: const Locale('en'),
    supportedLocales: AppStrings.supportedLocales,
    localizationsDelegates: const [
      AppStrings.delegate,
      GlobalMaterialLocalizations.delegate,
      GlobalWidgetsLocalizations.delegate,
      GlobalCupertinoLocalizations.delegate,
    ],
    home: child,
  );
}

void main() {
  testWidgets('billing screen lists subscriptions with summary AND mrr',
      (tester) async {
    final api = _FakeBillingApi();
    await tester.pumpWidget(
        _wrap(BillingScreen(session: _saasAdmin, apiClient: api)));
    await tester.pumpAndSettle();

    expect(find.text('Billing'), findsOneWidget);
    expect(find.text('Nile Cafe'), findsWidgets);
    expect(find.textContaining('Premium'), findsWidgets);
    expect(find.textContaining('E£299.00'), findsWidgets);
    expect(find.text('Active'), findsWidgets);
  });

  testWidgets('invoices tab banks an open invoice with the pay flow',
      (tester) async {
    final api = _FakeBillingApi();
    await tester.pumpWidget(
        _wrap(BillingScreen(session: _saasAdmin, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Invoices').last);
    await tester.pumpAndSettle();

    expect(find.textContaining('September plan'), findsOneWidget);
    expect(find.textContaining('E£300.00'), findsWidgets);

    await tester.tap(find.byIcon(Icons.payments_outlined));
    await tester.pumpAndSettle();

    expect(find.text('Pay'), findsWidgets);
    await tester.tap(find.text('Pay').last);
    await tester.pumpAndSettle();

    expect(api.payCalls, 1);
    expect(find.text('Invoice paid'), findsOneWidget);
  });

  testWidgets('users tab provisions a store user through the dialog',
      (tester) async {
    final api = _FakeBillingApi();
    await tester.pumpWidget(
        _wrap(BillingScreen(session: _saasAdmin, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Users').last);
    await tester.pumpAndSettle();

    expect(find.text('Sara'), findsOneWidget);
    expect(find.text('cashier@nile-cafe.com · Nile Cafe · cashier'),
        findsOneWidget);

    await tester.tap(find.text('Add user'));
    await tester.pumpAndSettle();

    await tester.enterText(
        find.byWidgetPredicate(
            (w) => w is TextField && w.keyboardType == TextInputType.emailAddress),
        'owner@nile-cafe.com');
    await tester.enterText(
        find.widgetWithText(TextField, 'Display name'), 'Omar');
    await tester.enterText(find.widgetWithText(TextField, 'Password'),
        'password123');
    await tester.pump();

    final confirm = find.byWidgetPredicate(
        (w) => w is FilledButton && w.onPressed != null);
    expect(confirm, findsWidgets);
    await tester.tap(confirm.last);
    await tester.pumpAndSettle();

    expect(api.createUserCalls, 1);
    expect(find.text('User created'), findsOneWidget);
  });
}