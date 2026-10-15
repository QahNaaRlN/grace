import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { LedgerScreen } from './LedgerScreen'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/shared/api/client', () => ({ apiClient: { GET: get, POST: vi.fn() } }))

const account = {
  id: 'a',
  name: 'Счёт',
  type: 'debit',
  currency: 'RUB',
  balanceMinor: 1000,
  balanceReconciled: true,
  revision: 1,
  hasOperations: false,
  lastReconciliation: null,
  lastImportConfirmedAt: null,
  createdAt: '2026-10-01T00:00:00Z',
}
const ledgerPaths = [
  '/api/v1/accounts',
  '/api/v1/categories',
  '/api/v1/financial-events',
  '/api/v1/receipts',
  '/api/v1/obligations',
  '/api/v1/analytics/cash-position',
]

function serve({ accounts = [account], failing }: { accounts?: unknown[]; failing?: string } = {}) {
  get.mockImplementation(async (path: string) =>
    path === failing
      ? { error: { message: 'Счета временно недоступны' } }
      : path === '/api/v1/accounts'
        ? { data: { items: accounts } }
        : path === '/api/v1/analytics/cash-position'
          ? { data: { asOf: '2026-10-15', through: '2026-10-31', currencies: [] } }
          : { data: { items: [] } },
  )
}

function show(props: Partial<Parameters<typeof LedgerScreen>[0]> = {}) {
  render(
    <MemoryRouter>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <LedgerScreen title="Счета и карты" {...props}>
          {({ accounts }) => <p>Счетов: {accounts.data?.length}</p>}
        </LedgerScreen>
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

beforeEach(() => get.mockReset())

describe('SPEC-0048 ledger screen', () => {
  it('waits for the ledger and then renders the page content', async () => {
    serve()
    show({ accountAction: 'always', operationAction: true })
    expect(await screen.findByText('Счетов: 1')).toBeVisible()
    expect(screen.getByRole('button', { name: 'Добавить счёт' })).toBeVisible()
    expect(screen.getByRole('button', { name: 'Добавить операцию' })).toBeEnabled()
  })

  it('invites to add the first account instead of rendering an empty ledger', async () => {
    serve({ accounts: [] })
    show({ accountAction: 'when-empty', operationAction: true })
    expect(await screen.findByRole('heading', { name: 'Начните с первого счёта' })).toBeVisible()
    // The header action and the invitation both open the account dialog.
    expect(screen.getAllByRole('button', { name: 'Добавить первый счёт' })).toHaveLength(2)
    expect(screen.queryByRole('button', { name: 'Добавить операцию' })).not.toBeInTheDocument()
    expect(screen.queryByText(/Счетов:/)).not.toBeInTheDocument()
  })

  it('never offers an account where accountAction is never (SPEC-0036)', async () => {
    serve({ accounts: [] })
    show({ gated: false })
    expect(await screen.findByText('Счетов: 0')).toBeVisible()
    expect(screen.queryByRole('button', { name: /Добавить .*счёт/ })).not.toBeInTheDocument()
  })

  it('reports a partial load and retries the ledger queries', async () => {
    serve({ failing: '/api/v1/accounts' })
    show()
    expect(await screen.findByText('Обзор загружен частично')).toBeVisible()
    expect(screen.getByText('Часть данных недоступна')).toBeVisible()
    serve()
    get.mockClear()
    await userEvent.click(screen.getByRole('button', { name: 'Повторить' }))
    await waitFor(() =>
      expect(get.mock.calls.map(([path]) => path).sort()).toEqual([...ledgerPaths].sort()),
    )
    expect(await screen.findByText('Счетов: 1')).toBeVisible()
  })
})
