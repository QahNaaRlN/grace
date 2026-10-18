import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { importBatchQueries } from '@/entities/import-batch'
import { queryHarness } from '@/test/query-harness'
import { useSaveImportDecision } from './save-import-decision'

const { patch } = vi.hoisted(() => ({ patch: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { PATCH: patch } }))

const input = {
  recordId: 'r',
  revision: 4,
  decision: { action: 'exclude' as const, reason: 'Дубль', resolutions: [] },
}

beforeEach(() => patch.mockReset())

it('saves the decision against the draft revision and reloads the drafts', async () => {
  patch.mockResolvedValue({ data: {} })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useSaveImportDecision('b'), { wrapper })
  await result.current.mutateAsync(input)
  expect(patch.mock.calls[0][1].body).toEqual({ expectedRevision: 4, decision: input.decision })
  expect(invalidated()).toEqual([importBatchQueries.all()])
})

it('explains a revision conflict and keeps the cache', async () => {
  patch.mockResolvedValue({ error: { code: 'revision_conflict' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useSaveImportDecision('b'), { wrapper })
  await expect(result.current.mutateAsync(input)).rejects.toThrow('Черновик изменился')
  expect(invalidated()).toEqual([])
})
