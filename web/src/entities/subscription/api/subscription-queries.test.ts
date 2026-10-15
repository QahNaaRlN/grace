import { beforeEach, expect, it, vi } from 'vitest'

import { fetchQuery, startsWith } from '@/test/fetch-query'
import { subscriptionExpenseQueries, subscriptionQueries } from './subscription-queries'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))

beforeEach(() => get.mockReset())

it('keeps expense history outside the list, so refreshing the list leaves it alone', () => {
  expect(subscriptionQueries.list().queryKey).toEqual(subscriptionQueries.all())
  expect(startsWith(subscriptionExpenseQueries.list('s').queryKey, subscriptionQueries.all())).toBe(
    false,
  )
})

it('shows the server message when the list cannot be read', async () => {
  get.mockResolvedValueOnce({ error: { message: 'Подписки недоступны' } })
  await expect(fetchQuery(subscriptionQueries.list())).rejects.toThrow('Подписки недоступны')
  get.mockResolvedValueOnce({ data: undefined })
  await expect(fetchQuery(subscriptionQueries.list())).resolves.toEqual([])
})
