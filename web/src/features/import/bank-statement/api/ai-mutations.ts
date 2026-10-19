import { useMutation, useQueryClient } from '@tanstack/react-query'

import { bankStatementQueries } from '@/entities/bank-statement'
import { importBatchQueries } from '@/entities/import-batch'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

type Preview = components['schemas']['StatementExtractionPreview']
export function useStatementAI(statementId: string) {
  const cache = useQueryClient()
  const preview = useMutation({
    retry: false,
    mutationFn: async (expectedRevision: number) => {
      const { data, error } = await apiClient.POST(
        '/api/imports/statements/{statementId}/extraction-previews',
        {
          params: { path: { statementId } },
          body: { expectedRevision },
        },
      )
      if (!data)
        throw new Error(
          error?.code === 'ai_disabled'
            ? 'AI для выписок выключен. Доступна ручная подготовка строк.'
            : (error?.message ?? 'Не удалось подготовить распознавание.'),
        )
      return data
    },
  })
  const extract = useMutation({
    retry: false,
    mutationFn: async ({ preview: p, key }: { preview: Preview; key: string }) => {
      const { data: consent, error } = await apiClient.POST(
        '/api/imports/statements/{statementId}/extraction-consents',
        {
          params: { path: { statementId }, header: { 'Idempotency-Key': `consent:${key}` } },
          body: { expectedRevision: p.sourceRevision, previewId: p.id, previewHash: p.hash },
        },
      )
      if (!consent) throw new Error(error?.message ?? 'Согласие не сохранено.')
      const result = await apiClient.POST('/api/imports/statements/{statementId}/extractions', {
        params: { path: { statementId }, header: { 'Idempotency-Key': key } },
        body: {
          expectedRevision: p.sourceRevision,
          previewId: p.id,
          previewHash: p.hash,
          consentId: consent.id,
        },
      })
      if (!result.data)
        throw new Error(
          result.error?.message ??
            'Ответ не получен. Проверьте историю попыток; повтор с этим ключом не вызывает модель заново.',
        )
      return result.data
    },
    onSuccess: async () => {
      await Promise.all([
        cache.invalidateQueries({ queryKey: bankStatementQueries.all() }),
        cache.invalidateQueries({ queryKey: importBatchQueries.all() }),
      ])
    },
  })
  return { preview, extract }
}
