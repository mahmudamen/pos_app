import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { DashboardPage } from './DashboardPage'

const OK = (body: unknown) =>
  Promise.resolve(
    new Response(JSON.stringify({ data: body }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    }),
  )

const SUMMARY = {
  active_plans: 4,
  subscriptions: { active: 12, trial: 3, past_due: 2, grace_period: 1 },
  mrr_minor: 4_250_000,
  currency: 'EGP',
  outstanding_minor: 150_000,
  providers: ['paymob', 'fawry'],
  recent_invoices: [
    {
      id: 'i1',
      tenant_name: 'Nile Books',
      amount_minor: 99_000,
      currency: 'EGP',
      status: 'paid',
      created_at: '2026-10-01T00:00:00Z',
    },
  ],
}

describe('DashboardPage', () => {
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

  function renderPage() {
    return render(
      <MemoryRouter initialEntries={['/']}>
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/tenants" element={<div>tenants</div>} />
        </Routes>
      </MemoryRouter>,
    )
  }

  it('leads with revenue, outstanding balance and subscription health', async () => {
    fetchMock.mockResolvedValueOnce(OK(SUMMARY))
    renderPage()

    expect(await screen.findByText('Monthly recurring revenue')).toBeInTheDocument()
    expect(screen.getByText('E£42500.00')).toBeInTheDocument()
    expect(screen.getByText('E£1500.00')).toBeInTheDocument()
    // Past due + grace is the number an operator acts on, so it is called out.
    expect(screen.getByText('3 need attention')).toBeInTheDocument()
    expect(screen.getByText('Awaiting payment')).toBeInTheDocument()
  })

  it('reads subscription states as proportions of one whole, not as a list', async () => {
    fetchMock.mockResolvedValueOnce(OK(SUMMARY))
    renderPage()

    await screen.findByText('Subscriptions by state')
    const meters = screen.getAllByRole('meter')
    // active 12, trial 3, past_due 2, grace_period 1 — total 18.
    expect(meters).toHaveLength(4)
    expect(meters[0]).toHaveAttribute('aria-valuenow', '12')
    expect(meters[0]).toHaveAttribute('aria-valuemax', '18')
    expect(screen.getByText('18 subscriptions across 4 states.')).toBeInTheDocument()
  })

  it('links an empty state onward instead of leaving a dead end', async () => {
    fetchMock.mockResolvedValueOnce(
      OK({
        ...SUMMARY,
        subscriptions: {},
        recent_invoices: [],
        providers: [],
      }),
    )
    renderPage()

    expect(await screen.findByText('No subscriptions yet')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Open tenants' })).toBeInTheDocument()
    expect(screen.getByText('No invoices yet')).toBeInTheDocument()
  })

  it('surfaces a load failure instead of rendering an empty dashboard', async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(JSON.stringify({ error: { code: 'forbidden', message: 'forbidden' } }), {
        status: 403,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
    renderPage()

    expect(await screen.findByRole('alert')).toHaveTextContent('forbidden')
  })
})