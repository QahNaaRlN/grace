import { createBrowserRouter } from 'react-router-dom'

import { HomePage } from '@/pages/home'
import { AssistantPage } from '@/pages/assistant'

/**
 * Маршрутизатор приложения с корневой страницей разработки.
 */
export const appRouter = createBrowserRouter([
  {
    path: '/',
    element: <HomePage />,
  },
  {
    path: '/assistant',
    element: <AssistantPage />,
  },
])
