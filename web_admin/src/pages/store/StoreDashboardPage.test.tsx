import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { StoreDashboardPage } from './StoreDashboardPage'
import type { DashboardSummary } from '../../types'

const { authState } = vi.hoisted(() => ({
  authState: { role: 'owner' },
}))

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({
    isAuthenticated: true,
    role: authState.role,
    user: null,
    signIn: vi.fn(),
    signOut: vi.fn(),
  }),
}))

const { apiMock } = vi.hoisted(() => {
  const m: Record<string, ReturnType<typeof vi.fn>> = {}
  m.dashboardSummary = vi.fn()
  return { apiMock: m }
})

vi.mock('../../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../../lib/api')>()
  return { ...mod, api: apiMock }
})

const summary: DashboardSummary = {
  date: '2026-09-22',
  vat_mode: 'inclusive',
  today: {
    revenue_minor: 284500,
    sales_count: 12,
    avg_sale_minor: 23708,
    items_sold: 31,
    tax_minor: 34000,
    cogs_minor: 120000,
    profit_minor: 164500,
  },
  top_products: [
    { product_name: 'Cola', sku: 'SKU-1', quantity: 8, revenue_minor: 10000 },
    { product_name: 'Bread', sku: 'SKU-2', quantity: 23, revenue_minor: 2300 },
  ],
  recent_sales: [
    {
      id: 'sale-1',
      status: 'completed',
      total_minor: 5000,
      currency: 'EGP',
      payment_method: 'cash',
      created_at: '2026-09-22T10:00:00Z',
    },
  ],
  per_cashier: [],
  payment_mix: [],
}

function renderPage() {
  return render(
    <MemoryRouter>
      <StoreDashboardPage />
    </MemoryRouter>,
  )
}

describe('StoreDashboardPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authState.role = 'owner'
    apiMock.dashboardSummary.mockResolvedValue(summary)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders stat cards from the dashboard summary', async () => {
    renderPage()

    expect(await screen.findByText("Today's revenue")).toBeInTheDocument()
    expect(screen.getByText('E£2845.00')).toBeInTheDocument()
    expect(screen.getByText('Sales today')).toBeInTheDocument()
    expect(screen.getByText('12')).toBeInTheDocument()
    expect(screen.getByText('Average sale')).toBeInTheDocument()
    expect(screen.getByText('E£237.08')).toBeInTheDocument()
    expect(screen.getByText('Items sold')).toBeInTheDocument()
    expect(screen.getByText('31')).toBeInTheDocument()
    expect(screen.getByText('Gross profit')).toBeInTheDocument()
    expect(screen.getByText('E£1645.00')).toBeInTheDocument()
  })

  it('renders recent sales and top products with a view-sales link', async () => {
    renderPage()

    expect(await screen.findByText('Recent sales')).toBeInTheDocument()
    expect(screen.getByText('cash')).toBeInTheDocument()
    expect(screen.getByText('Top products')).toBeInTheDocument()
    expect(screen.getByText('Cola')).toBeInTheDocument()
    expect(screen.getByText('Bread')).toBeInTheDocument()
    expect(screen.getByText('E£100.00')).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'View sales' }),
    ).toBeInTheDocument()
  })
})