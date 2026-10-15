import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { accountQueries } from '@/entities/account'
import { cashPositionQueries, periodSummaryQueries } from '@/entities/analytics-report'
import { financialEventQueries } from '@/entities/financial-event'
import { obligationQueries } from '@/entities/obligation'
import { queryHarness } from '@/test/query-harness'
import { useCreateFinancialEvent, type NewFinancialEvent } from './use-create-financial-event'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const transfer: NewFinancialEvent = {
  kind: 'transfer',
  key: 'key',
  common: {
    amountMinor: 500,
    merchant: '',
    description: 'Перевод',
    occurredAt: '2026-10-15T09:00:00Z',
  },
  accountId: 'from',
  categoryId: null,
  originalEventId: '',
  destinationAccountId: 'to',
}

beforeEach(() => post.mockReset())

it('posts a transfer with its key and refreshes balances, events, analytics and obligations', async () => {
  post.mockResolvedValue({ data: { id: 'event' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCreateFinancialEvent(), { wrapper })
  await result.current.mutateAsync(transfer)
  expect(post).toHaveBeenCalledWith('/api/v1/financial-events/transfers', {
    params: { header: { 'Idempotency-Key': 'key' } },
    body: {
      amountMinor: 500,
      description: 'Перевод',
      occurredAt: '2026-10-15T09:00:00Z',
      sourceAccountId: 'from',
      destinationAccountId: 'to',
    },
  })
  expect(invalidated()).toEqual([
    accountQueries.all(),
    financialEventQueries.all(),
    cashPositionQueries.all(),
    periodSummaryQueries.all(),
    obligationQueries.all(),
  ])
})

it('returns the API error and keeps the cache', async () => {
  post.mockResolvedValue({ error: { message: 'Счёт закрыт' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCreateFinancialEvent(), { wrapper })
  const outcome = await result.current.mutateAsync({ ...transfer, kind: 'purchase' })
  expect(post.mock.calls[0][0]).toBe('/api/v1/financial-events/purchases')
  expect(outcome.error).toEqual({ message: 'Счёт закрыт' })
  expect(invalidated()).toEqual([])
})
