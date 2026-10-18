import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { ApiError } from '../lib/api'
import { ErrorBanner } from '../components/ui'
import type { RegisterInput } from '../types'

// The backend's validBusinessTypes set (internal/transport/auth/register.go) —
// the same 17 verticals the Android onboarding picker offers.
const BUSINESS_TYPES: ReadonlyArray<[string, string]> = [
  ['coffee_shop', 'Coffee shop'],
  ['restaurant', 'Restaurant'],
  ['retail', 'Retail store'],
  ['book_store', 'Book store'],
  ['mobile_shop', 'Mobile phone shop'],
  ['computer_shop', 'Computer shop'],
  ['grocery', 'Grocery'],
  ['bakery', 'Bakery'],
  ['shawerma', 'Shawerma shop'],
  ['falafel', 'Falafel shop'],
  ['pharmacy', 'Pharmacy'],
  ['butcher', 'Butcher'],
  ['fruits_veg', 'Fruit & vegetable shop'],
  ['clothing', 'Clothing store'],
  ['sweets', 'Sweets shop'],
  ['jewelry', 'Jewelry store'],
  ['hardware', 'Hardware store'],
  ['beauty', 'Beauty & cosmetics'],
  ['supermarket', 'Supermarket'],
  ['auto_parts', 'Auto parts shop'],
  ['optics', 'Optician'],
  ['pets', 'Pet shop'],
  ['flowers', 'Flower shop'],
  ['laundry', 'Laundry & dry cleaning'],
]

export function RegisterPage() {
  const { register } = useAuth()
  const [storeName, setStoreName] = useState('')
  const [businessType, setBusinessType] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [phone, setPhone] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    const input: RegisterInput = {
      store_name: storeName.trim(),
      business_type: businessType,
      email: email.trim(),
      password,
      display_name: displayName.trim(),
    }
    if (phone.trim()) {
      input.phone = phone.trim()
    }
    try {
      await register(input)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(
          err.code === 'email_taken'
            ? 'An account with this email already exists — sign in instead.'
            : err.message || err.code,
        )
      } else {
        setError('Unable to reach the API. Is the backend running?')
      }
      setBusy(false)
    }
  }

  return (
    <div className="login-page">
      <form className="login-card card" onSubmit={onSubmit}>
        <div className="login-brand">
          <span className="brand-mark">P</span>
          <h1>Create your store</h1>
          <p>15-day free trial — no card required.</p>
        </div>
        <ErrorBanner error={error} />
        <label>
          Store name
          <input
            type="text"
            required
            value={storeName}
            autoComplete="organization"
            onChange={(e) => setStoreName(e.target.value)}
          />
        </label>
        <label>
          Business type
          <select
            required
            value={businessType}
            onChange={(e) => setBusinessType(e.target.value)}
          >
            <option value="" disabled>
              Choose your business
            </option>
            {BUSINESS_TYPES.map(([wire, label]) => (
              <option key={wire} value={wire}>
                {label}
              </option>
            ))}
          </select>
        </label>
        <label>
          Your name
          <input
            type="text"
            required
            value={displayName}
            autoComplete="name"
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </label>
        <label>
          Email
          <input
            type="email"
            required
            value={email}
            autoComplete="email"
            onChange={(e) => setEmail(e.target.value)}
          />
        </label>
        <label>
          Password
          <input
            type="password"
            required
            minLength={8}
            value={password}
            autoComplete="new-password"
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        <label>
          Phone <span className="muted">(optional)</span>
          <input
            type="tel"
            value={phone}
            autoComplete="tel"
            onChange={(e) => setPhone(e.target.value)}
          />
        </label>
        <button type="submit" className="btn primary" disabled={busy}>
          {busy ? 'Creating…' : 'Create my store'}
        </button>
        <p className="login-foot">
          Already have a store? <Link to="/">Sign in</Link>
        </p>
      </form>
    </div>
  )
}
