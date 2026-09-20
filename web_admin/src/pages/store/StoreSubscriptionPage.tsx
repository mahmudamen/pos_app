import { useEffect, useState } from 'react'
import { useAuth } from '../../auth/AuthContext'
import { isOwner } from '../../auth/roles'
import { ApiError, api } from '../../lib/api'
import type { Plan, StoreSubscription } from '../../types'
import { formatDate, PERIOD_LABELS } from '../../lib/billing'
import { ErrorBanner, Money, Spinner } from '../../components/ui'

const STATUS_COLOR: Record<string, string> = {
  trial: 'info',
  active: 'ok',
  grace_period: 'warn',
  past_due: 'warn',
  suspended: 'bad',
  cancelled: 'muted',
}

export function StoreSubscriptionPage() {
  const { role } = useAuth()
  const owner = isOwner(role)

  const [subscription, setSubscription] = useState<StoreSubscription | null>(null)
  const [plans, setPlans] = useState<Plan[]>([])
  const [selected, setSelected] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  async function load() {
    setError(null)
    try {
      const [sub, planList] = await Promise.all([
        api.subscription(),
        api.listPlans(),
      ])
      setSubscription(sub)
      const active = planList.filter((p) => p.is_active)
      setPlans(active)
      if (!selected) {
        setSelected(sub.plan?.code ?? '')
      }
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load subscription')
      }
    }
  }

  useEffect(() => {
    void load()
  }, [])

  async function changePlan() {
    if (!selected || selected === subscription?.plan?.code) {
      return
    }
    setBusy(true)
    setError(null)
    try {
      await api.changePlan(selected)
      await load()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Change plan failed')
      }
    } finally {
      setBusy(false)
    }
  }

  if (!subscription) {
    return error ? <ErrorBanner error={error} /> : <Spinner />
  }

  const currentCode = subscription.plan?.code
  const statusColor = STATUS_COLOR[subscription.status] ?? 'muted'

  return (
    <div className="page">
      <header className="page-head">
        <h2>Subscription</h2>
        <span className={`badge ${statusColor}`}>{subscription.status}</span>
      </header>
      <ErrorBanner error={error} />

      <section className="card">
        <h3>Current plan</h3>
        <p>
          <strong>{subscription.plan?.name ?? currentCode ?? '—'}</strong>{' '}
          <span className="badge info">{currentCode ?? '—'}</span>
        </p>
        <div className="form-grid">
          <label>
            Trial ends
            <span className="muted">{formatDate(subscription.trial_ends_at)}</span>
          </label>
          <label>
            Current period ends
            <span className="muted">
              {formatDate(subscription.current_period_end)}
            </span>
          </label>
          <label>
            Cancelled at period end
            <span className="muted">
              {subscription.cancel_at_period_end ? 'Yes' : 'No'}
            </span>
          </label>
        </div>
      </section>

      {owner && plans.length > 0 ? (
        <section className="card">
          <h3>Change plan</h3>
          <div className="form-grid">
            {plans.map((p) => (
              <label
                key={p.id}
                className={p.code === currentCode ? 'muted' : ''}
              >
                <input
                  type="radio"
                  name="plan"
                  value={p.code}
                  checked={selected === p.code}
                  disabled={p.code === currentCode}
                  onChange={() => setSelected(p.code)}
                />
                <strong>{p.name}</strong>
                <div className="muted small">
                  <Money minor={p.price_minor} currency={p.currency} /> /{' '}
                  {PERIOD_LABELS[p.billing_period]}
                </div>
                <div className="muted small">
                  {p.max_users === 0 ? '∞' : p.max_users} users ·{' '}
                  {p.max_products === 0 ? '∞' : p.max_products} products
                </div>
              </label>
            ))}
          </div>
          <div className="form-actions">
            <button
              className="btn primary"
              disabled={
                busy || !selected || selected === (subscription.plan?.code ?? '')
              }
              onClick={() => void changePlan()}
            >
              {busy ? 'Changing…' : 'Change plan'}
            </button>
          </div>
        </section>
      ) : owner ? (
        <section className="card">
          <p className="muted">No plans available.</p>
        </section>
      ) : null}
    </div>
  )
}