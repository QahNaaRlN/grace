import { queryOptions } from '@tanstack/react-query'

import { apiClient } from '@/shared/api'

/**
 * SPEC-0051. The card lives under the accounts root: any write that refreshes
 * accounts (an operation, a transfer, a reconciliation) refreshes the card status too.
 */
export const creditCardQueries = {
  all: () => ['accounts', 'credit-card'] as const,
  detail: (accountId: string) =>
    queryOptions({
      queryKey: [...creditCardQueries.all(), accountId] as const,
      queryFn: async () => {
        const { data, error } = await apiClient.GET('/api/v1/accounts/{accountId}/credit-card', {
          params: { path: { accountId } },
        })
        if (!data) throw new Error(error?.message ?? 'Не удалось загрузить условия карты.')
        return data
      },
    }),
}
