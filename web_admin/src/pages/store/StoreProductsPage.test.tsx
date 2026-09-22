import { describe, expect, it, beforeEach, afterEach, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { StoreProductsPage } from './StoreProductsPage'
import type { Category, Product } from '../../types'

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
  const methods = [
    'listProducts',
    'createProduct',
    'updateProduct',
    'listCategories',
  ]
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

const cat: Category = {
  id: 'c1',
  name: 'Drinks',
  slug: 'drinks',
  is_active: true,
}

const cola: Product = {
  id: 'p1',
  name: 'Cola',
  name_ar: 'كولا',
  sku: 'SKU-1',
  barcode: '1001',
  price_minor: 1250,
  currency: 'EGP',
  stock_quantity: 10,
  category_id: 'c1',
  is_active: true,
}

const water: Product = {
  id: 'p2',
  name: 'Water',
  name_ar: 'مياه',
  sku: 'SKU-2',
  price_minor: 500,
  currency: 'EGP',
  stock_quantity: 3,
  is_active: true,
}

function renderPage() {
  return render(
    <MemoryRouter>
      <StoreProductsPage />
    </MemoryRouter>,
  )
}

describe('StoreProductsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    authState.role = 'owner'
    apiMock.listCategories.mockResolvedValue([cat])
    apiMock.listProducts.mockImplementation(
      (_page: number, _limit: number, q = '') => {
        const needle = q.toLowerCase()
        const data = [cola, water].filter(
          (p) =>
            !needle ||
            p.name.toLowerCase().includes(needle) ||
            (p.name_ar ?? '').includes(needle),
        )
        return Promise.resolve({
          data,
          meta: { page: _page, limit: _limit ?? 50, total: data.length },
        })
      },
    )
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('renders the product list with prices and Arabic names', async () => {
    renderPage()
    expect(await screen.findByText('كولا')).toBeInTheDocument()
    expect(screen.getByText('مياه')).toBeInTheDocument()
    expect(screen.getByText('Cola')).toBeInTheDocument()
    expect(screen.getByText('E£12.50')).toBeInTheDocument()
    expect(screen.getByText('E£5.00')).toBeInTheDocument()
    expect(apiMock.listProducts).toHaveBeenCalledWith(1, 50, '')
  })

  it('re-queries the backend when the search box changes', async () => {
    renderPage()
    await screen.findByText('Cola')

    fireEvent.change(screen.getByPlaceholderText('Search products'), {
      target: { value: 'Cola' },
    })

    await waitFor(() =>
      expect(screen.queryByText('Water')).not.toBeInTheDocument(),
    )
    expect(apiMock.listProducts).toHaveBeenLastCalledWith(1, 50, 'Cola')
    expect(screen.getByText('Cola')).toBeInTheDocument()
  })

  it('creates a product with the Arabic fields through the form', async () => {
    apiMock.createProduct.mockResolvedValue({ id: 'p3' })
    renderPage()
    await screen.findByText('Cola')

    fireEvent.click(screen.getByRole('button', { name: 'New product' }))

    expect(screen.getByLabelText('Arabic name')).toBeInTheDocument()
    expect(screen.getByLabelText('Arabic description')).toBeInTheDocument()

    fireEvent.change(screen.getByLabelText('Name'), {
      target: { value: 'Tea' },
    })
    fireEvent.change(screen.getByLabelText('Arabic name'), {
      target: { value: 'شاي' },
    })
    fireEvent.change(screen.getByLabelText('Price (EGP)'), {
      target: { value: '8.00' },
    })
    fireEvent.change(screen.getByLabelText('SKU'), {
      target: { value: 'SKU-9' },
    })

    fireEvent.click(screen.getByRole('button', { name: 'Create product' }))

    await waitFor(() => expect(apiMock.createProduct).toHaveBeenCalledTimes(1))
    expect(apiMock.createProduct).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Tea',
        name_ar: 'شاي',
        sku: 'SKU-9',
        price_minor: 800,
        currency: 'EGP',
      }),
    )
  })

  it('hides product creation for a cashier', async () => {
    authState.role = 'cashier'
    renderPage()
    await screen.findByText('Cola')

    expect(
      screen.queryByRole('button', { name: 'New product' }),
    ).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Edit' })).not.toBeInTheDocument()
  })
})