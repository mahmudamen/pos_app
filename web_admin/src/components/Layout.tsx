import { NavLink, Outlet } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'

const NAV = [
  { to: '/', label: 'Dashboard', end: true },
  { to: '/tenants', label: 'Tenants', end: false },
  { to: '/plans', label: 'Plans', end: false },
  { to: '/subscriptions', label: 'Subscriptions', end: false },
  { to: '/invoices', label: 'Invoices', end: false },
]

export function Layout() {
  const { signOut } = useAuth()
  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">P</span>
          <div>
            <div className="brand-title">POS.Go</div>
            <div className="brand-sub">SaaS Admin</div>
          </div>
        </div>
        <nav className="nav">
          {NAV.map((item) => (
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