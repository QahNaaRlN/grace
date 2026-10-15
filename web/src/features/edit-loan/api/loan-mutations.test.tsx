import { renderHook, waitFor } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { cashPositionQueries } from '@/entities/analytics-report'
import { loanQueries, obligationQueries } from '@/entities/obligation'
import { queryHarness } from '@/test/query-harness'
import { useApplyLoanSchedule, useCopyLoanSchedule, useSaveLoanDetails } from './loan-mutations'

const { post, patch } = vi.hoisted(() => ({ post: vi.fn(), patch: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post, PATCH: patch } }))

const draft = { id: 'd', revision: 2, appliedAt: null, baseHash: 'h', payments: [] }
const details = loanQueries.details('o').queryKey

beforeEach(() => {
  post.mockReset()
  patch.mockReset()
})

it('applies a draft and refreshes the loan, obligations and the cash position', async () => {
  post.mockResolvedValue({ data: {} })
  const applied = vi.fn()
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useApplyLoanSchedule('o', { onSuccess: applied }), {
    wrapper,
  })
  await result.current.mutateAsync(draft)
  expect(post.mock.calls[0][1].body).toEqual({ revision: 2, confirm: true })
  expect(applied).toHaveBeenCalledOnce()
  await waitFor(() =>
    expect(invalidated()).toEqual([details, obligationQueries.all(), cashPositionQueries.all()]),
  )
})

it('copies the schedule and refreshes only the loan', async () => {
  post.mockResolvedValue({ data: draft })
  const copied = vi.fn()
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useCopyLoanSchedule('o', { onSuccess: copied }), {
    wrapper,
  })
  await result.current.mutateAsync()
  expect(copied).toHaveBeenCalledWith(draft)
  await waitFor(() => expect(invalidated()).toEqual([details]))
})

it('reports invalid input before writing and keeps the cache', async () => {
  const failed = vi.fn()
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useSaveLoanDetails('o', { onError: failed }), { wrapper })
  await expect(
    result.current.mutateAsync(() => {
      throw new Error('Проверьте сумму')
    }),
  ).rejects.toThrow('Проверьте сумму')
  expect(patch).not.toHaveBeenCalled()
  expect(failed).toHaveBeenCalled()
  expect(invalidated()).toEqual([])
})
