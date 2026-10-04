import { createBrowserRouter } from 'react-router-dom'

import { HomePage } from '@/pages/home'

/**
 * Маршрутизатор приложения с корневой страницей разработки.
 */
export const appRouter = createBrowserRouter([
  {
    path: '/',
    element: <HomePage />,
  },
])
