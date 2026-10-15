import { beforeEach, expect, it, vi } from 'vitest'

import { fetchQuery } from '@/test/fetch-query'
import { receiptQueries } from './receipt-queries'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))

beforeEach(() => get.mockReset())

it('reads receipts under the receipts root', async () => {
  expect(receiptQueries.list().queryKey).toEqual(receiptQueries.all())
  get.mockResolvedValueOnce({ data: { items: [{ id: 'r' }] } })
  await expect(fetchQuery(receiptQueries.list())).resolves.toEqual([{ id: 'r' }])
  get.mockResolvedValueOnce({ error: { message: 'down' } })
  await expect(fetchQuery(receiptQueries.list())).rejects.toThrow()
})
