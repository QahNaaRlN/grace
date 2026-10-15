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
