import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:pos_go_app/core/api_client.dart';
import 'package:pos_go_app/core/saas.dart';
import 'package:pos_go_app/core/session_store.dart';
import 'package:pos_go_app/features/saas/control_panel_screen.dart';
import 'package:pos_go_app/features/saas/tenant_analytics_screen.dart';
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

const _summary = SaasSummary(
  totalTenants: 2,
  totalUsers: 4,
  totalSales: 7,
  revenueMinor: 12500,
  byBusiness: [
    SaasBusinessCount(
        businessType: 'restaurant', tenants: 1, users: 2, revenueMinor: 8000),
    SaasBusinessCount(
        businessType: 'book_store', tenants: 1, users: 2, revenueMinor: 4500),
  ],
);

const _restaurant = SaasTenant(
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
  planFeatures: ['pos.basic', 'restaurant', 'pos.subdomain'],
  subdomain: 'nile-cafe.xamltech.com',
  subscriptionStatus: 'trial',
  status: 'active',
  revenueMinor: 8000,
  totalSales: 4,
  createdAt: '2026-09-01T09:00:00Z',
);

const _bookstore = SaasTenant(
  id: 't-2',
  name: 'Cairo Books',
  slug: 'cairo-books',
  businessType: 'book_store',
  countryCode: 'EG',
  currencyCode: 'EGP',
  defaultLanguage: 'ar',
  users: 2,
  products: 80,
  plan: 'premium',
  planFeatures: ['pos.basic', 'book_store', 'pos.subdomain'],
  subdomain: 'cairo-books.xamltech.com',
  subscriptionStatus: 'active',
  status: 'suspended',
  revenueMinor: 4500,
  totalSales: 3,
  createdAt: '2026-09-02T09:00:00Z',
);

class _FakeSaasApi extends ApiClient {
  _FakeSaasApi();

  List<SaasTenant> tenants = const [_restaurant, _bookstore];
  int activateCalls = 0;
  int suspendCalls = 0;
  int stopCalls = 0;
  int backupCalls = 0;
  SaasTenantsPage? lastPage;

  String? filterQuery;
  String? filterBusinessType;
  String? filterStatus;
  String? filterPlan;

  @override
  Future<SaasSummary> saasSummary(Session session) async => _summary;

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
    filterQuery = q;
    filterBusinessType = businessType;
    filterStatus = status;
    filterPlan = plan;
    final filtered = tenants.where((t) {
      if (q != null && q.isNotEmpty && !t.name.toLowerCase().contains(q)) {
        return false;
      }
      if (businessType != null &&
          businessType.isNotEmpty &&
          t.businessType != businessType) {
        return false;
      }
      if (status != null && status.isNotEmpty && t.status != status) {
        return false;
      }
      if (plan != null && plan.isNotEmpty && t.plan != plan) {
        return false;
      }
      return true;
    }).toList();
    final start = (page - 1) * limit;
    final slice = start >= filtered.length
        ? <SaasTenant>[]
        : filtered.sublist(start, (start + limit).clamp(0, filtered.length));
    lastPage = SaasTenantsPage(
      tenants: slice,
      total: filtered.length,
      page: page,
      limit: limit,
    );
    return lastPage!;
  }

  @override
  Future<TenantAnalytics> saasTenantAnalytics(
    Session session,
    String tenantId,
  ) async {
    final r = tenants.firstWhere((t) => t.id == tenantId);
    return TenantAnalytics(
      tenant: AnalyticsTenant(
        id: r.id,
        name: r.name,
        slug: r.slug,
        businessType: r.businessType,
        countryCode: r.countryCode,
        currencyCode: r.currencyCode,
        defaultLanguage: r.defaultLanguage,
        plan: r.plan,
        planFeatures: r.planFeatures,
        subscriptionStatus: r.subscriptionStatus,
        subdomain: r.subdomain,
        status: r.status,
        createdAt: r.createdAt,
      ),
      users: r.users,
      products: r.products,
      today: const TodayStats(
        date: '2026-09-17',
        revenueMinor: 1500,
        salesCount: 2,
        avgSaleMinor: 750,
        itemsSold: 6,
      ),
      revenueTrend: const [
        RevenueTrendPoint(day: '2026-09-11', revenueMinor: 200),
        RevenueTrendPoint(day: '2026-09-12', revenueMinor: 300),
      ],
      topProducts: const [
        TopProduct(productName: 'Cappuccino', sku: 'CAP-1', quantity: 4, revenueMinor: 1200),
      ],
      recentSales: const [
        AnalyticsRecentSale(
          id: 's-1',
          status: 'completed',
          totalMinor: 700,
          currency: 'EGP',
          paymentMethod: 'cash',
          cashier: 'Cashier',
          createdAt: '2026-09-17T10:30:00Z',
        ),
      ],
    );
  }

  @override
  Future<List<TrialEntitlement>> platformTrialEntitlements(
    Session session, {
    int limit = 100,
    int offset = 0,
  }) async {
    return const [
      TrialEntitlement(
        id: 'trial-1',
        organizationId: 'org-1',
        ownerUserId: 'u-1',
        accountId: 'a-1',
        trialType: 'signup',
        status: 'active',
        startedAt: '2026-09-01T00:00:00Z',
        expiresAt: '2026-09-15T00:00:00Z',
        trialDays: 14,
        consumedAt: '',
        source: 'website',
        eligibilityKey: '',
        reason: '',
      ),
    ];
  }

  @override
  Future<List<AuditEntry>> platformAudit(
    Session session, {
    int limit = 100,
    int offset = 0,
  }) async {
    return const [
      AuditEntry(
        id: 'e-1',
        createdAt: '2026-09-16T10:00:00Z',
        actorUserId: 'user-admin',
        accountId: 'a-1',
        tenantId: 't-1',
        action: 'organization.suspended',
        entityType: 'tenant',
        entityId: 't-1',
        reason: 'testing',
        ip: '1.2.3.4',
      ),
    ];
  }

  @override
  Future<void> platformActivateTenant(
    Session session,
    String tenantId,
  ) async {
    activateCalls++;
  }

  @override
  Future<void> platformSuspendTenant(
    Session session,
    String tenantId, {
    String reason = '',
  }) async {
    suspendCalls++;
  }

  @override
  Future<void> platformStopTenant(
    Session session,
    String tenantId, {
    String reason = '',
  }) async {
    stopCalls++;
  }

  @override
  Future<String?> platformBackupTenant(
    Session session,
    String tenantId,
  ) async {
    backupCalls++;
    return '{"tenant_id":"$tenantId","tables":{"products":[]}}';
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
  testWidgets('control panel overview shows per-type tenants/users/revenue',
      (tester) async {
    final api = _FakeSaasApi();
    await tester.pumpWidget(
        _wrap(ControlPanelScreen(session: _saasAdmin, apiClient: api)));
    await tester.pumpAndSettle();

    expect(find.text('Control Panel'), findsOneWidget);
    expect(find.text('2'), findsWidgets);
    expect(
      find.textContaining('restaurant'),
      findsWidgets,
    );
    expect(find.textContaining('E£80.00'), findsWidgets);
    expect(find.textContaining('E£45.00'), findsWidgets);
  });

  testWidgets('tenants tab lists tenants, filters by status and search',
      (tester) async {
    final api = _FakeSaasApi();
    await tester.pumpWidget(
        _wrap(ControlPanelScreen(session: _saasAdmin, apiClient: api)));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Tenants').last);
    await tester.pumpAndSettle();

    expect(find.text('Nile Cafe'), findsOneWidget);
    expect(find.text('Cairo Books'), findsOneWidget);
    expect(find.textContaining('Joined on Sep 1, 2026'), findsWidgets);

    await tester.tap(find.widgetWithText(FilterChip, 'Suspended'));
    await tester.pumpAndSettle();
    expect(api.filterStatus, 'suspended');
    expect(find.text('Nile Cafe'), findsNothing);
    expect(find.text('Cairo Books'), findsOneWidget);

    await tester.enterText(find.byType(TextField), 'nile');
    await tester.pump(const Duration(milliseconds: 400));
    await tester.pumpAndSettle();
    expect(api.filterQuery, 'nile');
    expect(api.filterStatus, 'suspended');
    expect(find.text('Nile Cafe'), findsNothing);
    expect(find.text('Cairo Books'), findsNothing);
  });

  testWidgets('tenant analytics uses tenant currency and dates, '
      'activate action for suspended tenants', (tester) async {
    final api = _FakeSaasApi();
    await tester.pumpWidget(_wrap(TenantAnalyticsScreen(
      session: _saasAdmin,
      apiClient: api,
      tenantId: 't-2',
      tenantName: 'Cairo Books',
    )));
    await tester.pumpAndSettle();

    expect(find.text('Cairo Books'), findsOneWidget);
    expect(find.textContaining('E£15.00'), findsWidgets);
    expect(find.textContaining('E£7.50'), findsWidgets);
    expect(find.textContaining('Sep 2, 2026'), findsWidgets);
    expect(find.text('Suspended'), findsWidgets);

    final activate = find.byTooltip('Activate tenant');
    expect(activate, findsOneWidget);
    await tester.tap(activate);
    await tester.pumpAndSettle();
    expect(api.activateCalls, 1);
    expect(find.text('Tenant activated'), findsOneWidget);
  });

  testWidgets('tenant analytics shows suspend action for active tenants',
      (tester) async {
    final api = _FakeSaasApi();
    await tester.pumpWidget(_wrap(TenantAnalyticsScreen(
      session: _saasAdmin,
      apiClient: api,
      tenantId: 't-1',
      tenantName: 'Nile Cafe',
    )));
    await tester.pumpAndSettle();

    expect(find.byTooltip('Suspend tenant'), findsOneWidget);
    expect(find.byTooltip('Activate tenant'), findsNothing);

    await tester.tap(find.byTooltip('Suspend tenant'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Suspend tenant').last);
    await tester.pumpAndSettle();
    expect(api.suspendCalls, 1);
    expect(find.text('Tenant suspended'), findsOneWidget);
  });

  testWidgets('tenant analytics stop action requires confirm and stops',
      (tester) async {
    final api = _FakeSaasApi();
    await tester.pumpWidget(_wrap(TenantAnalyticsScreen(
      session: _saasAdmin,
      apiClient: api,
      tenantId: 't-1',
      tenantName: 'Nile Cafe',
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Stop tenant'));
    await tester.pumpAndSettle();
    expect(find.byType(TextField), findsOneWidget);

    await tester.tap(find.widgetWithText(FilledButton, 'Stop tenant'));
    await tester.pumpAndSettle();
    expect(api.stopCalls, 1);
    expect(find.text('Tenant stopped'), findsOneWidget);
  });

  testWidgets('tenant analytics backup triggers export snackbar',
      (tester) async {
    final api = _FakeSaasApi();
    await tester.pumpWidget(_wrap(TenantAnalyticsScreen(
      session: _saasAdmin,
      apiClient: api,
      tenantId: 't-1',
      tenantName: 'Nile Cafe',
    )));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Download backup'));
    await tester.pumpAndSettle();
    expect(api.backupCalls, 1);
  });

  testWidgets('tenant analytics shows subdomain chip when plan includes it',
      (tester) async {
    final api = _FakeSaasApi();
    await tester.pumpWidget(_wrap(TenantAnalyticsScreen(
      session: _saasAdmin,
      apiClient: api,
      tenantId: 't-1',
      tenantName: 'Nile Cafe',
    )));
    await tester.pumpAndSettle();

    expect(find.text('nile-cafe.xamltech.com'), findsOneWidget);
    expect(find.byTooltip('Subdomain included'), findsOneWidget);
  });

  testWidgets('control panel list shows subdomain chip on tenant cards',
      (tester) async {
    final api = _FakeSaasApi();
    await tester.pumpWidget(
        _wrap(ControlPanelScreen(session: _saasAdmin, apiClient: api)));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Tenants').last);
    await tester.pumpAndSettle();

    expect(find.text('nile-cafe.xamltech.com'), findsOneWidget);
    expect(find.text('cairo-books.xamltech.com'), findsOneWidget);
  });
}