import { useMutation, useQueryClient } from '@tanstack/react-query'

import { accountQueries } from '@/entities/account'
import {
  cashPositionQueries,
  periodSummaryQueries,
  statisticsQueries,
} from '@/entities/analytics-report'
import { categoryQueries } from '@/entities/category'
import { financialEventQueries } from '@/entities/financial-event'
import { apiClient } from '@/shared/api'
import type { FinancialEvent } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export type Correction = components['schemas']['FinancialEventCorrection']

// A corrected or voided operation changes balances, analytics and category counts.
function useRefreshAfterChange() {
  const queryClient = useQueryClient()
  return async ({ data }: { data?: unknown }) => {
    if (!data) return
    await Promise.all(
      [
        financialEventQueries.all(),
        accountQueries.all(),
        cashPositionQueries.all(),
        statisticsQueries.all(),
        periodSummaryQueries.all(),
        categoryQueries.all(),
      ].map((queryKey) => queryClient.invalidateQueries({ queryKey })),
    )
  }
}

/** Sends only the changed fields with the revision; the key makes a retry safe. */
export function useCorrectOperation(event: FinancialEvent, idempotencyKey: string) {
  const onSuccess = useRefreshAfterChange()
  return useMutation({
    mutationFn: (body: Correction) =>
      apiClient.PATCH('/api/v1/financial-events/{eventId}', {
        params: { path: { eventId: event.id }, header: { 'Idempotency-Key': idempotencyKey } },
        body,
      }),
    onSuccess,
  })
}

/** Voids an operation with a reason; the record and its history stay. */
export function useVoidOperation(event: FinancialEvent, idempotencyKey: string) {
  const onSuccess = useRefreshAfterChange()
  return useMutation({
    mutationFn: (reason: string) =>
      apiClient.POST('/api/v1/financial-events/{eventId}/void', {
        params: { path: { eventId: event.id }, header: { 'Idempotency-Key': idempotencyKey } },
        body: { revision: event.revision, reason },
      }),
    onSuccess,
  })
}
