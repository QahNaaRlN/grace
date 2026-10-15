import { queryOptions } from '@tanstack/react-query'

import { apiClient } from '@/shared/api'

export const subscriptionQueries = {
  all: () => ['subscriptions'] as const,
  list: () =>
    queryOptions({
      queryKey: subscriptionQueries.all(),
      queryFn: async () => {
        const { data, error } = await apiClient.GET('/api/v1/subscriptions')
        if (error) throw new Error(error.message)
        return data?.items ?? []
      },
    }),
}

// Kept outside ['subscriptions'] so refreshing the list does not refetch an open history.
export const subscriptionExpenseQueries = {
  all: () => ['subscription-expenses'] as const,
  list: (subscriptionId: string | undefined) =>
    queryOptions({
      queryKey: [...subscriptionExpenseQueries.all(), subscriptionId] as const,
      queryFn: async () => {
        const { data, error } = await apiClient.GET(
          '/api/v1/subscriptions/{subscriptionId}/expenses',
          { params: { path: { subscriptionId: subscriptionId! } } },
        )
        if (error) throw new Error(error.message)
        return data?.items ?? []
      },
    }),
}
