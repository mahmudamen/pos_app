import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import App from './App'

describe('App', () => {
  it('shows the login page when unauthenticated', () => {
    vi.restoreAllMocks()
    localStorage.clear()
    window.history.pushState({}, '', '/admin/')
    render(<App />)
    expect(screen.getByRole('heading', { name: 'Sign in' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Sign in' })).toBeInTheDocument()
    expect(screen.getByLabelText('Tenant ID')).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
    expect(screen.getByLabelText('Password')).toBeInTheDocument()
    expect(
      screen.getByRole('link', { name: 'Create your store' }),
    ).toBeInTheDocument()
  })

  it('shows the register page at /admin/register when unauthenticated', () => {
    vi.restoreAllMocks()
    localStorage.clear()
    window.history.pushState({}, '', '/admin/register')
    render(<App />)
    expect(
      screen.getByRole('heading', { name: 'Create your store' }),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('Store name')).toBeInTheDocument()
    expect(screen.getByLabelText('Business type')).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Create my store' }),
    ).toBeInTheDocument()
    expect(
      screen.queryByLabelText('Tenant ID'),
    ).not.toBeInTheDocument()
    window.history.pushState({}, '', '/admin/')
  })
})