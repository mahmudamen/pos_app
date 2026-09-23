/// Wire models for the SaaS control-plane surface: platform summary,
/// per-tenant analytics drill-down, trial entitlement administration,
/// audit log, and the admin-adjustable trial policy.
library;

class SaasSummary {
  const SaasSummary({
    required this.totalTenants,
    required this.totalUsers,
    required this.totalSales,
    required this.revenueMinor,
    required this.byBusiness,
    this.supportedCountry = 'EG',
    this.supportedCurrency = 'EGP',
  });

  factory SaasSummary.fromJson(Map<String, dynamic> json) => SaasSummary(
        totalTenants: (json['total_tenants'] as num).toInt(),
        totalUsers: (json['total_users'] as num).toInt(),
        totalSales: (json['total_sales'] as num).toInt(),
        revenueMinor: (json['revenue_minor'] as num).toInt(),
        byBusiness: (json['by_business'] as List<dynamic>? ?? const [])
            .map((item) =>
                SaasBusinessCount.fromJson(item as Map<String, dynamic>))
            .toList(),
        supportedCountry: json['supported_country'] as String? ?? 'EG',
        supportedCurrency: json['supported_currency'] as String? ?? 'EGP',
      );

  final int totalTenants;
  final int totalUsers;
  final int totalSales;
  final int revenueMinor;
  final List<SaasBusinessCount> byBusiness;
  final String supportedCountry;
  final String supportedCurrency;
}

class SaasBusinessCount {
  const SaasBusinessCount({
    required this.businessType,
    required this.tenants,
    this.users = 0,
    this.revenueMinor = 0,
  });

  factory SaasBusinessCount.fromJson(Map<String, dynamic> json) =>
      SaasBusinessCount(
        businessType: json['business_type'] as String? ?? '',
        tenants: (json['tenants'] as num?)?.toInt() ?? 0,
        users: (json['users'] as num?)?.toInt() ?? 0,
        revenueMinor: (json['revenue_minor'] as num?)?.toInt() ?? 0,
      );

  final String businessType;
  final int tenants;
  final int users;
  final int revenueMinor;
}

class SaasTenant {
  const SaasTenant({
    required this.id,
    required this.name,
    required this.slug,
    required this.businessType,
    required this.countryCode,
    required this.currencyCode,
    required this.defaultLanguage,
    required this.users,
    required this.products,
    this.plan = 'standard',
    this.planFeatures = const [],
    this.subscriptionStatus = '',
    this.subdomain = '',
    this.maxUsers = 0,
    this.maxProducts = 0,
    this.status = '',
    this.revenueMinor = 0,
    this.totalSales = 0,
    this.createdAt = '',
    this.trialEndsAt = '',
    this.ownerUserId = '',
  });

  factory SaasTenant.fromJson(Map<String, dynamic> json) => SaasTenant(
        id: json['id'] as String,
        name: json['name'] as String,
        slug: json['slug'] as String? ?? '',
        businessType: json['business_type'] as String? ?? '',
        countryCode: json['country_code'] as String? ?? '',
        currencyCode: json['currency_code'] as String? ?? '',
        defaultLanguage: json['default_language'] as String? ?? '',
        users: (json['users'] as num?)?.toInt() ?? 0,
        products: (json['products'] as num?)?.toInt() ?? 0,
        plan: json['plan'] as String? ?? 'standard',
        planFeatures: (json['plan_features'] as List<dynamic>? ?? const [])
            .map((e) => e.toString())
            .toList(),
        subscriptionStatus: json['subscription_status'] as String? ?? '',
        subdomain: json['subdomain'] as String? ?? '',
        maxUsers: (json['max_users'] as num?)?.toInt() ?? 0,
        maxProducts: (json['max_products'] as num?)?.toInt() ?? 0,
        status: json['status'] as String? ?? '',
        revenueMinor: (json['revenue_minor'] as num?)?.toInt() ?? 0,
        totalSales: (json['total_sales'] as num?)?.toInt() ?? 0,
        createdAt: json['created_at'] as String? ?? '',
        trialEndsAt: json['trial_ends_at'] as String? ?? '',
        ownerUserId: json['owner_user_id'] as String? ?? '',
      );

  final String id;
  final String name;
  final String slug;
  final String businessType;
  final String countryCode;
  final String currencyCode;
  final String defaultLanguage;
  final int users;
  final int products;
  final String plan;
  final List<String> planFeatures;
  final String subscriptionStatus;
  final String subdomain;
  final int maxUsers;
  final int maxProducts;
  final String status;
  final int revenueMinor;
  final int totalSales;
  final String createdAt;
  final String trialEndsAt;
  final String ownerUserId;

  bool get hasSubdomain => planFeatures.contains('pos.subdomain');
}

class SaasTenantsPage {
  const SaasTenantsPage({
    required this.tenants,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<SaasTenant> tenants;
  final int total;
  final int page;
  final int limit;
}

class TenantAnalytics {
  const TenantAnalytics({
    required this.tenant,
    required this.users,
    required this.products,
    required this.today,
    required this.revenueTrend,
    required this.topProducts,
    required this.recentSales,
  });

  factory TenantAnalytics.fromJson(Map<String, dynamic> json) =>
      TenantAnalytics(
        tenant: AnalyticsTenant.fromJson(
            json['tenant'] as Map<String, dynamic>),
        users: (json['counts']['users'] as num?)?.toInt() ?? 0,
        products: (json['counts']['products'] as num?)?.toInt() ?? 0,
        today: TodayStats.fromJson(json['today'] as Map<String, dynamic>),
        revenueTrend: (json['revenue_trend'] as List<dynamic>? ?? const [])
            .map((e) => RevenueTrendPoint.fromJson(e as Map<String, dynamic>))
            .toList(),
        topProducts: (json['top_products'] as List<dynamic>? ?? const [])
            .map((e) => TopProduct.fromJson(e as Map<String, dynamic>))
            .toList(),
        recentSales: (json['recent_sales'] as List<dynamic>? ?? const [])
            .map((e) => AnalyticsRecentSale.fromJson(e as Map<String, dynamic>))
            .toList(),
      );

  final AnalyticsTenant tenant;
  final int users;
  final int products;
  final TodayStats today;
  final List<RevenueTrendPoint> revenueTrend;
  final List<TopProduct> topProducts;
  final List<AnalyticsRecentSale> recentSales;
}

class AnalyticsTenant {
  const AnalyticsTenant({
    required this.id,
    required this.name,
    required this.slug,
    required this.businessType,
    required this.countryCode,
    required this.currencyCode,
    required this.defaultLanguage,
    required this.plan,
    this.planFeatures = const [],
    this.subscriptionStatus = '',
    this.subdomain = '',
    this.maxUsers = 0,
    this.maxProducts = 0,
    this.status = '',
    this.createdAt = '',
    this.trialEndsAt = '',
  });

  factory AnalyticsTenant.fromJson(Map<String, dynamic> json) =>
      AnalyticsTenant(
        id: json['id'] as String,
        name: json['name'] as String,
        slug: json['slug'] as String? ?? '',
        businessType: json['business_type'] as String? ?? '',
        countryCode: json['country_code'] as String? ?? '',
        currencyCode: json['currency_code'] as String? ?? '',
        defaultLanguage: json['default_language'] as String? ?? '',
        plan: json['plan'] as String? ?? 'standard',
        planFeatures: (json['plan_features'] as List<dynamic>? ?? const [])
            .map((e) => e.toString())
            .toList(),
        subscriptionStatus: json['subscription_status'] as String? ?? '',
        subdomain: json['subdomain'] as String? ?? '',
        maxUsers: (json['max_users'] as num?)?.toInt() ?? 0,
        maxProducts: (json['max_products'] as num?)?.toInt() ?? 0,
        status: json['status'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
        trialEndsAt: json['trial_ends_at'] as String? ?? '',
      );

  final String id;
  final String name;
  final String slug;
  final String businessType;
  final String countryCode;
  final String currencyCode;
  final String defaultLanguage;
  final String plan;
  final List<String> planFeatures;
  final String subscriptionStatus;
  final String subdomain;
  final int maxUsers;
  final int maxProducts;
  final String status;
  final String createdAt;
  final String trialEndsAt;

  bool get hasSubdomain => planFeatures.contains('pos.subdomain');
}

class TodayStats {
  const TodayStats({
    required this.date,
    required this.revenueMinor,
    required this.salesCount,
    required this.avgSaleMinor,
    required this.itemsSold,
  });

  factory TodayStats.fromJson(Map<String, dynamic> json) => TodayStats(
        date: json['date'] as String? ?? '',
        revenueMinor: (json['revenue_minor'] as num?)?.toInt() ?? 0,
        salesCount: (json['sales_count'] as num?)?.toInt() ?? 0,
        avgSaleMinor: (json['avg_sale_minor'] as num?)?.toInt() ?? 0,
        itemsSold: (json['items_sold'] as num?)?.toInt() ?? 0,
      );

  final String date;
  final int revenueMinor;
  final int salesCount;
  final int avgSaleMinor;
  final int itemsSold;
}

class RevenueTrendPoint {
  const RevenueTrendPoint({required this.day, required this.revenueMinor});

  factory RevenueTrendPoint.fromJson(Map<String, dynamic> json) =>
      RevenueTrendPoint(
        day: json['day'] as String? ?? '',
        revenueMinor: (json['revenue_minor'] as num?)?.toInt() ?? 0,
      );

  final String day;
  final int revenueMinor;
}

class TopProduct {
  const TopProduct({
    required this.productName,
    required this.sku,
    required this.quantity,
    required this.revenueMinor,
  });

  factory TopProduct.fromJson(Map<String, dynamic> json) => TopProduct(
        productName: json['product_name'] as String? ?? '',
        sku: json['sku'] as String? ?? '',
        quantity: (json['quantity'] as num?)?.toInt() ?? 0,
        revenueMinor: (json['revenue_minor'] as num?)?.toInt() ?? 0,
      );

  final String productName;
  final String sku;
  final int quantity;
  final int revenueMinor;
}

class AnalyticsRecentSale {
  const AnalyticsRecentSale({
    required this.id,
    required this.status,
    required this.totalMinor,
    required this.currency,
    required this.paymentMethod,
    required this.cashier,
    required this.createdAt,
  });

  factory AnalyticsRecentSale.fromJson(Map<String, dynamic> json) =>
      AnalyticsRecentSale(
        id: json['id'] as String? ?? '',
        status: json['status'] as String? ?? '',
        totalMinor: (json['total_minor'] as num?)?.toInt() ?? 0,
        currency: json['currency'] as String? ?? 'EGP',
        paymentMethod: json['payment_method'] as String? ?? 'cash',
        cashier: json['cashier'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String status;
  final int totalMinor;
  final String currency;
  final String paymentMethod;
  final String cashier;
  final String createdAt;
}

class TrialEntitlement {
  const TrialEntitlement({
    required this.id,
    required this.organizationId,
    required this.ownerUserId,
    required this.accountId,
    required this.trialType,
    required this.status,
    required this.startedAt,
    required this.expiresAt,
    this.trialDays = 0,
    required this.consumedAt,
    required this.source,
    required this.eligibilityKey,
    required this.reason,
  });

  factory TrialEntitlement.fromJson(Map<String, dynamic> json) =>
      TrialEntitlement(
        id: json['id'] as String,
        organizationId: json['organization_id'] as String? ?? '',
        ownerUserId: json['owner_user_id'] as String? ?? '',
        accountId: json['account_id'] as String? ?? '',
        trialType: json['trial_type'] as String? ?? '',
        status: json['status'] as String? ?? '',
        startedAt: json['started_at'] as String? ?? '',
        expiresAt: json['expires_at'] as String? ?? '',
        trialDays: (json['trial_days'] as num?)?.toInt() ?? 0,
        consumedAt: json['consumed_at'] as String? ?? '',
        source: json['source'] as String? ?? '',
        eligibilityKey: json['eligibility_key'] as String? ?? '',
        reason: json['reason'] as String? ?? '',
      );

  final String id;
  final String organizationId;
  final String ownerUserId;
  final String accountId;
  final String trialType;
  final String status;
  final String startedAt;
  final String expiresAt;
  final int trialDays;
  final String consumedAt;
  final String source;
  final String eligibilityKey;
  final String reason;
}

class TrialEntitlementsPage {
  const TrialEntitlementsPage({required this.entitlements});

  factory TrialEntitlementsPage.fromJson(Map<String, dynamic> json) =>
      TrialEntitlementsPage(
        entitlements: (json['entitlements'] as List<dynamic>? ?? const [])
            .map((e) => TrialEntitlement.fromJson(e as Map<String, dynamic>))
            .toList(),
      );

  final List<TrialEntitlement> entitlements;
}

class AuditEntry {
  const AuditEntry({
    required this.id,
    required this.createdAt,
    required this.actorUserId,
    required this.accountId,
    required this.tenantId,
    required this.action,
    required this.entityType,
    required this.entityId,
    required this.reason,
    required this.ip,
  });

  factory AuditEntry.fromJson(Map<String, dynamic> json) => AuditEntry(
        id: json['id'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
        actorUserId: json['actor_user_id'] as String? ?? '',
        accountId: json['account_id'] as String? ?? '',
        tenantId: json['tenant_id'] as String? ?? '',
        action: json['action'] as String? ?? '',
        entityType: json['entity_type'] as String? ?? '',
        entityId: json['entity_id'] as String? ?? '',
        reason: json['reason'] as String? ?? '',
        ip: json['ip'] as String? ?? '',
      );

  final String id;
  final String createdAt;
  final String actorUserId;
  final String accountId;
  final String tenantId;
  final String action;
  final String entityType;
  final String entityId;
  final String reason;
  final String ip;
}

class AuditPage {
  const AuditPage({required this.entries});

  factory AuditPage.fromJson(Map<String, dynamic> json) {
    final data = json['data']?['entries'] as List<dynamic>? ??
        (json['entries'] as List<dynamic>? ?? const []);
    return AuditPage(
      entries: data
          .map((e) => AuditEntry.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  final List<AuditEntry> entries;
}

class TrialPolicy {
  const TrialPolicy({
    this.trialDurationDays = 14,
    this.trialScope = 'account',
    this.requireEmailVerification = false,
    this.requirePhoneVerification = false,
    this.requireDeviceIntegrity = false,
    this.maxOrganizationsPerAccount = 3,
    this.maxActiveInstallations = 10,
    this.suspiciousRegistrationPolicy = 'review',
    this.registerRatePerIpPerHour = 5,
    this.promoTrialsEnabled = false,
    this.offlineTrialPolicy = 'grace24h',
  });

  factory TrialPolicy.fromJson(Map<String, dynamic> json) => TrialPolicy(
        trialDurationDays:
            (json['trial_duration_days'] as num?)?.toInt() ?? 14,
        trialScope: json['trial_scope'] as String? ?? 'account',
        requireEmailVerification:
            json['require_email_verification'] as bool? ?? false,
        requirePhoneVerification:
            json['require_phone_verification'] as bool? ?? false,
        requireDeviceIntegrity:
            json['require_device_integrity'] as bool? ?? false,
        maxOrganizationsPerAccount:
            (json['max_organizations_per_account'] as num?)?.toInt() ?? 3,
        maxActiveInstallations:
            (json['max_active_installations'] as num?)?.toInt() ?? 10,
        suspiciousRegistrationPolicy:
            json['suspicious_registration_policy'] as String? ?? 'review',
        registerRatePerIpPerHour:
            (json['register_rate_per_ip_per_hour'] as num?)?.toInt() ?? 5,
        promoTrialsEnabled: json['promo_trials_enabled'] as bool? ?? false,
        offlineTrialPolicy:
            json['offline_trial_policy'] as String? ?? 'grace24h',
      );

  final int trialDurationDays;
  final String trialScope;
  final bool requireEmailVerification;
  final bool requirePhoneVerification;
  final bool requireDeviceIntegrity;
  final int maxOrganizationsPerAccount;
  final int maxActiveInstallations;
  final String suspiciousRegistrationPolicy;
  final int registerRatePerIpPerHour;
  final bool promoTrialsEnabled;
  final String offlineTrialPolicy;

  Map<String, dynamic> toJson() => {
        'trial_duration_days': trialDurationDays,
        'trial_scope': trialScope,
        'require_email_verification': requireEmailVerification,
        'require_phone_verification': requirePhoneVerification,
        'require_device_integrity': requireDeviceIntegrity,
        'max_organizations_per_account': maxOrganizationsPerAccount,
        'max_active_installations': maxActiveInstallations,
        'suspicious_registration_policy': suspiciousRegistrationPolicy,
        'register_rate_per_ip_per_hour': registerRatePerIpPerHour,
        'promo_trials_enabled': promoTrialsEnabled,
        'offline_trial_policy': offlineTrialPolicy,
      };
}