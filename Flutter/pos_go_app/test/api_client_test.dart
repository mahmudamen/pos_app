import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/customers.dart';
import 'package:pos_go_app/core/dashboard.dart';
import 'package:pos_go_app/core/inventory.dart';
import 'package:pos_go_app/core/payments.dart';
import 'package:pos_go_app/core/receipts.dart';
import 'package:pos_go_app/core/registers.dart';
import 'package:pos_go_app/core/restaurants.dart';
import 'package:pos_go_app/core/saas.dart';
import 'package:pos_go_app/core/security.dart';
import 'package:pos_go_app/core/session_store.dart';

void main() {
  test('Country parses the Go meta item', () {
    final country = Country.fromJson({
      'code': 'EG',
      'name_en': 'Egypt',
      'name_ar': 'مصر',
      'currency_code': 'EGP',
      'phone_code': '+20',
    });
    expect(country.code, 'EG');
    expect(country.nameEn, 'Egypt');
    expect(country.nameAr, 'مصر');
    expect(country.currencyCode, 'EGP');
    expect(country.phoneCode, '+20');
  });

  test('Currency parses the Go meta item', () {
    final currency = Currency.fromJson({
      'code': 'EGP',
      'name_en': 'Egyptian Pound',
      'name_ar': 'جنيه مصري',
      'symbol': 'E£',
      'digits_after_decimal': 2,
    });
    expect(currency.code, 'EGP');
    expect(currency.symbol, 'E£');
    expect(currency.digitsAfterDecimal, 2);
  });

  test('countries fetches the public meta endpoint', () async {
    final client = ApiClient(client: _MetaClient());
    final countries = await client.countries();
    expect(countries.single.code, 'EG');
    expect(countries.single.nameEn, 'Egypt');
  });

  test('currencies fetches the public meta endpoint', () async {
    final client = ApiClient(client: _MetaClient());
    final currencies = await client.currencies();
    expect(currencies.single.code, 'EGP');
    expect(currencies.single.symbol, 'E£');
  });

  test('SaasSummary parses platform totals', () {
    final summary = SaasSummary.fromJson({
      'total_tenants': 7,
      'total_users': 20,
      'total_sales': 3,
      'revenue_minor': 12050,
      'by_business': [
        {'business_type': 'restaurant', 'tenants': 1},
        {'business_type': 'general', 'tenants': 2},
      ],
      'supported_country': 'EG',
      'supported_currency': 'EGP',
    });
    expect(summary.totalTenants, 7);
    expect(summary.totalUsers, 20);
    expect(summary.totalSales, 3);
    expect(summary.revenueMinor, 12050);
    expect(summary.byBusiness, hasLength(2));
    expect(summary.byBusiness.first.businessType, 'restaurant');
  });

  test('SaasTenant parses tenant row with counts', () {
    final tenant = SaasTenant.fromJson({
      'id': 't1',
      'name': 'Demo Restaurant',
      'slug': 'demo-restaurant',
      'business_type': 'restaurant',
      'country_code': 'EG',
      'currency_code': 'EGP',
      'default_language': 'ar',
      'users': 3,
      'products': 14,
    });
    expect(tenant.businessType, 'restaurant');
    expect(tenant.users, 3);
    expect(tenant.products, 14);
  });

  test('SaasTenant parses plan, limits and status', () {
    final tenant = SaasTenant.fromJson({
      'id': 't1',
      'name': 'Demo',
      'slug': 'demo',
      'business_type': 'restaurant',
      'plan': 'trial',
      'max_users': 3,
      'max_products': 100,
      'status': 'suspended',
    });
    expect(tenant.plan, 'trial');
    expect(tenant.maxUsers, 3);
    expect(tenant.maxProducts, 100);
    expect(tenant.status, 'suspended');
  });

  test('TenantAnalytics parses report payload', () {
    final analytics = TenantAnalytics.fromJson({
      'tenant': {
        'id': 't1',
        'name': 'Demo Restaurant',
        'slug': 'demo-restaurant',
        'business_type': 'restaurant',
        'country_code': 'EG',
        'currency_code': 'EGP',
        'plan': 'trial',
        'status': 'active',
      },
      'counts': {'users': 3, 'products': 14},
      'today': {
        'date': '2026-09-17',
        'revenue_minor': 28000,
        'sales_count': 5,
        'avg_sale_minor': 5600,
        'items_sold': 9,
      },
      'revenue_trend': [
        {'day': '2026-09-11', 'revenue_minor': 1200},
      ],
      'top_products': [
        {'product_name': 'Espresso', 'sku': 'ESP-001', 'quantity': 4, 'revenue_minor': 1120},
      ],
      'recent_sales': [
        {
          'id': 's1',
          'status': 'completed',
          'total_minor': 5600,
          'currency': 'EGP',
          'payment_method': 'cash',
          'cashier': 'Amina',
          'created_at': '2026-09-17 10:00:00',
        },
      ],
    });
    expect(analytics.tenant.plan, 'trial');
    expect(analytics.users, 3);
    expect(analytics.products, 14);
    expect(analytics.today.revenueMinor, 28000);
    expect(analytics.today.salesCount, 5);
    expect(analytics.revenueTrend.single.day, '2026-09-11');
    expect(analytics.topProducts.single.productName, 'Espresso');
    expect(analytics.recentSales.single.cashier, 'Amina');
  });

  test('TrialEntitlement parses admin payload', () {
    final e = TrialEntitlement.fromJson({
      'id': 'e1',
      'organization_id': 'org1',
      'owner_user_id': 'u1',
      'account_id': 'acct1',
      'trial_type': 'standard',
      'status': 'active',
      'started_at': '2026-09-01T00:00:00Z',
      'expires_at': '2026-09-15T00:00:00Z',
      'trial_days': 14,
      'source': 'signup',
      'eligibility_key': 'org:org1',
      'reason': 'manual',
    });
    expect(e.status, 'active');
    expect(e.trialDays, 14);
    expect(e.source, 'signup');
    expect(e.organizationId, 'org1');
  });

  test('AuditPage parses audit entries', () {
    final page = AuditPage.fromJson({
      'data': {
        'entries': [
          {
            'id': 'a1',
            'created_at': '2026-09-17T10:00:00Z',
            'actor_user_id': 'u1',
            'account_id': '',
            'tenant_id': 't1',
            'action': 'trial_extended',
            'entity_type': 'trial',
            'entity_id': 'e1',
            'reason': 'admin extend by 7 days',
            'ip': '1.2.3.4',
          }
        ]
      }
    });
    expect(page.entries.single.action, 'trial_extended');
    expect(page.entries.single.reason, 'admin extend by 7 days');
  });

  test('TrialPolicy parses settings and round-trips', () {
    final policy = TrialPolicy.fromJson({
      'trial_duration_days': 14,
      'trial_scope': 'org',
      'require_email_verification': true,
      'max_organizations_per_account': 3,
      'offline_trial_policy': 'grace24h',
    });
    expect(policy.trialDurationDays, 14);
    expect(policy.requireEmailVerification, isTrue);
    expect(policy.toJson()['trial_duration_days'], 14);
    expect(policy.toJson()['require_email_verification'], isTrue);
    expect(policy.toJson()['offline_trial_policy'], 'grace24h');
  });

  test('saasTenantAnalytics fetches and parses drill-down', () async {
    final client = ApiClient(client: _SaasAnalyticsClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'u1',
      displayName: 'Platform Admin',
      tenantId: 't1',
      deviceId: 'd1',
      role: 'saas_admin',
    );
    final analytics = await client.saasTenantAnalytics(session, 't1');
    expect(analytics.tenant.name, 'Demo Restaurant');
    expect(analytics.today.revenueMinor, 28000);
  });

  test('platformTrialEntitlements fetches and parses entitlements', () async {
    final client = ApiClient(client: _PlatformTrialsClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'u1',
      displayName: 'Platform Admin',
      tenantId: 't1',
      deviceId: 'd1',
      role: 'saas_admin',
    );
    final trials = await client.platformTrialEntitlements(session);
    expect(trials.single.status, 'active');
  });

  test('trialChange extends with extra_days', () async {
    final client = ApiClient(client: _TrialExtendClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'u1',
      displayName: 'Platform Admin',
      tenantId: 't1',
      deviceId: 'd1',
      role: 'saas_admin',
    );
    final updated = await client.trialChange(session, 'e1', 'extend', extraDays: 7);
    expect(updated.status, 'active');
  });

  test('platformAudit fetches entries', () async {
    final client = ApiClient(client: _PlatformAuditClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'u1',
      displayName: 'Platform Admin',
      tenantId: 't1',
      deviceId: 'd1',
      role: 'saas_admin',
    );
    final entries = await client.platformAudit(session);
    expect(entries.single.action, 'trial_extended');
  });

  test('platformTrialSettings fetches and update sends policy', () async {
    final client = ApiClient(client: _TrialSettingsClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'u1',
      displayName: 'Platform Admin',
      tenantId: 't1',
      deviceId: 'd1',
      role: 'saas_admin',
    );
    final fetched = await client.platformTrialSettings(session);
    expect(fetched.trialDurationDays, 14);
    final updated = await client.platformUpdateTrialSettings(
        session, const TrialPolicy(trialDurationDays: 21));
    expect(updated.trialDurationDays, 21);
  });

  test('platformSuspendTenant posts and succeeds on 200', () async {
    final client = ApiClient(client: _SuspendTenantClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'u1',
      displayName: 'Platform Admin',
      tenantId: 't1',
      deviceId: 'd1',
      role: 'saas_admin',
    );
    await client.platformSuspendTenant(session, 't1', reason: 'abuse');
  });

  test('TenantSettings parses defaults when keys are missing', () {
    const raw = <String, dynamic>{};
    final settings = TenantSettings.fromJson(raw);
    expect(settings.defaultPaymentMethod, PaymentMethod.cash);
    expect(settings.showStockBadges, isTrue);
    expect(settings.receiptFooter, '');
    expect(settings.allowNegativeStock, isFalse);
  });

  test('TenantSettings parses configured values', () {
    final settings = TenantSettings.fromJson({
      'pos.default_payment_method': 'card',
      'pos.show_stock_badges': 'false',
      'pos.receipt_footer': 'Thanks!',
      'inventory.allow_negative_stock': 'true',
    });
    expect(settings.defaultPaymentMethod, PaymentMethod.card);
    expect(settings.showStockBadges, isFalse);
    expect(settings.receiptFooter, 'Thanks!');
    expect(settings.allowNegativeStock, isTrue);
    expect(settings.toUpdateMap(), {
      'pos.default_payment_method': 'card',
      'pos.show_stock_badges': 'false',
      'pos.receipt_footer': 'Thanks!',
      'inventory.allow_negative_stock': 'true',
    });
  });

  test('settings fetches tenant configuration', () async {
    final client = ApiClient(client: _SettingsClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final settings = await client.settings(session);

    expect(settings.defaultPaymentMethod, PaymentMethod.card);
    expect(settings.showStockBadges, isFalse);
    expect(settings.receiptFooter, 'Welcome to Demo Restaurant');
    expect(settings.allowNegativeStock, isTrue);
  });

  test('updateSettings sends the whitelisted payload', () async {
    final client = ApiClient(client: _UpdateSettingsClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Manager',
      tenantId: 'tenant-1',
    );

    final updated = await client.updateSettings(
      session,
      const TenantSettings(
        defaultPaymentMethod: PaymentMethod.mobile,
        showStockBadges: true,
        receiptFooter: 'Thanks for visiting!',
      ),
    );

    expect(updated.defaultPaymentMethod, PaymentMethod.mobile);
    expect(updated.showStockBadges, isTrue);
    expect(updated.receiptFooter, 'Thanks for visiting!');
  });

  test('session parses the Go API envelope', () {
    final session = Session.fromJson({
      'access_token': 'access',
      'refresh_token': 'refresh',
      'user': {'id': 'user-1', 'display_name': 'Cashier'},
      'tenant': {'id': 'tenant-1'},
    });

    expect(session.accessToken, 'access');
    expect(session.displayName, 'Cashier');
    expect(session.tenantId, 'tenant-1');
  });

  test('products parses the Go catalog response with authorization',
      () async {
    final client = ApiClient(client: _FakeClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final products = await client.products(session, search: 'coffee');

    expect(products, hasLength(1));
    expect(products.single.name, 'Espresso');
    expect(products.single.priceMinor, 280);
  });

  test('createSale sends product quantities and idempotency key',
      () async {
    final client = ApiClient(client: _SaleClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final sale = await client.createSale(
      session,
      const [SaleItemInput(productId: 'product-1', quantity: 2)],
      idempotencyKey: 'checkout-1',
    );

    expect(sale.id, 'sale-1');
    expect(sale.totalMinor, 560);
  });

  test('createSale sends split payments and parses the primary method',
      () async {
    final client = ApiClient(client: _SplitSaleClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final sale = await client.createSale(
      session,
      const [SaleItemInput(productId: 'product-1', quantity: 2)],
      idempotencyKey: 'checkout-2',
      payments: const [
        PaymentInput(method: PaymentMethod.card, amountMinor: 400),
        PaymentInput(method: PaymentMethod.mobile, amountMinor: 160),
      ],
    );

    expect(sale.paymentMethod, PaymentMethod.card);
  });

  test('createSale attaches session_id when a register session is open',
      () async {
    final client = ApiClient(client: _SessionSaleClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final sale = await client.createSale(
      session,
      const [SaleItemInput(productId: 'product-1', quantity: 1)],
      idempotencyKey: 'checkout-3',
      sessionId: 'session-1',
    );

    expect(sale.id, 'sale-3');
  });

  test('DashboardSummary parses today revenue, top products, cashiers and mix',
      () {
    final summary = DashboardSummary.fromJson(const {
      'date': '2026-09-11',
      'today': {
        'revenue_minor': 4125,
        'sales_count': 7,
        'avg_sale_minor': 589,
        'items_sold': 12,
      },
      'top_products': [
        {
          'product_name': 'Blueberry Muffin',
          'sku': 'MUF-001',
          'quantity': 4,
          'revenue_minor': 1200
        }
      ],
      'recent_sales': [
        {
          'id': 'sale-1',
          'status': 'completed',
          'total_minor': 600,
          'currency': 'EGP',
          'payment_method': 'cash',
          'created_at': '2026-09-11 00:11:17'
        }
      ],
      'per_cashier': [
        {
          'cashier': 'Restaurant Admin',
          'sales_count': 7,
          'revenue_minor': 4125
        }
      ],
      'payment_mix': [
        {'method': 'cash', 'amount_minor': 3000},
        {'method': 'card', 'amount_minor': 1125}
      ],
    });
    expect(summary.date, '2026-09-11');
    expect(summary.today.revenueMinor, 4125);
    expect(summary.today.salesCount, 7);
    expect(summary.today.avgSaleMinor, 589);
    expect(summary.today.itemsSold, 12);
    expect(summary.topProducts, hasLength(1));
    expect(summary.topProducts.first.productName, 'Blueberry Muffin');
    expect(summary.recentSales, hasLength(1));
    expect(summary.recentSales.first.paymentMethod, 'cash');
    expect(summary.perCashier, hasLength(1));
    expect(summary.perCashier.first.cashier, 'Restaurant Admin');
    expect(summary.paymentMix, hasLength(2));
    expect(summary.paymentMix.first.method, 'cash');
    expect(summary.paymentMix.first.amountMinor, 3000);
  });

  test('dashboardSummary fetches /v1/dashboard/summary', () async {
    final client = ApiClient(client: _DashboardClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Restaurant Admin',
      tenantId: 'tenant-1',
    );
    final summary = await client.dashboardSummary(session);
    expect(summary.today.revenueMinor, 4125);
    expect(summary.paymentMix, hasLength(2));
  });

  test('dashboardSummary surfaces API errors', () async {
    final client = ApiClient(client: _ErrorResponseClient('missing token'));
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );
expect(() => client.dashboardSummary(session),
        throwsA(isA<ApiException>()));
  });

  test('Customer parses the Go customer row with loyalty fields', () {
    final parsed = Customer.fromJson(const {
      'id': 'customer-1',
      'name': 'Ahmed Ali',
      'email': 'ahmed@example.com',
      'phone': '+201000000000',
      'loyalty_points': 4,
      'loyalty_points_total': 400,
      'created_at': '2026-09-11 00:11:17',
    });
    expect(parsed.id, 'customer-1');
    expect(parsed.name, 'Ahmed Ali');
    expect(parsed.email, 'ahmed@example.com');
    expect(parsed.phone, '+201000000000');
    expect(parsed.loyaltyPoints, 4);
    expect(parsed.loyaltyPointsTotal, 400);
  });

  test('listCustomers fetches /v1/customers with pagination meta', () async {
    final client = ApiClient(client: _CustomersListClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Restaurant Admin',
      tenantId: 'tenant-1',
    );
    final page = await client.listCustomers(session);
    expect(page.customers, hasLength(1));
    expect(page.total, 1);
    expect(page.customers.first.loyaltyPoints, 4);
  });

  test('createCustomer posts to /v1/customers and returns the row', () async {
    final client = ApiClient(client: _CreateCustomerClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Restaurant Admin',
      tenantId: 'tenant-1',
    );
    final customer = await client.createCustomer(
      session,
      name: 'Ahmed Ali',
      email: 'ahmed@example.com',
      phone: '+201000000000',
    );
    expect(customer.id, 'customer-1');
    expect(customer.loyaltyPoints, 0);
  });

  test('InventoryAdjustment parses the Go adjustment row', () {
    final parsed = InventoryAdjustment.fromJson(const {
      'id': 'adj-1',
      'product_id': 'product-1',
      'product_name': 'Cappuccino',
      'sku': 'CAP-001',
      'reason': 'restock',
      'quantity_delta': 10,
      'note': 'invoice #12',
      'created_by': 'Restaurant Admin',
      'created_at': '2026-09-14 12:00:00',
    });
    expect(parsed.id, 'adj-1');
    expect(parsed.productId, 'product-1');
    expect(parsed.productName, 'Cappuccino');
    expect(parsed.reason, 'restock');
    expect(parsed.quantityDelta, 10);
    expect(parsed.isCredit, isTrue);
    final loss =
        InventoryAdjustment.fromJson(const {'quantity_delta': -3});
    expect(loss.isCredit, isFalse);
  });

  test('createInventoryAdjustment posts /v1/inventory/adjustments', () async {
    final client = ApiClient(client: _CreateAdjustmentClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Branch Manager',
      tenantId: 'tenant-1',
    );
    final result = await client.createInventoryAdjustment(
      session,
      productId: 'product-1',
      reason: 'restock',
      quantityDelta: 12,
      note: 'supplier drop',
    );
    expect(result.adjustment.id, 'adj-1');
    expect(result.adjustment.reason, 'restock');
    expect(result.newStock, 52);
  });

  test('listInventoryAdjustments fetches /v1/inventory/adjustments', () async {
    final client = ApiClient(client: _AdjustmentsListClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Branch Manager',
      tenantId: 'tenant-1',
    );
    final page = await client.listInventoryAdjustments(session, page: 1);
    expect(page.items, isNotEmpty);
    expect(page.total, 2);
    expect(page.items.first.quantityDelta, -2);
  });

  test('syncPull fetches /v1/sync/pull with cursor + limit params', () async {
    final client = ApiClient(client: _SyncPullClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Restaurant Admin',
      tenantId: 'tenant-1',
    );
    final page = await client.syncPull(session, cursor: 28, limit: 5);
    expect(page.items, hasLength(2));
    expect(page.cursor, 130);
    expect(page.hasMore, isTrue);
    final first = page.items.first;
    expect(first.entity, 'categories');
    expect(first.id, 'category-1');
    expect(first.changeSeq, 28);
    expect(first.changeType, 'upsert');
    expect(first.data['name'], 'Drinks');
  });

  test('syncPull surfaces cursor_expired as an ApiException', () async {
    final client = ApiClient(client: _SyncExpiredClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Restaurant Admin',
      tenantId: 'tenant-1',
    );
    expect(
      () => client.syncPull(session, cursor: 1),
      throwsA(isA<ApiException>()),
    );
  });

  test('SyncRow parses a soft-deleted product row', () {
    const row = SyncRow(
      entity: 'products',
      id: 'product-1',
      changeSeq: 260,
      changeType: 'delete',
      data: {},
    );
    expect(row.entity, 'products');
    expect(row.changeSeq, 260);
    expect(row.changeType, 'delete');
  });

  test('syncPush POSTs commands and maps applied/replayed/conflict results',
      () async {
    final client = ApiClient(client: _SyncPushClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Restaurant Admin',
      tenantId: 'tenant-1',
    );
    final page = await client.syncPush(session, const [
      SyncPushCommand(
        commandId: 'cmd-1',
        operation: 'sale.create',
        payload: {
          'idempotency_key': 'offline-sale-1',
          'items': [
            {'product_id': 'product-1', 'quantity': 1}
          ],
          'payments': [
            {'method': 'cash', 'amount_minor': 350}
          ],
        },
      ),
    ]);
    expect(page.results, hasLength(3));
    expect(page.results[0].applied, isTrue);
    expect(page.results[0].replayed, isFalse);
    expect(page.results[0].result['id'], 'sale-42');
    expect(page.results[1].applied, isTrue);
    expect(page.results[1].replayed, isTrue);
    expect(page.results[2].conflicted, isTrue);
    expect(page.results[2].errorCode, 'insufficient_stock');
    expect(page.results[2].rejected, isFalse);
  });

  test('SyncPushResult default states are safe for rejected commands', () {
    const result = SyncPushResult(
      commandId: 'cmd-x',
      status: 'rejected',
      errorCode: 'unknown_command',
      errorDetail: 'unknown operation: foo.bar',
    );
    expect(result.rejected, isTrue);
    expect(result.applied, isFalse);
    expect(result.conflicted, isFalse);
    expect(result.result, isEmpty);
  });

  test('SaleReceipt parses the Go receipt payload', () {
    final receipt = SaleReceipt.fromJson({
      'tenant_name': 'Demo Store',
      'tenant_address': 'Cairo, Egypt',
      'sale_id': 'sale-42',
      'status': 'completed',
      'created_at': '2026-09-12 12:00:00',
      'cashier': 'Demo Manager',
      'device': 'register-1',
      'table_name': 'T1',
      'floor_name': 'Ground',
      'currency': 'EGP',
      'items': [
        {
          'name': 'Flat White',
          'sku': 'FW',
          'quantity': 2,
          'unit_price_minor': 1000,
          'total_minor': 2000,
        }
      ],
      'subtotal_minor': 2500,
      'discount_minor': 500,
      'tips_minor': 0,
      'total_minor': 2000,
      'payments': [
        {'method': 'card', 'amount_minor': 2000, 'tip_minor': 100}
      ],
      'loyalty_points_earned': 20,
      'idempotency_key': 'offline-sale-1',
    });
    expect(receipt.tenantName, 'Demo Store');
    expect(receipt.saleId, 'sale-42');
    expect(receipt.tableName, 'T1');
    expect(receipt.items, hasLength(1));
    expect(receipt.items.first.unitPriceMinor, 1000);
    expect(receipt.totalMinor, 2000);
    expect(receipt.payments.single.tipMinor, 100);
    expect(receipt.loyaltyPointsEarned, 20);
  });

  test('fetchReceipt GETs the JSON receipt endpoint', () async {
    final client = ApiClient(client: _ReceiptClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Restaurant Admin',
      tenantId: 'tenant-1',
    );
    final receipt = await client.fetchReceipt(session, 'sale-42');
    expect(receipt.saleId, 'sale-42');
    expect(receipt.tenantName, 'Demo Store');
    expect(receipt.totalMinor, 2000);
    expect(receipt.items, hasLength(1));
  });

  test('fetchReceiptPrint GETs ESC/POS bytes with layout knobs', () async {
    final client = ApiClient(client: _ReceiptPrintClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Restaurant Admin',
      tenantId: 'tenant-1',
    );
    final bytes =
        await client.fetchReceiptPrint(session, 'sale-42', cols: 24, cut: false, compact: true);
    expect(bytes, [0x1B, 0x40, 0x1D, 0x56, 0x00]);
  });

  test('fetchReceiptPrint throws on non-200', () async {
    final client =
        ApiClient(client: _RawErrorClient(500, '{"error":{"message":"printer boom"}}'));
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Restaurant Admin',
      tenantId: 'tenant-1',
    );
    try {
      await client.fetchReceiptPrint(session, 'sale-42');
      fail('should throw');
    } on ApiException catch (e) {
      expect(e.message, 'printer boom');
    }
  });

  test('RefundResult parses the refund payload', () {
    final refund = RefundResult.fromJson(const {
      'id': 'refund-1',
      'sale_id': 'sale-42',
      'refund_minor': 2000,
      'reason': 'defective item',
      'status': 'completed',
      'created_by': 'Restaurant Admin',
      'created_at': '2026-09-13 12:00:00',
    });
    expect(refund.id, 'refund-1');
    expect(refund.saleId, 'sale-42');
    expect(refund.refundMinor, 2000);
    expect(refund.reason, 'defective item');
    expect(refund.status, 'completed');
    expect(refund.createdBy, 'Restaurant Admin');
  });

  test('refundSale POSTs to the refund endpoint with an idempotency key',
      () async {
    final client = ApiClient(client: _RefundClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Restaurant Admin',
      tenantId: 'tenant-1',
    );
    final refund = await client.refundSale(
      session,
      'sale-42',
      reason: 'defective item',
      managerPin: '1234',
      idempotencyKey: 'refund-key-1',
    );
    expect(refund.saleId, 'sale-42');
    expect(refund.refundMinor, 2000);
    expect(refund.status, 'completed');
  });

  test('refundSale surfaces a non-201 as ApiException', () async {
    final client = ApiClient(client: _RefundForbiddenClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );
    await expectLater(
      client.refundSale(session, 'sale-42', idempotencyKey: 'refund-key-2'),
      throwsA(isA<ApiException>()
          .having((e) => e.message, 'message', contains('refunds'))),
    );
  });

  test('verifyPin posts the PIN and parses a valid result', () async {
    final client = ApiClient(client: _VerifyPinClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final result = await client.verifyPin(session, pin: '1234');

    expect(result.valid, isTrue);
    expect(result.hasPin, isTrue);
    expect(result.locked, isFalse);
  });

  test('verifyPin reports locked on a 423 pin_locked', () async {
    final client = ApiClient(client: _VerifyPinLockedClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final result = await client.verifyPin(session, pin: '9999');

    expect(result.locked, isTrue);
    expect(result.valid, isFalse);
  });

  test('verifyPin surfaces a server error as ApiException', () async {
    final client = ApiClient(client: _VerifyPinErrorClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    await expectLater(
      client.verifyPin(session, pin: '1234'),
      throwsA(isA<ApiException>()),
    );
  });

  test('unlockPin clears a cashier lockout with a 204', () async {
    final client = ApiClient(client: _UnlockPinClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Manager',
      tenantId: 'tenant-1',
    );

    await client.unlockPin(session, userId: 'user-2');
  });

  test('closeSession sends manager_pin when manager-gated', () async {
    final client = ApiClient(client: _CloseSessionPinClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final closed = await client.closeSession(session, 'session-1',
        closingCashMinor: 5300, managerPin: '1234');

    expect(closed.isOpen, isFalse);
    expect(closed.closingCashMinor, 5300);
    expect(closed.cashDifferenceMinor, 100);
  });

  test('createSale sends discount_minor and manager_pin when provided',
      () async {
    final client = ApiClient(client: _DiscountSaleClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Manager',
      tenantId: 'tenant-1',
    );

    final sale = await client.createSale(
      session,
      const [SaleItemInput(productId: 'product-1', quantity: 1)],
      idempotencyKey: 'checkout-9',
      discountMinor: 50,
      managerPin: '1234',
    );

    expect(sale.discountCapped, isTrue);
  });

  test('createSale omits discount and pin keys when absent', () async {
    final client = ApiClient(client: _PlainSaleClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final sale = await client.createSale(
      session,
      const [SaleItemInput(productId: 'product-1', quantity: 1)],
      idempotencyKey: 'checkout-10',
    );

    expect(sale.discountCapped, isFalse);
  });

  test('SaleResult parses discount_capped and discount_warning', () {
    final sale = SaleResult.fromJson({
      'id': 'sale-1',
      'subtotal_minor': 2000,
      'total_minor': 1950,
      'currency': 'EGP',
      'discount_capped': true,
      'discount_warning': 'capped at 5%',
    });

    expect(sale.discountCapped, isTrue);
    expect(sale.discountWarning, 'capped at 5%');
  });

  test('TenantSettings parses the ma_pos security and stock keys', () {
    final settings = TenantSettings.fromJson({
      'pos.discount_mode': 'block',
      'pos.max_discount_pct': '10',
      'pos.manager.discount': 'false',
      'pos.manager.close': 'false',
      'pos.stock_type': 'available',
      'pos.block_out_of_stock': 'true',
      'pos.low_stock_threshold': '3',
      'pos.low_stock_warning': 'true',
      'pos.validate_stock_payment': 'false',
      'pos.refresh_button': 'false',
    });

    expect(settings.discountMode, DiscountMode.block);
    expect(settings.maxDiscountPct, 10);
    expect(settings.managerDiscountOverride, isFalse);
    expect(settings.managerClosePin, isFalse);
    expect(settings.stockType, 'available');
    expect(settings.blockOutOfStock, isTrue);
    expect(settings.lowStockThreshold, 3);
    expect(settings.lowStockWarning, isTrue);
    expect(settings.validateStockPayment, isFalse);
    expect(settings.refreshButton, isFalse);
  });

  test('TenantSettings ma_pos keys fall back to defaults', () {
    const raw = <String, dynamic>{};
    final settings = TenantSettings.fromJson(raw);

    expect(settings.discountMode, DiscountMode.unknown);
    expect(settings.maxDiscountPct, 0);
    expect(settings.managerDiscountOverride, isTrue);
    expect(settings.managerClosePin, isTrue);
    expect(settings.stockType, 'on_hand');
    expect(settings.blockOutOfStock, isTrue);
    expect(settings.lowStockThreshold, 5);
    expect(settings.lowStockWarning, isTrue);
    expect(settings.validateStockPayment, isTrue);
    expect(settings.refreshButton, isTrue);
  });

  test('RegisterSession parses an open session with its live summary', () {
    final session = RegisterSession.fromJson(const {
      'id': 'session-1',
      'status': 'open',
      'opening_cash_minor': 5000,
      'opened_at': '2026-09-11 00:46:27',
      'opened_by': 'Restaurant Admin',
      'summary': {
        'sales_count': 1,
        'subtotal_minor': 600,
        'discount_minor': 0,
        'tax_minor': 0,
        'total_minor': 600,
        'cash_minor': 200,
        'card_minor': 400,
        'mobile_minor': 0,
        'expected_cash_minor': 5200,
      },
    });
    expect(session.isOpen, isTrue);
    expect(session.openingCashMinor, 5000);
    expect(session.openedBy, 'Restaurant Admin');
    expect(session.closingCashMinor, isNull);
    expect(session.summary.totalMinor, 600);
    expect(session.summary.cashMinor, 200);
    expect(session.summary.expectedCashMinor, 5200);
  });

  test('RegisterSession parses a closed session with reconciliation', () {
    final session = RegisterSession.fromJson(const {
      'id': 'session-1',
      'status': 'closed',
      'opening_cash_minor': 5000,
      'closing_cash_minor': 5300,
      'expected_cash_minor': 5200,
      'cash_difference_minor': 100,
      'opened_at': '2026-09-11 00:46:27',
      'closed_at': '2026-09-11 00:47:21',
      'opened_by': 'Restaurant Admin',
      'summary': {
        'sales_count': 1,
        'subtotal_minor': 600,
        'discount_minor': 0,
        'tax_minor': 0,
        'total_minor': 600,
        'cash_minor': 200,
        'card_minor': 400,
        'mobile_minor': 0,
        'expected_cash_minor': 5200,
      },
    });
    expect(session.isOpen, isFalse);
    expect(session.closingCashMinor, 5300);
    expect(session.expectedCashMinor, 5200);
    expect(session.cashDifferenceMinor, 100);
    expect(session.closedAt, isNotNull);
  });

  test('SessionListItem parses lean list rows', () {
    final item = SessionListItem.fromJson(const {
      'id': 'session-1',
      'status': 'closed',
      'opening_cash_minor': 5000,
      'closing_cash_minor': 5300,
      'expected_cash_minor': 5200,
      'cash_difference_minor': 100,
      'opened_at': '2026-09-11 00:46:27',
      'closed_at': '2026-09-11 00:47:21',
      'opened_by': 'Restaurant Admin',
      'sales_count': 1,
      'total_minor': 600,
    });
    expect(item.isOpen, isFalse);
    expect(item.salesCount, 1);
    expect(item.totalMinor, 600);
  });

  test('currentSession parses the terminal\'s open session', () async {
    final client = ApiClient(client: _CurrentSessionClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );
    final current = await client.currentSession(session);
    expect(current, isNotNull);
    expect(current!.id, 'session-1');
    expect(current.isOpen, isTrue);
  });

  test('currentSession returns null when the terminal has none', () async {
    final client = ApiClient(client: _NoSessionClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );
    expect(await client.currentSession(session), isNull);
  });

  test('openSession posts the starting cash and parses the new session',
      () async {
    final client = ApiClient(client: _OpenSessionClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );
    final opened = await client.openSession(session, openingCashMinor: 5000);
    expect(opened.id, 'session-1');
    expect(opened.isOpen, isTrue);
    expect(opened.openingCashMinor, 5000);
  });

  test('closeSession posts counted cash and parses the closed report',
      () async {
    final client = ApiClient(client: _CloseSessionClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );
    final closed =
        await client.closeSession(session, 'session-1', closingCashMinor: 5300);
    expect(closed.isOpen, isFalse);
    expect(closed.closingCashMinor, 5300);
    expect(closed.cashDifferenceMinor, 100);
  });

  test('sessions parses the paginated session history', () async {
    final client = ApiClient(client: _SessionsListClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );
    final page = await client.sessions(session, page: 1, limit: 5);
    expect(page.sessions.single.id, 'session-1');
    expect(page.total, 1);
    expect(page.page, 1);
    expect(page.limit, 5);
  });

  test('SaleResult defaults the payment method to cash', () {
    const raw = {'id': 'sale-1', 'subtotal_minor': 560, 'total_minor': 560, 'currency': 'EGP'};
    final sale = SaleResult.fromJson(raw);
    expect(sale.paymentMethod, PaymentMethod.cash);
  });

  test('SaleSummary parses payment method and defaults legacy rows to cash',
      () {
    final card = SaleSummary.fromJson({
      'id': 'sale-1',
      'status': 'completed',
      'subtotal_minor': 560,
      'total_minor': 560,
      'currency': 'EGP',
      'payment_method': 'card',
    });
    expect(card.paymentMethod, PaymentMethod.card);
    final legacy = SaleSummary.fromJson({
      'id': 'sale-0',
      'status': 'completed',
      'subtotal_minor': 300,
      'total_minor': 300,
      'currency': 'EGP',
    });
    expect(legacy.paymentMethod, PaymentMethod.cash);
  });

  test('refresh rotates both tokens', () async {
    final client = ApiClient(client: _DirectRefreshClient());
    const session = Session(
      accessToken: 'old-access',
      refreshToken: 'old-refresh',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final refreshed = await client.refresh(session);

    expect(refreshed.accessToken, 'new-access');
    expect(refreshed.refreshToken, 'new-refresh');
    expect(refreshed.userId, 'user-1');
  });

  test('refreshes once on unauthorized responses and retries with new token',
      () async {
    Session? refreshedSession;
    final client = ApiClient(
      client: _RefreshClient(),
      onSessionRefreshed: (session) async => refreshedSession = session,
    );
    const session = Session(
      accessToken: 'expired-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final products = await client.products(session);

    expect(products.single.id, 'product-1');
    expect(refreshedSession?.accessToken, 'new-access-token');
    expect(refreshedSession?.refreshToken, 'new-refresh-token');
  });

  test('createProduct sends catalog fields', () async {
    final client = ApiClient(client: _CreateProductClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Manager',
      tenantId: 'tenant-1',
    );

    final product = await client.createProduct(
      session,
      name: 'Espresso',
      sku: 'ESP-1',
      priceMinor: 280,
      currency: 'USD',
      stockQuantity: 4,
    );

    expect(product.name, 'Espresso');
    expect(product.stockQuantity, 4);
  });

  test('categories parses the Go categories response', () async {
    final client = ApiClient(client: _CategoriesClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final categories = await client.categories(session);

    expect(categories, hasLength(2));
    expect(categories.first.name, 'Drinks');
    expect(categories.first.slug, 'drinks');
    expect(categories.last.id, 'cat-2');
  });

  test('listSales parses the Go sales list response with pagination',
      () async {
    final client = ApiClient(client: _ListSalesClient());
    const session = Session(
      accessToken: 'access-token',
      refreshToken: 'refresh-token',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
    );

    final page = await client.listSales(session, page: 2, limit: 10);

    expect(page.sales, hasLength(1));
    expect(page.sales.first.id, 'sale-1');
    expect(page.sales.first.status, 'completed');
    expect(page.sales.first.totalMinor, 560);
    expect(page.total, 42);
    expect(page.page, 2);
    expect(page.limit, 10);
  });

  test('session stores and restores deviceId', () {
    const session = Session(
      accessToken: 'access',
      refreshToken: 'refresh',
      userId: 'user-1',
      displayName: 'Cashier',
      tenantId: 'tenant-1',
      deviceId: 'device-abc',
    );

    expect(session.deviceId, 'device-abc');

    final restored = session.copyWith();
    expect(restored.deviceId, 'device-abc');
  });

  test('SaleSummary parses Go sale list item', () {
    final sale = SaleSummary.fromJson({
      'id': 'sale-1',
      'status': 'completed',
      'subtotal_minor': 560,
      'total_minor': 560,
      'currency': 'USD',
      'created_at': '2025-01-15T10:30:00Z',
    });

    expect(sale.id, 'sale-1');
    expect(sale.status, 'completed');
    expect(sale.totalMinor, 560);
    expect(sale.createdAt, '2025-01-15T10:30:00Z');
  });

  test('Category parses Go category item', () {
    final cat = Category.fromJson({
      'id': 'cat-1',
      'name': 'Drinks',
      'slug': 'drinks',
    });

    expect(cat.id, 'cat-1');
    expect(cat.name, 'Drinks');
    expect(cat.slug, 'drinks');
  });

  group('login', () {
    test('posts credentials and parses session', () async {
      final client = ApiClient(client: _LoginClient());
      final session = await client.login(
        tenantId: 'tenant-1',
        email: 'cashier@example.com',
        password: 'pass',
        deviceId: 'dev-1',
        deviceName: 'iPhone',
      );
      expect(session.accessToken, 'access-token');
      expect(session.tenantId, 'tenant-1');
      expect(session.deviceId, 'dev-1');
    });

    test('throws ApiException on non-200', () async {
      final client = ApiClient(client: _ErrorClient(401));
      expect(
        () => client.login(
          tenantId: 't',
          email: 'e',
          password: 'p',
          deviceId: 'd',
          deviceName: 'n',
        ),
        throwsA(isA<ApiException>()),
      );
    });
  });

  group('register', () {
    test('creates a trial store and parses the trial session', () async {
      final client = ApiClient(client: _RegisterClient());
      final session = await client.register(
        storeName: 'My Coffee',
        businessType: 'coffee_shop',
        email: 'owner@mycoffee.com',
        password: 'secret-pass-1',
        displayName: 'Owner',
        deviceId: 'dev-1',
        deviceName: 'Counter 1',
      );
      expect(session.tenantId, 'tenant-new');
      expect(session.businessType, 'coffee_shop');
      expect(session.plan, 'trial');
      expect(session.trialEndsAt, '2026-09-29T00:00:00Z');
      expect(session.isTrial, isTrue);
      expect(session.role, 'owner');
    });

    test('maps a 403 trial_expired response onto the ApiException code',
        () async {
      final client = ApiClient(client: _TrialExpiredClient());
      try {
        await client.register(
          storeName: 's',
          businessType: 'coffee_shop',
          email: 'e@example.com',
          password: 'secret-123',
          displayName: 'o',
          deviceId: 'd',
          deviceName: 'n',
        );
        fail('register should throw on trial_expired');
      } on ApiException catch (error) {
        expect(error.isTrialExpired, isTrue);
      }
    });
  });

  group('logout', () {
    test('sends Bearer token and expects 204', () async {
      final client = ApiClient(client: _LogoutClient());
      const session = Session(
        accessToken: 'access-token',
        refreshToken: 'refresh-token',
        userId: 'u',
        displayName: 'Cashier',
        tenantId: 't',
      );
      await client.logout(session);
    });

    test('throws ApiException on non-204', () async {
      final client = ApiClient(client: _ErrorClient(500));
      const session = Session(
        accessToken: 'token',
        refreshToken: 'refresh',
        userId: 'u',
        displayName: 'Cashier',
        tenantId: 't',
      );
      expect(
        () => client.logout(session),
        throwsA(isA<ApiException>()),
      );
    });
  });

  group('updateProduct', () {
    test('sends PATCH fields and parses response', () async {
      final client = ApiClient(client: _UpdateProductClient());
      const session = Session(
        accessToken: 'access-token',
        refreshToken: 'refresh-token',
        userId: 'user-1',
        displayName: 'Manager',
        tenantId: 'tenant-1',
      );

      final product = await client.updateProduct(
        session,
        'product-1',
        name: 'Updated Espresso',
        priceMinor: 350,
      );

      expect(product.name, 'Updated Espresso');
      expect(product.priceMinor, 350);
    });
  });

  group('Product.fromJson', () {
    test('parses all fields', () {
      final product = Product.fromJson({
        'id': 'p-1',
        'name': 'Latte',
        'sku': 'LAT-1',
        'barcode': '123',
        'price_minor': 350,
        'currency': 'USD',
        'stock_quantity': 10,
        'category_id': 'cat-1',
        'cost_minor': 120,
        'image_url': 'https://example.com/latte.png',
        'unit': 'liter',
        'is_active': true,
      });
      expect(product.id, 'p-1');
      expect(product.name, 'Latte');
      expect(product.barcode, '123');
      expect(product.priceMinor, 350);
      expect(product.categoryId, 'cat-1');
      expect(product.costMinor, 120);
      expect(product.imageUrl, 'https://example.com/latte.png');
      expect(product.unit, 'liter');
      expect(product.isActive, true);
    });

    test('defaults optional fields', () {
      final product = Product.fromJson({
        'id': 'p-2',
        'name': 'Mocha',
        'sku': 'MOC-1',
        'price_minor': 400,
        'currency': 'USD',
        'stock_quantity': 5,
      });
      expect(product.barcode, '');
      expect(product.categoryId, '');
      expect(product.costMinor, 0);
      expect(product.imageUrl, '');
      expect(product.unit, 'piece');
      expect(product.isActive, true);
    });

    test('handles integer price_minor as num', () {
      final product = Product.fromJson({
        'id': 'p-3',
        'name': 'Tea',
        'sku': 'TEA-1',
        'barcode': '',
        'price_minor': 200,
        'currency': 'USD',
        'stock_quantity': 1,
      });
      expect(product.priceMinor, 200);
    });
  });

  group('_message edge cases', () {
    test('error message extracted from Go error envelope', () async {
      final client = ApiClient(client: _ErrorResponseClient('product not found'));
      const session = Session(
        accessToken: 't',
        refreshToken: 'r',
        userId: 'u',
        displayName: 'Cashier',
        tenantId: 'ten',
      );
      try {
        await client.products(session);
        fail('should throw');
      } on ApiException catch (e) {
        expect(e.message, 'product not found');
      }
    });

    test('fallback message when error field missing', () async {
      final client = ApiClient(client: _RawErrorClient(400, '{"oops": true}'));
      const session = Session(
        accessToken: 't',
        refreshToken: 'r',
        userId: 'u',
        displayName: 'Cashier',
        tenantId: 'ten',
      );
      try {
        await client.products(session);
        fail('should throw');
      } on ApiException catch (e) {
        expect(e.message, 'Request failed');
      }
    });

    test('fallback message for non-JSON body', () async {
      final client = ApiClient(client: _RawErrorClient(502, 'Bad Gateway'));
      const session = Session(
        accessToken: 't',
        refreshToken: 'r',
        userId: 'u',
        displayName: 'Cashier',
        tenantId: 'ten',
      );
      try {
        await client.products(session);
        fail('should throw');
      } on ApiException catch (e) {
        expect(e.message, contains('Request failed'));
        expect(e.message, contains('502'));
      }
    });

    test('floors fetches the restaurant floors list', () async {
      final client = ApiClient(client: _FloorsClient());
      const session = Session(
        accessToken: 'access-token',
        refreshToken: 'refresh-token',
        userId: 'user-1',
        displayName: 'Cashier',
        tenantId: 'tenant-1',
      );
      final floors = await client.floors(session);
      expect(floors, hasLength(2));
      expect(floors.first.name, 'Ground');
      expect(floors.last.sortOrder, 1);
    });

    test('tables fetches tables with status and floor name', () async {
      final client = ApiClient(client: _TablesClient());
      const session = Session(
        accessToken: 'access-token',
        refreshToken: 'refresh-token',
        userId: 'user-1',
        displayName: 'Cashier',
        tenantId: 'tenant-1',
      );
      final tables = await client.tables(session);
      expect(tables, hasLength(2));
      expect(tables.first.id, 'table-1');
      expect(tables.first.name, 'T01');
      expect(tables.first.floorName, 'Ground');
      expect(tables.first.isAvailable, isFalse);
      expect(tables.last.isAvailable, isTrue);
    });

    test('splitSale posts children and parses created ids', () async {
      final client = ApiClient(client: _SplitRequestClient());
      const session = Session(
        accessToken: 'access-token',
        refreshToken: 'refresh-token',
        userId: 'user-1',
        displayName: 'Cashier',
        tenantId: 'tenant-1',
      );
      final result = await client.splitSale(session, 'sale-9', const [
        [SplitBillLine(saleItemId: 'line-1', quantity: 2)],
        [SplitBillLine(saleItemId: 'line-2', quantity: 1)],
      ]);
      expect(result.parentSaleId, 'sale-9');
      expect(result.children, ['sale-9-a', 'sale-9-b']);
      expect(result.currency, 'EGP');
    });

    test('createSale sends table_id and tip on the tender line', () async {
      final client = ApiClient(client: _TableSaleClient());
      const session = Session(
        accessToken: 'access-token',
        refreshToken: 'refresh-token',
        userId: 'user-1',
        displayName: 'Cashier',
        tenantId: 'tenant-1',
      );
      final sale = await client.createSale(
        session,
        const [SaleItemInput(productId: 'product-1', quantity: 2)],
        idempotencyKey: 'checkout-table',
        payments: const [
          PaymentInput(method: PaymentMethod.cash, amountMinor: 2000, tipMinor: 300),
        ],
        tableId: 'table-1',
      );
      expect(sale.id, 'sale-8');
    });
  });

  group('client events', () {
    test('postClientEvent posts the event with a bearer token', () async {
      final client = ApiClient(client: _ClientEventClient());
      const session = Session(
        accessToken: 'access-token',
        refreshToken: 'refresh-token',
        userId: 'user-1',
        displayName: 'Cashier',
        tenantId: 'tenant-1',
      );

      await client.postClientEvent(
        session,
        'crash',
        appVersion: '0.1.0+1',
        screen: 'pos',
        stackTrace: 'trace line',
        payload: {'exception': 'boom'},
      );
    });

    test('postClientEvent throws on a non-201 response', () async {
      final client = ApiClient(client: _FailingClientEventClient());
      const session = Session(
        accessToken: 'access-token',
        refreshToken: 'refresh-token',
        userId: 'user-1',
        displayName: 'Cashier',
        tenantId: 'tenant-1',
      );

      expect(
        () => client.postClientEvent(session, 'crash'),
        throwsA(isA<ApiException>()),
      );
    });
  });
}

class _FakeClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.url.queryParameters['search'], 'coffee');
    const body =
        '{"data":[{"id":"product-1","name":"Espresso","sku":"ESP-1","barcode":"0123","price_minor":280,"currency":"USD","stock_quantity":4}],"meta":{"request_id":"test"}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _SaleClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.headers['Idempotency-Key'], 'checkout-1');
    expect(request.method, 'POST');
    const body =
        '{"data":{"id":"sale-1","subtotal_minor":560,"total_minor":560,"currency":"USD"},"meta":{"request_id":"test"}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _SplitSaleClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.headers['Idempotency-Key'], 'checkout-2');
    final body = await request.finalize().bytesToString();
    final payload = jsonDecode(body) as Map<String, dynamic>;
    expect(payload['payments'], const [
      {'method': 'card', 'amount_minor': 400},
      {'method': 'mobile', 'amount_minor': 160},
    ]);
    const response =
        '{"data":{"id":"sale-2","subtotal_minor":560,"total_minor":560,"currency":"EGP","payment_method":"card"},"meta":{"request_id":"test"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _DirectRefreshClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.headers['content-type'], 'application/json');
    const body =
        '{"data":{"access_token":"new-access","refresh_token":"new-refresh","expires_in":900}}';
    return http.StreamedResponse(Stream.value(body.codeUnits), 200);
  }
}

class _RefreshClient extends http.BaseClient {
  var productAttempts = 0;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    if (request.url.path.endsWith('/auth/refresh')) {
      const body =
          '{"data":{"access_token":"new-access-token","refresh_token":"new-refresh-token"}}';
      return http.StreamedResponse(
        Stream.value(body.codeUnits),
        200,
        headers: {'content-type': 'application/json'},
      );
    }
    productAttempts++;
    if (productAttempts == 1) {
      return http.StreamedResponse(Stream.value('{}'.codeUnits), 401);
    }
    expect(request.headers['Authorization'], 'Bearer new-access-token');
    const body =
        '{"data":[{"id":"product-1","name":"Espresso","sku":"ESP-1","barcode":"0123","price_minor":280,"currency":"USD","stock_quantity":4}]}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _CreateProductClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/products');
    expect(request.headers['Authorization'], 'Bearer access-token');
    const body =
        '{"data":{"id":"product-1","name":"Espresso","sku":"ESP-1","barcode":"","price_minor":280,"cost_minor":0,"currency":"USD","stock_quantity":4,"is_active":true}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _CategoriesClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.url.path, '/v1/categories');
    const body =
        '{"data":[{"id":"cat-1","name":"Drinks","slug":"drinks"},{"id":"cat-2","name":"Food","slug":"food"}],"meta":{"request_id":"test"}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _ListSalesClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.url.queryParameters['page'], '2');
    expect(request.url.queryParameters['limit'], '10');
    const body =
        '{"data":[{"id":"sale-1","status":"completed","subtotal_minor":560,"discount_minor":0,"tax_minor":0,"total_minor":560,"currency":"USD","created_at":"2025-01-15T10:30:00Z"}],"meta":{"request_id":"test","page":2,"limit":10,"total":42}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _LoginClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/auth/login');
    expect(request.method, 'POST');
    const body =
        '{"data":{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"device_id":"dev-1","user":{"id":"u-1","display_name":"Cashier"},"tenant":{"id":"tenant-1"}}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _RegisterClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/auth/register');
    expect(request.method, 'POST');
    const body =
        '{"data":{"access_token":"access-token","refresh_token":"refresh-token","expires_in":3600,"device_id":"dev-1","user":{"id":"u-1","display_name":"Owner","role":"owner","account_type":"standard"},"tenant":{"id":"tenant-new","business_type":"coffee_shop","country_code":"EG","currency_code":"EGP","default_language":"ar","plan":"trial","trial_ends_at":"2026-09-29T00:00:00Z"}}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _TrialExpiredClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/auth/register');
    return http.StreamedResponse(
      Stream.value(
          '{"error":{"code":"trial_expired","message":"trial has expired"}}'.codeUnits),
      403,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _LogoutClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/auth/logout');
    expect(request.headers['Authorization'], 'Bearer access-token');
    return http.StreamedResponse(Stream.value([]), 204);
  }
}

class _UpdateProductClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'PATCH');
    expect(request.url.path, '/v1/products/product-1');
    const body =
        '{"data":{"id":"product-1","name":"Updated Espresso","sku":"ESP-1","barcode":"","price_minor":350,"cost_minor":0,"currency":"USD","stock_quantity":4,"is_active":true}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _ErrorClient extends http.BaseClient {
  final int statusCode;
  _ErrorClient(this.statusCode);
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    return http.StreamedResponse(
      Stream.value('{"error":{"code":"unauthorized","message":"invalid credentials"}}'.codeUnits),
      statusCode,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _ErrorResponseClient extends http.BaseClient {
  final String message;
  _ErrorResponseClient(this.message);
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    return http.StreamedResponse(
      Stream.value('{"error":{"code":"not_found","message":"$message"}}'.codeUnits),
      404,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _RawErrorClient extends http.BaseClient {
  final int statusCode;
  final String body;
  _RawErrorClient(this.statusCode, this.body);
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      statusCode,
    );
  }
}

class _SettingsClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(request.url.path, '/v1/settings');
    expect(request.headers['Authorization'], 'Bearer access-token');
    const body =
        '{"data":{"pos.default_payment_method":"card","pos.show_stock_badges":"false","pos.receipt_footer":"Welcome to Demo Restaurant","inventory.allow_negative_stock":"true"},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _UpdateSettingsClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'PUT');
    expect(request.url.path, '/v1/settings');
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.headers['content-type'], 'application/json');
    final body = await request.finalize().bytesToString();
    expect(jsonDecode(body), {
      'settings': {
        'pos.default_payment_method': 'mobile',
        'pos.show_stock_badges': 'true',
        'pos.receipt_footer': 'Thanks for visiting!',
        'inventory.allow_negative_stock': 'false',
      }
    });
    const response =
        '{"data":{"pos.default_payment_method":"mobile","pos.show_stock_badges":"true","pos.receipt_footer":"Thanks for visiting!","inventory.allow_negative_stock":"false"},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _MetaClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    late final String body;
    if (request.url.path.endsWith('/meta/countries')) {
      body =
          '{"data":[{"code":"EG","name_en":"Egypt","name_ar":"مصر","currency_code":"EGP","phone_code":"+20"}],"meta":{"request_id":"t"}}';
    } else {
      body =
          '{"data":[{"code":"EGP","name_en":"Egyptian Pound","name_ar":"جنيه مصري","symbol":"E£","digits_after_decimal":2}],"meta":{"request_id":"t"}}';
    }
    final bytes = const Utf8Encoder().convert(body);
    return http.StreamedResponse(
      Stream.value(bytes),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

const _sessionJson =
    '{"data":{"id":"session-1","status":"open","opening_cash_minor":5000,"opened_at":"2026-09-11 00:46:27","opened_by":"Restaurant Admin","summary":{"sales_count":0,"subtotal_minor":0,"discount_minor":0,"tax_minor":0,"total_minor":0,"cash_minor":0,"card_minor":0,"mobile_minor":0,"expected_cash_minor":5000}},"meta":{"request_id":"t"}}';

class _CurrentSessionClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/registers/current');
    expect(request.headers['Authorization'], 'Bearer access-token');
    return http.StreamedResponse(
      Stream.value(_sessionJson.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _NoSessionClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/registers/current');
    expect(request.headers['Authorization'], 'Bearer access-token');
    return http.StreamedResponse(
      Stream.value(
          '{"error":{"code":"no_open_session","message":"no open session for this terminal"}}'.codeUnits),
      404,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _OpenSessionClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/registers/open');
    expect(request.headers['Authorization'], 'Bearer access-token');
    final body = await request.finalize().bytesToString();
    expect(jsonDecode(body), {'opening_cash_minor': 5000});
    return http.StreamedResponse(
      Stream.value(_sessionJson.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _CloseSessionClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/registers/session-1/close');
    expect(request.headers['Authorization'], 'Bearer access-token');
    final body = await request.finalize().bytesToString();
    expect(jsonDecode(body), {'closing_cash_minor': 5300});
    const response =
        '{"data":{"id":"session-1","status":"closed","opening_cash_minor":5000,"closing_cash_minor":5300,"expected_cash_minor":5200,"cash_difference_minor":100,"opened_at":"2026-09-11 00:46:27","closed_at":"2026-09-11 00:47:21","opened_by":"Restaurant Admin","summary":{"sales_count":1,"subtotal_minor":600,"discount_minor":0,"tax_minor":0,"total_minor":600,"cash_minor":200,"card_minor":400,"mobile_minor":0,"expected_cash_minor":5200}},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _SessionsListClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/registers');
    expect(request.headers['Authorization'], 'Bearer access-token');
    const response =
        '{"data":[{"id":"session-1","status":"closed","opening_cash_minor":5000,"closing_cash_minor":5300,"expected_cash_minor":5200,"cash_difference_minor":100,"opened_at":"2026-09-11 00:46:27","closed_at":"2026-09-11 00:47:21","opened_by":"Restaurant Admin","sales_count":1,"total_minor":600}],"meta":{"request_id":"t","page":1,"limit":5,"total":1}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

const _dashboardJson =
    '{"data":{"date":"2026-09-11","today":{"revenue_minor":4125,"sales_count":7,"avg_sale_minor":589,"items_sold":12},"top_products":[{"product_name":"Blueberry Muffin","sku":"MUF-001","quantity":4,"revenue_minor":1200}],"recent_sales":[{"id":"sale-1","status":"completed","total_minor":600,"currency":"EGP","payment_method":"cash","created_at":"2026-09-11 00:11:17"}],"per_cashier":[{"cashier":"Restaurant Admin","sales_count":7,"revenue_minor":4125}],"payment_mix":[{"method":"cash","amount_minor":3000},{"method":"card","amount_minor":1125}]},"meta":{"request_id":"t"}}';

class _DashboardClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(request.url.path, '/v1/dashboard/summary');
    expect(request.headers['Authorization'], 'Bearer access-token');
    return http.StreamedResponse(
      Stream.value(_dashboardJson.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _SessionSaleClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/sales');
    final body = await request.finalize().bytesToString();
    final payload = jsonDecode(body) as Map<String, dynamic>;
    expect(payload['session_id'], 'session-1');
    const response =
        '{"data":{"id":"sale-3","subtotal_minor":280,"total_minor":280,"currency":"EGP","payment_method":"cash"},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _CustomersListClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(request.url.path, '/v1/customers');
    expect(request.url.queryParameters['page'], '1');
    expect(request.headers['Authorization'], 'Bearer access-token');
    const response =
        '{"data":[{"id":"customer-1","name":"Ahmed Ali","email":"ahmed@example.com","phone":"+201000000000","loyalty_points":4,"loyalty_points_total":400,"created_at":"2026-09-11 00:11:17"}],"meta":{"request_id":"t","page":1,"limit":50,"total":1}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _CreateCustomerClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/customers');
    expect(request.headers['Authorization'], 'Bearer access-token');
    final body = await request.finalize().bytesToString();
    expect(jsonDecode(body), {
      'name': 'Ahmed Ali',
      'email': 'ahmed@example.com',
      'phone': '+201000000000',
    });
    const response =
        '{"data":{"id":"customer-1","name":"Ahmed Ali","email":"ahmed@example.com","phone":"+201000000000","loyalty_points":0,"loyalty_points_total":0,"created_at":"2026-09-11 00:11:17"},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _CreateAdjustmentClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/inventory/adjustments');
    expect(request.headers['Authorization'], 'Bearer access-token');
    final body = await request.finalize().bytesToString();
    expect(jsonDecode(body), {
      'product_id': 'product-1',
      'reason': 'restock',
      'quantity_delta': 12,
      'note': 'supplier drop',
    });
    const response =
        '{"data":{"id":"adj-1","product_id":"product-1","product_name":"Cappuccino","sku":"CAP-001","reason":"restock","quantity_delta":12,"note":"supplier drop","created_by":"Branch Manager","created_at":"2026-09-14 12:00:00"},"meta":{"request_id":"t","stock_quantity":52}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _AdjustmentsListClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(request.url.path, '/v1/inventory/adjustments');
    expect(request.url.queryParameters['page'], '1');
    expect(request.headers['Authorization'], 'Bearer access-token');
    const response =
        '{"data":[{"id":"adj-2","product_id":"product-2","product_name":"Croissant","sku":"CRN-001","reason":"damaged","quantity_delta":-2,"note":"","created_by":"Branch Manager","created_at":"2026-09-14 11:00:00"},{"id":"adj-1","product_id":"product-1","product_name":"Cappuccino","sku":"CAP-001","reason":"count","quantity_delta":5,"note":"","created_by":"Branch Manager","created_at":"2026-09-14 10:00:00"}],"meta":{"request_id":"t","page":1,"limit":50,"total":2}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _SyncPullClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(request.url.path, '/v1/sync/pull');
    expect(request.url.queryParameters['cursor'], '28');
    expect(request.url.queryParameters['limit'], '5');
    expect(request.headers['Authorization'], 'Bearer access-token');
    const response =
        '{"data":{"items":[{"entity":"categories","id":"category-1","change_seq":28,"change_type":"upsert","data":{"name":"Drinks","is_active":true}},{"entity":"products","id":"product-1","change_seq":130,"change_type":"upsert","data":{"name":"Cappuccino"}}],"cursor":130,"has_more":true},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _SyncPushClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/sync/push');
    expect(request.headers['Authorization'], 'Bearer access-token');
    final body = await request.finalize().bytesToString();
    final payload = jsonDecode(body) as Map<String, dynamic>;
    final commands = payload['commands'] as List;
    expect(commands, hasLength(1));
    final cmd = commands.first as Map<String, dynamic>;
    expect(cmd['command_id'], 'cmd-1');
    expect(cmd['operation'], 'sale.create');
    final cmdPayload = cmd['payload'] as Map<String, dynamic>;
    expect(cmdPayload['idempotency_key'], 'offline-sale-1');
    const response =
        '{"data":{"results":[{"command_id":"cmd-1","status":"applied","replayed":false,"result":{"id":"sale-42","total_minor":350}},{"command_id":"cmd-2","status":"applied","replayed":true},{"command_id":"cmd-3","status":"conflict","error_code":"insufficient_stock","error_detail":"stock is too low"}]},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _ReceiptPrintClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(request.url.path, '/v1/sales/sale-42/receipt/print');
    expect(request.url.queryParameters['cols'], '24');
    expect(request.url.queryParameters['cut'], '0');
    expect(request.url.queryParameters['compact'], '1');
    expect(request.headers['Accept'], 'application/vnd.escpos');
    expect(request.headers['Authorization'], 'Bearer access-token');
    const bytes = [0x1B, 0x40, 0x1D, 0x56, 0x00];
    return http.StreamedResponse(
      Stream.value(bytes),
      200,
      headers: {'content-type': 'application/vnd.escpos'},
    );
  }
}

class _ReceiptClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(request.url.path, '/v1/sales/sale-42/receipt');
    expect(request.headers['Authorization'], 'Bearer access-token');
    const response =
        '{"data":{"tenant_name":"Demo Store","tenant_address":"Cairo, Egypt","sale_id":"sale-42",'
        '"status":"completed","created_at":"2026-09-12 12:00:00","cashier":"Demo Manager",'
        '"device":"register-1","currency":"EGP","items":[{"name":"Flat White","sku":"FW",'
        '"quantity":2,"unit_price_minor":1000,"total_minor":2000}],"subtotal_minor":2500,'
        '"discount_minor":500,"tips_minor":0,"total_minor":2000,"payments":'
        '[{"method":"card","amount_minor":2000,"tip_minor":0}],"loyalty_points_earned":20,'
        '"idempotency_key":"key-1"},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _RefundClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/sales/sale-42/refund');
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.headers['Idempotency-Key'], 'refund-key-1');
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    expect(body['reason'], 'defective item');
    expect(body['manager_pin'], '1234');
    const response =
        '{"data":{"id":"refund-1","sale_id":"sale-42","refund_minor":2000,'
        '"reason":"defective item","status":"completed",'
        '"created_by":"Restaurant Admin","created_at":"2026-09-13 12:00:00"},'
        '"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _RefundForbiddenClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/sales/sale-42/refund');
    const response =
        '{"error":{"code":"permission_denied","message":"refunds not allowed for this role"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      403,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _SyncExpiredClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/sync/pull');
    const response =
        '{"error":{"code":"cursor_expired","message":"cursor is older than the retention window"}}';
return http.StreamedResponse(
      Stream.value(response.codeUnits),
       409,
       headers: {'content-type': 'application/json'},
     );
   }
 }

class _FloorsClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(request.url.path, '/v1/floors');
    expect(request.headers['Authorization'], 'Bearer access-token');
    const body =
        '{"data":[{"id":"floor-1","name":"Ground"},'
        '{"id":"floor-2","name":"First","sort_order":1}],'
        '"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _TablesClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(request.url.path, '/v1/tables');
    expect(request.headers['Authorization'], 'Bearer access-token');
    const body =
        '{"data":[{"id":"table-1","floor_id":"floor-1","name":"T01","seats":4,'
        '"status":"occupied","floor_name":"Ground"},'
        '{"id":"table-2","floor_id":"floor-2","name":"T02","seats":2,'
        '"status":"free","floor_name":"First"}],'
        '"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(body.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _SplitRequestClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/sales/sale-9/split');
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.headers['Idempotency-Key'], isNotNull);
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    final children = body['children'] as List<dynamic>;
    expect(children, hasLength(2));
    final lines =
        (children.first as Map<String, dynamic>)['lines'] as List<dynamic>;
    expect(lines.single, {'sale_item_id': 'line-1', 'quantity': 2});
    const response =
        '{"data":{"parent_sale_id":"sale-9",'
        '"children":["sale-9-a","sale-9-b"],"currency":"EGP"},'
        '"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _TableSaleClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/sales');
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.headers['Idempotency-Key'], 'checkout-table');
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    expect(body['table_id'], 'table-1');
    final payments = body['payments'] as List<dynamic>;
    expect(payments.single, {
      'method': 'cash',
      'amount_minor': 2000,
      'tip_minor': 300,
    });
    const response =
        '{"data":{"id":"sale-8","subtotal_minor":1700,"total_minor":1700,'
        '"currency":"EGP","payment_method":"cash","tips_minor":300},'
        '"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _VerifyPinClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/auth/verify-pin');
    expect(request.headers['Authorization'], 'Bearer access-token');
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    expect(body, {'pin': '1234'});
    const response =
        '{"data":{"valid":true,"has_pin":true,"attempts_left":5,'
        '"locked":false},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _VerifyPinLockedClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/auth/verify-pin');
    const response =
        '{"error":{"code":"pin_locked","message":"too many attempts",'
        '"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      423,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _VerifyPinErrorClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.url.path, '/v1/auth/verify-pin');
    const response =
        '{"error":{"code":"server_error","message":"boom","request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      500,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _UnlockPinClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/auth/unlock-pin');
    expect(request.headers['Authorization'], 'Bearer access-token');
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    expect(body, {'user_id': 'user-2'});
    return http.StreamedResponse(const Stream.empty(), 204);
  }
}

class _CloseSessionPinClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/registers/session-1/close');
    expect(request.headers['Authorization'], 'Bearer access-token');
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    expect(body, {'closing_cash_minor': 5300, 'manager_pin': '1234'});
    const response =
        '{"data":{"id":"session-1","status":"closed","opening_cash_minor":5000,'
        '"closing_cash_minor":5300,"expected_cash_minor":5200,'
        '"cash_difference_minor":100,"opened_at":"2026-09-11 00:46:27",'
        '"closed_at":"2026-09-11 00:47:21","opened_by":"Restaurant Admin",'
        '"summary":{}},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _DiscountSaleClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/sales');
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.headers['Idempotency-Key'], 'checkout-9');
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    expect(body['discount_minor'], 50);
    expect(body['manager_pin'], '1234');
    const response =
        '{"data":{"id":"sale-9","subtotal_minor":2000,"total_minor":1950,'
        '"currency":"EGP","payment_method":"cash","discount_capped":true,'
        '"discount_warning":"capped at 5%"},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _PlainSaleClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/sales');
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    expect(body.containsKey('discount_minor'), isFalse);
    expect(body.containsKey('manager_pin'), isFalse);
    const response =
        '{"data":{"id":"sale-10","subtotal_minor":2000,"total_minor":2000,'
        '"currency":"EGP","payment_method":"cash"},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _ClientEventClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/client/events');
    expect(request.headers['Authorization'], 'Bearer access-token');
    expect(request.headers['content-type'], 'application/json');
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    expect(body['event'], 'crash');
    expect(body['app_version'], '0.1.0+1');
    expect(body['screen'], 'pos');
    expect(body['stack_trace'], 'trace line');
    expect(body['payload'], {'exception': 'boom'});
    const response =
        '{"data":{"id":"event-1","event":"crash"},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      201,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _FailingClientEventClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    const response = '{"error":{"code":"internal_error","message":"nope"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      500,
      headers: {'content-type': 'application/json'},
    );
  }
}

const _analyticsJson =
    '{"data":{"tenant":{"id":"t1","name":"Demo Restaurant","slug":'
    '"demo-restaurant","business_type":"restaurant","country_code":"EG",'
    '"currency_code":"EGP","plan":"trial","status":"active"},"counts":'
    '{"users":3,"products":14},"today":{"date":"2026-09-17",'
    '"revenue_minor":28000,"sales_count":5,"avg_sale_minor":5600,'
    '"items_sold":9},"revenue_trend":[{"day":"2026-09-11",'
    '"revenue_minor":1200}],"top_products":[{"product_name":"Espresso",'
    '"sku":"ESP-001","quantity":4,"revenue_minor":1120}],'
    '"recent_sales":[{"id":"s1","status":"completed","total_minor":5600,'
    '"currency":"EGP","payment_method":"cash","cashier":"Amina",'
    '"created_at":"2026-09-17 10:00:00"}]},"meta":{"request_id":"t"}}';

class _SaasAnalyticsClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(request.url.path, '/v1/saas/tenants/t1/analytics');
    expect(request.headers['Authorization'], 'Bearer access-token');
    return http.StreamedResponse(
      Stream.value(_analyticsJson.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

const _entitlementJson =
    '{"data":{"entitlements":[{"id":"e1","organization_id":"org1",'
    '"owner_user_id":"u1","account_id":"acct1","trial_type":"standard",'
    '"status":"active","started_at":"2026-09-01T00:00:00Z",'
    '"expires_at":"2026-09-15T00:00:00Z","trial_days":14,"source":"signup",'
    '"eligibility_key":"org:org1","reason":"manual"}]},'
    '"meta":{"request_id":"t"}}';

class _PlatformTrialsClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(
      request.url.path,
      '/v1/platform/trial/entitlements',
    );
    expect(request.headers['Authorization'], 'Bearer access-token');
    return http.StreamedResponse(
      Stream.value(_entitlementJson.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _TrialExtendClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/platform/trial/entitlements/e1/extend');
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    expect(body['extra_days'], 7);
    const response =
        '{"data":{"id":"e1","organization_id":"org1","owner_user_id":"u1",'
        '"account_id":"acct1","trial_type":"standard","status":"active",'
        '"started_at":"2026-09-01T00:00:00Z","expires_at":"2026-09-22T00:00:00Z",'
        '"trial_days":21,"source":"admin","eligibility_key":"org:org1",'
        '"reason":"extended by admin"},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

const _auditJson =
    '{"data":{"entries":[{"id":"a1","created_at":"2026-09-17T10:00:00Z",'
    '"actor_user_id":"u1","account_id":"","tenant_id":"t1",'
    '"action":"trial_extended","entity_type":"trial","entity_id":"e1",'
    '"reason":"admin extend by 7 days","ip":"1.2.3.4"}]},'
    '"meta":{"request_id":"t"}}';

class _PlatformAuditClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'GET');
    expect(request.url.path, '/v1/platform/audit');
    expect(request.headers['Authorization'], 'Bearer access-token');
    return http.StreamedResponse(
      Stream.value(_auditJson.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

const _policyJson =
    '{"data":{"trial_duration_days":14,"trial_scope":"account",'
    '"require_email_verification":true,"require_phone_verification":false,'
    '"require_device_integrity":false,"max_organizations_per_account":3,'
    '"max_active_installations":10,"suspicious_registration_policy":"review",'
    '"register_rate_per_ip_per_hour":5,"promo_trials_enabled":false,'
    '"offline_trial_policy":"grace24h"},"meta":{"request_id":"t"}}';

class _TrialSettingsClient extends http.BaseClient {
  bool _updated = false;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.headers['Authorization'], 'Bearer access-token');
    if (request.method == 'GET') {
      expect(request.url.path, '/v1/platform/trial/settings');
      return http.StreamedResponse(
        Stream.value(_policyJson.codeUnits),
        200,
        headers: {'content-type': 'application/json'},
      );
    }
    expect(request.method, 'PUT');
    expect(request.url.path, '/v1/platform/trial/settings');
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    expect(body['trial_duration_days'], 21);
    _updated = true;
    const updatedJson =
        '{"data":{"trial_duration_days":21,"trial_scope":"account",'
        '"require_email_verification":true,"require_phone_verification":false,'
        '"require_device_integrity":false,"max_organizations_per_account":3,'
        '"max_active_installations":10,"suspicious_registration_policy":"review",'
        '"register_rate_per_ip_per_hour":5,"promo_trials_enabled":false,'
        '"offline_trial_policy":"grace24h"},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(updatedJson.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}

class _SuspendTenantClient extends http.BaseClient {
  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, 'POST');
    expect(request.url.path, '/v1/platform/tenants/t1/suspend');
    expect(request.headers['Authorization'], 'Bearer access-token');
    final body = jsonDecode(await request.finalize().bytesToString())
        as Map<String, dynamic>;
    expect(body['reason'], 'abuse');
    const response = '{"data":{"suspended":true},"meta":{"request_id":"t"}}';
    return http.StreamedResponse(
      Stream.value(response.codeUnits),
      200,
      headers: {'content-type': 'application/json'},
    );
  }
}
