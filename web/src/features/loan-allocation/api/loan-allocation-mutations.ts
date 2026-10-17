import { useMutation, useQueryClient } from '@tanstack/react-query'

import { periodSummaryQueries, statisticsQueries } from '@/entities/analytics-report'
import { loanAllocationQueries, loanQueries } from '@/entities/obligation'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export type LoanAllocation = components['schemas']['LoanAllocation']
type PartsEdit = components['schemas']['LoanPartsEdit']
type Callbacks = { onSuccess?: () => void; onError?: (error: Error) => void }

// A split changes the loan debt and the statistics of debt payments.
function useAllocationRefresh() {
  const queryClient = useQueryClient()
  return (saved: LoanAllocation) => {
    queryClient.setQueryData(loanAllocationQueries.event(saved.financialEventId).queryKey, saved)
    for (const queryKey of [
      loanQueries.details(saved.obligationId).queryKey,
      statisticsQueries.all(),
      periodSummaryQueries.all(),
    ])
      void queryClient.invalidateQueries({ queryKey })
  }
}

/** SPEC-0053: the owner's split of the part that paid a schedule row. */
export function useSaveLoanAllocation(eventId: string, callbacks: Callbacks = {}) {
  const refresh = useAllocationRefresh()
  return useMutation({
    // `prepare` validates the form inside the mutation, so input errors reach onError too.
    mutationFn: async (prepare: () => PartsEdit) => {
      const { data, error } = await apiClient.PUT(
        '/api/v1/financial-events/{eventId}/loan-allocation',
        { params: { path: { eventId } }, body: prepare() },
      )
      if (error) throw new Error(error.message)
      return data!
    },
    onSuccess: (saved) => {
      refresh(saved)
      callbacks.onSuccess?.()
    },
    onError: callbacks.onError,
  })
}

/** Returns the schedule part to the split derived from the row. */
export function useResetLoanAllocation(eventId: string, callbacks: Callbacks = {}) {
  const refresh = useAllocationRefresh()
  return useMutation({
    mutationFn: async (revision: number) => {
      const { data, error } = await apiClient.DELETE(
        '/api/v1/financial-events/{eventId}/loan-allocation',
        { params: { path: { eventId }, query: { revision } } },
      )
      if (error) throw new Error(error.message)
      return data!
    },
    onSuccess: (saved) => {
      refresh(saved)
      callbacks.onSuccess?.()
    },
    onError: callbacks.onError,
  })
}
