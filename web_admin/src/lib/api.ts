import type {
  AdjustmentReason,
  AuthUser,
  BillingSummary,
  Category,
  CategoryInput,
  DashboardSummary,
  InventoryAdjustment,
  InventoryAdjustmentInput,
  Invoice,
  InvoiceStatus,
  LoginResponse,
  Page,
  Plan,
  Product,
  ProductInput,
  ProductPatch,
  Provider,
  Purchase,
  PurchaseInput,
  RecentInvoice,
  SaleSummary,
  StoreSubscription,
  StoreUser,
  StoreUserInput,
  Subscription,
  SubscriptionStatus,
  Tenant,
  TenantAnalytics,
} from '../types'

export const ACCESS_TOKEN_KEY = 'pos_admin_token'
export const DEVICE_ID_KEY = 'pos_admin_device_id'
export const USER_KEY = 'pos_admin_user'

export class ApiError extends Error {
  status: number
  code: string
  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

const BASE_URL: string =
  (import.meta.env.VITE_API_BASE_URL as string | undefined) ?? ''

const SUBSCRIPTION_STATUSES: SubscriptionStatus[] = [
  'trial',
  'active',
  'grace_period',
  'past_due',
  'suspended',
  'cancelled',
]

const INVOICE_STATUSES: InvoiceStatus[] = ['open', 'paid', 'void', 'refunded']

export function deviceId(): string {
  let id = localStorage.getItem(DEVICE_ID_KEY)
  if (!id) {
    id = crypto.randomUUID()
    localStorage.setItem(DEVICE_ID_KEY, id)
  }
  return id
}

/**
 * Thin client over the backend's `{data, meta}` envelope. All methods talk to
 * /v1/* (same origin in production behind the nginx proxy, or VITE_API_BASE_URL).
 */
export class ApiClient {
  private baseUrl: string
  private storage: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

  constructor(
    baseUrl: string = BASE_URL,
    storage: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> = localStorage,
  ) {
    this.baseUrl = baseUrl
    this.storage = storage
  }

  private token(): string | null {
    return this.storage.getItem(ACCESS_TOKEN_KEY)
  }

  setToken(token: string | null): void {
    if (token) {
      this.storage.setItem(ACCESS_TOKEN_KEY, token)
    } else {
      this.storage.removeItem(ACCESS_TOKEN_KEY)
    }
  }

  hasToken(): boolean {
    return this.token() !== null
  }

  setUser(user: AuthUser): void {
    this.storage.setItem(USER_KEY, JSON.stringify(user))
  }

  readUser(): AuthUser | null {
    const raw = this.storage.getItem(USER_KEY)
    if (!raw) {
      return null
    }
    try {
      return JSON.parse(raw) as AuthUser
    } catch {
      return null
    }
  }

  clearUser(): void {
    this.storage.removeItem(USER_KEY)
  }

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
  ): Promise<T> {
    const token = this.token()
    const headers: Record<string, string> = {}
    if (body !== undefined) {
      headers['Content-Type'] = 'application/json'
    }
    if (token) {
      headers['Authorization'] = `Bearer ${token}`
    }
    const res = await fetch(`${this.baseUrl}/v1${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
    })
    const text = await res.text()
    let json: unknown = null
    if (text) {
      try {
        json = JSON.parse(text)
      } catch {
        json = null
      }
    }
    if (!res.ok) {
      const error = (json as { error?: { code?: string; message?: string } })
        ?.error
      throw new ApiError(
        res.status,
        error?.code ?? 'request_failed',
        error?.message ?? `HTTP ${res.status}`,
      )
    }
    return (json as { data?: T })?.data as T
  }

  // ---- auth -------------------------------------------------------------
  async login(
    tenantId: string,
    email: string,
    password: string,
    deviceName = 'web-admin',
  ): Promise<LoginResponse> {
    const data = await this.request<LoginResponse>('POST', '/auth/login', {
      tenant_id: tenantId,
      device_id: deviceId(),
      device_name: deviceName,
      email,
      password,
    })
    this.setToken(data.access_token)
    return data
  }

  // ---- saas platform ----------------------------------------------------
  listTenants(page = 1, limit = 50): Promise<Page<Tenant>> {
    return this.request<Page<Tenant>>(
      'GET',
      `/saas/tenants?page=${page}&limit=${limit}`,
    )
  }

  tenantAnalytics(id: string): Promise<TenantAnalytics> {
    return this.request<TenantAnalytics>('GET', `/saas/tenants/${id}/analytics`)
  }

  // ---- billing ----------------------------------------------------------
  billingSummary(): Promise<BillingSummary> {
    return this.request<BillingSummary>('GET', '/saas/billing/summary')
  }

  listPlans(): Promise<Plan[]> {
    return this.request<Plan[]>('GET', '/saas/plans')
  }

  createPlan(input: Partial<Plan>): Promise<{ id: string }> {
    return this.request<{ id: string }>('POST', '/saas/plans', input)
  }

  updatePlan(id: string, input: Partial<Plan>): Promise<{ updated: boolean }> {
    return this.request<{ updated: boolean }>('PATCH', `/saas/plans/${id}`, input)
  }

  deletePlan(id: string): Promise<void> {
    return this.request<void>('DELETE', `/saas/plans/${id}`)
  }

  listSubscriptions(
    page = 1,
    limit = 100,
    status = '',
  ): Promise<Page<Subscription>> {
    const qs = new URLSearchParams({ page: String(page), limit: String(limit) })
    if (status) {
      qs.set('status', status)
    }
    return this.request<Page<Subscription>>(
      'GET',
      `/saas/subscriptions?${qs.toString()}`,
    )
  }

  tenantSubscription(id: string): Promise<Subscription> {
    return this.request<Subscription>(
      'GET',
      `/saas/tenants/${id}/subscription`,
    )
  }

  assignSubscription(
    tenantId: string,
    planCode: string,
    status = 'trial',
    trialDays = 15,
  ): Promise<Record<string, unknown>> {
    return this.request<Record<string, unknown>>(
      'POST',
      `/saas/tenants/${tenantId}/subscription`,
      { plan_code: planCode, status, trial_days: trialDays },
    )
  }

  setSubscriptionStatus(
    id: string,
    status: SubscriptionStatus,
  ): Promise<{ id: string; status: string }> {
    return this.request<{ id: string; status: string }>(
      'POST',
      `/saas/subscriptions/${id}/status`,
      { status },
    )
  }

  changeSubscriptionPlan(
    id: string,
    planCode: string,
  ): Promise<{ id: string; plan_code: string }> {
    return this.request<{ id: string; plan_code: string }>(
      'POST',
      `/saas/subscriptions/${id}/change-plan`,
      { plan_code: planCode },
    )
  }

  listInvoices(page = 1, limit = 100, status = ''): Promise<Page<Invoice>> {
    const qs = new URLSearchParams({ page: String(page), limit: String(limit) })
    if (status) {
      qs.set('status', status)
    }
    return this.request<Page<Invoice>>('GET', `/saas/invoices?${qs.toString()}`)
  }

  createInvoice(
    tenantId: string,
    input: {
      amount_minor: number
      currency?: string
      description: string
      due_at?: string
      provider?: string
      subscription_id?: string
    },
  ): Promise<Record<string, unknown>> {
    return this.request<Record<string, unknown>>(
      'POST',
      `/saas/tenants/${tenantId}/invoices`,
      input,
    )
  }

  payInvoice(id: string, provider?: string): Promise<Record<string, unknown>> {
    return this.request<Record<string, unknown>>(
      'POST',
      `/saas/invoices/${id}/pay`,
      provider ? { provider } : {},
    )
  }

  voidInvoice(id: string): Promise<Record<string, unknown>> {
    return this.request<Record<string, unknown>>(
      'POST',
      `/saas/invoices/${id}/void`,
      {},
    )
  }

  refundInvoice(id: string): Promise<Record<string, unknown>> {
    return this.request<Record<string, unknown>>(
      'POST',
      `/saas/invoices/${id}/refund`,
      {},
    )
  }

  paymentProviders(): Promise<Provider[]> {
    return this.request<Provider[]>('GET', '/saas/payment-providers')
  }

  // ---- store console ----------------------------------------------------
  listProducts(page = 1, limit = 50, q = ''): Promise<Page<Product>> {
    const qs = new URLSearchParams({ page: String(page), limit: String(limit) })
    if (q) {
      qs.set('q', q)
    }
    return this.request<Page<Product>>('GET', `/products?${qs.toString()}`)
  }

  createProduct(input: ProductInput): Promise<{ id: string }> {
    return this.request<{ id: string }>('POST', '/products', input)
  }

  updateProduct(id: string, patch: ProductPatch): Promise<{ updated: boolean }> {
    return this.request<{ updated: boolean }>('PATCH', `/products/${id}`, patch)
  }

  listCategories(): Promise<Category[]> {
    return this.request<Category[]>('GET', '/categories')
  }

  createCategory(input: CategoryInput): Promise<{ id: string }> {
    return this.request<{ id: string }>('POST', '/categories', input)
  }

  updateCategory(
    id: string,
    patch: Partial<CategoryInput> & { is_active?: boolean },
  ): Promise<{ updated: boolean }> {
    return this.request<{ updated: boolean }>('PATCH', `/categories/${id}`, patch)
  }

  deleteCategory(id: string): Promise<void> {
    return this.request<void>('DELETE', `/categories/${id}`)
  }

  listUsers(page = 1, limit = 50): Promise<Page<StoreUser>> {
    return this.request<Page<StoreUser>>(
      'GET',
      `/users?page=${page}&limit=${limit}`,
    )
  }

  createUser(input: StoreUserInput): Promise<{ id: string }> {
    return this.request<{ id: string }>('POST', '/users', input)
  }

  listSales(page = 1, limit = 50): Promise<Page<SaleSummary>> {
    return this.request<Page<SaleSummary>>(
      'GET',
      `/sales?page=${page}&limit=${limit}`,
    )
  }

  listPurchases(): Promise<Purchase[]> {
    return this.request<Purchase[]>('GET', '/purchases')
  }

  createPurchase(input: PurchaseInput): Promise<{ id: string }> {
    return this.request<{ id: string }>('POST', '/purchases', input)
  }

  listInventoryAdjustments(
    page = 1,
    limit = 50,
  ): Promise<Page<InventoryAdjustment>> {
    return this.request<Page<InventoryAdjustment>>(
      'GET',
      `/inventory/adjustments?page=${page}&limit=${limit}`,
    )
  }

  createInventoryAdjustment(
    input: InventoryAdjustmentInput,
  ): Promise<{ id: string }> {
    return this.request<{ id: string }>(
      'POST',
      '/inventory/adjustments',
      input,
    )
  }

  dashboardSummary(): Promise<DashboardSummary> {
    return this.request<DashboardSummary>('GET', '/dashboard/summary')
  }

  subscription(): Promise<StoreSubscription> {
    return this.request<StoreSubscription>('GET', '/subscription')
  }

  changePlan(planCode: string): Promise<{ id: string; plan_code: string }> {
    return this.request<{ id: string; plan_code: string }>(
      'POST',
      '/subscription/change-plan',
      { plan_code: planCode },
    )
  }
}

export const REASONS: AdjustmentReason[] = ['damaged', 'restock', 'count']

export const api = new ApiClient()

export { SUBSCRIPTION_STATUSES, INVOICE_STATUSES }
export type { RecentInvoice }