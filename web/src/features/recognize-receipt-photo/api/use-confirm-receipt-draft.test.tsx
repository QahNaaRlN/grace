import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { accountQueries } from '@/entities/account'
import {
  cashPositionQueries,
  periodSummaryQueries,
  statisticsQueries,
} from '@/entities/analytics-report'
import { financialEventQueries } from '@/entities/financial-event'
import { receiptQueries } from '@/entities/receipt'
import { queryHarness } from '@/test/query-harness'
import type { ReceiptDraftPreview } from './receipt-drafts'
import { useConfirmReceiptDraft } from './use-confirm-receipt-draft'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const preview = { id: 'preview', hash: 'hash' } as ReceiptDraftPreview
const confirmation = { draftId: 'draft', preview, key: 'key' }

beforeEach(() => post.mockReset())

it('confirms the reviewed preview with its key and refreshes ledger, receipts and analytics', async () => {
  post.mockResolvedValue({ data: { id: 'draft', status: 'confirmed' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useConfirmReceiptDraft(), { wrapper })
  await expect(result.current.mutateAsync(confirmation)).resolves.toEqual({
    id: 'draft',
    status: 'confirmed',
  })
  expect(post).toHaveBeenCalledWith('/api/receipt-drafts/{draftId}/confirmations', {
    params: { path: { draftId: 'draft' }, header: { 'Idempotency-Key': 'key' } },
    body: { previewId: 'preview', hash: 'hash' },
  })
  expect(invalidated()).toEqual([
    accountQueries.all(),
    financialEventQueries.all(),
    receiptQueries.all(),
    cashPositionQueries.all(),
    periodSummaryQueries.all(),
    statisticsQueries.all(),
  ])
})

it('rejects a failed confirmation and keeps the cache', async () => {
  post.mockResolvedValue({ error: { message: 'Книга изменилась' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useConfirmReceiptDraft(), { wrapper })
  await expect(result.current.mutateAsync(confirmation)).rejects.toThrow(
    'Подтверждение не получено.',
  )
  expect(invalidated()).toEqual([])
})
