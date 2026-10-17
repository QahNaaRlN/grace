import { useMutation, useQueryClient } from '@tanstack/react-query'

import {
  cashPositionQueries,
  periodSummaryQueries,
  statisticsQueries,
} from '@/entities/analytics-report'
import { loanAllocationQueries, loanQueries, obligationQueries } from '@/entities/obligation'
import { apiClient } from '@/shared/api'
import type { components, paths } from '@/shared/api/generated/schema'
import { createIdempotencyKey } from '@/shared/lib/idempotency'

type Draft = components['schemas']['PaymentScheduleDraft']
type LoanUpdate =
  paths['/api/v1/loans/{obligationId}']['patch']['requestBody']['content']['application/json']
type DraftUpdate =
  paths['/api/v1/loans/{obligationId}/schedule-drafts/{draftId}']['patch']['requestBody']['content']['application/json']
type Callbacks<T> = { onSuccess?: (data: T) => void; onError?: (error: Error) => void }
type Calculation = components['schemas']['PaymentSchedulePartsCalculation']
type SnapshotInput = components['schemas']['LoanDebtSnapshotInput']
type PrepaymentInput = components['schemas']['CreateLoanPrepayment']

function useLoanRefresh(obligationId: string) {
  const queryClient = useQueryClient()
  return () =>
    void queryClient.invalidateQueries({ queryKey: loanQueries.details(obligationId).queryKey })
}

/** Saves the loan agreement details and refreshes the loan card. */
export function useSaveLoanDetails(obligationId: string, callbacks: Callbacks<void> = {}) {
  const refresh = useLoanRefresh(obligationId)
  return useMutation({
    // `prepare` validates the form inside the mutation, so input errors reach onError too.
    mutationFn: async (prepare: () => LoanUpdate) => {
      const { error } = await apiClient.PATCH('/api/v1/loans/{obligationId}', {
        params: { path: { obligationId } },
        body: prepare(),
      })
      if (error) throw new Error(error.message)
    },
    onSuccess: () => {
      callbacks.onSuccess?.()
      refresh()
    },
    onError: callbacks.onError,
  })
}

/** Copies the current schedule into an editable draft. */
export function useCopyLoanSchedule(obligationId: string, callbacks: Callbacks<Draft> = {}) {
  const refresh = useLoanRefresh(obligationId)
  return useMutation({
    mutationFn: async () => {
      const { data, error } = await apiClient.POST('/api/v1/loans/{obligationId}/schedule-drafts', {
        params: { path: { obligationId } },
      })
      if (error) throw new Error(error.message)
      return data!
    },
    onSuccess: (draft) => {
      callbacks.onSuccess?.(draft)
      refresh()
    },
    onError: callbacks.onError,
  })
}

/** Uploads an original loan document as a new version. */
export function useUploadLoanDocument(obligationId: string, callbacks: Callbacks<void> = {}) {
  const refresh = useLoanRefresh(obligationId)
  return useMutation({
    mutationFn: async (file: File) => {
      if (file.size > 16 * 1024 * 1024) throw new Error('Файл должен быть не больше 16 МиБ.')
      const { error } = await apiClient.POST('/api/v1/loans/{obligationId}/documents', {
        params: { path: { obligationId } },
        body: { file: '' },
        bodySerializer: () => {
          const form = new FormData()
          form.append('file', file)
          return form
        },
      })
      if (error) throw new Error(error.message)
    },
    onSuccess: () => {
      callbacks.onSuccess?.()
      refresh()
    },
    onError: callbacks.onError,
  })
}

/** Saves a schedule draft; the current schedule and balances do not change. */
export function useSaveScheduleDraft(
  obligationId: string,
  draftId: string,
  callbacks: Callbacks<Draft> = {},
) {
  const refresh = useLoanRefresh(obligationId)
  return useMutation({
    mutationFn: async (prepare: () => DraftUpdate) => {
      const { data, error } = await apiClient.PATCH(
        '/api/v1/loans/{obligationId}/schedule-drafts/{draftId}',
        { params: { path: { obligationId, draftId } }, body: prepare() },
      )
      if (error) throw new Error(error.message)
      return data!
    },
    onSuccess: (saved) => {
      callbacks.onSuccess?.(saved)
      refresh()
    },
    onError: callbacks.onError,
  })
}

/** Applies a draft as the new schedule: obligations and the cash position change. */
export function useApplyLoanSchedule(obligationId: string, callbacks: Callbacks<void> = {}) {
  const queryClient = useQueryClient()
  const refresh = useLoanRefresh(obligationId)
  return useMutation({
    mutationFn: async (draft: Draft | null) => {
      if (!draft) return
      const { error } = await apiClient.POST(
        '/api/v1/loans/{obligationId}/schedule-drafts/{draftId}/apply',
        {
          params: { path: { obligationId, draftId: draft.id } },
          body: { revision: draft.revision, confirm: true },
        },
      )
      if (error) throw new Error(error.message)
    },
    onSuccess: () => {
      callbacks.onSuccess?.()
      refresh()
      void queryClient.invalidateQueries({ queryKey: obligationQueries.all() })
      void queryClient.invalidateQueries({ queryKey: cashPositionQueries.all() })
    },
    onError: callbacks.onError,
  })
}

/** SPEC-0053: Grace fills principal and interest of a saved draft by the repayment type. */
export function useCalculateScheduleParts(
  obligationId: string,
  callbacks: Callbacks<Calculation> = {},
) {
  const refresh = useLoanRefresh(obligationId)
  return useMutation({
    mutationFn: async (draft: Draft) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/loans/{obligationId}/schedule-drafts/{draftId}/calculate-parts',
        {
          params: { path: { obligationId, draftId: draft.id } },
          body: { revision: draft.revision },
        },
      )
      if (error) throw new Error(error.message)
      return data!
    },
    onSuccess: (result) => {
      callbacks.onSuccess?.(result)
      refresh()
    },
    onError: callbacks.onError,
  })
}

// Debt figures feed the loan card, the operation card and the statistics.
function useDebtRefresh(obligationId: string) {
  const queryClient = useQueryClient()
  const refresh = useLoanRefresh(obligationId)
  return () => {
    refresh()
    for (const queryKey of [
      loanAllocationQueries.all(),
      statisticsQueries.all(),
      periodSummaryQueries.all(),
    ])
      void queryClient.invalidateQueries({ queryKey })
  }
}

/** SPEC-0053: saves the outstanding debt the owner copied from the bank. */
export function useCreateLoanSnapshot(obligationId: string, callbacks: Callbacks<void> = {}) {
  const refresh = useDebtRefresh(obligationId)
  return useMutation({
    mutationFn: async (prepare: () => SnapshotInput) => {
      const { error } = await apiClient.POST('/api/v1/loans/{obligationId}/debt-snapshots', {
        params: {
          path: { obligationId },
          header: { 'Idempotency-Key': createIdempotencyKey() },
        },
        body: prepare(),
      })
      if (error) throw new Error(error.message)
    },
    onSuccess: () => {
      callbacks.onSuccess?.()
      refresh()
    },
    onError: callbacks.onError,
  })
}

/** SPEC-0053: links a debt payment to the loan as an early repayment. */
export function useCreateLoanPrepayment(obligationId: string, callbacks: Callbacks<void> = {}) {
  const queryClient = useQueryClient()
  const refresh = useDebtRefresh(obligationId)
  return useMutation({
    mutationFn: async (prepare: () => PrepaymentInput) => {
      const { error } = await apiClient.POST('/api/v1/loans/{obligationId}/prepayments', {
        params: {
          path: { obligationId },
          header: { 'Idempotency-Key': createIdempotencyKey() },
        },
        body: prepare(),
      })
      if (error) throw new Error(error.message)
    },
    onSuccess: () => {
      callbacks.onSuccess?.()
      refresh()
      void queryClient.invalidateQueries({ queryKey: obligationQueries.all() })
    },
    onError: callbacks.onError,
  })
}
