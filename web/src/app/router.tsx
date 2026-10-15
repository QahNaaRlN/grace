import { createBrowserRouter, Outlet, type RouteObject } from 'react-router-dom'

import { AccountsPage } from '@/pages/accounts'
import { AnalyticsPage } from '@/pages/analytics'
import { AssistantPage } from '@/pages/assistant'
import { CategoriesPage } from '@/pages/categories'
import { DashboardPage } from '@/pages/dashboard'
import { HealthPage } from '@/pages/health'
import { ImportsPage } from '@/pages/imports'
import { LoginPage } from '@/pages/login'
import { OperationsPage } from '@/pages/operations'
import { PasswordEntryPage } from '@/pages/password-entry'
import { PaymentsPage } from '@/pages/payments'
import { ProfilePage } from '@/pages/profile'
import { SubscriptionsPage } from '@/pages/subscriptions'
import { AppLayout } from '@/widgets/app-shell'
import { AuthBoundary } from './AuthBoundary'

/**
 * Маршруты приложения: публичные страницы входа и layout route с проверкой доступа и
 * оболочкой, внутри которого каждая страница — свой срез `pages/*`.
 */
export const appRoutes: RouteObject[] = [
  {
    element: (
      <AuthBoundary>
        <AppLayout>
          <Outlet />
        </AppLayout>
      </AuthBoundary>
    ),
    children: [
      { path: '/', element: <DashboardPage /> },
      { path: '/operations', element: <OperationsPage /> },
      { path: '/payments', element: <PaymentsPage /> },
      { path: '/accounts', element: <AccountsPage /> },
      { path: '/categories', element: <CategoriesPage /> },
      { path: '/analytics', element: <AnalyticsPage /> },
      { path: '/health', element: <HealthPage /> },
      { path: '/profile', element: <ProfilePage /> },
      { path: '/subscriptions', element: <SubscriptionsPage /> },
      { path: '/assistant', element: <AssistantPage /> },
      { path: '/imports', element: <ImportsPage /> },
    ],
  },
  { path: '/login', element: <LoginPage /> },
  { path: '/register', element: <PasswordEntryPage view="register" /> },
  { path: '/forgot-password', element: <PasswordEntryPage view="forgot" /> },
  { path: '/reset-password', element: <PasswordEntryPage view="reset" /> },
]

export const appRouter = createBrowserRouter(appRoutes)
