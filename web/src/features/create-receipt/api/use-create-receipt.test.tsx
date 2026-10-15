import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { receiptQueries } from '@/entities/receipt'
import { queryHarness } from '@/test/query-harness'
import { useCreateReceipt, type NewReceipt } from './use-create-receipt'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const body: NewReceipt = {
  purchaseEventId: 'p',
  seller: 'Магазин',
  issuedAt: '2026-10-15T09:00:00Z',
  fiscalDocumentNumber: '1',
  items: [{ name: 'Хлеб', quantityMilli: 1000, unitPriceMinor: 100, lineTotalMinor: 100 }],
}

beforeEach(() => post.mockReset())

it('refreshes receipts after the lines are saved', async () => {
  post.mockResolvedValue({ data: { id: 'r' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCreateReceipt(), { wrapper })
  await result.current.mutateAsync(body)
  expect(invalidated()).toEqual([receiptQueries.all()])
})

it('keeps the cache when the receipt is rejected', async () => {
  post.mockResolvedValue({ error: { message: 'Сумма не совпала' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCreateReceipt(), { wrapper })
  await result.current.mutateAsync(body)
  expect(invalidated()).toEqual([])
})
