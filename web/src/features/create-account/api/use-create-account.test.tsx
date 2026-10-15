import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { accountQueries } from '@/entities/account'
import { cashPositionQueries } from '@/entities/analytics-report'
import { queryHarness } from '@/test/query-harness'
import { useCreateAccount, type NewAccount } from './use-create-account'

const { post } = vi.hoisted(() => ({ post: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post } }))

const body: NewAccount = {
  name: 'Карта',
  type: 'debit',
  currency: 'RUB',
  openingBalanceMinor: 100,
  balanceReconciled: true,
}

beforeEach(() => post.mockReset())

it('refreshes accounts and the cash position after an account is created', async () => {
  post.mockResolvedValue({ data: { id: 'a' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCreateAccount(), { wrapper })
  await result.current.mutateAsync(body)
  expect(post).toHaveBeenCalledWith('/api/v1/accounts', { body })
  expect(invalidated()).toEqual([accountQueries.all(), cashPositionQueries.all()])
})

it('keeps the cache when the server rejects the account', async () => {
  post.mockResolvedValue({ error: { message: 'Нет' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCreateAccount(), { wrapper })
  expect((await result.current.mutateAsync(body)).error).toEqual({ message: 'Нет' })
  expect(invalidated()).toEqual([])
})
