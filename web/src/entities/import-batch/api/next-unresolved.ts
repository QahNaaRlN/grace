import { apiClient } from '@/shared/api'

/** SPEC-0044: the next unresolved row in sheet order after `after`, or null when none is left. */
export async function nextUnresolvedRecord(batchId: string, after?: string) {
  const { data } = await apiClient.GET('/api/imports/batches/{batchId}/records/next-unresolved', {
    params: { path: { batchId }, query: { after: after || undefined } },
  })
  if (!data) throw new Error('Не удалось найти следующую строку.')
  return data.record ?? null
}
