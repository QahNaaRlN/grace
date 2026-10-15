import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { statisticsQueries } from '@/entities/analytics-report'
import { categoryQueries } from '@/entities/category'
import { financialEventQueries } from '@/entities/financial-event'
import type { Category } from '@/shared/api'
import { queryHarness } from '@/test/query-harness'
import { useDeleteCategory, useSaveCategory, useToggleCategoryHidden } from './category-mutations'

const client = vi.hoisted(() => ({ POST: vi.fn(), PATCH: vi.fn(), PUT: vi.fn(), DELETE: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: client }))

const category = {
  id: 'c',
  name: 'Спорт',
  kind: 'expense',
  icon: null,
  hidden: false,
  operationCount: 2,
  createdAt: '2026-10-01T00:00:00Z',
} as Category
const refreshed = [categoryQueries.all(), financialEventQueries.all(), statisticsQueries.all()]

beforeEach(() => Object.values(client).forEach((fn) => fn.mockReset()))

it('refreshes categories, operations and statistics after a save and a hide', async () => {
  client.POST.mockResolvedValue({ data: { id: 'n' } })
  client.PUT.mockResolvedValue({ data: {} })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(
    () => ({ save: useSaveCategory(null), hide: useToggleCategoryHidden() }),
    { wrapper },
  )
  await result.current.save.mutateAsync({ name: 'Спорт', icon: null })
  await result.current.hide.mutateAsync(category)
  expect(client.POST).toHaveBeenCalledWith('/api/v1/categories', { body: { name: 'Спорт' } })
  expect(invalidated()).toEqual([...refreshed, ...refreshed])
})

it('deletes with the previewed count and the move target', async () => {
  client.POST.mockResolvedValue({ data: { operations: 2 } })
  client.DELETE.mockResolvedValue({ data: {} })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useDeleteCategory(category), { wrapper })
  await result.current.mutateAsync('target')
  expect(client.DELETE.mock.calls[0][1].params.query).toEqual({
    expectedOperations: 2,
    moveTo: 'target',
  })
  expect(invalidated()).toEqual(refreshed)
})

it('stops a deletion when the count changed and keeps the cache', async () => {
  client.POST.mockResolvedValue({ data: { operations: 3 } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useDeleteCategory(category), { wrapper })
  await expect(result.current.mutateAsync('target')).rejects.toThrow(
    'Количество операций изменилось',
  )
  expect(client.DELETE).not.toHaveBeenCalled()
  expect(invalidated()).toEqual([])
})
