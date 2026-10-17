import { useMutation, useQueryClient } from '@tanstack/react-query'

import { cashPositionQueries } from '@/entities/analytics-report'
import { loanQueries, obligationQueries } from '@/entities/obligation'
import { apiClient } from '@/shared/api'
import { createIdempotencyKey } from '@/shared/lib/idempotency'

export type Settlement = {
  obligationId: string
  paymentId: string
  financialEventId: string
  // SPEC-0053: a loan payment above the row remainder repays the loan early.
  prepayRemainder?: boolean
}

/** Links an actual payment to a scheduled one; refreshes obligations and the cash position. */
export function useSettlePayment() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ obligationId, paymentId, financialEventId, prepayRemainder }: Settlement) =>
      apiClient.POST(
        '/api/v1/obligations/{obligationId}/scheduled-payments/{paymentId}/settlements',
        {
          params: {
            path: { obligationId, paymentId },
            header: { 'Idempotency-Key': createIdempotencyKey() },
          },
          body: prepayRemainder ? { financialEventId, prepayRemainder } : { financialEventId },
        },
      ),
    onSuccess: async ({ error }) => {
      if (error) return
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: obligationQueries.all() }),
        queryClient.invalidateQueries({ queryKey: cashPositionQueries.all() }),
        queryClient.invalidateQueries({ queryKey: loanQueries.all() }),
      ])
    },
  })
}
