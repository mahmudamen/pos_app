// Arabic-first display helpers for the store console. Names prefer the
// bilingual `name_ar` field when the merchant filled it in; units are mapped
// through a small Arabic dictionary with the code acronym as a stable fallback.

const UNIT_AR: Record<string, string> = {
  kg: 'كجم',
  kilo: 'كيلو',
  g: 'جم',
  pcs: 'قطعة',
  piece: 'قطعة',
  box: 'علبة',
  dozen: 'دستة',
  liter: 'لتر',
  l: 'لتر',
  m: 'متر',
  meter: 'متر',
}

/** Prefer the Arabic localized name; fall back to the canonical English one. */
export function localizedName(nameAr?: string | null, name?: string | null): string {
  const ar = nameAr?.trim()
  if (ar) return ar
  return (name ?? '—').trim() || '—'
}

/** Localized unit label (e.g. kg → كجم) with the code kept when unknown. */
export function localizedUnit(unit?: string | null): string {
  const u = unit?.trim()
  if (!u) return '—'
  const lower = u.toLowerCase()
  return UNIT_AR[lower] ?? u
}

export const ARABIC_UNITS = UNIT_AR