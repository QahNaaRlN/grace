import { useMutation, useQueryClient } from '@tanstack/react-query'

import { subscriptionQueries } from '@/entities/subscription'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export type Subscription = components['schemas']['Subscription']
export type SubscriptionInput = components['schemas']['SubscriptionInput']
type Callbacks = { onSuccess?: () => void; onError?: (error: Error) => void }

function useRefreshSubscriptions() {
  const queryClient = useQueryClient()
  return () => queryClient.invalidateQueries({ queryKey: subscriptionQueries.all() })
}

/** Pauses an active subscription or resumes a paused one; the list refreshes before it settles. */
export function useToggleSubscriptionPause(callbacks: Callbacks = {}) {
  const refresh = useRefreshSubscriptions()
  return useMutation({
    mutationFn: async (s: Subscription) => {
      const { error } = await apiClient.PATCH('/api/v1/subscriptions/{subscriptionId}', {
        params: { path: { subscriptionId: s.id } },
        body: {
          name: s.name,
          amountMinor: s.amountMinor,
          currency: s.currency,
          period: s.period,
          startsOn: s.startsOn,
          endsOn: s.endsOn,
          accountId: s.accountId,
          paymentInstrumentId: s.paymentInstrumentId,
          categoryId: s.categoryId,
          reminders: s.reminders,
          paused: !s.paused,
          revision: s.revision,
        },
      })
      if (error) throw new Error(error.message)
    },
    onSuccess: refresh,
    onError: callbacks.onError,
  })
}

/** Links an existing purchase to a subscription; it does not create another charge. */
export function useLinkSubscriptionExpense(callbacks: Callbacks = {}) {
  const refresh = useRefreshSubscriptions()
  return useMutation({
    mutationFn: async ({
      subscription,
      financialEventId,
    }: {
      subscription: Subscription | null
      financialEventId: string
    }) => {
      if (!subscription) return
      const { error } = await apiClient.POST('/api/v1/subscriptions/{subscriptionId}/expenses', {
        params: { path: { subscriptionId: subscription.id } },
        body: { financialEventId },
      })
      if (error) throw new Error(error.message)
    },
    onSuccess: () => {
      callbacks.onSuccess?.()
      void refresh()
    },
    onError: callbacks.onError,
  })
}

/**
 * Creates a subscription (`initial` is undefined) or updates one. `prepare` validates the form
 * inside the mutation, so input errors reach onError as well.
 */
export function useSaveSubscription(
  initial: Subscription | undefined,
  creationKey: string,
  callbacks: Callbacks = {},
) {
  const refresh = useRefreshSubscriptions()
  return useMutation({
    mutationFn: async (prepare: () => SubscriptionInput) => {
      const body = prepare()
      const result = initial
        ? await apiClient.PATCH('/api/v1/subscriptions/{subscriptionId}', {
            params: { path: { subscriptionId: initial.id } },
            body,
          })
        : await apiClient.POST('/api/v1/subscriptions', {
            params: { header: { 'Idempotency-Key': creationKey } },
            body,
          })
      if (result.error) throw new Error(result.error.message)
    },
    onSuccess: () => {
      callbacks.onSuccess?.()
      void refresh()
    },
    onError: callbacks.onError,
  })
}
