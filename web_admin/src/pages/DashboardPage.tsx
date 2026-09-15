import { useEffect, useState } from 'react'
import { ApiError, api } from '../lib/api'
import type { BillingSummary } from '../types'
import {
  formatMoney,
  SUBSCRIPTION_STATUS_LABELS,
  isSubscriptionStatus,
} from '../lib/billing'
import { ErrorBanner, InvoiceBadge, Money, Spinner, Stat } from '../components/ui'

export function DashboardPage() {
  const [summary, setSummary] = useState<BillingSummary | null>(null)
  const [error, setError] = useState<string | null>(null)

  async function load() {
    setError(null)
    try {
      setSummary(await api.billingSummary())
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else {
        setError('Failed to load billing summary')
      }
    }
  }

  useEffect(() => {
    void load()
  }, [])

  if (error) {
    return <ErrorBanner error={error} />
  }
  if (!summary) {
    return <Spinner />
  }

  const subscriptions = Object.entries(summary.subscriptions ?? {})
    .filter(([k, v]) => isSubscriptionStatus(k) && v > 0)
    .sort((a, b) => b[1] - a[1])

  const mrrText = formatMoney(summary.mrr_minor, summary.currency)
  const outstandingText = formatMoney(summary.outstanding_minor, summary.currency)

  return (
    <div className="page">
      <header className="page-head">
        <h2>Billing dashboard</h2>
        <button className="btn" onClick={() => void load()}>
          Refresh
        </button>
      </header>

      <div className="stat-grid">
        <Stat label="Monthly recurring revenue" value={mrrText} sub={summary.currency} />
        <Stat label="Outstanding invoices" value={outstandingText} />
        <Stat label="Active plans" value={String(summary.active_plans)} />
        <Stat label="Active subscriptions" value={String(summary.subscriptions?.active ?? 0)} />
      </div>

      <div className="grid-2">
        <section className="card">
          <h3>Subscriptions by state</h3>
          {subscriptions.length === 0 ? (
            <p className="muted">No subscriptions yet.</p>
          ) : (
            <table className="table">
              <tbody>
                {subscriptions.map(([status, count]) => (
                  <tr key={status}>
                    <td>{SUBSCRIPTION_STATUS_LABELS[status as keyof typeof SUBSCRIPTION_STATUS_LABELS]}</td>
                    <td className="num">{count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>

        <section className="card">
          <h3>Payment providers</h3>
          <ul className="providers">
            {summary.providers?.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
          <h3>Recent invoices</h3>
          {summary.recent_invoices?.length ? (
            <div className="recent-invoices">
              {summary.recent_invoices.map((inv) => (
                <div key={inv.id} className="recent-line">
                  <span className="recent-name">{inv.tenant_name}</span>
                  <InvoiceBadge status={inv.status} />
                  <Money minor={inv.amount_minor} currency={inv.currency} muted />
                </div>
              ))}
            </div>
          ) : (
            <p className="muted">No invoices yet.</p>
          )}
        </section>
      </div>
    </div>
  )
}