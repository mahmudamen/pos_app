import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { isSaasAdmin } from '../auth/roles'

type NavItem = { to: string; label: string; end: boolean }

const SAAS_NAV: NavItem[] = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/tenants', label: 'Tenants', end: false },
  { to: '/plans', label: 'Plans', end: false },
  { to: '/subscriptions', label: 'Subscriptions', end: false },
  { to: '/invoices', label: 'Invoices', end: false },
]

const STORE_NAV: NavItem[] = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/products', label: 'Products', end: false },
  { to: '/categories', label: 'Categories', end: false },
  { to: '/employees', label: 'Employees', end: false },
  { to: '/sales', label: 'Sales', end: false },
  { to: '/purchases', label: 'Purchases', end: false },
  { to: '/inventory', label: 'Inventory', end: false },
  { to: '/subscription', label: 'Subscription', end: false },
]

const CASHIER_NAV: NavItem[] = STORE_NAV.filter(
  (item) => !['/employees', '/purchases', '/subscription'].includes(item.to),
)

export function Layout() {
  const { role, signOut } = useAuth()
  const saas = isSaasAdmin(role)
  const nav = saas ? SAAS_NAV : role === 'cashier' ? CASHIER_NAV : STORE_NAV
  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">P</span>
          <div>
            <div className="brand-title">POS.Go</div>
            <div className="brand-sub">{saas ? 'SaaS Admin' : 'Store Console'}</div>
          </div>
        </div>
        <nav className="nav">
          {nav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              className={({ isActive }) => (isActive ? 'nav-link active' : 'nav-link')}
            >
              {item.label}
            </NavLink>
          ))}
        </nav>
        <div className="sidebar-footer">
          <button className="nav-link signout" onClick={signOut}>
            Sign out
          </button>
        </div>
      </aside>
      <main className="main">
        <Outlet />
      </main>
    </div>
  )
}