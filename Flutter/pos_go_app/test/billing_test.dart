import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/billing.dart';
import 'package:pos_go_app/core/session_store.dart';

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

const _subJson = '''{
  "id": "sub-1",
  "tenant_id": "t-1",
  "tenant_name": "Nile Cafe",
  "tenant_slug": "nile-cafe",
  "plan_id": "p-1",
  "plan_code": "premium",
  "plan_name": "Premium",
  "price_minor": 29900,
  "currency": "EGP",
  "billing_period": "monthly",
  "status": "active",
  "provider": "manual",
  "trial_ends_at": "2026-09-01T00:00:00Z",
  "current_period_end": "2026-10-01T00:00:00Z",
  "cancel_at_period_end": false,
  "created_at": "2026-09-01T09:00:00Z"
}''';

const _invoiceJson = '''{
  "id": "inv-1",
  "tenant_id": "t-1",
  "tenant_name": "Nile Cafe",
  "amount_minor": 29900,
  "currency": "EGP",
  "status": "paid",
  "provider": "manual",
  "provider_ref": "ch_123",
  "description": "September plan",
  "due_at": "2026-09-30",
  "paid_at": "2026-09-01T09:30:00Z",
  "created_at": "2026-09-01T09:00:00Z"
}''';

class _BillingClient extends http.BaseClient {
  _BillingClient(this.path, {required this.method});

  final String path;
  final String method;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    expect(request.method, method);
    expect(request.url.path, path);
    expect(request.headers['Authorization'], 'Bearer access-token');
    late final String body;
    String data = '{}';
    int status = 200;
    if (path.endsWith('/saas/billing/summary')) {
      data =
          '{"active_plans":2,"subscriptions":{"active":3,"trial":1},"mrr_minor":89700,"currency":"EGP","outstanding_minor":15000,"providers":["manual","card"],"recent_invoices":[$_invoiceJson]}';
    } else if (path.endsWith('/saas/plans')) {
      data =
          '[{"id":"p-1","code":"premium","name":"Premium","price_minor":29900,"currency":"EGP","billing_period":"monthly","is_active":true}]';
    } else if (path.endsWith('/saas/payment-providers')) {
      data =
          '[{"name":"manual","description":"Manual payout"},{"name":"card","description":"Card gateway"}]';
    } else if (path.endsWith('/saas/subscriptions')) {
      body =
          '{"data":[$_subJson],"meta":{"total":1,"page":1,"limit":50}}';
      return _respond(body);
    } else if (path.endsWith('/saas/invoices')) {
      body =
          '{"data":[$_invoiceJson],"meta":{"total":1,"page":1,"limit":50}}';
      return _respond(body);
    } else if (path.endsWith('/subscription')) {
      data = _subJson;
      status = 201;
    } else if (path.endsWith('/status')) {
      data = _subJson;
    } else if (path.endsWith('/change-plan')) {
      data = _subJson;
    } else if (path.endsWith('/invoices')) {
      data = _invoiceJson;
      status = 201;
    } else if (path.endsWith('/pay')) {
      data = _invoiceJson;
    } else if (path.endsWith('/void')) {
      data = '{${_invoiceJson.substring(1, _invoiceJson.length - 1)},"status":"void"}';
    } else if (path.endsWith('/refund')) {
      data = '{${_invoiceJson.substring(1, _invoiceJson.length - 1)},"status":"refunded"}';
    } else if (path.endsWith('/saas/users')) {
      body =
          '{"data":[{"id":"u-1","tenant_id":"t-1","tenant_name":"Nile Cafe","email":"cashier@nile-cafe.com","display_name":"Sara","role":"cashier","account_type":"standard","is_active":true,"access_level":"cashier"}],"meta":{"total":1,"page":1,"limit":50}}';
      return _respond(body);
    } else if (path.contains('/saas/tenants/') && path.endsWith('/users')) {
      data =
          '{"id":"u-2","tenant_id":"t-1","tenant_name":"Nile Cafe","email":"owner@nile-cafe.com","display_name":"Omar","role":"owner","account_type":"standard","is_active":true,"access_level":"manager"}';
      status = 201;
    } else {
      fail('unexpected request path: ${request.url.path}');
    }
    body = jsonEncode({'data': jsonDecode(data)});
    return _respond(body, status: status);
  }

  http.StreamedResponse _respond(String body, {int status = 200}) {
    return http.StreamedResponse(
      Stream.value(const Utf8Encoder().convert(body)),
      status,
      headers: {'content-type': 'application/json'},
    );
  }
}

void main() {
  const session = _saasAdmin;

  group('billing models', () {
    test('BillingPlan parses the wire plan', () {
      final plan = BillingPlan.fromJson({
        'id': 'p-1',
        'code': 'premium',
        'name': 'Premium',
        'price_minor': 29900,
        'currency': 'EGP',
        'billing_period': 'monthly',
        'is_active': true,
      });
      expect(plan.code, 'premium');
      expect(plan.priceMinor, 29900);
      expect(plan.isActive, isTrue);
    });

    test('BillingSubscription parses the wire subscription', () {
      final sub = BillingSubscription.fromJson(jsonDecode(_subJson));
      expect(sub.id, 'sub-1');
      expect(sub.tenantName, 'Nile Cafe');
      expect(sub.planCode, 'premium');
      expect(sub.priceMinor, 29900);
      expect(sub.status, 'active');
      expect(sub.cancelAtPeriodEnd, isFalse);
    });

    test('BillingInvoice parses wire status and helper getters', () {
      final inv = BillingInvoice.fromJson(jsonDecode(_invoiceJson));
      expect(inv.amountMinor, 29900);
      expect(inv.provider, 'manual');
      expect(inv.status, 'paid');
      expect(inv.isOpen, isFalse);
      expect(inv.isPaid, isTrue);

      final open = BillingInvoice.fromJson(
          jsonDecode('{${_invoiceJson.substring(1, _invoiceJson.length - 1)},"status":"open"}'));
      expect(open.isOpen, isTrue);
      expect(open.isPaid, isFalse);
    });

    test('BillingSummaryData parses counts, amounts and recent invoices', () {
      final summary = BillingSummaryData.fromJson({
        'active_plans': 3,
        'subscriptions': {'active': 2, 'trial': 1},
        'mrr_minor': 89700,
        'currency': 'EGP',
        'outstanding_minor': 15000,
        'providers': ['manual', 'card'],
        'recent_invoices': [jsonDecode(_invoiceJson)],
      });
      expect(summary.mrrMinor, 89700);
      expect(summary.outstandingMinor, 15000);
      expect(summary.subscriptionCounts['active'], 2);
      expect(summary.recentInvoices.single.amountMinor, 29900);
    });

    test('PaymentProvider and SaasUser parse the wire payloads', () {
      final provider = PaymentProvider.fromJson({'name': 'card'});
      expect(provider.name, 'card');

      final user = SaasUser.fromJson({
        'id': 'u-1',
        'tenant_id': 't-1',
        'tenant_name': 'Nile Cafe',
        'email': 'cashier@nile-cafe.com',
        'display_name': 'Sara',
        'role': 'cashier',
        'is_active': true,
        'access_level': 'cashier',
      });
      expect(user.displayName, 'Sara');
      expect(user.role, 'cashier');
      expect(user.isActive, isTrue);
    });

    test('lifecycle const lists expose the platform states', () {
      expect(subscriptionStatuses, contains('active'));
      expect(invoiceStatuses, contains('paid'));
      expect(platformUserRoles, contains('owner'));
      expect(platformAccountTypes, contains('guest'));
    });
  });

  group('billing API methods', () {
    test('billingSummary fetches /v1/saas/billing/summary', () async {
      final api = ApiClient(client: _BillingClient('/v1/saas/billing/summary', method: 'GET'));
      final summary = await api.billingSummary(session);
      expect(summary.activePlans, 2);
      expect(summary.mrrMinor, 89700);
      expect(summary.recentInvoices.single.id, 'inv-1');
    });

    test('billingPlans fetches /v1/saas/plans', () async {
      final api = ApiClient(client: _BillingClient('/v1/saas/plans', method: 'GET'));
      final plans = await api.billingPlans(session);
      expect(plans.single.code, 'premium');
      expect(plans.single.priceMinor, 29900);
    });

    test('paymentProviders fetches /v1/saas/payment-providers', () async {
      final api = ApiClient(client: _BillingClient('/v1/saas/payment-providers', method: 'GET'));
      final providers = await api.paymentProviders(session);
      expect(providers.map((p) => p.name), contains('card'));
    });

    test('billingSubscriptions fetches paginated list', () async {
      final api = ApiClient(client: _BillingClient('/v1/saas/subscriptions', method: 'GET'));
      final page = await api.billingSubscriptions(session);
      expect(page.subscriptions.single.tenantName, 'Nile Cafe');
      expect(page.total, 1);
    });

    test('assignSubscription POSTs plan to the tenant subscription', () async {
      final api = ApiClient(client: _BillingClient('/v1/saas/tenants/t-1/subscription', method: 'POST'));
      final sub = await api.assignSubscription(session, 't-1',
          planCode: 'premium', status: 'active', trialDays: 14);
      expect(sub.status, 'active');
      expect(sub.planCode, 'premium');
    });

    test('setSubscriptionStatus and changeSubscriptionPlan POST to sub routes',
        () async {
      final statusApi = ApiClient(client: _BillingClient('/v1/saas/subscriptions/sub-1/status', method: 'POST'));
      final updated = await statusApi.setSubscriptionStatus(session, 'sub-1', 'suspended');
      expect(updated.status, 'active');

      final planApi = ApiClient(client: _BillingClient('/v1/saas/subscriptions/sub-1/change-plan', method: 'POST'));
      final changed = await planApi.changeSubscriptionPlan(session, 'sub-1', 'basic');
      expect(changed.planCode, 'premium');
    });

    test('billingInvoices fetches the paginated invoice list', () async {
      final api = ApiClient(client: _BillingClient('/v1/saas/invoices', method: 'GET'));
      final page = await api.billingInvoices(session);
      expect(page.invoices.single.amountMinor, 29900);
      expect(page.invoices.single.isPaid, isTrue);
    });

    test('createInvoice POSTs to the tenant invoices route', () async {
      final api = ApiClient(client: _BillingClient('/v1/saas/tenants/t-1/invoices', method: 'POST'));
      final inv = await api.createInvoice(
        session,
        't-1',
        amountMinor: 29900,
        description: 'September plan',
        provider: 'manual',
        dueAt: '2026-09-30',
      );
      expect(inv.amountMinor, 29900);
      expect(inv.providerRef, 'ch_123');
    });

    test('payInvoice, voidInvoice and refundInvoice route the payment flow',
        () async {
      final payApi = ApiClient(client: _BillingClient('/v1/saas/invoices/inv-1/pay', method: 'POST'));
      final paid = await payApi.payInvoice(session, 'inv-1', provider: 'manual');
      expect(paid.isPaid, isTrue);

      final voidApi = ApiClient(client: _BillingClient('/v1/saas/invoices/inv-2/void', method: 'POST'));
      final voided = await voidApi.voidInvoice(session, 'inv-2');
      expect(voided.status, 'void');

      final refundApi = ApiClient(client: _BillingClient('/v1/saas/invoices/inv-3/refund', method: 'POST'));
      final refunded = await refundApi.refundInvoice(session, 'inv-3');
      expect(refunded.status, 'refunded');
    });

    test('saasUsers lists and createSaasUser provisions a store user',
        () async {
      final listApi = ApiClient(client: _BillingClient('/v1/saas/users', method: 'GET'));
      final users = await listApi.saasUsers(session);
      expect(users.users.single.displayName, 'Sara');
      expect(users.total, 1);

      final createApi = ApiClient(client: _BillingClient('/v1/saas/tenants/t-1/users', method: 'POST'));
      final created = await createApi.createSaasUser(
        session,
        't-1',
        email: 'owner@nile-cafe.com',
        displayName: 'Omar',
        password: 'password123',
        role: 'owner',
        accountType: 'standard',
      );
      expect(created.role, 'owner');
      expect(created.isActive, isTrue);
    });
  });
}