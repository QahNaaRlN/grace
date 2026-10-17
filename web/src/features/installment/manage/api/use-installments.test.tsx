import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'
import { accountQueries } from '@/entities/account'
import {
  cashPositionQueries,
  periodSummaryQueries,
  statisticsQueries,
} from '@/entities/analytics-report'
import { financialEventQueries } from '@/entities/financial-event'
import { obligationQueries, paymentReminderQueries } from '@/entities/obligation'
import { installmentQueries } from '@/entities/installment'
import { queryHarness } from '@/test/query-harness'
import {
  useAllocateInstallment,
  useCreateInstallment,
  type InstallmentInput,
} from './use-installments'

const { post, put } = vi.hoisted(() => ({ post: vi.fn(), put: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post, PUT: put } }))
beforeEach(() => {
  post.mockReset()
  put.mockReset()
})
const body: InstallmentInput = {
  name: 'Synthetic BNPL',
  mode: 'standalone',
  provider: 'Synthetic',
  currency: 'RUB',
  openedOn: '2026-10-01',
  principalMinor: 1200000,
  purchase: { categoryId: 'category' },
  payments: [{ dueOn: '2026-10-01', amountMinor: 1200000 }],
}
it('sends one confirmed purchase command and refreshes every dependent screen after success', async () => {
  post.mockResolvedValue({ data: { id: 'plan' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(useCreateInstallment, { wrapper })
  await result.current.mutateAsync({ body, key: 'stable-key' })
  expect(post).toHaveBeenCalledWith('/api/v1/installment-plans', {
    params: { header: { 'Idempotency-Key': 'stable-key' } },
    body,
  })
  expect(invalidated()).toEqual([
    accountQueries.all(),
    obligationQueries.all(),
    installmentQueries.all(),
    financialEventQueries.all(),
    cashPositionQueries.all(),
    periodSummaryQueries.all(),
    statisticsQueries.all(),
    paymentReminderQueries.all(),
  ])
})
it('keeps caches intact when a stale allocation is rejected', async () => {
  put.mockResolvedValue({ error: { message: 'Обновите данные' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useAllocateInstallment('event'), { wrapper })
  await expect(
    result.current.mutateAsync({
      body: {
        revision: 1,
        reason: 'Bank confirmed',
        items: [{ paymentId: 'row', amountMinor: 300000 }],
      },
      key: 'retry-key',
    }),
  ).rejects.toThrow('Обновите данные')
  expect(invalidated()).toEqual([])
})
