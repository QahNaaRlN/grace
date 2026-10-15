import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import type { Mock } from 'vitest'

export const testAccount = {
  id: 'a',
  name: 'Счёт',
  type: 'debit',
  currency: 'RUB',
  balanceMinor: 100000,
  balanceReconciled: true,
  revision: 1,
  hasOperations: false,
  lastReconciliation: null,
  lastImportConfirmedAt: null,
  createdAt: '2026-10-01T00:00:00Z',
  paymentInstruments: [],
}

/**
 * Answers the ledger reads of a money page: one account, empty lists and an empty cash
 * position. `extra` overrides single paths.
 */
export function serveLedger(get: Mock, extra: Record<string, unknown> = {}) {
  get.mockImplementation(async (path: string) =>
    path in extra
      ? { data: extra[path] }
      : path === '/api/v1/accounts'
        ? { data: { items: [testAccount] } }
        : path === '/api/v1/analytics/cash-position'
          ? { data: { asOf: '2026-10-15', through: '2026-10-31', currencies: [] } }
          : path === '/api/v1/analytics/statistics'
            ? { data: { currencies: [] } }
            : { data: { items: [] } },
  )
}

/** Renders a page at its URL with a fresh query client. */
export function renderPage(page: ReactNode, url: string) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        {page}
      </QueryClientProvider>
    </MemoryRouter>,
  )
}
