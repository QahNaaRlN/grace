import { useMutation, useQueryClient } from '@tanstack/react-query'
import { accountQueries } from '@/entities/account'
import {
  cashPositionQueries,
  periodSummaryQueries,
  statisticsQueries,
} from '@/entities/analytics-report'
import { financialEventQueries } from '@/entities/financial-event'
import { installmentQueries } from '@/entities/installment'
import { obligationQueries, paymentReminderQueries } from '@/entities/obligation'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'
import { createIdempotencyKey } from '@/shared/lib/idempotency'

type Schemas = components['schemas']
export type InstallmentInput = Schemas['CreateInstallmentPlan']
export function useInstallmentMutation<T, R>(mutationFn: (input: T) => Promise<R>) {
  const client = useQueryClient()
  return useMutation({
    mutationFn,
    onSuccess: () => {
      for (const queryKey of [
        accountQueries.all(),
        obligationQueries.all(),
        installmentQueries.all(),
        financialEventQueries.all(),
        cashPositionQueries.all(),
        periodSummaryQueries.all(),
        statisticsQueries.all(),
        paymentReminderQueries.all(),
      ]) {
        void client.invalidateQueries({ queryKey })
      }
    },
  })
}
export function useCreateInstallment() {
  return useInstallmentMutation(async ({ body, key }: { body: InstallmentInput; key: string }) => {
    const { data, error } = await apiClient.POST('/api/v1/installment-plans', {
      params: { header: { 'Idempotency-Key': key } },
      body,
    })
    if (!data) throw new Error(error?.message ?? 'Не удалось сохранить рассрочку.')
    return data
  })
}
export function useAllocateInstallment(eventId: string) {
  return useInstallmentMutation(
    async ({ body, key }: { body: Schemas['InstallmentAllocationInput']; key: string }) => {
      const { data, error } = await apiClient.PUT(
        '/api/v1/financial-events/{eventId}/installment-allocation',
        { params: { path: { eventId }, header: { 'Idempotency-Key': key } }, body },
      )
      if (!data) throw new Error(error?.message ?? 'Распределение не сохранено.')
      return data
    },
  )
}
export function useInstallmentTransfer() {
  return useInstallmentMutation(
    async (body: {
      sourceAccountId: string
      destinationAccountId: string
      amountMinor: number
      occurredAt: string
      description: string
      key: string
    }) => {
      const { key, ...payload } = body
      const { data, error } = await apiClient.POST('/api/v1/financial-events/transfers', {
        params: { header: { 'Idempotency-Key': key } },
        body: payload,
      })
      if (!data) throw new Error(error?.message ?? 'Перевод не сохранён.')
      return data
    },
  )
}
export function useInstallmentDraft(planId: string) {
  return useInstallmentMutation(async (revision: number) => {
    const { data, error } = await apiClient.POST(
      '/api/v1/installment-plans/{planId}/schedule-drafts',
      {
        params: { path: { planId }, header: { 'Idempotency-Key': createIdempotencyKey() } },
        body: { revision },
      },
    )
    if (!data) throw new Error(error?.message ?? 'Копия графика недоступна.')
    return data
  })
}
export function useSaveInstallmentDraft(planId: string) {
  return useInstallmentMutation(
    async ({
      draftId,
      revision,
      payments,
      key,
    }: {
      draftId: string
      revision: number
      payments: Schemas['InstallmentRow'][]
      key: string
    }) => {
      const { data, error } = await apiClient.PUT(
        '/api/v1/installment-plans/{planId}/schedule-drafts/{draftId}',
        {
          params: { path: { planId, draftId }, header: { 'Idempotency-Key': key } },
          body: { revision, payments },
        },
      )
      if (!data) throw new Error(error?.message ?? 'График не сохранён.')
      return data
    },
  )
}
export function useApplyInstallmentDraft(planId: string) {
  return useInstallmentMutation(
    async ({
      draftId,
      revision,
      planRevision,
      key,
    }: {
      draftId: string
      revision: number
      planRevision: number
      key: string
    }) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/installment-plans/{planId}/schedule-drafts/{draftId}/apply',
        {
          params: { path: { planId, draftId }, header: { 'Idempotency-Key': key } },
          body: { revision, planRevision },
        },
      )
      if (!data) throw new Error(error?.message ?? 'График изменился. Обновите копию.')
      return data
    },
  )
}
