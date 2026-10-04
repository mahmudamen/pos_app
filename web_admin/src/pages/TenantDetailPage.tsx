import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError, api } from '../lib/api'
import type { Invoice, Plan, Subscription, TenantAnalytics } from '../types'
import { businessTypeLabel, formatMoney, PERIOD_LABELS } from '../lib/billing'
import {
  ErrorBanner,
  InvoiceBadge,
  Money,
  Spinner,
  Stat,
  SubscriptionBadge,
  EmptyState,
  Meter,
  PageHead,
} from '../components/ui'

const TRIAL_DAYS = 15

export function TenantDetailPage() {
  const { id = '' } = useParams()
  const [analytics, setAnalytics] = useState<TenantAnalytics | null>(null)
  const [subscription, setSubscription] = useState<Subscription | null>(null)
  const [invoices, setInvoices] = useState<Invoice[]>([])
  const [plans, setPlans] = useState<Plan[]>([])
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  // assign-subscription form
  const [assignOpen, setAssignOpen] = useState(false)
  const [assignPlan, setAssignPlan] = useState('')
  const [trialDays, setTrialDays] = useState(TRIAL_DAYS)

  // new-invoice form
  const [invoiceOpen, setInvoiceOpen] = useState(false)
  const [invoiceAmount, setInvoiceAmount] = useState('')
  const [invoiceDescription, setInvoiceDescription] = useState('')

  async function load() {
    setError(null)
    try {
      const analyticsRes = await api.tenantAnalytics(id)
      const plansRes = await api.listPlans()
      const invRes = await api.listInvoices(1, 100)
      setAnalytics(analyticsRes)
      setPlans(plansRes.filter((p) => p.is_active))
      setInvoices(invRes.data.filter((i) => i.tenant_id === id))
      try {
        setSubscription(await api.tenantSubscription(id))
      } catch {
        setSubscription(null)
      }
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load tenant')
      }
    }
  }

  useEffect(() => {
    void load()
  }, [id])

  async function run(action: () => Promise<unknown>) {
    setBusy(true)
    setError(null)
    try {
      await action()
      await load()
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Request failed')
      }
    } finally {
      setBusy(false)
    }
  }

  async function assignSubscription() {
    await run(() =>
      api.assignSubscription(id, assignPlan, 'trial', trialDays),
    )
    setAssignOpen(false)
    setAssignPlan('')
  }

  async function createInvoice() {
    const amountMinor = Math.round(parseFloat(invoiceAmount) * 100)
    if (!Number.isFinite(amountMinor) || amountMinor <= 0) {
      setError('Enter a valid amount')
      return
    }
    await run(() =>
      api.createInvoice(id, {
        amount_minor: amountMinor,
        description: invoiceDescription || 'Manual invoice',
      }),
    )
    setInvoiceOpen(false)
    setInvoiceAmount('')
    setInvoiceDescription('')
  }

  if (error && !analytics) {
    return <ErrorBanner error={error} />
  }
  if (!analytics) {
    return <Spinner />
  }

  return (
    <div className="page">
      <PageHead title={analytics.name} sub={`${businessTypeLabel(analytics.business_type)} · ${analytics.slug}`}>
        {subscription ? (
          <SubscriptionBadge status={subscription.status} />
        ) : (
          <button className="btn primary" onClick={() => setAssignOpen(true)} disabled={busy}>
            Assign subscription
          </button>
        )}
        <button className="btn" onClick={() => setInvoiceOpen(true)} disabled={busy}>
          New invoice
        </button>
      </PageHead>

      <ErrorBanner error={error} />

      {assignOpen ? (
        <section className="card form-row">
          <select value={assignPlan} onChange={(e) => setAssignPlan(e.target.value)} aria-label="Plan">
            <option value="">Select plan…</option>
            {plans.map((p) => (
              <option key={p.id} value={p.code}>
                {p.name} ({p.code}) —{' '}
                {formatMoney(p.price_minor, p.currency)} / {PERIOD_LABELS[p.billing_period]}
              </option>
            ))}
          </select>
          <label className="inline">
            Trial days
            <input
              type="number"
              min={0}
              max={90}
              value={trialDays}
              onChange={(e) => setTrialDays(parseInt(e.target.value, 10) || 0)}
            />
          </label>
          <button
            className="btn primary"
            disabled={!assignPlan || busy}
            onClick={() => void assignSubscription()}
          >
            Assign
          </button>
          <button className="btn" onClick={() => setAssignOpen(false)}>
            Cancel
          </button>
        </section>
      ) : null}

      {invoiceOpen ? (
        <section className="card form-row">
          <input
            type="text"
            inputMode="decimal"
            placeholder="Amount (e.g. 150.00)"
            value={invoiceAmount}
            onChange={(e) => setInvoiceAmount(e.target.value)}
          />
          <input
            type="text"
            placeholder="Description"
            value={invoiceDescription}
            onChange={(e) => setInvoiceDescription(e.target.value)}
          />
          <button
            className="btn primary"
            disabled={busy || !invoiceAmount}
            onClick={() => void createInvoice()}
          >
            Create
          </button>
          <button className="btn" onClick={() => setInvoiceOpen(false)}>
            Cancel
          </button>
        </section>
      ) : null}

      <div className="stat-grid">
        <Stat
          label="Users"
          value={String(analytics.users)}
          sub={analytics.max_users === 0 ? 'unlimited plan cap' : `of ${analytics.max_users} seats`}
        />
        <Stat
          label="Products"
          value={String(analytics.products)}
          sub={analytics.max_products === 0 ? 'unlimited plan cap' : `of ${analytics.max_products} slots`}
        />
        <Stat label="Sales today" value={String(analytics.today_sales)} tone="accent" />
        <Stat label="Revenue today" value={formatMoney(analytics.today_revenue_minor, 'EGP')} />
      </div>

      <div className="grid-3" style={{ marginBottom: 'var(--s5)' }}>
        <section className="card">
          <h3>Seat usage</h3>
          <Meter
            label="Users"
            value={analytics.users}
            max={analytics.max_users}
            tone="accent"
            hint={analytics.max_users === 0 ? 'Plan has no seat limit' : undefined}
          />
        </section>
        <section className="card">
          <h3>Catalog usage</h3>
          <Meter
            label="Products"
            value={analytics.products}
            max={analytics.max_products}
            tone="info"
            hint={analytics.max_products === 0 ? 'Plan has no product limit' : undefined}
          />
        </section>
        <section className="card">
          <h3>Plan</h3>
          <p>
            <span className="badge accent">{analytics.plan || '—'}</span>
          </p>
          <p className="small muted" style={{ margin: 0 }}>
            {subscription
              ? `${subscription.plan_name} · ${PERIOD_LABELS[subscription.billing_period]}`
              : 'No subscription assigned'}
          </p>
        </section>
      </div>

      <div className="grid-2">
        <section className="card">
          <h3>Subscription</h3>
          {subscription ? (
            <>
              <p>
                <span className="badge info">{subscription.plan_name}</span>{' '}
                <SubscriptionBadge status={subscription.status} />
              </p>
              <table className="table">
                <tbody>
                  <tr><td>Price</td><td className="num"><Money minor={subscription.price_minor} currency={subscription.currency} /></td></tr>
                  <tr><td>Provider</td><td>{subscription.provider}</td></tr>
                  <tr><td>Period</td><td>{PERIOD_LABELS[subscription.billing_period]} · starts {subscription.current_period_start.slice(0, 10)}</td></tr>
                  <tr><td>Created</td><td>{subscription.created_at.slice(0, 10)}</td></tr>
                </tbody>
              </table>
            </>
          ) : (
            <EmptyState
              title="No subscription"
              hint="Assign a plan so this merchant can transact."
              action={
                <button className="btn" onClick={() => setAssignOpen(true)} disabled={busy}>
                  Assign subscription
                </button>
              }
            />
          )}
        </section>

        <section className="card">
          <h3>Today</h3>
          <div className="stat-grid inner">
            <Stat label="Revenue" value={formatMoney(analytics.today_revenue_minor, 'EGP')} />
            <Stat label="Sales" value={String(analytics.today_sales)} />
          </div>
          <h3>7-day trend</h3>
          {analytics.revenue_trend.length ? (
            <div className="bars">
              {analytics.revenue_trend.map((p) => (
                <div key={p.day} className="bar-col" title={`${p.day}: ${formatMoney(p.revenue_minor)}`}>
                  <div
                    className="bar"
                    style={{ height: `${barHeight(p.revenue_minor, analytics.revenue_trend)}%` }}
                  />
                  <span className="bar-label">{p.day.slice(5)}</span>
                </div>
              ))}
            </div>
          ) : (
            <p className="muted">No sales yet.</p>
          )}
        </section>
      </div>

      <section className="card">
        <h3>Invoices</h3>
        {invoices.length === 0 ? (
          <EmptyState
            title="No invoices"
            hint="Raise one to bill this merchant."
            action={
              <button className="btn" onClick={() => setInvoiceOpen(true)} disabled={busy}>
                New invoice
              </button>
            }
          />
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Description</th>
                <th>Status</th>
                <th>Amount</th>
              </tr>
            </thead>
            <tbody>
              {invoices.map((inv) => (
                <tr key={inv.id}>
                  <td>{inv.created_at.slice(0, 10)}</td>
                  <td>{inv.description || <span className="muted">—</span>}</td>
                  <td><InvoiceBadge status={inv.status} /></td>
                  <td className="num"><Money minor={inv.amount_minor} currency={inv.currency} muted /></td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section className="card">
        <h3>Key sales metrics</h3>
        <div className="grid-2">
          <div>
            <h4>Top products</h4>
            {analytics.top_products.length ? (
              <ol className="simple-list">
                {analytics.top_products.map((p) => (
                  <li key={p.product_name}>
                    {p.product_name} · {p.quantity} sold ·{' '}
                    <Money minor={p.revenue_minor} currency="EGP" muted />
                  </li>
                ))}
              </ol>
            ) : (
              <EmptyState title="No products sold yet" hint="This merchant has not recorded a sale." />
            )}
          </div>
          <div>
            <h4>Recent sales</h4>
            {analytics.recent_sales.length ? (
              <ol className="simple-list">
                {analytics.recent_sales.slice(0, 5).map((s) => (
                  <li key={s.id}>
                    {s.cashier || 'cash'} · {s.payment_method} ·{' '}
                    <Money minor={s.total_minor} currency={s.currency} muted />
                  </li>
                ))}
              </ol>
            ) : (
              <EmptyState title="No recent sales" hint="Completed sales show up here." />
            )}
          </div>
        </div>
      </section>
    </div>
  )
}

function barHeight(value: number, all: { revenue_minor: number }[]): number {
  const max = Math.max(...all.map((p) => p.revenue_minor), 1)
  return Math.max(4, Math.round((value / max) * 100))
}

