import { infiniteQueryOptions, queryOptions } from '@tanstack/react-query'

import { apiClient } from '@/shared/api'
import type { FinancialEvent } from '@/shared/api'

/** URL filter of the operations list; empty strings mean "not set". */
export type FinancialEventFilter = Record<
  'q' | 'category' | 'account' | 'kind' | 'from' | 'to' | 'voided',
  string
>

export const financialEventQueries = {
  all: () => ['financial-events'] as const,
  list: () =>
    queryOptions({
      queryKey: [...financialEventQueries.all(), 'list'] as const,
      queryFn: async () => {
        const { data, error } = await apiClient.GET('/api/v1/financial-events')
        if (!data) throw new Error(error?.message ?? 'Не удалось загрузить финансовые события')
        return data.items
      },
    }),
  page: (filter: FinancialEventFilter) =>
    infiniteQueryOptions({
      queryKey: [...financialEventQueries.all(), 'page', filter] as const,
      initialPageParam: '',
      queryFn: async ({ pageParam }) => {
        const { data, error } = await apiClient.GET('/api/v1/financial-events', {
          params: {
            query: {
              limit: 50,
              cursor: pageParam || undefined,
              q: filter.q || undefined,
              categoryId: filter.category || undefined,
              accountId: filter.account || undefined,
              kind: (filter.kind || undefined) as FinancialEvent['kind'] | undefined,
              from: filter.from || undefined,
              to: filter.to || undefined,
              voided: filter.voided === 'true' || undefined,
            },
          },
        })
        if (!data) throw new Error(error?.message ?? 'Не удалось загрузить операции.')
        return data
      },
      getNextPageParam: (last) => last.nextCursor ?? undefined,
    }),
  history: (event: Pick<FinancialEvent, 'id' | 'revision'>) =>
    queryOptions({
      queryKey: [...financialEventQueries.all(), 'history', event.id, event.revision] as const,
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/v1/financial-events/{eventId}/history', {
          params: { path: { eventId: event.id } },
        })
        if (!data) throw new Error('История недоступна.')
        return data.items
      },
    }),
  importOptions: () =>
    queryOptions({
      queryKey: [...financialEventQueries.all(), 'import-options'] as const,
      queryFn: async () => (await apiClient.GET('/api/v1/financial-events')).data,
    }),
  subscriptionPicker: () =>
    queryOptions({
      queryKey: [...financialEventQueries.all(), 'subscription-picker'] as const,
      queryFn: async () => {
        const { data, error } = await apiClient.GET('/api/v1/financial-events')
        if (error) throw new Error(error.message)
        return data?.items ?? []
      },
    }),
}
