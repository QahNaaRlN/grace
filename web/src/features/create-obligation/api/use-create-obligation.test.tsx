import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { cashPositionQueries } from '@/entities/analytics-report'
import { obligationQueries } from '@/entities/obligation'
import { queryHarness } from '@/test/query-harness'
import { useCreateObligation, type NewObligation } from './use-create-obligation'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const body: NewObligation = {
  name: 'Ноутбук',
  kind: 'installment',
  currency: 'RUB',
  totalMinor: 1000,
  payments: [{ dueOn: '2026-11-01', amountMinor: 1000 }],
}

beforeEach(() => post.mockReset())

it('refreshes obligations and the cash position after a new obligation', async () => {
  post.mockResolvedValue({ data: { id: 'o' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCreateObligation(), { wrapper })
  await result.current.mutateAsync(body)
  expect(post.mock.calls[0][1].params.header['Idempotency-Key']).toBeTruthy()
  expect(invalidated()).toEqual([obligationQueries.all(), cashPositionQueries.all()])
})

it('keeps the cache when the obligation is rejected', async () => {
  post.mockResolvedValue({ error: { message: 'Нет' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCreateObligation(), { wrapper })
  await result.current.mutateAsync(body)
  expect(invalidated()).toEqual([])
})
