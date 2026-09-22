import { describe, expect, it } from 'vitest'
import { ARABIC_UNITS, localizedName, localizedUnit } from './ar'

describe('i18n/ar display helpers', () => {
  it('prefers the Arabic name when present', () => {
    expect(localizedName('كولا', 'Cola')).toBe('كولا')
    expect(localizedName('  ', 'Cola')).toBe('Cola')
    expect(localizedName(null, 'Cola')).toBe('Cola')
    expect(localizedName('', '')).toBe('—')
  })

  it('maps common units to Arabic labels with stable fallbacks', () => {
    expect(localizedUnit('kg')).toBe('كجم')
    expect(localizedUnit('pcs')).toBe('قطعة')
    expect(localizedUnit('box')).toBe('علبة')
    expect(localizedUnit('piece')).toBe('قطعة')
    expect(localizedUnit('liter')).toBe('لتر')
    expect(localizedUnit('unmapped_unit')).toBe('unmapped_unit')
    expect(localizedUnit('')).toBe('—')
    expect(localizedUnit(null)).toBe('—')
    expect(ARABIC_UNITS.mesto).toBeUndefined()
  })

  it('maps both kilos and case-insensitive variant', () => {
    expect(localizedUnit('KG')).toBe('كجم')
    expect(localizedUnit('كيلو')).toBe('كيلو')
  })
})