import { useMutation, useQueryClient } from '@tanstack/react-query'

import { accountQueries } from '@/entities/account'
import { cashPositionQueries, periodSummaryQueries } from '@/entities/analytics-report'
import { financialEventQueries } from '@/entities/financial-event'
import { apiClient } from '@/shared/api'

export type CardRepayment = {
  fromAccountId: string
  cardAccountId: string
  amountMinor: number
  occurredAt: string
  /** Idempotency key of the reviewed repayment: a retry must not pay twice. */
  key: string
}

/**
 * SPEC-0051: repaying a card is a transfer between own accounts. Money leaves the
 * paying account, the card debt goes down, and nothing is counted as spending again.
 * The card status lives under the accounts root and is refreshed with it.
 */
export function useRepayCreditCard() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({
      fromAccountId,
      cardAccountId,
      amountMinor,
      occurredAt,
      key,
    }: CardRepayment) => ({
      error: (
        await apiClient.POST('/api/v1/financial-events/transfers', {
          params: { header: { 'Idempotency-Key': key } },
          body: {
            amountMinor,
            description: 'Погашение кредитной карты',
            occurredAt,
            sourceAccountId: fromAccountId,
            destinationAccountId: cardAccountId,
          },
        })
      ).error,
    }),
    onSuccess: async ({ error }) => {
      if (error) return
      await Promise.all(
        [
          accountQueries.all(),
          financialEventQueries.all(),
          cashPositionQueries.all(),
          periodSummaryQueries.all(),
        ].map((queryKey) => queryClient.invalidateQueries({ queryKey })),
      )
    },
  })
}
