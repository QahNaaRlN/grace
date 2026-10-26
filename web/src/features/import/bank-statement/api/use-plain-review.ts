import { useMutation, useQueryClient } from '@tanstack/react-query'
import { bankStatementQueries } from '@/entities/bank-statement'
import { importBatchQueries } from '@/entities/import-batch'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export function usePlainReview(id: string) {
  const cache = useQueryClient()
  return useMutation({
    mutationFn: async (body: components['schemas']['PlainStatementReviewInput']) => {
      const { data, error } = await apiClient.POST(
        '/api/imports/statements/{statementId}/plain-reviews',
        {
          params: { path: { statementId: id }, header: { 'Idempotency-Key': crypto.randomUUID() } },
          body,
        },
      )
      if (!data) throw new Error(error?.message ?? 'Сверка не сохранена. Обновите документ.')
      return data
    },
    onSuccess: async () => {
      await Promise.all([
        cache.invalidateQueries({ queryKey: bankStatementQueries.all() }),
        cache.invalidateQueries({ queryKey: importBatchQueries.all() }),
      ])
    },
  })
}
