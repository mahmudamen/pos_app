import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { isSaasAdmin } from '../auth/roles'
import { BrandMark, Icon } from './icons'

type NavItem = { to: string; label: string; end: boolean; icon: string }

/**
 * Grouped rather than one flat list: the console mixes the platform's own
 * money (plans, subscriptions, invoices) with the merchants it serves, and an
 * operator switching between those two jobs should not have to re-read the
 * whole rail each time.
 */
const SAAS_NAV: { group: string; items: NavItem[] }[] = [
  {
    group: 'Platform',
    items: [
      { to: '/', label: 'Dashboard', end: true, icon: 'dashboard' },
      { to: '/tenants', label: 'Tenants', end: false, icon: 'tenants' },
      { to: '/support', label: 'Support inbox', end: false, icon: 'support' },
    ],
  },
  {
    group: 'Billing',
    items: [
      { to: '/plans', label: 'Plans', end: false, icon: 'plans' },
      { to: '/subscriptions', label: 'Subscriptions', end: false, icon: 'subscriptions' },
      { to: '/invoices', label: 'Invoices', end: false, icon: 'invoices' },
    ],
  },
]

const STORE_NAV: { group: string; items: NavItem[] }[] = [
  {
    group: 'Store',
    items: [
      { to: '/', label: 'Dashboard', end: true, icon: 'dashboard' },
      { to: '/products', label: 'Products', end: false, icon: 'products' },
      { to: '/categories', label: 'Categories', end: false, icon: 'categories' },
      { to: '/inventory', label: 'Inventory', end: false, icon: 'inventory' },
    ],
  },
  {
    group: 'Trading',
    items: [
      { to: '/sales', label: 'Sales', end: false, icon: 'sales' },
      { to: '/purchases', label: 'Purchases', end: false, icon: 'purchases' },
      { to: '/employees', label: 'Employees', end: false, icon: 'employees' },
    ],
  },
  {
    group: 'Account',
    items: [
      { to: '/subscription', label: 'Subscription', end: false, icon: 'subscription' },
      { to: '/community', label: 'Community', end: false, icon: 'community' },
    ],
  },
]

const CASHIER_HIDDEN = ['/employees', '/purchases', '/subscription']

function groupsFor(role: string | null | undefined) {
  const saas = isSaasAdmin(role)
  if (saas) {
    return SAAS_NAV
  }
  if (role === 'cashier') {
    return STORE_NAV.map((g) => ({
      ...g,
      items: g.items.filter((item) => !CASHIER_HIDDEN.includes(item.to)),
    })).filter((g) => g.items.length > 0)
  }
  return STORE_NAV
}

/** Initials for the identity monogram, capped at two letters. */
function initials(name: string | undefined, fallback: string): string {
  const source = (name ?? fallback).trim()
  const parts = source.split(/[\s@._-]+/).filter(Boolean)
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }
  return source.slice(0, 2).toUpperCase()
}

export function Layout() {
  const { user, role, signOut } = useAuth()
  const saas = isSaasAdmin(role)
  const groups = groupsFor(role)
  const who = user?.display_name ?? user?.email ?? 'Operator'

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <BrandMark />
          <div className="brand-text">
            <div className="brand-title">POS.Go</div>
            <div className="brand-sub">{saas ? 'SaaS Admin' : 'Store Console'}</div>
          </div>
        </div>
        <nav className="nav">
          {groups.map((group) => (
            <div key={group.group}>
              <div className="nav-group">{group.group}</div>
              {group.items.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  end={item.end}
                  className={({ isActive }) => (isActive ? 'nav-link active' : 'nav-link')}
                >
                  <Icon name={item.icon} />
                  <span className="nav-label">{item.label}</span>
                </NavLink>
              ))}
            </div>
          ))}
        </nav>
        <div className="sidebar-footer">
          {/* Who is signed in matters here: stop and backup act on a named
              merchant's live data, so the operator should never be guessing
              which account the session belongs to. */}
          <div className="identity">
            <div className="identity-monogram" aria-hidden="true">
              {initials(user?.display_name, who)}
            </div>
            <div className="identity-text">
              <div className="identity-name" title={who}>
                {who}
              </div>
              <div className="identity-role">{role ?? 'operator'}</div>
            </div>
          </div>
          <button className="nav-link signout" onClick={signOut}>
            <Icon name="signout" />
            <span className="nav-label">Sign out</span>
          </button>
        </div>
      </aside>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}
