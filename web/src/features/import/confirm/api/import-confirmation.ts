import { useMutation, useQueryClient } from '@tanstack/react-query'

import { accountQueries } from '@/entities/account'
import { cashPositionQueries, periodSummaryQueries } from '@/entities/analytics-report'
import { financialEventQueries } from '@/entities/financial-event'
import { importBatchQueries } from '@/entities/import-batch'
import { apiClient } from '@/shared/api'
import type { components, paths } from '@/shared/api/generated/schema'

export type ImportPreview = components['schemas']['ImportPreview']
export type ImportReceipt = components['schemas']['ImportReceipt']
type PreviewRequest =
  paths['/api/imports/batches/{batchId}/previews']['post']['requestBody']['content']['application/json']

const staleCodes = ['preview_expired', 'preview_changed', 'revision_conflict']

/** Asks Go what a confirmation would write; nothing is written yet. */
export async function previewImport(batchId: string, body: PreviewRequest) {
  const { data, error } = await apiClient.POST('/api/imports/batches/{batchId}/previews', {
    params: { path: { batchId }, header: { 'Idempotency-Key': crypto.randomUUID() } },
    body,
  })
  if (!data)
    throw new Error(
      error?.code === 'revision_conflict'
        ? 'Черновик изменился. Обновите страницу.'
        : error?.code === 'selection_too_large'
          ? 'За раз можно проверить не больше 1 000 строк. Сузьте фильтр и повторите.'
          : 'Не удалось подготовить предварительный просмотр.',
    )
  return data
}

/**
 * Confirms a preview. `onConfirmed` runs before the refresh of drafts, balances, events and
 * analytics; `onStale` runs before the drafts are reloaded when the preview is out of date.
 */
export function useConfirmImport(
  batchId: string,
  callbacks: { onConfirmed: (receipt: ImportReceipt) => void; onStale: () => void },
) {
  const queryClient = useQueryClient()
  const refresh = (keys: readonly (readonly unknown[])[]) =>
    Promise.all(keys.map((queryKey) => queryClient.invalidateQueries({ queryKey })))
  return useMutation({
    mutationFn: async ({ preview, key }: { preview: ImportPreview; key: string }) => {
      const { data, error } = await apiClient.POST('/api/imports/batches/{batchId}/confirmations', {
        params: { path: { batchId }, header: { 'Idempotency-Key': key } },
        body: { previewId: preview.id, previewHash: preview.hash },
      })
      return { data, stale: !data && staleCodes.includes(error?.code ?? '') }
    },
    onSuccess: async ({ data, stale }) => {
      if (data) {
        callbacks.onConfirmed(data)
        await refresh([
          importBatchQueries.all(),
          accountQueries.all(),
          financialEventQueries.all(),
          periodSummaryQueries.all(),
          cashPositionQueries.all(),
        ])
      } else if (stale) {
        callbacks.onStale()
        await refresh([importBatchQueries.all()])
      }
    },
  })
}
