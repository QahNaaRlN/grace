import { useMutation, useQueryClient } from '@tanstack/react-query'

import { importBatchQueries } from '@/entities/import-batch'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export type DraftUpload = {
  file: File
  kind: 'csv' | 'json'
  sourceId: string
  /** One key per chosen file: «Повторить» resends the same bytes with the same key. */
  idempotencyKey: string
}

async function upload({ file, kind, sourceId, idempotencyKey }: DraftUpload) {
  const header = { 'Idempotency-Key': idempotencyKey }
  if (kind === 'csv') {
    const { data, error } = await apiClient.POST('/api/imports/csv', {
      body: '',
      bodySerializer: () => file,
      headers: { 'Content-Type': 'text/csv' },
      params: { query: { sourceId, fileName: file.name }, header },
    })
    if (!data)
      throw new Error(
        error?.code === 'invalid_csv'
          ? 'Не удалось прочитать CSV. Экспортируйте полный лист «Операции» в UTF-8 с исходными строками и столбцами A:O.'
          : (error?.message ?? 'CSV не сохранён.'),
      )
    return data.id
  }
  // Pass original bytes: parsing/re-serializing JSON would round source decimals.
  const { data, error } = await apiClient.POST('/api/imports/batches', {
    body: {} as components['schemas']['ImportSnapshot'],
    bodySerializer: () => file,
    params: { header, query: { fileName: file.name } },
  })
  if (!data)
    throw new Error(error?.message ?? 'Снимок не сохранён. Проверьте формат и размер файла.')
  return data.id
}

/** Saves a CSV or JSON file as an import draft; resolves with the batch id after a refresh. */
export function useCreateImportDraft() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: upload,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: importBatchQueries.all() }),
  })
}
