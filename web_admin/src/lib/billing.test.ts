import { describe, expect, it } from 'vitest'
import {
  addPeriod,
  allowedTransitions,
  businessTypeLabel,
  canTransition,
  decimalToMinor,
  formatDate,
  formatMoney,
  isBillable,
  minorToDecimal,
  monthlyEquivalentMinor,
  planMrr,
} from './billing'

describe('formatMoney', () => {
  it('formats EGP with the E£ symbol and 2 decimals', () => {
    expect(formatMoney(650)).toBe('E£6.50')
    expect(formatMoney(0)).toBe('E£0.00')
    expect(formatMoney(123456)).toBe('E£1234.56')
  })

  it('handles negative and non-EGP currencies', () => {
    expect(formatMoney(-500)).toBe('E£-5.00')
    expect(formatMoney(9900, 'USD')).toBe('$99.00')
    expect(formatMoney(100, 'SAR')).toBe('SAR1.00')
  })

  it('pads single fractional digits', () => {
    expect(formatMoney(5)).toBe('E£0.05')
  })
})

describe('decimal <-> minor units', () => {
  it('round-trips plain decimals', () => {
    expect(decimalToMinor('99.00')).toBe(9900)
    expect(decimalToMinor('0.5')).toBe(50)
    expect(decimalToMinor('0')).toBe(0)
    expect(decimalToMinor('1234.56')).toBe(123456)
  })

  it('rejects malformed input', () => {
    expect(decimalToMinor('1.234')).toBeNull()
    expect(decimalToMinor('abc')).toBeNull()
    expect(decimalToMinor('-5')).toBeNull()
    expect(decimalToMinor('')).toBeNull()
  })

  it('minorToDecimal produces a safe two-place string', () => {
    expect(minorToDecimal(9900)).toBe('99.00')
    expect(minorToDecimal(2005)).toBe('20.05')
  })
})

describe('monthlyEquivalentMinor', () => {
  it('is identity for monthly plans', () => {
    expect(monthlyEquivalentMinor(8000, 'monthly')).toBe(8000)
  })

  it('divides yearly prices and rounds to minor units', () => {
    expect(monthlyEquivalentMinor(120000, 'yearly')).toBe(10000)
    expect(monthlyEquivalentMinor(1, 'yearly')).toBe(0)
    expect(monthlyEquivalentMinor(99000, 'yearly')).toBe(8250)
  })
})

describe('subscription state machine helpers', () => {
  it('allows the same transitions as the backend', () => {
    expect(allowedTransitions('trial')).toEqual(['active', 'cancelled'])
    expect(allowedTransitions('active')).toEqual([
      'grace_period',
      'suspended',
      'cancelled',
    ])
    expect(allowedTransitions('cancelled')).toEqual([])
  })

  it('canTransition rejects unsupported hops', () => {
    expect(canTransition('active', 'grace_period')).toBe(true)
    expect(canTransition('active', 'trial')).toBe(false)
    expect(canTransition('cancelled', 'active')).toBe(false)
  })

  it('isBillable covers MRR states only', () => {
    expect(isBillable('trial')).toBe(true)
    expect(isBillable('active')).toBe(true)
    expect(isBillable('grace_period')).toBe(true)
    expect(isBillable('past_due')).toBe(true)
    expect(isBillable('suspended')).toBe(false)
    expect(isBillable('cancelled')).toBe(false)
  })
})

describe('dates', () => {
  it('addPeriod adds a month or a year', () => {
    const base = new Date(Date.UTC(2026, 0, 31))
    expect(addPeriod(base, 'monthly').toISOString().slice(0, 10)).toBe('2026-03-03')
    expect(addPeriod(base, 'yearly').toISOString().slice(0, 10)).toBe('2027-01-31')
  })

  it('formats ISO dates and returns a dash for empty', () => {
    expect(formatDate(undefined)).toBe('—')
    expect(formatDate('')).toBe('—')
    expect(formatDate('nonsense')).toBe('nonsense')
  })
})

describe('planMrr', () => {
  it('sums monthly-equivalent prices of active plans only', () => {
    const plans = [
      { price_minor: 1000, billing_period: 'monthly', is_active: true } as never,
      { price_minor: 120000, billing_period: 'yearly', is_active: true } as never,
      { price_minor: 5000, billing_period: 'monthly', is_active: false } as never,
    ]
    expect(planMrr(plans)).toBe(11000)
  })
})

describe('businessTypeLabel', () => {
  it('title-cases snake_case types', () => {
    expect(businessTypeLabel('coffee_shop')).toBe('Coffee Shop')
    expect(businessTypeLabel('book_store')).toBe('Book Store')
  })
})