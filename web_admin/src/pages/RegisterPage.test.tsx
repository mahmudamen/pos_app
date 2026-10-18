import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../auth/AuthContext'
import { RegisterPage } from './RegisterPage'

function renderPage(initialEntry = '/register') {
  return render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <AuthProvider>
        <RegisterPage />
      </AuthProvider>
    </MemoryRouter>,
  )
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
    new Response(JSON.stringify({ error: { code, message } }), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  )

const SESSION = {
  access_token: 'reg-token',
  refresh_token: 'reg-refresh',
  expires_in: 900,
  user: {
    id: 'u-reg',
    display_name: 'Nadia',
    role: 'owner',
    account_type: 'standard',
  },
  tenant: { id: 'ten-reg', slug: 'nile-grocers' },
}

function fillForm() {
  fireEvent.change(screen.getByLabelText('Store name'), {
    target: { value: 'Nile Grocers' },
  })
  fireEvent.change(screen.getByLabelText('Business type'), {
    target: { value: 'grocery' },
  })
  fireEvent.change(screen.getByLabelText('Your name'), {
    target: { value: 'Nadia' },
  })
  fireEvent.change(screen.getByLabelText('Email'), {
    target: { value: 'nadia@nile.example' },
  })
  fireEvent.change(screen.getByLabelText('Password'), {
    target: { value: 'supersecret1' },
  })
}

describe('RegisterPage', () => {
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

  it('renders the signup form with the 17 business types', () => {
    fetchMock.mockResolvedValue(OK({}))
    renderPage()

    expect(
      screen.getByRole('heading', { name: 'Create your store' }),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Store name')).toBeInTheDocument()
    expect(screen.getByLabelText('Your name')).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
    expect(screen.getByLabelText('Password')).toBeInTheDocument()
    expect(screen.queryByLabelText(/^Phone/)).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Create my store' }),
    ).toBeInTheDocument()

    const options = screen.getAllByRole('option')
    // placeholder + 24 verticals
    expect(options).toHaveLength(25)
    expect(
      options.map((o) => o.textContent).filter(Boolean),
    ).toEqual(
      expect.arrayContaining([
        'Coffee shop',
        'Restaurant',
        'Pharmacy',
        'Hardware store',
        'Fruit & vegetable shop',
        'Beauty & cosmetics',
        'Supermarket',
        'Auto parts shop',
        'Optician',
        'Pet shop',
        'Flower shop',
        'Laundry & dry cleaning',
      ]),
    )
    expect(fetchMock).not.toHaveBeenCalled()
    expect(
      screen.getByRole('link', { name: 'Sign in' }),
    ).toBeInTheDocument()
  })

  it('submits the register payload and stores the session', async () => {
    fetchMock.mockResolvedValueOnce(OK(SESSION, 201))
    renderPage()
    fillForm()
    fireEvent.click(screen.getByRole('button', { name: 'Create my store' }))

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(1)
    })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('/v1/auth/register')
    const body = JSON.parse(init.body as string)
    expect(body).toMatchObject({
      store_name: 'Nile Grocers',
      business_type: 'grocery',
      display_name: 'Nadia',
      email: 'nadia@nile.example',
      password: 'supersecret1',
      device_name: 'web-admin',
    })
    expect(typeof body.device_id).toBe('string')
    expect(body.device_id.length).toBeGreaterThan(0)
    expect(body.phone).toBeUndefined()

    await waitFor(() => {
      expect(localStorage.getItem('pos_admin_token')).toBe('reg-token')
    })
    const user = JSON.parse(localStorage.getItem('pos_admin_user') ?? '{}')
    expect(user.id).toBe('u-reg')
    expect(user.role).toBe('owner')
    expect(user.tenant_id).toBe('ten-reg')
  })

  it('surfaces a duplicate email as a friendly error', async () => {
    fetchMock.mockResolvedValueOnce(
      ERR(
        'email_taken',
        'an account with this email already exists',
        409,
      ),
    )
    renderPage()
    fillForm()
    fireEvent.click(screen.getByRole('button', { name: 'Create my store' }))

    const banner = await screen.findByRole('alert')
    expect(banner).toHaveTextContent(/already exists/i)
    expect(banner).toHaveTextContent(/sign in instead/i)
    expect(localStorage.getItem('pos_admin_token')).toBeNull()
    expect(
      screen.getByRole('button', { name: 'Create my store' }),
    ).toBeEnabled()
  })

  it('reports backend validation errors inline', async () => {
    fetchMock.mockResolvedValueOnce(
      ERR('validation_error', 'business type is not supported', 400),
    )
    renderPage()
    fillForm()
    fireEvent.click(screen.getByRole('button', { name: 'Create my store' }))

    const banner = await screen.findByRole('alert')
    expect(banner).toHaveTextContent('business type is not supported')
    expect(
      screen.getByRole('button', { name: 'Create my store' }),
    ).toBeEnabled()
  })
})
