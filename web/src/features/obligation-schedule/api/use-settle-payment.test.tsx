import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { cashPositionQueries } from '@/entities/analytics-report'
import { loanQueries, obligationQueries } from '@/entities/obligation'
import { queryHarness } from '@/test/query-harness'
import { useSettlePayment } from './use-settle-payment'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const settlement = { obligationId: 'o', paymentId: 'p', financialEventId: 'e' }

beforeEach(() => post.mockReset())

it('refreshes obligations, the cash position and loan debt after a payment is linked', async () => {
  post.mockResolvedValue({ data: {} })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useSettlePayment(), { wrapper })
  await result.current.mutateAsync(settlement)
  expect(post.mock.calls[0][1].params.path).toEqual({ obligationId: 'o', paymentId: 'p' })
  expect(post.mock.calls[0][1].body).toEqual({ financialEventId: 'e' })
  expect(invalidated()).toEqual([
    obligationQueries.all(),
    cashPositionQueries.all(),
    loanQueries.all(),
  ])
})

it('SPEC-0053: asks to book the amount above the row as a prepayment', async () => {
  post.mockResolvedValue({ data: {} })
  const { wrapper } = queryHarness()
  const { result } = renderHook(() => useSettlePayment(), { wrapper })
  await result.current.mutateAsync({ ...settlement, prepayRemainder: true })
  expect(post.mock.calls[0][1].body).toEqual({ financialEventId: 'e', prepayRemainder: true })
})

it('keeps the cache when the link is rejected', async () => {
  post.mockResolvedValue({ error: { message: 'Уже связан' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useSettlePayment(), { wrapper })
  await result.current.mutateAsync(settlement)
  expect(invalidated()).toEqual([])
})
