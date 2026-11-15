import { Link, Navigate, Route, Routes } from 'react-router-dom'
import { AuthProvider, ProtectedRoute, RoleGate } from './auth/AuthProvider'
import { AppShell } from './components/layout/AppShell'
import { LoginPage } from './features/auth/LoginPage'
import { DashboardPage } from './features/dashboard/DashboardPage'
import { ProductListPage } from './features/products/ProductListPage'
import { ProductFormPage } from './features/products/ProductFormPage'
import { ProductDetailPage } from './features/products/ProductDetailPage'
import { InventoryPage } from './features/inventory/InventoryPage'
import { ProductInventoryPage } from './features/inventory/ProductInventoryPage'
import { ReceivingPage } from './features/receiving/ReceivingPage'
import { RequestListPage } from './features/requests/RequestListPage'
import { RequestCreatePage } from './features/requests/RequestCreatePage'
import { RequestDetailPage } from './features/requests/RequestDetailPage'
import { ReturnListPage } from './features/returns/ReturnListPage'
import { ReturnCreatePage } from './features/returns/ReturnCreatePage'
import { ReturnDetailPage } from './features/returns/ReturnDetailPage'
import { AdjustmentPage } from './features/adjustments/AdjustmentPage'
import { ReportsPage } from './features/reports/ReportsPage'
import { UserListPage } from './features/users/UserListPage'
import { UserFormPage } from './features/users/UserFormPage'
import { NotificationsPage } from './features/notifications/NotificationsPage'
import { AuditHistoryPage } from './features/audit/AuditHistoryPage'
import { Button } from './components/ui'
import { useT } from './i18n/I18nProvider'

function ForbiddenPage() {
  const { t } = useT()
  return (
    <div className="flex flex-col items-center justify-center py-24 text-center">
      <p className="text-5xl font-bold text-slate-300">403</p>
      <h1 className="mt-2 text-xl font-semibold">{t('errors.accessDenied')}</h1>
      <p className="mt-1 text-sm text-slate-500">{t('errors.roleNotAllowed')}</p>
      <Link className="mt-4" to="/dashboard"><Button variant="outline">{t('errors.backToDashboard')}</Button></Link>
    </div>
  )
}

function NotFoundPage() {
  const { t } = useT()
  return (
    <div className="flex flex-col items-center justify-center py-24 text-center">
      <p className="text-5xl font-bold text-slate-300">404</p>
      <h1 className="mt-2 text-xl font-semibold">{t('errors.pageNotFound')}</h1>
      <Link className="mt-4" to="/dashboard"><Button variant="outline">{t('errors.backToDashboard')}</Button></Link>
    </div>
  )
}

export default function App() {
  return (
    <AuthProvider>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          element={
            <ProtectedRoute>
              <AppShell />
            </ProtectedRoute>
          }
        >
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/products" element={<ProductListPage />} />
          <Route path="/products/new" element={<RoleGate roles={['WAREHOUSE_ADMIN']}><ProductFormPage /></RoleGate>} />
          <Route path="/products/:id" element={<ProductDetailPage />} />
          <Route path="/products/:id/edit" element={<RoleGate roles={['WAREHOUSE_ADMIN', 'SUPERVISOR']}><ProductFormPage /></RoleGate>} />
          <Route path="/inventory" element={<InventoryPage />} />
          <Route path="/inventory/:productId" element={<ProductInventoryPage />} />
          <Route path="/receiving" element={<RoleGate roles={['WAREHOUSE_ADMIN']}><ReceivingPage /></RoleGate>} />
          <Route path="/requests" element={<RequestListPage />} />
          <Route path="/requests/new" element={<RoleGate roles={['DEPARTMENT_USER']}><RequestCreatePage /></RoleGate>} />
          <Route path="/requests/:id" element={<RequestDetailPage />} />
          <Route path="/returns" element={<ReturnListPage />} />
          <Route path="/returns/new" element={<RoleGate roles={['DEPARTMENT_USER']}><ReturnCreatePage /></RoleGate>} />
          <Route path="/returns/:id" element={<ReturnDetailPage />} />
          <Route path="/adjustments" element={<RoleGate roles={['WAREHOUSE_ADMIN']}><AdjustmentPage /></RoleGate>} />
          <Route path="/reports" element={<ReportsPage />} />
          <Route path="/audit" element={<RoleGate roles={['WAREHOUSE_ADMIN', 'SUPERVISOR']}><AuditHistoryPage /></RoleGate>} />
          <Route path="/users" element={<RoleGate roles={['WAREHOUSE_ADMIN']}><UserListPage /></RoleGate>} />
          <Route path="/users/new" element={<RoleGate roles={['WAREHOUSE_ADMIN']}><UserFormPage /></RoleGate>} />
          <Route path="/users/:id/edit" element={<RoleGate roles={['WAREHOUSE_ADMIN']}><UserFormPage /></RoleGate>} />
          <Route path="/notifications" element={<NotificationsPage />} />
          <Route path="/403" element={<ForbiddenPage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Route>
      </Routes>
    </AuthProvider>
  )
}
