export type BillingPeriod = 'monthly' | 'yearly'

export type SubscriptionStatus =
  | 'trial'
  | 'active'
  | 'grace_period'
  | 'past_due'
  | 'suspended'
  | 'cancelled'

export type InvoiceStatus = 'open' | 'paid' | 'void' | 'refunded'

export interface Plan {
  id: string
  code: string
  name: string
  description: string
  price_minor: number
  currency: string
  billing_period: BillingPeriod
  features: string[]
  max_users: number
  max_products: number
  is_active: boolean
}

export interface Subscription {
  id: string
  tenant_id: string
  tenant_name: string
  tenant_slug: string
  plan_id: string
  plan_code: string
  plan_name: string
  price_minor: number
  currency: string
  billing_period: BillingPeriod
  status: SubscriptionStatus
  provider: string
  trial_ends_at?: string
  current_period_start: string
  current_period_end?: string
  cancel_at_period_end: boolean
  cancelled_at?: string
  created_at: string
}

export interface Invoice {
  id: string
  tenant_id: string
  tenant_name: string
  amount_minor: number
  currency: string
  status: InvoiceStatus
  provider: string
  provider_ref?: string
  description: string
  due_at?: string
  paid_at?: string
  created_at: string
}

export interface Provider {
  name: string
  description: string
}

export interface RecentInvoice {
  id: string
  tenant_name: string
  amount_minor: number
  currency: string
  status: InvoiceStatus
  created_at: string
}

export interface BillingSummary {
  active_plans: number
  subscriptions: Record<SubscriptionStatus, number>
  mrr_minor: number
  currency: string
  outstanding_minor: number
  providers: string[]
  recent_invoices: RecentInvoice[]
}

export interface Tenant {
  id: string
  name: string
  slug: string
  business_type: string
  country_code: string
  currency_code: string
  default_language: string
  plan: string
  max_users: number
  max_products: number
  users: number
  products: number
}

export interface RevenuePoint {
  day: string
  revenue_minor: number
}

export interface TopProduct {
  product_name: string
  sku: string
  quantity: number
  revenue_minor: number
}

export interface RecentSale {
  id: string
  status: string
  total_minor: number
  currency: string
  payment_method: string
  cashier: string
  created_at: string
}

export interface TenantAnalytics {
  id: string
  name: string
  slug: string
  business_type: string
  plan: string
  max_users: number
  max_products: number
  users: number
  products: number
  date: string
  today_revenue_minor: number
  today_sales: number
  today_items: number
  revenue_trend: RevenuePoint[]
  top_products: TopProduct[]
  recent_sales: RecentSale[]
}

export interface PageMeta {
  request_id?: string
  page: number
  limit: number
  total: number
}

export interface Page<T> {
  data: T[]
  meta: PageMeta
}

export interface LoginResponse {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
  user: Record<string, unknown>
}