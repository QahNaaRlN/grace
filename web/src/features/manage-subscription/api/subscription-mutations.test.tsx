import { renderHook, waitFor } from '@testing-library/react'
import { beforeEach, expect, it, vi } from 'vitest'

import { subscriptionQueries } from '@/entities/subscription'
import { queryHarness } from '@/test/query-harness'
import {
  useLinkSubscriptionExpense,
  useSaveSubscription,
  useToggleSubscriptionPause,
  type Subscription,
} from './subscription-mutations'

const { post, patch } = vi.hoisted(() => ({ post: vi.fn(), patch: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { POST: post, PATCH: patch } }))

const subscription = { id: 's', name: 'Связь', paused: false, revision: 1 } as Subscription

beforeEach(() => {
  post.mockReset()
  patch.mockReset()
})

it('refreshes subscriptions after a pause, a link and a save', async () => {
  patch.mockResolvedValue({ data: {} })
  post.mockResolvedValue({ data: {} })
  const linked = vi.fn()
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(
    () => ({
      pause: useToggleSubscriptionPause(),
      link: useLinkSubscriptionExpense({ onSuccess: linked }),
      save: useSaveSubscription(undefined, 'key'),
    }),
    { wrapper },
  )
  await result.current.pause.mutateAsync(subscription)
  expect(patch.mock.calls[0][1].body).toMatchObject({ paused: true, revision: 1 })
  await result.current.link.mutateAsync({ subscription, financialEventId: 'e' })
  expect(linked).toHaveBeenCalledOnce()
  await result.current.save.mutateAsync(() => ({}) as never)
  expect(post.mock.calls[1][1].params.header).toEqual({ 'Idempotency-Key': 'key' })
  await waitFor(() =>
    expect(invalidated()).toEqual([
      subscriptionQueries.all(),
      subscriptionQueries.all(),
      subscriptionQueries.all(),
    ]),
  )
})

it('keeps the cache when the form is invalid', async () => {
  const failed = vi.fn()
  const { wrapper, invalidated } = queryHarness()
  const { result } = renderHook(() => useSaveSubscription(undefined, 'key', { onError: failed }), {
    wrapper,
  })
  await expect(
    result.current.mutateAsync(() => {
      throw new Error('Выберите категорию расхода.')
    }),
  ).rejects.toThrow('Выберите категорию')
  expect(post).not.toHaveBeenCalled()
  expect(invalidated()).toEqual([])
})
