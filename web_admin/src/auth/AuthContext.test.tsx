import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { AuthProvider, useAuth } from './AuthContext'
import App from '../App'

const { apiMock } = vi.hoisted(() => {
  const methods = [
    'login',
    'logout',
    'hasToken',
    'setToken',
    'setUser',
    'readUser',
    'clearUser',
    'dashboardSummary',
    'billingSummary',
  ]
  const m: Record<string, ReturnType<typeof vi.fn>> = {}
  for (const k of methods) {
    m[k] = vi.fn()
  }
  m.hasToken = vi.fn(() => localStorage.getItem('pos_admin_token') !== null)
  m.readUser = vi.fn(() => {
    const raw = localStorage.getItem('pos_admin_user')
    if (!raw) {
      return null
    }
    try {
      return JSON.parse(raw)
    } catch {
      return null
    }
  })
  m.setToken = vi.fn((token: string | null) => {
    if (token) {
      localStorage.setItem('pos_admin_token', token)
    } else {
      localStorage.removeItem('pos_admin_token')
    }
  })
  m.setUser = vi.fn((u: unknown) => {
    localStorage.setItem('pos_admin_user', JSON.stringify(u))
  })
  m.clearUser = vi.fn(() => localStorage.removeItem('pos_admin_user'))
  return { apiMock: m }
})

vi.mock('../lib/api', async (importOriginal) => {
  const mod = await importOriginal<typeof import('../lib/api')>()
  return { ...mod, api: apiMock }
})

function userData(role: string) {
  return {
    id: 'u1',
    tenant_id: 'ten-1',
    display_name: 'Ali',
    role,
    account_type: 'standard',
    permissions: {},
  }
}

function Harness() {
  const { role, user, signIn, signOut } = useAuth()
  return (
    <div>
      <span data-testid="role">{role ?? 'none'}</span>
      <span data-testid="name">{user?.display_name ?? ''}</span>
      <span data-testid="tenant">{user?.tenant_id ?? ''}</span>
      <button onClick={() => void signIn('ten-1', 'ali@demo.com', 'admin')}>
        sign in
      </button>
      <button onClick={signOut}>sign out</button>
    </div>
  )
}

describe('AuthContext', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    localStorage.clear()
  })

  it('persists role + user after signIn and clears them on signOut', async () => {
    apiMock.login.mockResolvedValueOnce({
      access_token: 'tok',
      refresh_token: 'ref',
      token_type: 'Bearer',
      expires_in: 3600,
      user: userData('owner'),
    })
    render(
      <AuthProvider>
        <Harness />
      </AuthProvider>,
    )
    expect(screen.getByTestId('role')).toHaveTextContent('none')

    fireEvent.click(screen.getByRole('button', { name: 'sign in' }))
    await waitFor(() =>
      expect(screen.getByTestId('role')).toHaveTextContent('owner'),
    )
    expect(screen.getByTestId('name')).toHaveTextContent('Ali')
    expect(screen.getByTestId('tenant')).toHaveTextContent('ten-1')
    expect(apiMock.login).toHaveBeenCalledWith('ten-1', 'ali@demo.com', 'admin')
    const stored = JSON.parse(localStorage.getItem('pos_admin_user') ?? '{}')
    expect(stored.role).toBe('owner')

    fireEvent.click(screen.getByRole('button', { name: 'sign out' }))
    await waitFor(() =>
      expect(screen.getByTestId('role')).toHaveTextContent('none'),
    )
    expect(localStorage.getItem('pos_admin_user')).toBeNull()
    expect(localStorage.getItem('pos_admin_token')).toBeNull()
  })

  it('restores role from persisted user on boot', () => {
    localStorage.setItem('pos_admin_token', 'tok')
    localStorage.setItem(
      'pos_admin_user',
      JSON.stringify({ ...userData('cashier'), tenant_id: 'ten-2' }),
    )
    render(
      <AuthProvider>
        <Harness />
      </AuthProvider>,
    )
    expect(screen.getByTestId('role')).toHaveTextContent('cashier')
    expect(screen.getByTestId('tenant')).toHaveTextContent('ten-2')
  })
})

describe('role-aware console routing', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
  })

  afterEach(() => {
    localStorage.clear()
  })

  it('shows the store console for a manager login', () => {
    localStorage.setItem('pos_admin_token', 'tok')
    localStorage.setItem(
      'pos_admin_user',
      JSON.stringify(userData('manager')),
    )
    render(<App />)

    expect(screen.getByText('Store Console')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Products' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Categories' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Employees' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Sales' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Purchases' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Inventory' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Subscription' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Tenants' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Plans' })).not.toBeInTheDocument()
  })

  it('hides employee/purchase/subscription nav for a cashier login', () => {
    localStorage.setItem('pos_admin_token', 'tok')
    localStorage.setItem(
      'pos_admin_user',
      JSON.stringify(userData('cashier')),
    )
    render(<App />)

    expect(screen.getByText('Store Console')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Products' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Sales' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Inventory' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Employees' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Purchases' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Subscription' })).not.toBeInTheDocument()
  })

  it('keeps the saas admin console for a saas_admin login', () => {
    localStorage.setItem('pos_admin_token', 'tok')
    localStorage.setItem(
      'pos_admin_user',
      JSON.stringify(userData('saas_admin')),
    )
    render(<App />)

    expect(screen.getByText('SaaS Admin')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Tenants' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Plans' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Subscriptions' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Invoices' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Products' })).not.toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Employees' })).not.toBeInTheDocument()
  })
})