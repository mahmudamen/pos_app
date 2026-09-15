import type { BillingPeriod, InvoiceStatus, Plan, SubscriptionStatus } from '../types'

export const CURRENCY_SYMBOLS: Record<string, string> = {
  EGP: 'E£',
  USD: '$',
  EUR: '€',
  GBP: '£',
  SAR: 'SAR',
  AED: 'AED',
}

const SUBS: SubscriptionStatus[] = [
  'trial',
  'active',
  'grace_period',
  'past_due',
  'suspended',
  'cancelled',
]

const INVS: InvoiceStatus[] = ['open', 'paid', 'void', 'refunded']

export function isSubscriptionStatus(v: string): v is SubscriptionStatus {
  return (SUBS as string[]).includes(v)
}

export function isInvoiceStatus(v: string): v is InvoiceStatus {
  return (INVS as string[]).includes(v)
}

export function formatMoney(minor: number, currency = 'EGP'): string {
  const symbol = CURRENCY_SYMBOLS[currency] ?? `${currency} `
  const whole = Math.floor(minor / 100)
  const frac = Math.abs(minor % 100)
  const fracStr = String(frac).padStart(2, '0')
  return `${symbol}${whole}.${fracStr}`
}

/** Convert a minor-unit price to a decimal string safe for input fields. */
export function minorToDecimal(minor: number): string {
  const sign = minor < 0 ? '-' : ''
  const abs = Math.abs(minor)
  return `${sign}${Math.floor(abs / 100)}.${String(abs % 100).padStart(2, '0')}`
}

/** Parse a decimal string into minor units. Returns null for invalid input. */
export function decimalToMinor(value: string): number | null {
  const trimmed = value.trim()
  if (!/^\d+(\.\d{1,2})?$/.test(trimmed)) {
    return null
  }
  const [whole, frac = ''] = trimmed.split('.')
  const wholeMinor = parseInt(whole, 10) * 100
  const fracMinor = parseInt(frac.padEnd(2, '0'), 10) || 0
  return wholeMinor + fracMinor
}

/** Monthly-equivalent price so yearly plans can be compared on the same axis. */
export function monthlyEquivalentMinor(
  priceMinor: number,
  period: BillingPeriod,
): number {
  if (period === 'yearly') {
    return Math.round(priceMinor / 12)
  }
  return priceMinor
}

export const SUBSCRIPTION_STATUS_LABELS: Record<SubscriptionStatus, string> = {
  trial: 'Trial',
  active: 'Active',
  grace_period: 'Grace period',
  past_due: 'Past due',
  suspended: 'Suspended',
  cancelled: 'Cancelled',
}

export const INVOICE_STATUS_LABELS: Record<InvoiceStatus, string> = {
  open: 'Open',
  paid: 'Paid',
  void: 'Void',
  refunded: 'Refunded',
}

export const PERIOD_LABELS: Record<BillingPeriod, string> = {
  monthly: 'Monthly',
  yearly: 'Yearly',
}

/**
 * Which subscription transitions the control plane permits. Mirrors the
 * backend state machine (internal/billing/subscription.go ValidateTransition).
 */
export function allowedTransitions(
  from: SubscriptionStatus,
): SubscriptionStatus[] {
  switch (from) {
    case 'trial':
    case 'grace_period':
    case 'past_due':
    case 'suspended':
      return ['active', 'cancelled']
    case 'active':
      return ['grace_period', 'suspended', 'cancelled']
    case 'cancelled':
      return []
    default:
      return []
  }
}

export function canTransition(from: SubscriptionStatus, to: SubscriptionStatus): boolean {
  return allowedTransitions(from).includes(to)
}

/** True when a subscription is billable (counts toward MRR in the summary). */
export function isBillable(status: SubscriptionStatus): boolean {
  return (
    status === 'trial' ||
    status === 'active' ||
    status === 'grace_period' ||
    status === 'past_due'
  )
}

/** Next natural billing date, best-effort from a plan period + start time. */
export function addPeriod(from: Date, period: BillingPeriod): Date {
  const d = new Date(from)
  if (period === 'yearly') {
    d.setUTCFullYear(d.getUTCFullYear() + 1)
  } else {
    d.setUTCMonth(d.getUTCMonth() + 1)
  }
  return d
}

export function formatDate(iso: string | undefined): string {
  if (!iso) {
    return '—'
  }
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) {
    return iso
  }
  return d.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

/** Sum MRR of a plan list at its monthly-equivalent unit price. */
export function planMrr(plans: Plan[]): number {
  return plans.reduce(
    (sum, p) => sum + (p.is_active ? monthlyEquivalentMinor(p.price_minor, p.billing_period) : 0),
    0,
  )
}

export function businessTypeLabel(t: string): string {
  return t
    .split('_')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ')
}