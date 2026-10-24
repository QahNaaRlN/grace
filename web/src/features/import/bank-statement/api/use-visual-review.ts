import { useMutation, useQueryClient } from '@tanstack/react-query'
import { bankStatementQueries } from '@/entities/bank-statement'
import { importBatchQueries } from '@/entities/import-batch'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export function useVisualReview(
  id: string,
  onSaved: (s: components['schemas']['BankStatement']) => void,
) {
  const cache = useQueryClient()
  return useMutation({
    mutationFn: async (body: components['schemas']['VisualReviewInput']) => {
      const result = await apiClient.POST('/api/imports/statements/{statementId}/visual-reviews', {
        params: { path: { statementId: id }, header: { 'Idempotency-Key': crypto.randomUUID() } },
        body,
      })
      if (!result.data) throw new Error(result.error?.message ?? 'Проверка не сохранена.')
      return result.data
    },
    onSuccess: async (next) => {
      onSaved(next)
      await Promise.all([
        cache.invalidateQueries({ queryKey: bankStatementQueries.all() }),
        cache.invalidateQueries({ queryKey: importBatchQueries.all() }),
      ])
    },
  })
}
