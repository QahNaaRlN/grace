import { queryOptions } from '@tanstack/react-query'
import { apiClient } from '@/shared/api'
import type { components } from '@/shared/api/generated/schema'

export type InstallmentPlan = components['schemas']['InstallmentPlan']
export type InstallmentDraft = components['schemas']['InstallmentScheduleDraft']
export type InstallmentAllocation = components['schemas']['InstallmentAllocationView']
export const installmentQueries = {
  all: () => ['obligations', 'installments'] as const,
  list: () =>
    queryOptions({
      queryKey: [...installmentQueries.all(), 'list'],
      queryFn: async () => {
        const { data, error } = await apiClient.GET('/api/v1/installment-plans')
        if (!data) throw new Error(error?.message ?? 'Рассрочки недоступны.')
        return data.items
      },
    }),
  allocation: (eventId: string) =>
    queryOptions({
      queryKey: [...installmentQueries.all(), 'allocation', eventId],
      queryFn: async () => {
        const { data, error } = await apiClient.GET(
          '/api/v1/financial-events/{eventId}/installment-allocation',
          { params: { path: { eventId } } },
        )
        if (!data) throw new Error(error?.message ?? 'Распределение недоступно.')
        return data
      },
    }),
}
