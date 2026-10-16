import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { accountQueries } from '@/entities/account'
import { cashPositionQueries, periodSummaryQueries } from '@/entities/analytics-report'
import { creditCardQueries } from '@/entities/credit-card'
import { financialEventQueries } from '@/entities/financial-event'
import { APP_UTC_OFFSET } from '@/shared/config'
import { queryHarness } from '@/test/query-harness'
import { startsWith } from '@/test/fetch-query'
import { useRepayCreditCard } from './use-repay-credit-card'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const repayment = {
  fromAccountId: 'salary',
  cardAccountId: 'card',
  amountMinor: 900_000,
  occurredAt: `2026-09-25T10:00:00${APP_UTC_OFFSET}`,
  key: 'k',
}

beforeEach(() => post.mockReset())

it('pays the card as a transfer and refreshes balances, operations and the card', async () => {
  post.mockResolvedValue({ data: { id: 'e' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useRepayCreditCard(), { wrapper })
  await result.current.mutateAsync(repayment)
  expect(post).toHaveBeenCalledWith('/api/v1/financial-events/transfers', {
    params: { header: { 'Idempotency-Key': 'k' } },
    body: {
      amountMinor: 900_000,
      description: 'Погашение кредитной карты',
      occurredAt: `2026-09-25T10:00:00${APP_UTC_OFFSET}`,
      sourceAccountId: 'salary',
      destinationAccountId: 'card',
    },
  })
  expect(invalidated()).toEqual([
    accountQueries.all(),
    financialEventQueries.all(),
    cashPositionQueries.all(),
    periodSummaryQueries.all(),
  ])
  expect(startsWith(creditCardQueries.all(), accountQueries.all())).toBe(true)
})

it('returns the API error and keeps the cache', async () => {
  post.mockResolvedValue({ error: { message: 'Недостаточно данных' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useRepayCreditCard(), { wrapper })
  expect((await result.current.mutateAsync(repayment)).error).toEqual({
    message: 'Недостаточно данных',
  })
  expect(invalidated()).toEqual([])
})
