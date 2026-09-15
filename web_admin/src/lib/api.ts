import type {
  BillingSummary,
  Invoice,
  InvoiceStatus,
  LoginResponse,
  Page,
  Plan,
  Provider,
  RecentInvoice,
  Subscription,
  SubscriptionStatus,
  Tenant,
  TenantAnalytics,
} from '../types'

export const ACCESS_TOKEN_KEY = 'pos_admin_token'
export const DEVICE_ID_KEY = 'pos_admin_device_id'

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
}

export const api = new ApiClient()

export { SUBSCRIPTION_STATUSES, INVOICE_STATUSES }
export type { RecentInvoice }