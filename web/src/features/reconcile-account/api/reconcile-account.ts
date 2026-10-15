import { useMutation, useQueryClient } from '@tanstack/react-query'

import { accountQueries } from '@/entities/account'
import {
  cashPositionQueries,
  periodSummaryQueries,
  statisticsQueries,
} from '@/entities/analytics-report'
import { financialEventQueries } from '@/entities/financial-event'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export type ReconciliationPreview = components['schemas']['ReconciliationPreview']
type Stated = { asOf: string; statedMinor: number }

/** Grace computes its balance at the stated moment; nothing is written. */
export function previewReconciliation(accountId: string, body: Stated) {
  return apiClient.POST('/api/v1/accounts/{accountId}/reconciliations/preview', {
    params: { path: { accountId } },
    body,
  })
}

/**
 * Confirms a previewed reconciliation. A balance feeds lists, the overview and analytics,
 * so all of them are refreshed after a success.
 */
export function useConfirmReconciliation(accountId: string, idempotencyKey: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (preview: ReconciliationPreview) =>
      apiClient.POST('/api/v1/accounts/{accountId}/reconciliations', {
        params: { path: { accountId }, header: { 'Idempotency-Key': idempotencyKey } },
        body: {
          asOf: preview.asOf,
          statedMinor: preview.statedMinor,
          previewHash: preview.previewHash,
        },
      }),
    onSuccess: async ({ data }) => {
      if (!data) return
      await Promise.all(
        [
          accountQueries.all(),
          financialEventQueries.all(),
          cashPositionQueries.all(),
          statisticsQueries.all(),
          periodSummaryQueries.all(),
        ].map((queryKey) => queryClient.invalidateQueries({ queryKey })),
      )
    },
  })
}
