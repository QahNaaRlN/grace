import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { creditCardQueries } from '@/entities/credit-card'
import { queryHarness } from '@/test/query-harness'
import { useCardReminders, useRecordCardSnapshot, useSaveCardTerms } from './use-credit-card'

const { post, patch, put } = vi.hoisted(() => ({ post: vi.fn(), patch: vi.fn(), put: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post, PATCH: patch, PUT: put } }))

const terms = {
  effectiveFrom: '2026-09-01',
  graceModel: 'statement' as const,
  statementDay: 5,
  paymentDays: 25,
  minPercentIsCap: true,
  minIncludesInterest: false,
  note: '',
  revision: 0,
}

beforeEach(() => {
  post.mockReset()
  patch.mockReset()
  put.mockReset()
})

it('creates a version with its key and refreshes only the card', async () => {
  post.mockResolvedValue({ data: { accountId: 'a' } })
  const { wrapper, invalidated, client } = queryHarness()
  const { result } = renderHook(() => useSaveCardTerms('a', 'key'), { wrapper })
  await result.current.mutateAsync({ body: terms })
  expect(post).toHaveBeenCalledWith('/api/v1/accounts/{accountId}/credit-card/terms', {
    params: { path: { accountId: 'a' }, header: { 'Idempotency-Key': 'key' } },
    body: terms,
  })
  expect(invalidated()).toEqual([creditCardQueries.all()])
  expect(client.getQueryData(creditCardQueries.detail('a').queryKey)).toEqual({ accountId: 'a' })
})

it('edits a version by id', async () => {
  patch.mockResolvedValue({ data: { accountId: 'a' } })
  const { wrapper } = queryHarness()
  const { result } = renderHook(() => useSaveCardTerms('a', 'key'), { wrapper })
  await result.current.mutateAsync({ termsId: 't', body: { ...terms, revision: 2 } })
  expect(patch.mock.calls[0][1].params.path).toEqual({ accountId: 'a', termsId: 't' })
})

it('keeps the cache when a snapshot or setting is refused', async () => {
  post.mockResolvedValue({ error: { message: 'Проверьте даты' } })
  put.mockResolvedValue({ error: { message: 'Запись недоступна' } })
  const { wrapper, invalidated } = queryHarness()
  const snapshot = renderHook(() => useRecordCardSnapshot('a', 'k'), { wrapper })
  await snapshot.result.current.mutateAsync({ takenOn: '2026-09-25', debtMinor: 1, note: '' })
  const reminders = renderHook(() => useCardReminders('a'), { wrapper })
  await reminders.result.current.mutateAsync(true)
  expect(put).toHaveBeenCalledWith('/api/v1/accounts/{accountId}/credit-card/settings', {
    params: { path: { accountId: 'a' } },
    body: { reminders: true },
  })
  expect(invalidated()).toEqual([])
})
