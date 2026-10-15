import { useMutation, useQueryClient } from '@tanstack/react-query'

import { accountQueries } from '@/entities/account'
import { cashPositionQueries, periodSummaryQueries } from '@/entities/analytics-report'
import { financialEventQueries } from '@/entities/financial-event'
import { obligationQueries } from '@/entities/obligation'
import { apiClient } from '@/shared/api'

type Common = { amountMinor: number; merchant: string; description: string; occurredAt: string }

export type NewFinancialEvent = {
  kind: 'purchase' | 'income' | 'transfer' | 'refund' | 'debt_payment'
  /** Idempotency key of the reviewed operation: a retry must not create a second one. */
  key: string
  common: Common
  accountId: string
  categoryId: string | null
  originalEventId: string
  destinationAccountId: string
}

async function post({ kind, key, common, ...links }: NewFinancialEvent) {
  const params = { header: { 'Idempotency-Key': key } }
  const booked = { ...common, accountId: links.accountId, categoryId: links.categoryId }
  if (kind === 'purchase')
    return apiClient.POST('/api/v1/financial-events/purchases', { params, body: booked })
  if (kind === 'income')
    return apiClient.POST('/api/v1/financial-events/incomes', { params, body: booked })
  if (kind === 'debt_payment')
    return apiClient.POST('/api/v1/financial-events/debt-payments', { params, body: booked })
  if (kind === 'refund')
    return apiClient.POST('/api/v1/financial-events/refunds', {
      params,
      body: { ...common, originalEventId: links.originalEventId },
    })
  return apiClient.POST('/api/v1/financial-events/transfers', {
    params,
    body: {
      amountMinor: common.amountMinor,
      description: common.description,
      occurredAt: common.occurredAt,
      sourceAccountId: links.accountId,
      destinationAccountId: links.destinationAccountId,
    },
  })
}

/**
 * Records a confirmed operation. The result keeps the API error for the form; after a
 * success balances, events, analytics and obligations are refreshed before it resolves.
 */
export function useCreateFinancialEvent() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (event: NewFinancialEvent) => ({ error: (await post(event)).error }),
    onSuccess: async ({ error }) => {
      if (error) return
      await Promise.all(
        [
          accountQueries.all(),
          financialEventQueries.all(),
          cashPositionQueries.all(),
          periodSummaryQueries.all(),
          obligationQueries.all(),
        ].map((queryKey) => queryClient.invalidateQueries({ queryKey })),
      )
    },
  })
}
