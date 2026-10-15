import { beforeEach, expect, it, vi } from 'vitest'

import { fetchQuery, startsWith } from '@/test/fetch-query'
import { categoryQueries } from './category-queries'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))

beforeEach(() => get.mockReset())

it('keeps the list and the import options under categories', () => {
  expect(categoryQueries.list().queryKey).toEqual(categoryQueries.all())
  expect(startsWith(categoryQueries.importOptions().queryKey, categoryQueries.all())).toBe(true)
  expect(categoryQueries.importOptions().queryKey).not.toEqual(categoryQueries.list().queryKey)
})

it('loads the categories and reports a failed read', async () => {
  get.mockResolvedValueOnce({ data: { items: [{ id: 'c' }] } })
  await expect(fetchQuery(categoryQueries.list())).resolves.toEqual([{ id: 'c' }])
  get.mockResolvedValueOnce({ error: { message: 'down' } })
  await expect(fetchQuery(categoryQueries.list())).rejects.toThrow()
})
