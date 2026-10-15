import { queryOptions } from '@tanstack/react-query'

import { apiClient } from '@/shared/api'
import type { CashPosition, PeriodSummary } from '@/shared/api'
import type { paths } from '@/shared/api/generated/schema'
import { dayStart } from '@/shared/lib/date'

type Granularity = NonNullable<
  paths['/api/v1/analytics/statistics']['get']['parameters']['query']
>['granularity']

export const statisticsQueries = {
  all: () => ['statistics'] as const,
  range: (from: string, to: string, granularity: Granularity) =>
    queryOptions({
      queryKey: [...statisticsQueries.all(), from, to, granularity] as const,
      queryFn: async () => {
        const { data, error } = await apiClient.GET('/api/v1/analytics/statistics', {
          params: { query: { from, to, granularity } },
        })
        if (!data || !Array.isArray(data.currencies))
          throw new Error(error?.message ?? 'Не удалось загрузить статистику')
        return data
      },
    }),
}

export const periodSummaryQueries = {
  all: () => ['period-summary'] as const,
  period: (fromDate: string, toDate: string) =>
    queryOptions<PeriodSummary>({
      queryKey: [...periodSummaryQueries.all(), fromDate, toDate] as const,
      queryFn: async () => {
        const { data, error } = await apiClient.GET('/api/v1/analytics/period-summary', {
          params: { query: { from: dayStart(fromDate), to: dayStart(toDate) } },
        })
        if (!data) throw new Error(error?.message ?? 'Не удалось загрузить сводку периода')
        return data
      },
    }),
}

export const cashPositionQueries = {
  all: () => ['cash-position'] as const,
  through: (through: string) =>
    queryOptions<CashPosition>({
      queryKey: [...cashPositionQueries.all(), through] as const,
      queryFn: async () => {
        const { data, error } = await apiClient.GET('/api/v1/analytics/cash-position', {
          params: { query: { through } },
        })
        if (!data) throw new Error(error?.message ?? 'Не удалось загрузить доступность денег')
        return data
      },
    }),
}
