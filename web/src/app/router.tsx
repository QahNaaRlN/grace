import { createBrowserRouter } from 'react-router-dom'

import { HomePage } from '@/pages/home'
import { AssistantPage } from '@/pages/assistant'
import { LoginPage } from '@/pages/login/LoginPage'
import { ImportsPage } from '@/pages/imports/ImportsPage'
import { AuthBoundary } from './AuthBoundary'

/**
 * Маршрутизатор приложения с корневой страницей разработки.
 */
export const appRouter = createBrowserRouter([
  {
    path: '/',
    element: <AuthBoundary><HomePage /></AuthBoundary>,
  },
  {
    path: '/assistant',
    element: <AuthBoundary><AssistantPage /></AuthBoundary>,
  },
  { path: '/login', element: <LoginPage /> },
  { path: '/imports', element: <AuthBoundary><ImportsPage /></AuthBoundary> },
])
