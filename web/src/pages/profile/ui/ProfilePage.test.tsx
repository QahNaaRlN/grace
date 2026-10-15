import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ProfilePage } from './ProfilePage'

const { post, get } = vi.hoisted(() => ({ post: vi.fn(), get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post, GET: get } }))
beforeEach(() => {
  post.mockReset()
  get.mockReset()
})
describe('SPEC-0036 header account action', () => {
  it('does not offer the first account on the profile page', async () => {
    get.mockImplementation(async (path: string) => ({
      data:
        path === '/api/auth/profile'
          ? {
              id: 'u',
              displayName: 'Number One',
              firstName: '',
              lastName: '',
              email: null,
              avatarUrl: null,
            }
          : path === '/api/v1/analytics/cash-position'
            ? { asOf: '2026-10-06', through: '2026-10-31', currencies: [] }
            : { items: [] },
    }))
    render(
      <MemoryRouter initialEntries={['/profile']}>
        <QueryClientProvider
          client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
        >
          <ProfilePage />
        </QueryClientProvider>
      </MemoryRouter>,
    )
    expect(await screen.findByRole('heading', { name: 'Личные данные' })).toBeVisible()
    expect(screen.queryByRole('button', { name: /Добавить .*счёт/ })).not.toBeInTheDocument()
  })
})
