import { expect, it, vi } from 'vitest'

import type { FinancialEvent } from '@/shared/api'
import { startsWith } from '@/test/fetch-query'
import { financialEventQueries } from './financial-event-queries'

vi.mock('@/shared/api/client', () => ({ apiClient: { GET: vi.fn() } }))

it('keeps the list, pages, history and pickers under financial-events', () => {
  const event = { id: 'e', revision: 2 } as FinancialEvent
  for (const key of [
    financialEventQueries.list().queryKey,
    financialEventQueries.history(event).queryKey,
    financialEventQueries.importOptions().queryKey,
    financialEventQueries.subscriptionPicker().queryKey,
  ])
    expect(startsWith(key, financialEventQueries.all())).toBe(true)
})

it('gives each revision of an operation its own history key', () => {
  const first = financialEventQueries.history({ id: 'e', revision: 2 } as FinancialEvent).queryKey
  const next = financialEventQueries.history({ id: 'e', revision: 3 } as FinancialEvent).queryKey
  expect(first).not.toEqual(next)
})
