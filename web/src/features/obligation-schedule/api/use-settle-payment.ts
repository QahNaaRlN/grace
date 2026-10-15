import { useMutation, useQueryClient } from '@tanstack/react-query'

import { cashPositionQueries } from '@/entities/analytics-report'
import { obligationQueries } from '@/entities/obligation'
import { apiClient } from '@/shared/api'
import { createIdempotencyKey } from '@/shared/lib/idempotency'

export type Settlement = { obligationId: string; paymentId: string; financialEventId: string }

/** Links an actual payment to a scheduled one; refreshes obligations and the cash position. */
export function useSettlePayment() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ obligationId, paymentId, financialEventId }: Settlement) =>
      apiClient.POST(
        '/api/v1/obligations/{obligationId}/scheduled-payments/{paymentId}/settlements',
        {
          params: {
            path: { obligationId, paymentId },
            header: { 'Idempotency-Key': createIdempotencyKey() },
          },
          body: { financialEventId },
        },
      ),
    onSuccess: async ({ error }) => {
      if (error) return
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: obligationQueries.all() }),
        queryClient.invalidateQueries({ queryKey: cashPositionQueries.all() }),
      ])
    },
  })
}
