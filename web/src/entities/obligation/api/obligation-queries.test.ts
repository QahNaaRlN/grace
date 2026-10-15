import { beforeEach, expect, it, vi } from 'vitest'

import { fetchQuery, startsWith } from '@/test/fetch-query'
import { loanQueries, obligationQueries, paymentReminderQueries } from './obligation-queries'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get } }))

beforeEach(() => get.mockReset())

it('keeps obligations, loans and reminders under separate roots', () => {
  expect(startsWith(obligationQueries.list('2026-10-15').queryKey, obligationQueries.all())).toBe(
    true,
  )
  expect(startsWith(loanQueries.details('o').queryKey, loanQueries.all())).toBe(true)
  expect(paymentReminderQueries.list().queryKey).toEqual(paymentReminderQueries.all())
  expect(startsWith(loanQueries.all(), obligationQueries.all())).toBe(false)
})

it('reads obligations as of a day and shows the loan error from the server', async () => {
  get.mockResolvedValueOnce({ data: { items: [] } })
  await fetchQuery(obligationQueries.list('2026-10-15'))
  expect(get).toHaveBeenCalledWith('/api/v1/obligations', {
    params: { query: { asOf: '2026-10-15' } },
  })
  get.mockResolvedValueOnce({ error: { message: 'Кредит не найден' } })
  await expect(fetchQuery(loanQueries.details('o'))).rejects.toThrow('Кредит не найден')
})
