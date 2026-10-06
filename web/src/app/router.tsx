import { createBrowserRouter } from 'react-router-dom'

import { HomePage } from '@/pages/home'
import { AssistantPage } from '@/pages/assistant'
import { LoginPage } from '@/pages/login/LoginPage'
import { AuthBoundary } from './AuthBoundary'

/**
 * Маршрутизатор приложения с корневой страницей разработки.
 */
export const appRouter = createBrowserRouter([
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
  ...[
    'operations',
    'payments',
    'accounts',
    'analytics',
    'health',
    'profile',
  ].map((page) => ({
    path: `/${page}`,
    element: (
      <AuthBoundary>
        <HomePage />
      </AuthBoundary>
    ),
  })),
  { path: '/login', element: <LoginPage /> },
])
