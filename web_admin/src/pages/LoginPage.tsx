import { useState } from 'react'
import { useAuth } from '../auth/AuthContext'
import { ApiError } from '../lib/api'
import { ErrorBanner } from '../components/ui'

const DEFAULT_TENANT =
  (import.meta.env.VITE_SAAS_TENANT_ID as string | undefined) ?? ''

export function LoginPage() {
  const { signIn } = useAuth()
  const [tenant, setTenant] = useState(DEFAULT_TENANT)
  const [email, setEmail] = useState(
    (import.meta.env.VITE_SAAS_EMAIL as string | undefined) ?? '',
  )
  const [password, setPassword] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setBusy(true)
    try {
      await signIn(tenant.trim(), email.trim(), password)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message || err.code)
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
          <h1>POS.Go Admin</h1>
          <p>Subscriptions, plans &amp; billing control plane</p>
        </div>
        <ErrorBanner error={error} />
        <label>
          Tenant ID
          <input
            type="text"
            required
            value={tenant}
            placeholder="platform tenant uuid"
            onChange={(e) => setTenant(e.target.value)}
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
            value={password}
            autoComplete="current-password"
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        <button type="submit" className="btn primary" disabled={busy}>
          {busy ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  )
}