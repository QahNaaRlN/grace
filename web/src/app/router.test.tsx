/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider, type RouteObject } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'

import { apiClient } from '@/shared/api'

import { appRoutes } from './router'

vi.mock('@/shared/api/client', () => ({ apiClient: { GET: vi.fn(), POST: vi.fn() } }))

// SPEC-0048 AC8: the URL set of main e623775.
const protectedPaths = [
  '/',
  '/operations',
  '/payments',
  '/accounts',
  '/categories',
  '/analytics',
  '/health',
  '/profile',
  '/subscriptions',
  '/assistant',
  '/imports',
]
const publicPaths = ['/login', '/register', '/forgot-password', '/reset-password']

function paths(routes: RouteObject[]): string[] {
  return routes.flatMap((route) => [
    ...(route.path ? [route.path] : []),
    ...paths(route.children ?? []),
  ])
}

describe('SPEC-0048 router', () => {
  it('keeps the URL set of e623775', () => {
    expect(paths(appRoutes).sort()).toEqual([...protectedPaths, ...publicPaths].sort())
  })

  it('puts every protected page under one layout route and keeps sign-in pages outside', () => {
    const layouts = appRoutes.filter((route) => route.children)
    expect(layouts).toHaveLength(1)
    expect(layouts[0].path).toBeUndefined()
    expect(paths(layouts[0].children ?? []).sort()).toEqual([...protectedPaths].sort())
    expect(appRoutes.filter((route) => !route.children).map((route) => route.path)).toEqual(
      publicPaths,
    )
  })

  // SPEC-0052 AC2: every page is a route-level chunk.
  it('loads every page through route lazy, not a static import', () => {
    const source = readFileSync(join(process.cwd(), 'src/app/router.tsx'), 'utf8')
    expect(source).not.toMatch(/from '@\/pages\//)
    const leaves = (routes: RouteObject[]): RouteObject[] =>
      routes.flatMap((route) => (route.children ? leaves(route.children) : [route]))
    for (const route of leaves(appRoutes)) {
      expect(route.lazy, route.path).toBeTypeOf('function')
      expect(route.element ?? route.Component, route.path).toBeUndefined()
    }
  })

  it('renders a protected page inside the shell after the session check', async () => {
    vi.mocked(apiClient.GET).mockImplementation((async (path: string) =>
      path === '/api/auth/me'
        ? {
            data: { user: { id: 'user', name: 'Alice', role: 'member' }, csrfToken: 'csrf' },
            response: new Response(null, { status: 200 }),
          }
        : { data: { items: [] }, response: new Response(null, { status: 200 }) }) as never)
    const router = createMemoryRouter(appRoutes, { initialEntries: ['/categories'] })
    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <RouterProvider router={router} />
      </QueryClientProvider>,
    )
    expect(await screen.findByRole('heading', { name: 'Категории', level: 1 })).toBeVisible()
    expect(screen.getByRole('navigation', { name: 'Навигация' })).toBeInTheDocument()
  })

  // SPEC-0052 AC4: the shared password-entry chunk keeps the view of each URL.
  it('renders a sign-in page with its view', async () => {
    const router = createMemoryRouter(appRoutes, { initialEntries: ['/register'] })
    render(
      <QueryClientProvider client={new QueryClient()}>
        <RouterProvider router={router} />
      </QueryClientProvider>,
    )
    expect(await screen.findByText('Регистрация в Grace')).toBeVisible()
  })
})
