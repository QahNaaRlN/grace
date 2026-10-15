import { useMutation, useQueryClient } from '@tanstack/react-query'

import { accountQueries } from '@/entities/account'
import {
  cashPositionQueries,
  periodSummaryQueries,
  statisticsQueries,
} from '@/entities/analytics-report'
import { financialEventQueries } from '@/entities/financial-event'
import { receiptQueries } from '@/entities/receipt'
import { confirmReceipt, type ReceiptDraftPreview } from './receipt-drafts'

export type ReceiptDraftConfirmation = {
  draftId: string
  preview: ReceiptDraftPreview
  /** Idempotency key of the reviewed preview: a retry must not book a second expense. */
  key: string
}

/**
 * Confirms a receipt draft, which books or links a purchase and stores its receipt
 * (SPEC-0049). After a success balances, events, receipts and analytics are refreshed
 * before it resolves; a failed confirmation throws and leaves the cache alone.
 */
export function useConfirmReceiptDraft() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ draftId, preview, key }: ReceiptDraftConfirmation) =>
      confirmReceipt(draftId, preview, key),
    onSuccess: () =>
      Promise.all(
        [
          accountQueries.all(),
          financialEventQueries.all(),
          receiptQueries.all(),
          cashPositionQueries.all(),
          periodSummaryQueries.all(),
          statisticsQueries.all(),
        ].map((queryKey) => queryClient.invalidateQueries({ queryKey })),
      ),
  })
}
