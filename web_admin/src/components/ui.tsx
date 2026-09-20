import type { InvoiceStatus, SubscriptionStatus } from '../types'
import { formatMoney, INVOICE_STATUS_LABELS, SUBSCRIPTION_STATUS_LABELS } from '../lib/billing'

export function Spinner() {
  return <span className="spinner" aria-label="loading" />
}

export function ErrorBanner({ error }: { error: string | null }) {
  if (!error) {
    return null
  }
  return (
    <div className="error-banner" role="alert">
      {error}
    </div>
  )
}

export function Money({
  minor,
  currency,
  muted,
}: {
  minor: number
  currency: string
  muted?: boolean
}) {
  return <span className={muted ? 'money muted' : 'money'}>{formatMoney(minor, currency)}</span>
}

export function Stat({
  label,
  value,
  sub,
}: {
  label: string
  value: React.ReactNode
  sub?: string
}) {
  return (
    <div className="stat card">
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
      {sub ? <div className="stat-sub">{sub}</div> : null}
    </div>
  )
}

const SUB_COLOR: Record<SubscriptionStatus, string> = {
  trial: 'info',
  active: 'ok',
  grace_period: 'warn',
  past_due: 'warn',
  suspended: 'bad',
  cancelled: 'muted',
}

const INV_COLOR: Record<InvoiceStatus, string> = {
  open: 'warn',
  paid: 'ok',
  void: 'muted',
  refunded: 'info',
}

export function SubscriptionBadge({ status }: { status: SubscriptionStatus }) {
  return (
    <span className={`badge ${SUB_COLOR[status] ?? 'muted'}`}>
      {SUBSCRIPTION_STATUS_LABELS[status] ?? status}
    </span>
  )
}

export function InvoiceBadge({ status }: { status: InvoiceStatus }) {
  return (
    <span className={`badge ${INV_COLOR[status] ?? 'muted'}`}>
      {INVOICE_STATUS_LABELS[status] ?? status}
    </span>
  )
}