import { createBrowserRouter } from 'react-router-dom'

import { HomePage } from '@/pages/home'
import { AssistantPage } from '@/pages/assistant'
import { LoginPage } from '@/pages/login/LoginPage'
import { ImportsPage } from '@/pages/imports/ImportsPage'
import { AuthBoundary } from './AuthBoundary'
import { PasswordEntry } from '@/pages/login/PasswordEntry'
import { SubscriptionsPage } from '@/pages/subscriptions/SubscriptionsPage'

/**
 * Маршрутизатор приложения с корневой страницей разработки.
 */
export const appRouter = createBrowserRouter([
  {
    path: '/subscriptions',
    element: (
      <AuthBoundary>
        <SubscriptionsPage />
      </AuthBoundary>
    ),
  },
  {
    path: '/',
    element: (
      <AuthBoundary>
        <HomePage />
      </AuthBoundary>
    ),
  },
  {
    path: '/assistant',
    element: (
      <AuthBoundary>
        <AssistantPage />
      </AuthBoundary>
    ),
  },
  ...['operations', 'payments', 'accounts', 'categories', 'analytics', 'health', 'profile'].map(
    (page) => ({
      path: `/${page}`,
      element: (
        <AuthBoundary>
          <HomePage />
        </AuthBoundary>
      ),
    }),
  ),
  { path: '/login', element: <LoginPage /> },
  { path: '/register', element: <PasswordEntry view="register" /> },
  { path: '/forgot-password', element: <PasswordEntry view="forgot" /> },
  { path: '/reset-password', element: <PasswordEntry view="reset" /> },
  {
    path: '/imports',
    element: (
      <AuthBoundary>
        <ImportsPage />
      </AuthBoundary>
    ),
  },
])
