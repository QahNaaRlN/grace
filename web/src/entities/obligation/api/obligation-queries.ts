import { queryOptions } from '@tanstack/react-query'

import { apiClient } from '@/shared/api'

export const obligationQueries = {
  all: () => ['obligations'] as const,
  list: (asOf: string) =>
    queryOptions({
      queryKey: [...obligationQueries.all(), asOf] as const,
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/v1/obligations', {
          params: { query: { asOf } },
        })
        if (!data) throw new Error('Не удалось загрузить обязательства')
        return data.items
      },
    }),
}

export const loanQueries = {
  all: () => ['loan-details'] as const,
  details: (obligationId: string) =>
    queryOptions({
      queryKey: [...loanQueries.all(), obligationId] as const,
      queryFn: async () => {
        const { data, error } = await apiClient.GET('/api/v1/loans/{obligationId}', {
          params: { path: { obligationId } },
        })
        if (error) throw new Error(error.message)
        return data!
      },
    }),
}

// SPEC-0053: how a debt payment splits for its loan; null when it is not linked to one.
export const loanAllocationQueries = {
  all: () => ['loan-allocation'] as const,
  event: (eventId: string) =>
    queryOptions({
      queryKey: [...loanAllocationQueries.all(), eventId] as const,
      queryFn: async () => {
        const { data, error, response } = await apiClient.GET(
          '/api/v1/financial-events/{eventId}/loan-allocation',
          { params: { path: { eventId } } },
        )
        if (response.status === 404) return null
        if (error) throw new Error(error.message)
        return data ?? null
      },
    }),
}

export const paymentReminderQueries = {
  all: () => ['payment-reminders'] as const,
  list: () =>
    queryOptions({
      queryKey: paymentReminderQueries.all(),
      queryFn: async () => {
        const { data, error } = await apiClient.GET('/api/v1/payment-reminders')
        if (error) throw new Error(error.message)
        return data?.items ?? []
      },
    }),
}
