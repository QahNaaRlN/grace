import { renderHook } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { accountQueries, type PaymentCard } from '@/entities/account'
import { queryHarness } from '@/test/query-harness'
import { useSaveCard, useToggleCard, useUpdateAccount } from './account-mutations'

const { post, patch } = vi.hoisted(() => ({ post: vi.fn(), patch: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post, PATCH: patch } }))

const card = {
  id: 'c',
  name: 'Карта',
  bank: 'Банк',
  last4: '0012',
} as PaymentCard
const fields = { name: 'Карта', bank: 'Банк', last4: '0012' }

beforeEach(() => {
  post.mockReset()
  patch.mockReset()
})

it('refreshes accounts after a rename, a new card and a card toggle', async () => {
  patch.mockResolvedValue({ data: {} })
  post.mockResolvedValue({ data: {} })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(
    () => ({
      update: useUpdateAccount('a'),
      add: useSaveCard('a', null),
      toggle: useToggleCard('a'),
    }),
    { wrapper },
  )
  await result.current.update.mutateAsync({ revision: 1, name: 'Новое' })
  await result.current.add.mutateAsync(fields)
  await result.current.toggle.mutateAsync(card)
  expect(post.mock.calls[0][0]).toBe('/api/v1/accounts/{accountId}/payment-instruments')
  expect(patch.mock.calls[1][1].body).toEqual({ disabled: true })
  expect(invalidated()).toEqual([accountQueries.all(), accountQueries.all(), accountQueries.all()])
})

it('changes an existing card in place and keeps the cache on a rejection', async () => {
  patch.mockResolvedValue({ error: { message: 'Конфликт' } })
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useSaveCard('a', card), { wrapper })
  await result.current.mutateAsync(fields)
  expect(patch.mock.calls[0][1].params.path).toEqual({ accountId: 'a', cardId: 'c' })
  expect(invalidated()).toEqual([])
})
