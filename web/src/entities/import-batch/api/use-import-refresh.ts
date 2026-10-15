import { useQueryClient } from '@tanstack/react-query'

import { importBatchQueries } from './import-batch-queries'

/** «Повторить» after a failed import read: reloads drafts or the month reports. */
export function useImportRefresh() {
  const queryClient = useQueryClient()
  return {
    all: () => queryClient.invalidateQueries({ queryKey: importBatchQueries.all() }),
    months: () => queryClient.invalidateQueries({ queryKey: importBatchQueries.months() }),
  }
}
