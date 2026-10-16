import { createBrowserRouter, Outlet, type RouteObject } from 'react-router-dom'

import { AppLayout } from '@/widgets/app-shell'
import { AuthBoundary } from './AuthBoundary'
import { lazyPage } from './lazyPage'

/**
 * Страница загружается своим чанком при первом переходе на неё (SPEC-0052). При ошибке
 * загрузки открывается адрес, на который шла навигация, а при первой загрузке — текущий.
 */
const page = (load: Parameters<typeof lazyPage>[0]) =>
  lazyPage(load, () => appRouter.state.navigation.location ?? appRouter.state.location)

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
      {
        path: '/',
        lazy: page(async () => ({ Component: (await import('@/pages/dashboard')).DashboardPage })),
      },
      {
        path: '/operations',
        lazy: page(async () => ({
          Component: (await import('@/pages/operations')).OperationsPage,
        })),
      },
      {
        path: '/payments',
        lazy: page(async () => ({ Component: (await import('@/pages/payments')).PaymentsPage })),
      },
      {
        path: '/accounts',
        lazy: page(async () => ({ Component: (await import('@/pages/accounts')).AccountsPage })),
      },
      {
        path: '/categories',
        lazy: page(async () => ({
          Component: (await import('@/pages/categories')).CategoriesPage,
        })),
      },
      {
        path: '/analytics',
        lazy: page(async () => ({ Component: (await import('@/pages/analytics')).AnalyticsPage })),
      },
      {
        path: '/health',
        lazy: page(async () => ({ Component: (await import('@/pages/health')).HealthPage })),
      },
      {
        path: '/profile',
        lazy: page(async () => ({ Component: (await import('@/pages/profile')).ProfilePage })),
      },
      {
        path: '/subscriptions',
        lazy: page(async () => ({
          Component: (await import('@/pages/subscriptions')).SubscriptionsPage,
        })),
      },
      {
        path: '/assistant',
        lazy: page(async () => ({ Component: (await import('@/pages/assistant')).AssistantPage })),
      },
      {
        path: '/imports',
        lazy: page(async () => ({ Component: (await import('@/pages/imports')).ImportsPage })),
      },
    ],
  },
  {
    path: '/login',
    lazy: page(async () => ({ Component: (await import('@/pages/login')).LoginPage })),
  },
  {
    path: '/register',
    lazy: page(async () => {
      const { PasswordEntryPage } = await import('@/pages/password-entry')
      return { element: <PasswordEntryPage view="register" /> }
    }),
  },
  {
    path: '/forgot-password',
    lazy: page(async () => {
      const { PasswordEntryPage } = await import('@/pages/password-entry')
      return { element: <PasswordEntryPage view="forgot" /> }
    }),
  },
  {
    path: '/reset-password',
    lazy: page(async () => {
      const { PasswordEntryPage } = await import('@/pages/password-entry')
      return { element: <PasswordEntryPage view="reset" /> }
    }),
  },
]

export const appRouter = createBrowserRouter(appRoutes)
