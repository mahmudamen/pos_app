import type { InvoiceStatus, SubscriptionStatus, TenantStatus } from '../types'
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

export type StatTone = 'plain' | 'accent' | 'info' | 'warn'

/**
 * A KPI tile. The tint is soft rather than solid and each hue carries its own
 * subject, so four tiles read as four different facts instead of four boxes.
 */
export function Stat({
  label,
  value,
  sub,
  tone = 'plain',
}: {
  label: string
  value: React.ReactNode
  sub?: string
  tone?: StatTone
}) {
  return (
    <div className={tone === 'plain' ? 'stat' : `stat tint-${tone}`}>
      <div className="stat-label">{label}</div>
      <div className="stat-value">{value}</div>
      {sub ? <div className="stat-sub">{sub}</div> : null}
    </div>
  )
}

/**
 * A single value inside its contextual range, so the operator reads proportion
 * rather than counting four numbers against each other.
 */
export function Meter({
  label,
  value,
  max,
  tone = 'accent',
  hint,
}: {
  label: string
  value: number
  max: number
  tone?: 'accent' | 'info' | 'warn' | 'success'
  hint?: string
}) {
  const safeMax = max > 0 ? max : 0
  const pct = safeMax > 0 ? Math.min(100, Math.round((value / safeMax) * 100)) : 0
  return (
    <div className="meter-row">
      <span className="meter-name">{label}</span>
      <span className="meter-value">{value}</span>
      <div
        className="meter-track"
        role="meter"
        aria-valuenow={value}
        aria-valuemin={0}
        aria-valuemax={safeMax}
        aria-label={label}
      >
        <div className="meter-fill" style={{ width: `${pct}%`, ['--meter' as string]: `var(--${tone})` }} />
      </div>
      {hint ? <span className="small muted" style={{ gridColumn: '1 / -1' }}>{hint}</span> : null}
    </div>
  )
}

/**
 * Dead ends cost more than empty tables: every empty state names what is
 * missing and offers the next move.
 */
export function EmptyState({
  title,
  hint,
  action,
}: {
  title: string
  hint?: string
  action?: React.ReactNode
}) {
  return (
    <div className="empty">
      <div className="empty-title">{title}</div>
      {hint ? <p className="empty-hint">{hint}</p> : null}
      {action}
    </div>
  )
}

export function PageHead({
  title,
  sub,
  children,
}: {
  title: string
  sub?: React.ReactNode
  children?: React.ReactNode
}) {
  return (
    <header className="page-head">
      <div>
        <h2>{title}</h2>
        {sub ? <p className="page-sub">{sub}</p> : null}
      </div>
      {children ? <div className="head-actions">{children}</div> : null}
    </header>
  )
}

/**
 * Holds the table's shape while it loads, so the page does not jump under the
 * operator and a table is visibly on its way.
 */
export function TableSkeleton({ rows = 5, cols = 4 }: { rows?: number; cols?: number }) {
  return (
    <tbody>
      {Array.from({ length: rows }, (_, r) => (
        <tr key={r}>
          {Array.from({ length: cols }, (_, c) => (
            <td key={c}>
              <div className="skeleton" style={{ width: `${[70, 45, 60, 35][c % 4]}%` }} />
            </td>
          ))}
        </tr>
      ))}
    </tbody>
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

/**
 * Tenant lifecycle, which is a different axis from subscription state: a tenant
 * can have a paid subscription and still be stopped by the platform.
 */
const TENANT_STATUS_COLOR: Record<TenantStatus, string> = {
  active: 'ok',
  trial: 'info',
  suspended: 'warn',
  closed: 'muted',
  disabled: 'bad',
}

export function SubscriptionBadge({ status }: { status: SubscriptionStatus }) {
  return (
    <span className={`badge ${SUB_COLOR[status] ?? 'muted'}`}>
      <span className="badge-dot" aria-hidden="true" />
      {SUBSCRIPTION_STATUS_LABELS[status] ?? status}
    </span>
  )
}

export function TenantStatusBadge({ status }: { status: TenantStatus }) {
  return (
    <span className={`badge ${TENANT_STATUS_COLOR[status] ?? 'muted'}`}>
      <span className="badge-dot" aria-hidden="true" />
      {status}
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