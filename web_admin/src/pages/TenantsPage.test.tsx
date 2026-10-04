import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { TenantsPage } from './TenantsPage'

/** The real paginated envelope: rows in `data`, the counter in a sibling `meta`. */
const PAGE = (rows: unknown[], meta: Record<string, unknown> = {}) =>
  Promise.resolve(
    new Response(
      JSON.stringify({
        data: rows,
        meta: { page: 1, limit: 50, total: rows.length, request_id: 'r', ...meta },
      }),
      { status: 200, headers: { 'Content-Type': 'application/json' } },
    ),
  )

const TENANT = {
  id: 't1',
  name: 'Nile Books',
  slug: 'nile-books',
  business_type: 'book_store',
  country_code: 'EG',
  currency_code: 'EGP',
  default_language: 'ar',
  plan: 'starter',
  max_users: 5,
  max_products: 100,
  users: 3,
  products: 40,
  status: 'active',
  subscription_status: 'active',
  revenue_minor: 125_000,
  total_sales: 87,
}

describe('TenantsPage', () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    localStorage.clear()
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  function renderAt(url: string) {
    return render(
      <MemoryRouter initialEntries={[url]}>
        <Routes>
          <Route path="/tenants" element={<TenantsPage />} />
          <Route path="/tenants/:id" element={<div>tenant detail</div>} />
        </Routes>
      </MemoryRouter>,
    )
  }

  /** The URL the last tenants request went to. */
  function lastTenantsUrl(): URL {
    const call = fetchMock.mock.calls.at(-1)
    return new URL(String(call?.[0]), 'https://api.test')
  }

  it('filters on the server so the search spans every tenant, not one page', async () => {
    fetchMock.mockResolvedValueOnce(PAGE([TENANT]))
    renderAt('/tenants?q=nile&status=active&business_type=book_store&plan=starter')

    await screen.findByText('Nile Books')
    const url = lastTenantsUrl()
    expect(url.searchParams.get('q')).toBe('nile')
    expect(url.searchParams.get('status')).toBe('active')
    expect(url.searchParams.get('business_type')).toBe('book_store')
    expect(url.searchParams.get('plan')).toBe('starter')
  })

  it('returns to page one when a filter changes, and sends it to the API', async () => {
    fetchMock
      .mockResolvedValueOnce(PAGE([TENANT], { page: 3, total: 200 }))
      .mockResolvedValueOnce(PAGE([TENANT]))
    renderAt('/tenants?page=3')
    await screen.findByText('Nile Books')
    expect(lastTenantsUrl().searchParams.get('page')).toBe('3')

    fireEvent.change(screen.getByLabelText('Status'), { target: { value: 'suspended' } })

    await waitFor(() => {
      expect(lastTenantsUrl().searchParams.get('status')).toBe('suspended')
    })
    // A filter narrows the result set, so paging restarts.
    expect(lastTenantsUrl().searchParams.get('page')).toBe('1')
  })

  it('shows lifecycle, sales and revenue instead of only usage counts', async () => {
    fetchMock.mockResolvedValueOnce(PAGE([TENANT]))
    renderAt('/tenants')

    expect(await screen.findByText('Nile Books')).toBeInTheDocument()
    expect(screen.getByText('active')).toBeInTheDocument()
    expect(screen.getByText('87')).toBeInTheDocument()
    expect(screen.getByText('E£1250.00')).toBeInTheDocument()
    // Usage reads against the plan ceiling, and exposes it to assistive tech.
    const users = screen.getByRole('meter', { name: 'users' })
    expect(users).toHaveAttribute('aria-valuenow', '3')
    expect(users).toHaveAttribute('aria-valuemax', '5')
  })

  it('explains an unlimited plan cap instead of rendering "40 / 0"', async () => {
    fetchMock.mockResolvedValueOnce(PAGE([{ ...TENANT, max_products: 0 }]))
    renderAt('/tenants')

    await screen.findByText('Nile Books')
    expect(screen.getByText('40 · unlimited')).toBeInTheDocument()
    expect(screen.queryByRole('meter', { name: 'products' })).not.toBeInTheDocument()
  })

  it('offers a way out of an empty filtered result', async () => {
    fetchMock.mockResolvedValue(PAGE([], { total: 0 }))
    renderAt('/tenants?q=nope')

    expect(await screen.findByText('No tenants match these filters')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Clear filters' })).toBeInTheDocument()
  })

  it('names the missing thing when there are no tenants at all', async () => {
    fetchMock.mockResolvedValue(PAGE([], { total: 0 }))
    renderAt('/tenants')

    expect(await screen.findByText('No tenants yet')).toBeInTheDocument()
  })
})