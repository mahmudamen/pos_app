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

export type UserRole =
  | 'owner'
  | 'manager'
  | 'cashier'
  | 'saas_admin'
  | 'superadmin'

export interface AuthUser {
  id: string
  tenant_id: string
  email?: string
  display_name: string
  role: string
  account_type?: string
  permissions: Record<string, unknown>
}

// ---- store console ------------------------------------------------------

export interface Product {
  id: string
  name: string
  name_ar?: string
  description?: string
  description_ar?: string
  sku: string
  barcode?: string
  price_minor: number
  currency: string
  stock_quantity: number
  category_id?: string
  category_name?: string
  is_active: boolean
}

export interface ProductInput {
  name: string
  name_ar?: string
  description?: string
  description_ar?: string
  sku?: string
  barcode?: string
  price_minor: number
  currency?: string
  category_id?: string
  is_active?: boolean
}

export type ProductPatch = Partial<ProductInput>

export interface Category {
  id: string
  name: string
  name_ar?: string
  slug: string
  is_active: boolean
}

export interface CategoryInput {
  name: string
  name_ar?: string
}

export interface StoreUser {
  id: string
  email: string
  display_name: string
  role: string
  is_active: boolean
  permissions?: Record<string, unknown>
}

export interface StoreUserInput {
  email: string
  display_name: string
  password: string
  role: UserRole
}

export interface SaleSummary {
  id: string
  status: string
  subtotal_minor: number
  discount_minor: number
  tax_minor: number
  total_minor: number
  currency: string
  payment_method: string
  created_at: string
}

export interface PurchaseItemInput {
  product_id?: string
  product_name?: string
  quantity: number
  unit?: string
  unit_price_minor: number
}

export interface PurchaseInput {
  supplier: string
  invoice_no?: string
  currency?: string
  items: PurchaseItemInput[]
  tax_minor?: number
}

export interface Purchase {
  id: string
  supplier: string
  invoice_no?: string
  currency: string
  subtotal_minor: number
  tax_minor: number
  total_minor: number
  item_count: number
  created_by: string
  created_at: string
}

export type AdjustmentReason = 'damaged' | 'restock' | 'count'

export interface InventoryAdjustment {
  id: string
  product_id: string
  product_name?: string
  sku?: string
  reason: AdjustmentReason
  quantity_delta: number
  note?: string
  created_by: string
  created_at: string
}

export interface InventoryAdjustmentInput {
  product_id: string
  reason: AdjustmentReason
  quantity_delta: number
  note?: string
}

export interface DashboardToday {
  revenue_minor: number
  sales_count: number
  avg_sale_minor: number
  items_sold: number
  tax_minor: number
  cogs_minor: number
  profit_minor: number
}

export interface DashboardTopProduct {
  product_name: string
  sku?: string
  quantity: number
  revenue_minor: number
}

export interface DashboardRecentSale {
  id: string
  status: string
  total_minor: number
  currency: string
  payment_method: string
  created_at: string
}

export interface PerCashier {
  cashier_id?: string
  cashier: string
  sales_count: number
  revenue_minor: number
}

export interface PaymentMix {
  method: string
  amount_minor: number
}

export interface DashboardSummary {
  date: string
  vat_mode?: string
  today: DashboardToday
  top_products: DashboardTopProduct[]
  recent_sales: DashboardRecentSale[]
  per_cashier: PerCashier[]
  payment_mix: PaymentMix[]
}

export interface StoreSubscription {
  id: string
  status: string
  plan?: {
    code: string
    name: string
    billing_period?: string
    price_minor?: number
    currency?: string
  }
  trial_ends_at?: string
  current_period_end?: string
  cancel_at_period_end?: boolean
}