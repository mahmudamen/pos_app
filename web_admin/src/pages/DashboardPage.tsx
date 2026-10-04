import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiError, api } from '../lib/api'
import type { BillingSummary } from '../types'
import {
  formatMoney,
  SUBSCRIPTION_STATUS_LABELS,
  isSubscriptionStatus,
} from '../lib/billing'
import {
  EmptyState,
  ErrorBanner,
  InvoiceBadge,
  Meter,
  Money,
  PageHead,
  Spinner,
  Stat,
} from '../components/ui'

/** Health first: what an operator opens this screen to answer. */
export function DashboardPage() {
  const [summary, setSummary] = useState<BillingSummary | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [refreshedAt, setRefreshedAt] = useState<Date | null>(null)

  async function load() {
    setError(null)
    try {
      setSummary(await api.billingSummary())
      setRefreshedAt(new Date())
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

  if (error && !summary) {
    return <ErrorBanner error={error} />
  }
  if (!summary) {
    return <Spinner />
  }

  const subscriptions = Object.entries(summary.subscriptions ?? {})
    .filter(([k, v]) => isSubscriptionStatus(k) && v > 0)
    .sort((a, b) => b[1] - a[1])
  const subscriptionTotal = subscriptions.reduce((sum, [, count]) => sum + count, 0)
  const needsAttention = (summary.subscriptions?.past_due ?? 0) + (summary.subscriptions?.grace_period ?? 0)

  const mrrText = formatMoney(summary.mrr_minor, summary.currency)
  const outstandingText = formatMoney(summary.outstanding_minor, summary.currency)

  // Subscription states are parts of one whole, so they read as meters rather
  // than a two-column table the operator has to total up mentally.
  const toneFor = (status: string) =>
    status === 'active'
      ? 'success'
      : status === 'cancelled'
        ? 'info'
        : status === 'trial'
          ? 'accent'
          : 'warn'

  return (
    <div className="page">
      <PageHead
        title="Billing dashboard"
        sub={
          refreshedAt
            ? `Updated ${refreshedAt.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`
            : 'Platform revenue and subscription health'
        }
      >
        <button className="btn" onClick={() => void load()}>
          Refresh
        </button>
      </PageHead>

      <ErrorBanner error={error} />

      <div className="stat-grid">
        <Stat label="Monthly recurring revenue" value={mrrText} sub={summary.currency} tone="accent" />
        <Stat
          label="Outstanding invoices"
          value={outstandingText}
          sub={summary.outstanding_minor > 0 ? 'Awaiting payment' : 'All invoices settled'}
          tone={summary.outstanding_minor > 0 ? 'warn' : 'plain'}
        />
        <Stat label="Active plans" value={String(summary.active_plans)} />
        <Stat
          label="Active subscriptions"
          value={String(summary.subscriptions?.active ?? 0)}
          sub={
            needsAttention > 0
              ? `${needsAttention} need${needsAttention === 1 ? 's' : ''} attention`
              : 'None past due'
          }
          tone={needsAttention > 0 ? 'warn' : 'plain'}
        />
      </div>

      <div className="grid-2">
        <section className="card">
          <h3>Subscriptions by state</h3>
          {subscriptions.length === 0 ? (
            <EmptyState
              title="No subscriptions yet"
              hint="Once a merchant completes checkout or lands on a trial, its state shows up here."
              action={
                <Link className="btn" to="/tenants">
                  Open tenants
                </Link>
              }
            />
          ) : (
            <>
              <div className="meter-list">
                {subscriptions.map(([status, count]) => (
                  <Meter
                    key={status}
                    label={SUBSCRIPTION_STATUS_LABELS[status as keyof typeof SUBSCRIPTION_STATUS_LABELS]}
                    value={count}
                    max={subscriptionTotal}
                    tone={toneFor(status) as 'accent' | 'success' | 'info' | 'warn'}
                  />
                ))}
              </div>
              <p className="small muted" style={{ marginTop: 'var(--s3)' }}>
                {subscriptionTotal} subscriptions across{' '}
                {subscriptions.length} {subscriptions.length === 1 ? 'state' : 'states'}.
              </p>
            </>
          )}
        </section>

        <section className="card">
          <h3>Payment providers</h3>
          {summary.providers?.length ? (
            <ul className="providers" style={{ marginBottom: 'var(--s4)' }}>
              {summary.providers.map((p) => (
                <li key={p}>{p}</li>
              ))}
            </ul>
          ) : (
            <p className="muted small" style={{ marginBottom: 'var(--s4)' }}>
              No payment provider recorded yet.
            </p>
          )}

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
            <EmptyState title="No invoices yet" hint="Issued invoices appear here as soon as one is raised." />
          )}
        </section>
      </div>
    </div>
  )
}