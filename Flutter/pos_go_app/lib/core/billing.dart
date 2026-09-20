/// Wire models for the SaaS billing control plane: the plan catalog,
/// tenant subscriptions, invoices/charges through the payment gateway,
/// the billing summary, and platform-wide user administration.
library;

class BillingPlan {
  const BillingPlan({
    required this.id,
    required this.code,
    required this.name,
    required this.priceMinor,
    required this.currency,
    required this.billingPeriod,
    this.description = '',
    this.features = const [],
    this.maxUsers = 0,
    this.maxProducts = 0,
    this.isActive = true,
  });

  factory BillingPlan.fromJson(Map<String, dynamic> json) => BillingPlan(
        id: json['id'] as String,
        code: json['code'] as String? ?? '',
        name: json['name'] as String? ?? '',
        description: json['description'] as String? ?? '',
        priceMinor: (json['price_minor'] as num?)?.toInt() ?? 0,
        currency: json['currency'] as String? ?? 'EGP',
        billingPeriod: json['billing_period'] as String? ?? 'monthly',
        features: (json['features'] as List<dynamic>? ?? const [])
            .map((e) => e.toString())
            .toList(),
        maxUsers: (json['max_users'] as num?)?.toInt() ?? 0,
        maxProducts: (json['max_products'] as num?)?.toInt() ?? 0,
        isActive: json['is_active'] as bool? ?? true,
      );

  final String id;
  final String code;
  final String name;
  final String description;
  final int priceMinor;
  final String currency;
  final String billingPeriod;
  final List<String> features;
  final int maxUsers;
  final int maxProducts;
  final bool isActive;
}

class BillingSubscription {
  const BillingSubscription({
    required this.id,
    required this.tenantId,
    required this.tenantName,
    required this.planCode,
    required this.planName,
    required this.status,
    required this.provider,
    required this.createdAt,
    this.tenantSlug = '',
    this.planId = '',
    this.priceMinor = 0,
    this.currency = 'EGP',
    this.billingPeriod = 'monthly',
    this.trialEndsAt = '',
    this.currentPeriodStart = '',
    this.currentPeriodEnd = '',
    this.cancelAtPeriodEnd = false,
    this.cancelledAt = '',
  });

  factory BillingSubscription.fromJson(Map<String, dynamic> json) =>
      BillingSubscription(
        id: json['id'] as String,
        tenantId: json['tenant_id'] as String? ?? '',
        tenantName: json['tenant_name'] as String? ?? '',
        tenantSlug: json['tenant_slug'] as String? ?? '',
        planId: json['plan_id'] as String? ?? '',
        planCode: json['plan_code'] as String? ?? '',
        planName: json['plan_name'] as String? ?? '',
        priceMinor: (json['price_minor'] as num?)?.toInt() ?? 0,
        currency: json['currency'] as String? ?? 'EGP',
        billingPeriod: json['billing_period'] as String? ?? 'monthly',
        status: json['status'] as String? ?? '',
        provider: json['provider'] as String? ?? '',
        trialEndsAt: json['trial_ends_at'] as String? ?? '',
        currentPeriodStart: json['current_period_start'] as String? ?? '',
        currentPeriodEnd: json['current_period_end'] as String? ?? '',
        cancelAtPeriodEnd: json['cancel_at_period_end'] as bool? ?? false,
        cancelledAt: json['cancelled_at'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String tenantId;
  final String tenantName;
  final String tenantSlug;
  final String planId;
  final String planCode;
  final String planName;
  final int priceMinor;
  final String currency;
  final String billingPeriod;
  final String status;
  final String provider;
  final String trialEndsAt;
  final String currentPeriodStart;
  final String currentPeriodEnd;
  final bool cancelAtPeriodEnd;
  final String cancelledAt;
  final String createdAt;
}

class BillingSubscriptionsPage {
  const BillingSubscriptionsPage({
    required this.subscriptions,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<BillingSubscription> subscriptions;
  final int total;
  final int page;
  final int limit;
}

class BillingInvoice {
  const BillingInvoice({
    required this.id,
    required this.tenantId,
    required this.tenantName,
    required this.amountMinor,
    required this.currency,
    required this.status,
    required this.provider,
    required this.description,
    required this.createdAt,
    this.providerRef = '',
    this.dueAt = '',
    this.paidAt = '',
  });

  factory BillingInvoice.fromJson(Map<String, dynamic> json) => BillingInvoice(
        id: json['id'] as String,
        tenantId: json['tenant_id'] as String? ?? '',
        tenantName: json['tenant_name'] as String? ?? '',
        amountMinor: (json['amount_minor'] as num?)?.toInt() ?? 0,
        currency: json['currency'] as String? ?? 'EGP',
        status: json['status'] as String? ?? '',
        provider: json['provider'] as String? ?? '',
        providerRef: json['provider_ref'] as String? ?? '',
        description: json['description'] as String? ?? '',
        dueAt: json['due_at'] as String? ?? '',
        paidAt: json['paid_at'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String tenantId;
  final String tenantName;
  final int amountMinor;
  final String currency;
  final String status;
  final String provider;
  final String providerRef;
  final String description;
  final String dueAt;
  final String paidAt;
  final String createdAt;

  bool get isOpen => status == 'open';
  bool get isPaid => status == 'paid';
}

class BillingInvoicesPage {
  const BillingInvoicesPage({
    required this.invoices,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<BillingInvoice> invoices;
  final int total;
  final int page;
  final int limit;
}

class BillingRecentInvoice {
  const BillingRecentInvoice({
    required this.id,
    required this.tenantName,
    required this.amountMinor,
    required this.currency,
    required this.status,
    required this.createdAt,
  });

  factory BillingRecentInvoice.fromJson(Map<String, dynamic> json) =>
      BillingRecentInvoice(
        id: json['id'] as String,
        tenantName: json['tenant_name'] as String? ?? '',
        amountMinor: (json['amount_minor'] as num?)?.toInt() ?? 0,
        currency: json['currency'] as String? ?? 'EGP',
        status: json['status'] as String? ?? '',
        createdAt: json['created_at'] as String? ?? '',
      );

  final String id;
  final String tenantName;
  final int amountMinor;
  final String currency;
  final String status;
  final String createdAt;
}

class BillingSummaryData {
  const BillingSummaryData({
    required this.activePlans,
    required this.subscriptionCounts,
    required this.mrrMinor,
    required this.currency,
    required this.outstandingMinor,
    required this.providers,
    required this.recentInvoices,
  });

  factory BillingSummaryData.fromJson(Map<String, dynamic> json) {
    final subs = json['subscriptions'] as Map<String, dynamic>? ?? const {};
    return BillingSummaryData(
      activePlans: (json['active_plans'] as num?)?.toInt() ?? 0,
      subscriptionCounts: {
        for (final e in subs.entries) e.key: (e.value as num?)?.toInt() ?? 0,
      },
      mrrMinor: (json['mrr_minor'] as num?)?.toInt() ?? 0,
      currency: json['currency'] as String? ?? 'EGP',
      outstandingMinor: (json['outstanding_minor'] as num?)?.toInt() ?? 0,
      providers: (json['providers'] as List<dynamic>? ?? const [])
          .map((e) => e.toString())
          .toList(),
      recentInvoices: (json['recent_invoices'] as List<dynamic>? ?? const [])
          .map((e) => BillingRecentInvoice.fromJson(e as Map<String, dynamic>))
          .toList(),
    );
  }

  final int activePlans;
  final Map<String, int> subscriptionCounts;
  final int mrrMinor;
  final String currency;
  final int outstandingMinor;
  final List<String> providers;
  final List<BillingRecentInvoice> recentInvoices;
}

class PaymentProvider {
  const PaymentProvider({required this.name, this.description = ''});

  factory PaymentProvider.fromJson(Map<String, dynamic> json) =>
      PaymentProvider(
        name: json['name'] as String? ?? '',
        description: json['description'] as String? ?? '',
      );

  final String name;
  final String description;
}

class SaasUser {
  const SaasUser({
    required this.id,
    required this.tenantId,
    required this.tenantName,
    required this.email,
    required this.displayName,
    required this.role,
    required this.isActive,
    this.accountType = '',
    this.accessLevel = '',
  });

  factory SaasUser.fromJson(Map<String, dynamic> json) => SaasUser(
        id: json['id'] as String,
        tenantId: json['tenant_id'] as String? ?? '',
        tenantName: json['tenant_name'] as String? ?? '',
        email: json['email'] as String? ?? '',
        displayName: json['display_name'] as String? ?? '',
        role: json['role'] as String? ?? '',
        accountType: json['account_type'] as String? ?? '',
        isActive: json['is_active'] as bool? ?? true,
        accessLevel: json['access_level'] as String? ?? '',
      );

  final String id;
  final String tenantId;
  final String tenantName;
  final String email;
  final String displayName;
  final String role;
  final String accountType;
  final bool isActive;
  final String accessLevel;
}

class SaasUsersPage {
  const SaasUsersPage({
    required this.users,
    required this.total,
    required this.page,
    required this.limit,
  });

  final List<SaasUser> users;
  final int total;
  final int page;
  final int limit;
}

/// Subscription lifecycle states the platform UI can drive.
const subscriptionStatuses = [
  'trial',
  'active',
  'grace_period',
  'past_due',
  'suspended',
  'cancelled',
];

/// Invoice lifecycle states the platform UI can drive.
const invoiceStatuses = ['open', 'paid', 'void', 'refunded'];

/// Roles the platform can assign to a store user.
const platformUserRoles = ['owner', 'manager', 'cashier'];

/// Account types the platform can assign to a store user.
const platformAccountTypes = ['standard', 'demo', 'guest'];