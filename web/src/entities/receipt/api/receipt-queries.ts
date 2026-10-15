import { queryOptions } from '@tanstack/react-query'

import { apiClient } from '@/shared/api'

export const receiptQueries = {
  all: () => ['receipts'] as const,
  list: () =>
    queryOptions({
      queryKey: receiptQueries.all(),
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/v1/receipts')
        if (!data) throw new Error('Не удалось загрузить чеки')
        return data.items
      },
    }),
}
