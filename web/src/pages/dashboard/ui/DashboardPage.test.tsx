import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { DashboardPage } from './DashboardPage'

const { post, get } = vi.hoisted(() => ({ post: vi.fn(), get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post, GET: get } }))
function wrap(element: React.ReactNode) {
  return render(
    <MemoryRouter>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        {element}
      </QueryClientProvider>
    </MemoryRouter>,
  )
}
beforeEach(() => {
  post.mockReset()
  get.mockReset()
})
describe('SPEC-0020', () => {
  it('offers the first account without showing invented balances', async () => {
    get.mockImplementation(async (path: string) => ({
      data:
        path === '/api/v1/analytics/cash-position'
          ? { asOf: '2026-10-06', through: '2026-10-31', currencies: [] }
          : { items: [] },
    }))
    wrap(<DashboardPage />)
    expect(await screen.findByRole('heading', { name: 'Начните с первого счёта' })).toBeVisible()
    expect(
      screen.queryByRole('heading', { name: 'После известных платежей' }),
    ).not.toBeInTheDocument()
    const user = userEvent.setup()
    await user.click(screen.getAllByRole('button', { name: 'Добавить первый счёт' })[0])
    expect(screen.getByRole('heading', { name: 'Новый счёт' })).toBeVisible()
    expect(post).not.toHaveBeenCalled()
  })
})
