import { useMutation, useQueryClient } from '@tanstack/react-query'

import { receiptQueries } from '@/entities/receipt'
import { apiClient } from '@/shared/api'
import type { paths } from '@/shared/api/generated/schema'
import { createIdempotencyKey } from '@/shared/lib/idempotency'

export type NewReceipt =
  paths['/api/v1/receipts']['post']['requestBody']['content']['application/json']

/** Adds receipt lines to a purchase; refreshes receipts after a success. */
export function useCreateReceipt() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (body: NewReceipt) =>
      apiClient.POST('/api/v1/receipts', {
        params: { header: { 'Idempotency-Key': createIdempotencyKey() } },
        body,
      }),
    onSuccess: async ({ error }) => {
      if (error) return
      await queryClient.invalidateQueries({ queryKey: receiptQueries.all() })
    },
  })
}
