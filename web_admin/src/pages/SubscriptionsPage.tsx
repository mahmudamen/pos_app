import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiError, api, SUBSCRIPTION_STATUSES } from '../lib/api'
import type { Plan, Subscription, SubscriptionStatus } from '../types'
import {
  allowedTransitions,
  formatMoney,
  PERIOD_LABELS,
  SUBSCRIPTION_STATUS_LABELS,
} from '../lib/billing'
import { ErrorBanner, Spinner, SubscriptionBadge } from '../components/ui'

export function SubscriptionsPage() {
  const [subs, setSubs] = useState<Subscription[]>([])
  const [plans, setPlans] = useState<Plan[]>([])
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState<SubscriptionStatus | ''>('')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)

  async function load() {
    setError(null)
    try {
      const subRes = await api.listSubscriptions(page, 100, status)
      const planRes = await api.listPlans()
      setSubs(subRes.data)
      setPlans(planRes.filter((p) => p.is_active))
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load subscriptions')
      }
    }
  }

  useEffect(() => {
    void load()
  }, [page, status])

  async function transition(sub: Subscription, target: SubscriptionStatus) {
    setBusy(sub.id)
    setError(null)
    try {
      await api.setSubscriptionStatus(sub.id, target)
      await load()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      }
    } finally {
      setBusy(null)
    }
  }

  return (
    <div className="page">
      <header className="page-head">
        <h2>Subscriptions</h2>
        <select
          className="status-filter"
          value={status}
          onChange={(e) => { setPage(1); setStatus(e.target.value as SubscriptionStatus | '') }}
          aria-label="Filter by status"
        >
          <option value="">All</option>
          {SUBSCRIPTION_STATUSES.map((s) => (
            <option key={s} value={s}>
              {SUBSCRIPTION_STATUS_LABELS[s]}
            </option>
          ))}
        </select>
      </header>
      <ErrorBanner error={error} />
      {!subs.length && !error ? (
        <Spinner />
      ) : (
        <table className="table card">
          <thead>
            <tr>
              <th>Tenant</th>
              <th>Plan</th>
              <th>Status</th>
              <th>Price / period</th>
              <th>Provider</th>
              <th>Period start</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {subs.map((s) => {
              const allowed = allowedTransitions(s.status)
              return (
                <tr key={s.id}>
                  <td>
                    <Link className="table-link" to={`/tenants/${s.tenant_id}`}>
                      {s.tenant_name}
                    </Link>
                  </td>
                  <td>{s.plan_code}</td>
                  <td><SubscriptionBadge status={s.status} /></td>
                  <td>
                    {formatMoney(s.price_minor, s.currency)} / {PERIOD_LABELS[s.billing_period]}
                  </td>
                  <td>{s.provider}</td>
                  <td className="muted">{s.current_period_start.slice(0, 10)}</td>
                  <td className="actions">
                    {allowed.map((t) => (
                      <button
                        key={t}
                        className="btn small"
                        disabled={busy !== null}
                        onClick={() => void transition(s, t)}
                      >
                        {SUBSCRIPTION_STATUS_LABELS[t]}
                      </button>
                    ))}
                    {s.status !== 'cancelled' && plans.length > 0 ? (
                      <ChangePlanButton subscription={s} plans={plans} onDone={() => void load()} />
                    ) : null}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
    </div>
  )
}

function ChangePlanButton({
  subscription,
  plans,
  onDone,
}: {
  subscription: Subscription
  plans: Plan[]
  onDone: () => void
}) {
  const [open, setOpen] = useState(false)
  const [selected, setSelected] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function doChange() {
    if (!selected) {
      return
    }
    setBusy(true)
    setError(null)
    try {
      await api.changeSubscriptionPlan(subscription.id, selected)
      setOpen(false)
      onDone()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      }
    } finally {
      setBusy(false)
    }
  }

  if (!open) {
    return (
      <button className="btn small" onClick={() => setOpen(true)}>
        Change plan
      </button>
    )
  }

  return (
    <div className="inline-change">
      <select value={selected} onChange={(e) => setSelected(e.target.value)} aria-label="New plan">
        <option value="">Select…</option>
        {plans
          .filter((p) => p.code !== subscription.plan_code)
          .map((p) => (
            <option key={p.id} value={p.code}>
              {p.name}
            </option>
          ))}
      </select>
      <button className="btn small primary" disabled={!selected || busy} onClick={() => void doChange()}>
        Go
      </button>
      <button className="btn small" onClick={() => setOpen(false)}>
        ✕
      </button>
      {error ? <div className="error-banner inline">{error}</div> : null}
    </div>
  )
}