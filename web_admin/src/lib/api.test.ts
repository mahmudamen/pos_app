import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ACCESS_TOKEN_KEY, ApiClient, ApiError, DEVICE_ID_KEY, deviceId } from './api'

class MemoryStorage implements Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> {
  private map = new Map<string, string>()
  getItem(k: string) {
    return this.map.get(k) ?? null
  }
  setItem(k: string, v: string) {
    this.map.set(k, v)
  }
  removeItem(k: string) {
    this.map.delete(k)
  }
  clear() {
    this.map.clear()
  }
}

const OK = (body: unknown, status = 200) =>
  Promise.resolve(
    new Response(JSON.stringify({ data: body }), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )

const ERR = (code: string, message: string, status: number) =>
  Promise.resolve(
    new Response(
      JSON.stringify({ error: { code, message, request_id: 'r' } }),
      { status, headers: { 'Content-Type': 'application/json' } },
    ),
  )

describe('ApiClient', () => {
  let fetchMock: ReturnType<typeof vi.fn>
  let storage: MemoryStorage
  let client: ApiClient

  beforeEach(() => {
    storage = new MemoryStorage()
    storage.clear()
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    client = new ApiClient('http://api.test', storage)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  it('posts to /v1/auth/login and persists the access token', async () => {
    fetchMock.mockResolvedValueOnce(
      OK({ access_token: 'tok', refresh_token: 'ref', token_type: 'Bearer', expires_in: 3600, user: {} }),
    )
    await client.login('ten-1', 'admin@posgo.saas', 'secret')

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('http://api.test/v1/auth/login')
    expect(JSON.parse(String(init.body))).toEqual({
      tenant_id: 'ten-1',
      device_id: expect.any(String),
      device_name: 'web-admin',
      email: 'admin@posgo.saas',
      password: 'secret',
    })
    expect(storage.getItem(ACCESS_TOKEN_KEY)).toBe('tok')
    expect(client.hasToken()).toBe(true)
  })

  it('unwraps the data envelope and strips /v1 from the path', async () => {
    fetchMock.mockResolvedValueOnce(
      OK({ active_plans: 3, mrr_minor: 1200, currency: 'EGP' }),
    )
    const summary = await client.billingSummary()
    expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/v1/saas/billing/summary')
    expect(summary.active_plans).toBe(3)
  })

  it('sends the bearer token once one exists', async () => {
    storage.setItem(ACCESS_TOKEN_KEY, 'tok')
    fetchMock.mockResolvedValueOnce(OK([]))
    await client.listPlans()
    const headers = fetchMock.mock.calls[0][1].headers as Record<string, string>
    expect(headers['Authorization']).toBe('Bearer tok')
  })

  it('maps error envelopes to ApiError with code + message', async () => {
    fetchMock.mockResolvedValueOnce(ERR('plan_downgrade_blocked', 'over limits', 409))
    const err = await client.changeSubscriptionPlan('sub-1', 'starter').catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect((err as ApiError).status).toBe(409)
    expect((err as ApiError).code).toBe('plan_downgrade_blocked')
    expect((err as ApiError).message).toBe('over limits')
  })

  it('throws a generic error for malformed responses', async () => {
    fetchMock.mockResolvedValueOnce(Promise.resolve(new Response('boom', { status: 500 })))
    const err = await client.listPlans().catch((e: unknown) => e)
    expect((err as ApiError).code).toBe('request_failed')
    expect((err as ApiError).status).toBe(500)
  })

  it('builds subscription query params from page/limit/status', async () => {
    fetchMock.mockResolvedValueOnce(OK({ data: [], meta: { page: 1, limit: 100, total: 0 } }))
    await client.listSubscriptions(2, 50, 'active')
    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toContain('page=2')
    expect(url).toContain('limit=50')
    expect(url).toContain('status=active')
  })

  it('payInvoice posts an empty object by default and a provider when given', async () => {
    fetchMock.mockResolvedValueOnce(OK({ id: 'inv', status: 'paid' }))
    await client.payInvoice('inv-1')
    let body = fetchMock.mock.calls[0][1].body as string
    expect(JSON.parse(body)).toEqual({})
    expect(fetchMock.mock.calls[0][0]).toBe('http://api.test/v1/saas/invoices/inv-1/pay')

    fetchMock.mockResolvedValueOnce(OK({ id: 'inv', status: 'paid' }))
    await client.payInvoice('inv-1', 'mock')
    body = fetchMock.mock.calls[1][1].body as string
    expect(JSON.parse(body)).toEqual({ provider: 'mock' })
  })

  it('returns undefined for 204 DELETE responses', async () => {
    fetchMock.mockResolvedValueOnce(Promise.resolve(new Response(null, { status: 204 })))
    const result = await client.deletePlan('plan-1')
    expect(result).toBeUndefined()
  })
})

describe('deviceId', () => {
  it('generates once and reuses', () => {
    const storage = new MemoryStorage()
    vi.stubGlobal('localStorage', storage)
    const a = deviceId()
    const b = deviceId()
    expect(a).toBe(b)
    expect(storage.getItem(DEVICE_ID_KEY)).toBe(a)
    vi.unstubAllGlobals()
  })
})