import { useMutation, useQueryClient } from '@tanstack/react-query'

import { importBatchQueries } from '@/entities/import-batch'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export type ImportDecision = components['schemas']['ImportDecision']

/** Saves a decision in the draft only; the financial journal changes after a confirmation. */
export function useSaveImportDecision(batchId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      recordId,
      revision,
      decision,
    }: {
      recordId: string
      revision: number
      decision: ImportDecision
    }) => {
      const { data, error } = await apiClient.PATCH(
        '/api/imports/batches/{batchId}/records/{recordId}',
        {
          params: {
            path: { batchId, recordId },
            header: { 'Idempotency-Key': crypto.randomUUID() },
          },
          body: { expectedRevision: revision, decision },
        },
      )
      if (!data)
        throw new Error(
          error?.code === 'revision_conflict'
            ? 'Черновик изменился. Обновите страницу.'
            : 'Не удалось сохранить решение. Проверьте поля.',
        )
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: importBatchQueries.all() }),
  })
}
