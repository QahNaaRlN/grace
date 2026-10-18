import { queryOptions } from '@tanstack/react-query'

import { apiClient } from '@/shared/api'

export const bankStatementQueries = {
  all: () => ['bank-statements'] as const,
  detail: (statementId: string) =>
    queryOptions({
      queryKey: [...bankStatementQueries.all(), statementId],
      queryFn: async () => {
        const { data, error } = await apiClient.GET('/api/imports/statements/{statementId}', {
          params: { path: { statementId } },
        })
        if (!data) throw new Error(error?.message ?? 'Выписка недоступна.')
        return data
      },
    }),
}

export const statementAttemptQueries = {
  list: (statementId: string) =>
    queryOptions({
      queryKey: [...bankStatementQueries.all(), statementId, 'attempts'],
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/imports/statements/{statementId}/extractions', {
          params: { path: { statementId } },
        })
        if (!data) throw new Error('История распознавания недоступна.')
        return data.items
      },
    }),
}
