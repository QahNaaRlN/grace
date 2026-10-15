import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { queryHarness } from '@/test/query-harness'
import { startsWith } from '@/test/fetch-query'
import { importBatchQueries } from './import-batch-queries'
import { nextUnresolvedRecord } from './next-unresolved'
import { useImportRefresh } from './use-import-refresh'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))

beforeEach(() => get.mockReset())

it('keeps batches, records and month reports under imports', () => {
  for (const key of [
    importBatchQueries.batch('b').queryKey,
    importBatchQueries.record('b', 'r').queryKey,
    importBatchQueries.receipt('b', 'c').queryKey,
    importBatchQueries.month('b', '2026-09').queryKey,
  ])
    expect(startsWith(key, importBatchQueries.all())).toBe(true)
  expect(
    startsWith(importBatchQueries.month('b', '2026-09').queryKey, importBatchQueries.months()),
  ).toBe(true)
})

it('retries a failed read by refreshing all drafts or only the month reports', async () => {
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useImportRefresh(), { wrapper })
  await result.current.all()
  await result.current.months()
  expect(invalidated()).toEqual([importBatchQueries.all(), importBatchQueries.months()])
})

it('finds the next unresolved row or reports that none is left', async () => {
  get.mockResolvedValueOnce({ data: { record: { id: 'r2' } } })
  await expect(nextUnresolvedRecord('b', 'r1')).resolves.toEqual({ id: 'r2' })
  expect(get.mock.calls[0][1].params).toEqual({ path: { batchId: 'b' }, query: { after: 'r1' } })
  get.mockResolvedValueOnce({ data: { record: null } })
  await expect(nextUnresolvedRecord('b')).resolves.toBeNull()
  get.mockResolvedValueOnce({ error: {} })
  await expect(nextUnresolvedRecord('b')).rejects.toThrow('Не удалось найти следующую строку.')
})
