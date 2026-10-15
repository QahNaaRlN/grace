import { beforeEach, expect, it, vi } from 'vitest'

import { fetchQuery, startsWith } from '@/test/fetch-query'
import {
  cashPositionQueries,
  periodSummaryQueries,
  statisticsQueries,
} from './analytics-report-queries'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))

beforeEach(() => get.mockReset())

it('keeps each report under its own root, so a refresh asks only for that report', () => {
  expect(
    startsWith(
      statisticsQueries.range('2026-01-01', '2026-02-01', 'month').queryKey,
      statisticsQueries.all(),
    ),
  ).toBe(true)
  expect(
    startsWith(
      periodSummaryQueries.period('2026-01-01', '2026-02-01').queryKey,
      periodSummaryQueries.all(),
    ),
  ).toBe(true)
  expect(
    startsWith(cashPositionQueries.through('2026-10-31').queryKey, cashPositionQueries.all()),
  ).toBe(true)
  expect(cashPositionQueries.all()).not.toEqual(statisticsQueries.all())
})

it('asks for the cash position through the chosen date', async () => {
  get.mockResolvedValue({ data: { asOf: '2026-10-15', through: '2026-10-31', currencies: [] } })
  await expect(fetchQuery(cashPositionQueries.through('2026-10-31'))).resolves.toMatchObject({
    currencies: [],
  })
  expect(JSON.stringify(get.mock.calls[0])).toContain('2026-10-31')
})
