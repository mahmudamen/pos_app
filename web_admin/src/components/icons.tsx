/**
 * Inline icon set and the POS.Go brand mark.
 *
 * Hand-drawn 24px stroke paths rather than an icon package: the console ships
 * as one bundle behind nginx with no runtime dependencies, and the nav needs
 * fifteen glyphs — a dependency would cost more than it saves. Geometry follows
 * the Feather conventions (round caps, 1.75 stroke, 24px box).
 */

const PATHS: Record<string, string[]> = {
  dashboard: ['M3 3h7v7H3z', 'M14 3h7v7h-7z', 'M14 14h7v7h-7z', 'M3 14h7v7H3z'],
  tenants: ['M3 21h18', 'M5 21V7l7-4 7 4v14', 'M9 9h.01', 'M9 13h.01', 'M9 17h.01', 'M15 13h.01', 'M15 17h.01'],
  plans: ['M12 2 2 7l10 5 10-5-10-5z', 'M2 17l10 5 10-5', 'M2 12l10 5 10-5'],
  subscriptions: ['M21 2v6h-6', 'M3 12a9 9 0 0 1 15-6.7L21 8', 'M3 22v-6h6', 'M21 12a9 9 0 0 1-15 6.7L3 16'],
  invoices: ['M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z', 'M14 2v6h6', 'M16 13H8', 'M16 17H8', 'M10 9H8'],
  products: ['M20.6 13.4 13.4 20.6a2 2 0 0 1-2.8 0L2 12V2h10l8.6 8.6a2 2 0 0 1 0 2.8z', 'M7 7h.01'],
  categories: ['M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z'],
  employees: ['M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2', 'M9 3a4 4 0 1 1 0 8 4 4 0 0 1 0-8z', 'M22 21v-2a4 4 0 0 0-3-3.9'],
  sales: ['M18 20V10', 'M12 20V4', 'M6 20v-6'],
  purchases: [
    'M9 20a1 1 0 1 0 0 2 1 1 0 0 0 0-2z',
    'M20 20a1 1 0 1 0 0 2 1 1 0 0 0 0-2z',
    'M1 1h4l2.7 13.4a2 2 0 0 0 2 1.6h9.7a2 2 0 0 0 2-1.6L23 6H6',
  ],
  inventory: ['M21 8v13H3V8', 'M1 3h22v5H1z', 'M10 12h4'],
  subscription: ['M2 6h20v12H2z', 'M2 10h20'],
  community: ['M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20z', 'M2 12h20', 'M12 2a15 15 0 0 1 4 10 15 15 0 0 1-4 10 15 15 0 0 1-4-10 15 15 0 0 1 4-10z'],
  signout: ['M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4', 'M16 17l5-5-5-5', 'M21 12H9'],
}

export type IconName = keyof typeof PATHS

export function Icon({ name, className = 'nav-icon' }: { name: string; className?: string }) {
  const paths = PATHS[name]
  if (!paths) {
    return null
  }
  return (
    <svg className={className} viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      {paths.map((d, i) => (
        <path key={i} d={d} />
      ))}
    </svg>
  )
}

/**
 * The bolt from the shipped POS.Go app icon (public/favicon.svg). It keeps the
 * icon's own violet — a logo is identity, not interface accent, and the console
 * accent is the brand green from the landing page.
 */
export function BrandMark() {
  return (
    <span className="brand-mark">
      <svg viewBox="0 0 48 46" aria-hidden="true" focusable="false">
        <path
          fill="#863bff"
          d="M25.946 44.938c-.664.845-2.021.375-2.021-.698V33.937a2.26 2.26 0 0 0-2.262-2.262H10.287c-.92 0-1.456-1.04-.92-1.788l7.48-10.471c1.07-1.497 0-3.578-1.842-3.578H1.237c-.92 0-1.456-1.04-.92-1.788L10.013.474c.214-.297.556-.474.92-.474h28.894c.92 0 1.456 1.04.92 1.788l-7.48 10.471c-1.07 1.498 0 3.579 1.842 3.579h11.377c.943 0 1.473 1.088.89 1.83L25.947 44.94z"
        />
      </svg>
    </span>
  )
}
