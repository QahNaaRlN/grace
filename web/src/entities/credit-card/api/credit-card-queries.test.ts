import { beforeEach, expect, it, vi } from 'vitest'

import { fetchQuery, startsWith } from '@/test/fetch-query'
import { creditCardQueries } from './credit-card-queries'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))

beforeEach(() => get.mockReset())

it('keeps the card under the accounts root so ledger writes refresh it', () => {
  expect(startsWith(creditCardQueries.detail('a').queryKey, creditCardQueries.all())).toBe(true)
  expect(startsWith(creditCardQueries.all(), ['accounts'])).toBe(true)
})

it('reads one card and shows the server error', async () => {
  get.mockResolvedValueOnce({ data: { accountId: 'a' } })
  await expect(fetchQuery(creditCardQueries.detail('a'))).resolves.toEqual({ accountId: 'a' })
  expect(get).toHaveBeenCalledWith('/api/v1/accounts/{accountId}/credit-card', {
    params: { path: { accountId: 'a' } },
  })
  get.mockResolvedValueOnce({
    error: { message: 'Условия карты доступны только для кредитного счёта.' },
  })
  await expect(fetchQuery(creditCardQueries.detail('a'))).rejects.toThrow('только для кредитного')
})
