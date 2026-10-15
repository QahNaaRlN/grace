import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { accountQueries } from '@/entities/account'
import {
  cashPositionQueries,
  periodSummaryQueries,
  statisticsQueries,
} from '@/entities/analytics-report'
import { financialEventQueries } from '@/entities/financial-event'
import { queryHarness } from '@/test/query-harness'
import { useConfirmReconciliation, type ReconciliationPreview } from './reconcile-account'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const preview = {
  asOf: '2026-10-15T09:00:00Z',
  statedMinor: 1000,
  previewHash: 'hash',
} as ReconciliationPreview

beforeEach(() => post.mockReset())

it('confirms with the preview hash and the key, then refreshes everything a balance feeds', async () => {
  post.mockResolvedValue({ data: { reconciliation: { outcome: 'matched' } } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useConfirmReconciliation('a', 'key'), { wrapper })
  await result.current.mutateAsync(preview)
  expect(post.mock.calls[0][1]).toEqual({
    params: { path: { accountId: 'a' }, header: { 'Idempotency-Key': 'key' } },
    body: { asOf: preview.asOf, statedMinor: 1000, previewHash: 'hash' },
  })
  expect(invalidated()).toEqual([
    accountQueries.all(),
    financialEventQueries.all(),
    cashPositionQueries.all(),
    statisticsQueries.all(),
    periodSummaryQueries.all(),
  ])
})

it('keeps the cache when the preview is stale', async () => {
  post.mockResolvedValue({ error: { message: 'Данные изменились' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useConfirmReconciliation('a', 'key'), { wrapper })
  await result.current.mutateAsync(preview)
  expect(invalidated()).toEqual([])
})
