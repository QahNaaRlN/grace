import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { accountQueries } from '@/entities/account'
import { cashPositionQueries, periodSummaryQueries } from '@/entities/analytics-report'
import { financialEventQueries } from '@/entities/financial-event'
import { importBatchQueries } from '@/entities/import-batch'
import { queryHarness } from '@/test/query-harness'
import { useConfirmImport, type ImportPreview } from './import-confirmation'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const preview = { id: 'p', hash: 'h' } as ImportPreview

beforeEach(() => post.mockReset())

function setup() {
  const order: string[] = []
  const harness = queryHarness()
  vi.mocked(harness.client.invalidateQueries).mockImplementation(async (filters) => {
    order.push(JSON.stringify(filters?.queryKey))
  })
  const { result } = renderHook(
    () =>
      useConfirmImport('b', {
        onConfirmed: () => order.push('confirmed'),
        onStale: () => order.push('stale'),
      }),
    { wrapper: harness.wrapper },
  )
  return { order, result }
}

it('shows the receipt first, then refreshes drafts, balances, events and analytics', async () => {
  post.mockResolvedValue({ data: { id: 'receipt', outcomes: [] } })
  const { order, result } = setup()
  await result.current.mutateAsync({ preview, key: 'key' })
  expect(post.mock.calls[0][1].body).toEqual({ previewId: 'p', previewHash: 'h' })
  expect(order).toEqual([
    'confirmed',
    ...[
      importBatchQueries.all(),
      accountQueries.all(),
      financialEventQueries.all(),
      periodSummaryQueries.all(),
      cashPositionQueries.all(),
    ].map((key) => JSON.stringify(key)),
  ])
})

it('drops a stale preview and reloads only the drafts', async () => {
  post.mockResolvedValue({ error: { code: 'preview_expired' } })
  const { order, result } = setup()
  expect((await result.current.mutateAsync({ preview, key: 'key' })).stale).toBe(true)
  expect(order).toEqual(['stale', JSON.stringify(importBatchQueries.all())])
})

it('keeps the cache on another failure so the same key can be retried', async () => {
  post.mockResolvedValue({ error: { code: 'internal' } })
  const { order, result } = setup()
  expect(await result.current.mutateAsync({ preview, key: 'key' })).toEqual({
    data: undefined,
    stale: false,
  })
  expect(order).toEqual([])
})
