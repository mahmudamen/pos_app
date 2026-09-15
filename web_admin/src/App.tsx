import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { AuthProvider, useAuth } from './auth/AuthContext'
import { Layout } from './components/Layout'
import { DashboardPage } from './pages/DashboardPage'
import { InvoicesPage } from './pages/InvoicesPage'
import { LoginPage } from './pages/LoginPage'
import { PlansPage } from './pages/PlansPage'
import { SubscriptionsPage } from './pages/SubscriptionsPage'
import { TenantsPage } from './pages/TenantsPage'
import { TenantDetailPage } from './pages/TenantDetailPage'

function Guard({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth()
  return isAuthenticated ? <>{children}</> : <LoginPage />
}

export default function App() {
  const basename = import.meta.env.BASE_URL
  return (
    <BrowserRouter basename={basename}>
      <AuthProvider>
        <Guard>
          <Routes>
            <Route element={<Layout />}>
              <Route index element={<DashboardPage />} />
              <Route path="tenants" element={<TenantsPage />} />
              <Route path="tenants/:id" element={<TenantDetailPage />} />
              <Route path="plans" element={<PlansPage />} />
              <Route path="subscriptions" element={<SubscriptionsPage />} />
              <Route path="invoices" element={<InvoicesPage />} />
            </Route>
          </Routes>
        </Guard>
      </AuthProvider>
    </BrowserRouter>
  )
}