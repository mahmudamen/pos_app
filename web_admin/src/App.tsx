import { BrowserRouter, Route, Routes } from 'react-router-dom'
import { AuthProvider, useAuth } from './auth/AuthContext'
import { isSaasAdmin } from './auth/roles'
import { Layout } from './components/Layout'
import { DashboardPage } from './pages/DashboardPage'
import { InvoicesPage } from './pages/InvoicesPage'
import { LoginPage } from './pages/LoginPage'
import { PlansPage } from './pages/PlansPage'
import { SubscriptionsPage } from './pages/SubscriptionsPage'
import { TenantsPage } from './pages/TenantsPage'
import { TenantDetailPage } from './pages/TenantDetailPage'
import { StoreCategoriesPage } from './pages/store/StoreCategoriesPage'
import { StoreDashboardPage } from './pages/store/StoreDashboardPage'
import { StoreEmployeesPage } from './pages/store/StoreEmployeesPage'
import { StoreInventoryPage } from './pages/store/StoreInventoryPage'
import { StoreProductsPage } from './pages/store/StoreProductsPage'
import { StorePurchasesPage } from './pages/store/StorePurchasesPage'
import { StoreSalesPage } from './pages/store/StoreSalesPage'
import { StoreSubscriptionPage } from './pages/store/StoreSubscriptionPage'

function Guard({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuth()
  return isAuthenticated ? <>{children}</> : <LoginPage />
}

function AppRoutes() {
  const { role } = useAuth()
  const saas = isSaasAdmin(role)

  if (saas) {
    return (
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
    )
  }

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<StoreDashboardPage />} />
        <Route path="products" element={<StoreProductsPage />} />
        <Route path="categories" element={<StoreCategoriesPage />} />
        {role === 'cashier' ? null : (
          <>
            <Route path="employees" element={<StoreEmployeesPage />} />
            <Route path="purchases" element={<StorePurchasesPage />} />
            <Route path="subscription" element={<StoreSubscriptionPage />} />
          </>
        )}
        <Route path="sales" element={<StoreSalesPage />} />
        <Route path="inventory" element={<StoreInventoryPage />} />
      </Route>
    </Routes>
  )
}

export default function App() {
  const basename = import.meta.env.BASE_URL
  return (
    <BrowserRouter basename={basename}>
      <AuthProvider>
        <Guard>
          <AppRoutes />
        </Guard>
      </AuthProvider>
    </BrowserRouter>
  )
}