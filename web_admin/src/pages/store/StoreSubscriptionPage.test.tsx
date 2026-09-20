import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { StoreSubscriptionPage } from './StoreSubscriptionPage'
import type { Plan, StoreSubscription } from '../../types'

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
  const methods = ['subscription', 'changePlan', 'listPlans']
  const m: Record<string, ReturnType<typeof vi.fn>> = {}
  for (const k of methods) {
    m[k] = vi.fn()
  }
  return { apiMock: m }
})

vi.mock('../../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../../lib/api')>()
  return { ...mod, api: apiMock }
})

const current: StoreSubscription = {
  id: 'sub-1',
  status: 'active',
  plan: { code: 'starter', name: 'Starter' },
  trial_ends_at: '2026-10-01T00:00:00Z',
  current_period_end: '2026-11-01T00:00:00Z',
  cancel_at_period_end: false,
}

const plans: Plan[] = [
  {
    id: 'pl-1',
    code: 'starter',
    name: 'Starter',
    description: '',
    price_minor: 1000,
    currency: 'EGP',
    billing_period: 'monthly',
    features: [],
    max_users: 2,
    max_products: 100,
    is_active: true,
  },
  {
    id: 'pl-2',
    code: 'pro',
    name: 'Pro',
    description: '',
    price_minor: 5000,
    currency: 'EGP',
    billing_period: 'monthly',
    features: [],
    max_users: 10,
    max_products: 1000,
    is_active: true,
  },
]

function renderPage() {
  return render(
    <MemoryRouter>
      <StoreSubscriptionPage />
    </MemoryRouter>,
  )
}

describe('StoreSubscriptionPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authState.role = 'owner'
    apiMock.subscription.mockResolvedValue(current)
    apiMock.listPlans.mockResolvedValue(plans)
    apiMock.changePlan.mockResolvedValue({ id: 'sub-1', plan_code: 'pro' })
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders the current plan and the selectable plans', async () => {
    renderPage()
    expect((await screen.findAllByText('Starter')).length).toBeGreaterThan(0)
    expect(screen.getByText('Pro')).toBeInTheDocument()
    expect(screen.getByText('E£10.00')).toBeInTheDocument()
    expect(screen.getByText('E£50.00')).toBeInTheDocument()
  })

  it('submits a plan change via POST /v1/subscription/change-plan', async () => {
    renderPage()
    await screen.findByText('Pro')

    fireEvent.click(screen.getByRole('radio', { name: /Pro/ }))
    fireEvent.click(screen.getByRole('button', { name: 'Change plan' }))

    await waitFor(() => expect(apiMock.changePlan).toHaveBeenCalledTimes(1))
    expect(apiMock.changePlan).toHaveBeenCalledWith('pro')
  })
})