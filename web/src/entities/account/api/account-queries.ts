import { queryOptions } from '@tanstack/react-query'

import { apiClient } from '@/shared/api'
import type { Account } from '@/shared/api'

export const accountQueries = {
  all: () => ['accounts'] as const,
  list: () =>
    queryOptions({
      queryKey: accountQueries.all(),
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/v1/accounts')
        if (!data) throw new Error('Не удалось загрузить счета')
        return data.items
      },
    }),
  // The import review reads the raw response under its own key, so it never
  // overwrites the list cache with a different shape.
  importOptions: () =>
    queryOptions({
      queryKey: [...accountQueries.all(), 'import-options'] as const,
      queryFn: async () => (await apiClient.GET('/api/v1/accounts')).data,
    }),
  reconciliations: (account: Account) =>
    queryOptions({
      queryKey: [
        ...accountQueries.all(),
        account.id,
        'reconciliations',
        account.revision,
        account.lastReconciliation?.createdAt,
      ] as const,
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/v1/accounts/{accountId}/reconciliations', {
          params: { path: { accountId: account.id } },
        })
        if (!data) throw new Error('История сверок недоступна.')
        return data.items
      },
    }),
}
