import { beforeEach, expect, it, vi } from 'vitest'

import type { Account } from '@/shared/api'
import { fetchQuery, startsWith } from '@/test/fetch-query'
import { accountQueries } from './account-queries'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))

const account = { id: 'a', revision: 2, lastReconciliation: null } as unknown as Account

beforeEach(() => get.mockReset())

it('keeps every account key under accounts so one invalidation refreshes them all', () => {
  for (const key of [
    accountQueries.list().queryKey,
    accountQueries.importOptions().queryKey,
    accountQueries.reconciliations(account).queryKey,
  ])
    expect(startsWith(key, accountQueries.all())).toBe(true)
})

it('loads the list items and turns a failed read into an error', async () => {
  get.mockResolvedValueOnce({ data: { items: [account] } })
  await expect(fetchQuery(accountQueries.list())).resolves.toEqual([account])
  expect(get).toHaveBeenCalledWith('/api/v1/accounts')
  get.mockResolvedValueOnce({ error: { message: 'down' } })
  await expect(fetchQuery(accountQueries.list())).rejects.toThrow('Не удалось загрузить счета')
})
