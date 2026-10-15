import type { StatisticsBucket } from '@/shared/api'
import { APP_TIME_ZONE } from '@/shared/config'

export function defaultStatisticsPeriod() {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone: APP_TIME_ZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(new Date())
  const get = (key: string) => Number(parts.find((p) => p.type === key)?.value)
  const day = new Date(Date.UTC(get('year'), get('month') - 1, get('day')))
  return {
    from: new Date(Date.UTC(get('year'), get('month') - 6, 1)).toISOString().slice(0, 10),
    to: new Date(day.getTime() + 86400000).toISOString().slice(0, 10),
  }
}

export const dayText = (day: string) =>
  new Intl.DateTimeFormat('ru-RU', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  }).format(new Date(day + 'T00:00:00Z'))

export const rangeText = (bucket: StatisticsBucket) =>
  `${dayText(bucket.from)} — до ${dayText(bucket.to)}, не включая`

export const shortDate = (day: string) =>
  new Intl.DateTimeFormat('ru-RU', { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(
    new Date(day + 'T00:00:00Z'),
  )

export type RowKey = 'purchasesMinor' | 'incomeMinor' | 'outflowsMinor'
