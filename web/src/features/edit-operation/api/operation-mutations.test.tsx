import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { accountQueries } from '@/entities/account'
import {
  cashPositionQueries,
  periodSummaryQueries,
  statisticsQueries,
} from '@/entities/analytics-report'
import { categoryQueries } from '@/entities/category'
import { financialEventQueries } from '@/entities/financial-event'
import type { FinancialEvent } from '@/shared/api'
import { queryHarness } from '@/test/query-harness'
import { useCorrectOperation, useVoidOperation } from './operation-mutations'

const { post, patch } = vi.hoisted(() => ({ post: vi.fn(), patch: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post, PATCH: patch } }))

const event = { id: 'e', revision: 3 } as FinancialEvent
const refreshed = [
  financialEventQueries.all(),
  accountQueries.all(),
  cashPositionQueries.all(),
  statisticsQueries.all(),
  periodSummaryQueries.all(),
  categoryQueries.all(),
]

beforeEach(() => {
  post.mockReset()
  patch.mockReset()
})

it('corrects with the key and refreshes events, balances, analytics and categories', async () => {
  patch.mockResolvedValue({ data: { id: 'e', revision: 4 } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCorrectOperation(event, 'key'), { wrapper })
  await result.current.mutateAsync({ revision: 3, merchant: 'Новый' })
  expect(patch.mock.calls[0][1].params.header).toEqual({ 'Idempotency-Key': 'key' })
  expect(invalidated()).toEqual(refreshed)
})

it('voids with the revision and the reason', async () => {
  post.mockResolvedValue({ data: { id: 'e', voidedAt: '2026-10-15T09:00:00Z' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useVoidOperation(event, 'key'), { wrapper })
  await result.current.mutateAsync('Дубль')
  expect(post.mock.calls[0][1].body).toEqual({ revision: 3, reason: 'Дубль' })
  expect(invalidated()).toEqual(refreshed)
})

it('keeps the cache when a linked operation blocks the change', async () => {
  patch.mockResolvedValue({ error: { message: 'Сначала отвяжите чек' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCorrectOperation(event, 'key'), { wrapper })
  await result.current.mutateAsync({ revision: 3, amountMinor: 10 })
  expect(invalidated()).toEqual([])
})
