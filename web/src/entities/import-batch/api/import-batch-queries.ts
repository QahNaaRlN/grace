import { infiniteQueryOptions, queryOptions } from '@tanstack/react-query'

import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

/** Filter and page of the record review; the whole object is part of the key. */
export type ImportRecordsView = {
  status: components['schemas']['ImportRecordStatus'] | ''
  domain: components['schemas']['ImportRecordDomain']
  q: string
  sort: 'row' | 'date' | '-date' | 'amount' | '-amount'
  cursor: string
  limit: 50 | 100
}

export const importBatchQueries = {
  all: () => ['imports'] as const,
  batches: () =>
    infiniteQueryOptions({
      queryKey: [...importBatchQueries.all(), 'batches'] as const,
      initialPageParam: '',
      queryFn: async ({ pageParam }) => {
        const { data } = await apiClient.GET('/api/imports/batches', {
          params: { query: { cursor: pageParam || undefined, limit: 20 } },
        })
        if (!data) throw new Error('Не удалось загрузить историю импортов.')
        return data
      },
      getNextPageParam: (last) => last.nextCursor || undefined,
    }),
  batch: (batchId: string) =>
    queryOptions({
      queryKey: [...importBatchQueries.all(), 'batch', batchId] as const,
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/imports/batches/{batchId}', {
          params: { path: { batchId } },
        })
        if (!data) throw new Error('Черновик недоступен.')
        return data
      },
    }),
  records: (batchId: string, view: ImportRecordsView) =>
    queryOptions({
      queryKey: [...importBatchQueries.all(), 'records', batchId, view] as const,
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/imports/batches/{batchId}/records', {
          params: {
            path: { batchId },
            query: {
              cursor: view.cursor || undefined,
              limit: view.limit,
              status: view.status || undefined,
              domain: view.domain,
              q: view.q || undefined,
              sort: view.sort,
            },
          },
        })
        if (!data) throw new Error('Не удалось загрузить строки.')
        return data
      },
    }),
  record: (batchId: string, recordId: string) =>
    queryOptions({
      queryKey: [...importBatchQueries.all(), 'record', batchId, recordId] as const,
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/imports/batches/{batchId}/records/{recordId}', {
          params: { path: { batchId, recordId } },
        })
        if (!data) throw new Error('Строка недоступна.')
        return data
      },
    }),
  receipt: (batchId: string, receiptId: string) =>
    queryOptions({
      queryKey: [...importBatchQueries.all(), 'receipt', batchId, receiptId] as const,
      queryFn: async () => {
        const { data } = await apiClient.GET(
          '/api/imports/batches/{batchId}/confirmations/{confirmationId}',
          { params: { path: { batchId, confirmationId: receiptId } } },
        )
        if (!data) throw new Error('Подтверждение недоступно.')
        return data
      },
    }),
  months: () => [...importBatchQueries.all(), 'month'] as const,
  month: (batchId: string, month: string) =>
    queryOptions({
      queryKey: [...importBatchQueries.months(), batchId, month] as const,
      queryFn: async () => {
        const { data } = await apiClient.GET('/api/imports/batches/{batchId}/month', {
          params: { path: { batchId }, query: { month } },
        })
        if (!data) throw new Error('Не удалось получить сверку месяца.')
        return data
      },
    }),
}
