import { useMutation, useQueryClient } from '@tanstack/react-query'

import { cashPositionQueries } from '@/entities/analytics-report'
import { obligationQueries } from '@/entities/obligation'
import { apiClient } from '@/shared/api'
import type { paths } from '@/shared/api/generated/schema'
import { createIdempotencyKey } from '@/shared/lib/idempotency'

export type NewObligation =
  paths['/api/v1/obligations']['post']['requestBody']['content']['application/json']

/** Creates an obligation from a checked schedule; refreshes obligations and the cash position. */
export function useCreateObligation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: NewObligation) =>
      apiClient.POST('/api/v1/obligations', {
        params: { header: { 'Idempotency-Key': createIdempotencyKey() } },
        body,
      }),
    onSuccess: async ({ error }) => {
      if (error) return
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: obligationQueries.all() }),
        queryClient.invalidateQueries({ queryKey: cashPositionQueries.all() }),
      ])
    },
  })
}
